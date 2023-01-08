package neo

import (
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"strings"
)

type Config struct {
	IncludeComments bool
	IncludeArgNames bool

	extractor *Extractor
}

func (c *Config) Extract(ob interface{}) *Shape {
	if c.extractor == nil {
		c.extractor = &Extractor{
			Config:   c,
			seen:     map[ID]*Shape{},
			packages: map[string]*Package{},
		}
	}
	return c.extractor.Extract(ob)
}

type Extractor struct {
	Config *Config

	seen     map[ID]*Shape
	packages map[string]*Package
}

func (e *Extractor) Extract(ob interface{}) *Shape {
	// TODO: only handling *T

	rt := reflect.TypeOf(ob)
	rv := reflect.ValueOf(ob)
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

	Number  int
	Package *Package
	e       *Extractor
}

func (s *Shape) Equal(another *Shape) bool {
	return s.ID == another.ID
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
