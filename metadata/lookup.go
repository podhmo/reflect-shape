package metadata

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/podhmo/commentof"
	"github.com/podhmo/commentof/collect"
	"github.com/podhmo/reflect-shape/metadata/internal/unsaferuntime"
	"golang.org/x/tools/go/packages"
)

// ErrNotFound is the error metadata is not found.
var ErrNotFound = fmt.Errorf("not found")

// ErrNotSupported is the error metadata is not supported, yet
var ErrNotSupported = fmt.Errorf("not supported")

var DEBUG = false

func init() {
	if ok, _ := strconv.ParseBool(os.Getenv("DEBUG")); ok {
		DEBUG = true
	}
}

type Lookup struct {
	Fset     *token.FileSet
	accessor *unsaferuntime.Accessor

	IncludeGoTestFiles bool
	IncludeUnexported  bool

	cache map[string]*packageRef // TODO: lock
}

func NewLookup(fset *token.FileSet) *Lookup {
	return &Lookup{
		Fset:               fset,
		accessor:           unsaferuntime.New(),
		IncludeGoTestFiles: false,
		IncludeUnexported:  false,
		cache:              map[string]*packageRef{},
	}
}

type Func struct {
	pc   uintptr
	Raw  *collect.Func
	Recv string
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

func (m *Func) ArgComments() map[string]string {
	comments := make(map[string]string, len(m.Raw.ParamNames))
	for _, id := range m.Raw.ParamNames {
		p := m.Raw.Params[id]
		doc := p.Doc
		if doc == "" {
			doc = p.Comment
		}
		comments[id] = strings.TrimSpace(doc)
	}
	return comments
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
	return l.LookupFromFuncForPC(pc)
}

func (l *Lookup) LookupFromFuncForPC(pc uintptr) (*Func, error) {
	rfunc := l.accessor.FuncForPC(pc)
	if rfunc == nil {
		return nil, fmt.Errorf("cannot find runtime.Func")
	}

	filename, _ := rfunc.FileLine(rfunc.Entry())

	// /<pkg name>.<function name>
	// /<pkg name>.<recv>.<method name>
	// /<pkg name>.<recv>.<method name>-fm

	parts := strings.Split(rfunc.Name(), "/")
	last := parts[len(parts)-1]
	pkgname, name, isFunc := strings.Cut(last, ".")
	_ = pkgname
	if !isFunc {
		return nil, fmt.Errorf("unexpected func: %v", rfunc.Name())
	}

	recv, name, isMethod := strings.Cut(name, ".")
	if isMethod {
		recv = strings.Trim(recv, "(*)")
	} else {
		name = recv
		recv = ""
	}
	// log.Printf("pkgname:%-15s\trecv:%-10s\tname:%s\n", pkgname, recv, name)

	pkgpath := rfuncPkgpath(rfunc)
	p0, ok := l.cache[pkgpath]
	if ok {
		if p0.fullset {
			if p0.err != nil {
				return nil, p0.err
			}

			if isMethod {
				ob, ok := p0.Types[recv]
				if !ok {
					// anonymous function? (TODO: correct check)
					if _, ok := p0.Functions[name]; !ok {
						return nil, fmt.Errorf("lookup metadata of anonymous function %s, %w", rfunc.Name(), ErrNotSupported)
					}
					return nil, fmt.Errorf("lookup metadata of method %s, %w", rfunc.Name(), ErrNotFound)
				}
				result, ok := ob.Methods[name]
				if !ok {
					return nil, fmt.Errorf("lookup metadata of method %s, %w", rfunc.Name(), ErrNotFound)
				}
				if DEBUG {
					log.Println("\tOK func cache (full)", rfunc.Name())
				}
				return &Func{pc: pc, Raw: result, Recv: recv}, nil
			} else {
				result, ok := p0.Functions[name]
				if !ok {
					return nil, fmt.Errorf("lookup metadata of %s is failed %w", rfunc.Name(), ErrNotFound)
				}
				if DEBUG {
					log.Println("\tOK func cache (full)", rfunc.Name())
				}
				return &Func{Raw: result}, nil
			}
		}

		for _, visitedFile := range p0.FileNames {
			if visitedFile == filename {
				f, ok := p0.Files[filename]
				if !ok {
					break
				}
				result, ok := f.Functions[name]
				if !ok {
					return nil, fmt.Errorf("lookup metadata of %s is failed.. %w", rfunc.Name(), ErrNotFound)
				}
				if DEBUG {
					log.Println("\tOK func cache", rfunc.Name())
				}
				return &Func{pc: pc, Raw: result}, nil
			}
		}
	}

	f, err := parser.ParseFile(l.Fset, filename, nil, parser.ParseComments)
	if f == nil {
		l.cache[pkgpath] = &packageRef{fullset: false, err: err} // error cache
		return nil, err
	}

	p, err := commentof.File(l.Fset, f, commentof.WithIncludeUnexported(l.IncludeUnexported), func(b *collect.PackageBuilder) {
		if p0 != nil {
			b.Package = p0.Package // merge
		}
	})
	if !ok && p != nil {
		l.cache[pkgpath] = &packageRef{fullset: false, Package: p}
	}
	if err != nil {
		return nil, err
	}

	if DEBUG {
		log.Println("\tNG func cache", rfunc.Name())
	}
	if isMethod {
		ob, ok := p.Types[recv]
		if !ok {
			// anonymous function? (TODO: correct check)
			if _, ok := p.Functions[name]; !ok {
				return nil, fmt.Errorf("lookup metadata of anonymous function %s, %w", rfunc.Name(), ErrNotSupported)
			}
			return nil, fmt.Errorf("lookup metadata of method %s, %w", rfunc.Name(), ErrNotFound)
		}
		result, ok := ob.Methods[name]
		if !ok {
			return nil, fmt.Errorf("lookup metadata of method %s, %w", rfunc.Name(), ErrNotFound)
		}
		return &Func{pc: pc, Raw: result, Recv: recv}, nil
	} else {
		result, ok := p.Functions[name]
		if !ok {
			return nil, fmt.Errorf("lookup metadata of function %s, %w", rfunc.Name(), ErrNotFound)
		}
		return &Func{pc: pc, Raw: result}, nil
	}
}

func rfuncPkgpath(rfunc *runtime.Func) string {
	parts := strings.Split(rfunc.Name(), ".")
	return strings.Join(parts[:len(parts)-1], ".")
}

type Type struct {
	Raw *collect.Object
}

func (s *Type) Name() string {
	return s.Raw.Name
}

func (s *Type) Doc() string {
	doc := s.Raw.Doc
	if doc == "" {
		doc = s.Raw.Comment
	}
	return strings.TrimSpace(doc)
}

func (s *Type) FieldComments() map[string]string {
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

func (l *Lookup) LookupFromType(ob interface{}) (*Type, error) {
	rt := reflect.TypeOf(ob)
	return l.LookupFromTypeForReflectType(rt)
}
func (l *Lookup) LookupFromTypeForReflectType(rt reflect.Type) (*Type, error) {
	obname, _, _ := strings.Cut(rt.Name(), "[") // for generics
	pkgpath := rt.PkgPath()

	if pkgpath == "main" {
		binfo, ok := debug.ReadBuildInfo()
		if !ok {
			log.Println("debug.ReadBuildInfo() is failed")
			return nil, ErrNotFound
		}
		pkgpath = binfo.Path
	}

	if p, ok := l.cache[pkgpath]; ok && p.fullset {
		if p.err != nil {
			return nil, p.err
		}

		result, ok := p.Types[obname]
		if !ok {
			result, ok = p.Interfaces[obname]
			if !ok {
				return nil, fmt.Errorf("lookup metadata of %v is failed %w", rt, ErrNotFound)
			}
		}
		if DEBUG {
			log.Println("OK package cache", pkgpath)
		}
		return &Type{Raw: result}, nil
	}

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

	patterns := []string{pkgpath}
	if strings.HasSuffix(pkgpath, "_test") {
		patterns = []string{strings.TrimSuffix(pkgpath, "_test")} // for go test
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("packages.Load() %w", err)
	}

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			for _, err := range pkg.Errors {
				log.Printf("lookup package error (%s) %+v", pkg, err)
			}
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

		ref := &packageRef{fullset: true}
		l.cache[pkg.PkgPath] = ref
		p, err := commentof.Package(l.Fset, tree, commentof.WithIncludeUnexported(l.IncludeUnexported))
		if err != nil {
			ref.err = err
			return nil, fmt.Errorf("collect: dir=%s, name=%s, %w", pkg.PkgPath, obname, err)
		}
		ref.Package = p

		result, ok := p.Types[obname]
		if !ok {
			result, ok = p.Interfaces[obname]
			if !ok {
				continue
			}
		}
		if DEBUG {
			log.Println("NG package cache", pkgpath)
		}
		return &Type{Raw: result}, nil
	}
	return nil, fmt.Errorf("lookup metadata of %v is failed %w", rt, ErrNotFound)
}

type packageRef struct {
	*collect.Package

	fullset bool
	err     error
}
