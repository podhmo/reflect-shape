package neo

import (
	"fmt"
	"reflect"

	"github.com/podhmo/reflect-shape/metadata"
)

type ID struct {
	rt reflect.Type
	pc uintptr
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

func rzero(rt reflect.Type) reflect.Value {
	// TODO: fixme
	return reflect.New(rt)
}
