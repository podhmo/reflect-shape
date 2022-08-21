package metadata

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/podhmo/commentof"
	"github.com/podhmo/commentof/collect"
	"golang.org/x/tools/go/packages"
)

// TODO: cache

// ErrNotFound is the error metadata is not found.
var ErrNotFound = fmt.Errorf("not found")

type Lookup struct {
	Fset *token.FileSet

	cache     map[string]*packageRef
	mu        sync.Mutex
	buildinfo *debug.BuildInfo

	IncludeGoTestFiles bool
}

type packageRef struct {
	*collect.Package

	fullset bool
	err     error
}

func NewLookup(fset *token.FileSet) *Lookup {
	return &Lookup{
		Fset:               fset,
		IncludeGoTestFiles: false,
		cache:              map[string]*packageRef{},
	}
}

type Func struct {
	pc  uintptr
	Raw *collect.Func
}

func (m *Func) Fullname() string {
	return runtime.FuncForPC(m.pc).Name()
}

func (m *Func) Name() string {
	return m.Raw.Name
}

func (m *Func) Doc() string {
	return strings.TrimSpace(m.Raw.Doc)
}

func (m *Func) Args() []string {
	names := make([]string, len(m.Raw.ParamNames))
	for i, id := range m.Raw.ParamNames {
		names[i] = m.Raw.Params[id].Name
	}
	return names
}

func (m *Func) Returns() []string {
	names := make([]string, len(m.Raw.ReturnNames))
	for i, id := range m.Raw.ReturnNames {
		names[i] = m.Raw.Returns[id].Name
	}
	return names
}

func (l *Lookup) LookupFromFunc(fn interface{}) (*Func, error) {
	pc := reflect.ValueOf(fn).Pointer()
	rfunc := runtime.FuncForPC(pc)
	if rfunc == nil {
		return nil, fmt.Errorf("cannot find runtime.Func")
	}

	filename, _ := rfunc.FileLine(rfunc.Entry())
	fullname := rfunc.Name()
	parts := strings.Split(fullname, ".")
	pkgpath := strings.Join(parts[:len(parts)-1], ".")
	funcname := parts[len(parts)-1]

	l.mu.Lock()
	p0, ok := l.cache[pkgpath]
	if ok && p0.fullset {
		defer l.mu.Unlock()
		if p0.err != nil {
			return nil, p0.err
		}
		result, ok := p0.Functions[funcname]
		if !ok {
			return nil, fmt.Errorf("lookup metadata of %s is failed %w", fullname, ErrNotFound)
		}
		return &Func{Raw: result}, nil
	}
	defer l.mu.Unlock()

	if ok && p0 != nil {
		for _, visitedFile := range p0.FileNames {
			if visitedFile == filename {
				f, ok := p0.Files[funcname]
				if !ok {
					break
				}
				result, ok := f.Functions[funcname]
				if !ok {
					return nil, fmt.Errorf("lookup metadata of %T is failed %w", funcname, ErrNotFound)
				}
				return &Func{pc: pc, Raw: result}, nil
			}
		}
	}

	tree, err := parser.ParseFile(l.Fset, filename, nil, parser.ParseComments)
	if tree == nil {
		l.cache[pkgpath] = &packageRef{fullset: false, err: err}
		return nil, err
	}

	p, err := commentof.File(l.Fset, tree, func(b *collect.PackageBuilder) {
		if p0 != nil {
			b.Package = p0.Package // merge Package
		}
	})
	if !ok && p != nil {
		l.cache[pkgpath] = &packageRef{fullset: false, Package: p}
	}
	if err != nil {
		return nil, err
	}

	result, ok := p.Functions[funcname]
	if !ok {
		return nil, fmt.Errorf("lookup metadata of %T is failed %w", funcname, ErrNotFound)
	}
	return &Func{pc: pc, Raw: result}, nil
}

type Struct struct {
	Raw *collect.Object
}

func (s *Struct) Name() string {
	return s.Raw.Name
}

func (s *Struct) Doc() string {
	doc := s.Raw.Doc
	if doc == "" {
		doc = s.Raw.Comment
	}
	return strings.TrimSpace(doc)
}

func (s *Struct) FieldComments() map[string]string {
	comments := make(map[string]string, len(s.Raw.Fields))
	for _, f := range s.Raw.Fields {
		doc := f.Doc
		if doc == "" {
			doc = f.Comment
		}
		comments[f.Name] = strings.TrimSpace(doc)
	}
	return comments
}

func (l *Lookup) LookupFromStruct(ob interface{}) (*Struct, error) {
	rt := reflect.TypeOf(ob)
	obname := rt.Name()
	pkgpath := rt.PkgPath()
	if pkgpath == "main" {
		if l.buildinfo == nil {
			binfo, ok := debug.ReadBuildInfo()
			if !ok {
				log.Println("debug.ReadBuildInfo() is failed")
				return nil, ErrNotFound
			}
			l.buildinfo = binfo
		}
		pkgpath = l.buildinfo.Path
	}

	l.mu.Lock()
	if p, ok := l.cache[pkgpath]; ok && p.fullset {
		defer l.mu.Unlock()
		if p.err != nil {
			return nil, p.err
		}

		result, ok := p.Structs[obname]
		if !ok {
			return nil, fmt.Errorf("lookup metadata of %T is failed %w", ob, ErrNotFound)
		}
		return &Struct{Raw: result}, nil
	}
	l.mu.Unlock()

	cfg := &packages.Config{
		Fset:  l.Fset,
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedSyntax,
		Tests: l.IncludeGoTestFiles, // TODO: support <name>_test package
		ParseFile: func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
			// TODO: debug print
			const mode = parser.ParseComments //| parser.AllErrors
			return parser.ParseFile(fset, filename, src, mode)
		},
	}

	pkgs, err := packages.Load(cfg, pkgpath)
	if err != nil {
		return nil, fmt.Errorf("packages.Load() %w", err)
	}

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			for _, err := range pkg.Errors {
				log.Printf("lookup package error (%s) %+v", pkg, err)
			}
			func() {
				l.mu.Lock()
				defer l.mu.Unlock()
				ref := &packageRef{fullset: true, err: pkg.Errors[0]}
				l.cache[pkg.PkgPath] = ref
			}()
			continue
		}

		if pkg.PkgPath != pkgpath {
			continue
		}

		tree := &ast.Package{Name: pkg.Name, Files: map[string]*ast.File{}}
		for _, f := range pkg.Syntax {
			filename := l.Fset.File(f.Pos()).Name()
			tree.Files[filename] = f
		}

		func() {
			l.mu.Lock()
			defer l.mu.Unlock()

			ref := &packageRef{fullset: true}
			p, err := commentof.Package(l.Fset, tree)
			if err != nil {
				ref.err = err
			}
			ref.Package = p
			l.cache[pkg.PkgPath] = ref
		}()

		result, ok := l.cache[pkgpath].Structs[obname]
		if !ok {
			continue
		}
		return &Struct{Raw: result}, nil
	}
	return nil, fmt.Errorf("lookup metadata of %T is failed %w", ob, ErrNotFound)
}
