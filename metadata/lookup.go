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

	"github.com/podhmo/commentof"
	"github.com/podhmo/commentof/collect"
	"golang.org/x/tools/go/packages"
)

// ErrNotFound is the error metadata is not found.
var ErrNotFound = fmt.Errorf("not found")

type Lookup struct {
	Fset *token.FileSet
}

func NewLookup(fset *token.FileSet) *Lookup {
	return &Lookup{
		Fset: fset,
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
	return strings.TrimSpace(m.Raw.Doc) // todo: handling comment
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
	funcname := rfunc.Name()
	if strings.Contains(funcname, ".") {
		parts := strings.Split(funcname, ".")
		funcname = parts[len(parts)-1]
	}

	f, err := parser.ParseFile(l.Fset, filename, nil, parser.ParseComments)
	if f == nil {
		return nil, err
	}

	// TODO: package cache
	p, err := commentof.File(l.Fset, f)
	if err != nil {
		return nil, err
	}
	result, ok := p.Functions[funcname]
	if !ok {
		return nil, fmt.Errorf("function not found,")
	}
	return &Func{pc: pc, Raw: result}, nil
}

type Struct struct {
	Raw *collect.Object
}

func (l *Lookup) LookupFromStruct(ob interface{}) (*Struct, error) {
	rt := reflect.TypeOf(ob)
	obname := rt.Name()
	pkgpath := rt.PkgPath()
	if pkgpath == "main" {
		binfo, ok := debug.ReadBuildInfo()
		if !ok {
			log.Println("debug.ReadBuildInfo() is failed")
			return nil, ErrNotFound
		}
		pkgpath = binfo.Path
	}

	cfg := &packages.Config{
		Fset: l.Fset,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax,
	}
	pkgs, err := packages.Load(cfg, pkgpath)
	if err != nil {
		return nil, fmt.Errorf("packages.Load() %w", err)
	}

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			for _, err := range pkg.Errors {
				log.Printf("pkg %s error: %+v", pkg, err)
			}
			continue
		}
		if pkg.ID != pkgpath {
			continue
		}
		tree := &ast.Package{Name: pkg.Name, Files: map[string]*ast.File{}}
		for _, f := range pkg.Syntax {
			filename := l.Fset.File(f.Pos()).Name()
			tree.Files[filename] = f
		}

		p, err := commentof.Package(l.Fset, tree)
		if err != nil {
			return nil, fmt.Errorf("collect: dir=%s, name=%s, %w", pkg.PkgPath, obname, err)
		}
		result, ok := p.Structs[rt.Name()]
		if !ok {
			continue
		}
		return &Struct{Raw: result}, nil
	}
	return nil, ErrNotFound
}
