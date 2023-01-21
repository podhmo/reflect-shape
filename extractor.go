package reflectshape

import (
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"github.com/podhmo/reflect-shape/metadata"
)

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
	lv := 0
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
		rv = rv.Elem()
		lv++
	}

	id := ID{rt: rt}
	if rt.Kind() == reflect.Func {
		id.pc = rv.Pointer() // distinguish same signature function
	}

	shape, ok := e.seen[id]
	if ok {
		if lv == 0 {
			return shape
		}
		copied := *shape
		copied.Lv = lv
		return &copied
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
			name = fmt.Sprintf("%s.%s", strings.Trim(parts[len(parts)-2], "(*)"), strings.TrimSuffix(parts[len(parts)-1], "-fm"))
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
			Path:  pkgPath,
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

	if lv == 0 {
		return shape
	}
	copied := *shape
	copied.Lv = lv
	return &copied
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
