package neo

import (
	"fmt"
	"go/token"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"github.com/podhmo/reflect-shape/metadata"
)

type Config struct {
	IncludeComments bool
	IncludeArgNames bool

	extractor *Extractor
	lookup    *metadata.Lookup
}

func (c *Config) Extract(ob interface{}) *Shape {
	if c.lookup == nil {
		c.lookup = metadata.NewLookup(token.NewFileSet())
	}
	if c.extractor == nil {
		c.extractor = &Extractor{
			Config:   c,
			Lookup:   c.lookup,
			seen:     map[ID]*Shape{},
			packages: map[string]*Package{},
		}
	}
	return c.extractor.Extract(ob)
}

type Extractor struct {
	Config *Config
	Lookup *metadata.Lookup

	seen     map[ID]*Shape
	packages map[string]*Package
}

func (e *Extractor) Extract(ob interface{}) *Shape {
	// TODO: only handling *T
	rt := reflect.TypeOf(ob)
	rv := reflect.ValueOf(ob)
	return e.extract(rt, rv)
}

func (e *Extractor) extract(rt reflect.Type, rv reflect.Value) *Shape {
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
		rv = rv.Elem()
	}

	id := ID{rt: rt}
	if rt.Kind() == reflect.Func {
		id.pc = rv.Pointer() // distinguish same signature function
	}

	shape, ok := e.seen[id]
	if ok {
		return shape
	}

	name := rt.Name()
	pkgPath := rt.PkgPath()
	isMethod := false
	if pkgPath == "" && id.pc != 0 {
		fullname := runtime.FuncForPC(id.pc).Name()
		parts := strings.Split(fullname, ".")

		if strings.HasSuffix(fullname, "-fm") {
			isMethod = true
			// @@ github.com/podhmo/reflect-shape/neo_test.S0.M-fm
			// @@ github.com/podhmo/reflect-shape/neo_test.(*S1).M-fm
			pkgPath = strings.Join(parts[:len(parts)-2], ".")
			name = fmt.Sprintf("%s.%s", strings.Trim(parts[len(parts)-2], "(*)"), parts[len(parts)-1])
		} else {
			// @@ github.com/podhmo/reflect-shape/neo_test.F1
			// @@ github.com/podhmo/reflect-shape/neo_test.S0
			pkgPath = strings.Join(parts[:len(parts)-1], ".")
			name = parts[len(parts)-1]
		}
	}

	pkg, ok := e.packages[pkgPath]
	if !ok {
		parts := strings.Split(pkgPath, "/") // todo fix
		pkgName := parts[len(parts)-1]
		pkg = &Package{
			Name:  pkgName,
			Path:  rt.PkgPath(),
			scope: &Scope{shapes: map[string]*Shape{}},
		}
		e.packages[pkgPath] = pkg
	}

	shape = &Shape{
		Name:         name,
		Kind:         rt.Kind(),
		ID:           id,
		Type:         rt,
		DefaultValue: rv,
		Number:       len(e.seen),
		IsMethod:     isMethod,
		Package:      pkg,
		e:            e,
	}
	e.seen[id] = shape
	pkg.scope.shapes[name] = shape
	return shape
}

type Shape struct {
	Name     string
	Kind     reflect.Kind
	IsMethod bool

	ID           ID
	Type         reflect.Type
	DefaultValue reflect.Value

	Number  int // If all shapes are from the same extractor, this value can be used as ID
	Package *Package
	e       *Extractor
}

func (s *Shape) Equal(another *Shape) bool {
	return s.ID == another.ID
}

func (s *Shape) MustFunc() *Func {
	if s.Kind != reflect.Func && s.ID.pc == 0 {
		panic(fmt.Sprintf("shape %v is not func kind, %s", s, s.Kind))
	}
	lookup := s.e.Lookup
	metadata, err := lookup.LookupFromFuncForPC(s.ID.pc)
	if err != nil {
		panic(err)
	}
	// TODO: fill all data
	return &Func{Shape: s, metadata: metadata}
}

type Func struct {
	Shape    *Shape
	metadata *metadata.Func
}

func (f *Func) Name() string {
	return f.Shape.Name
}

func (f *Func) IsMethod() bool {
	return f.Shape.IsMethod
}
func (f *Func) IsVariadic() bool {
	return f.Shape.Type.IsVariadic()
}

func (f *Func) Args() VarList {
	typ := f.Shape.Type
	r := make([]*Var, typ.NumIn())
	args := f.metadata.Args()
	for i := 0; i < typ.NumIn(); i++ {
		rt := typ.In(i)
		rv := rzero(rt)
		shape := f.Shape.e.extract(rt, rv)
		r[i] = &Var{Name: args[i], Shape: shape}
	}
	return VarList(r)
}

func (f *Func) Returns() VarList {
	typ := f.Shape.Type
	r := make([]*Var, typ.NumOut())
	args := f.metadata.Returns()
	for i := 0; i < typ.NumOut(); i++ {
		rt := typ.Out(i)
		rv := rzero(rt)
		shape := f.Shape.e.extract(rt, rv)
		r[i] = &Var{Name: args[i], Shape: shape}
	}
	return VarList(r)
}

func (f *Func) Doc() string {
	return f.metadata.Doc()
}
func (f *Func) Recv() string {
	return f.metadata.Recv
}

func (f *Func) String() string {
	return fmt.Sprintf("&Func{Name: %q, Args: %v, Returns: %v}", f.Name(), f.metadata.Args(), f.metadata.Returns())
}

type VarList []*Var

func (vl VarList) String() string {
	parts := make([]string, len(vl))
	for i, v := range vl {
		parts[i] = fmt.Sprintf("%+v", v)
	}
	return fmt.Sprintf("%+v", parts)
}

type Var struct {
	Name  string
	Shape *Shape
}

type ID struct {
	rt reflect.Type
	pc uintptr
}

type Package struct {
	Name string
	Path string

	scope *Scope
}

func (p *Package) Scope() *Scope {
	return p.scope
}

type Scope struct {
	shapes map[string]*Shape
}

func (s *Scope) Names() []string {
	return s.names(false)
}

func (s *Scope) NamesWithMethod() []string {
	return s.names(true)
}

func (s *Scope) names(withMethod bool) []string {
	// anonymous function is not supported yet
	r := make([]string, 0, len(s.shapes))
	for name, s := range s.shapes {
		if !withMethod && s.IsMethod {
			continue
		}
		r = append(r, name)
	}
	sort.Strings(r)
	return r
}

var (
	rnil = reflect.ValueOf(nil)
)

func rzero(rt reflect.Type) reflect.Value {
	// TODO: fixme
	return reflect.New(rt)
}
