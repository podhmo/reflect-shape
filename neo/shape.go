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

func (s *Shape) String() string {
	return fmt.Sprintf("&Shape#%d{Name: %-12q, Kind: %v, Type: %v, Package: %v}", s.Number, s.Name, s.Kind, s.Type, s.Package.Name)
}

func (s *Shape) MustStruct() *Struct {
	if s.Kind != reflect.Struct {
		panic(fmt.Sprintf("shape %v is not Struct kind, %s", s, s.Kind))
	}
	lookup := s.e.Lookup
	metadata, err := lookup.LookupFromTypeForReflectType(s.Type)
	if err != nil {
		panic(err)
	}
	return &Struct{Shape: s, metadata: metadata}
}

func (s *Shape) MustInterface() *Interface {
	if s.Kind != reflect.Interface {
		panic(fmt.Sprintf("shape %v is not Interface kind, %s", s, s.Kind))
	}
	lookup := s.e.Lookup
	metadata, err := lookup.LookupFromTypeForReflectType(s.Type)
	if err != nil {
		panic(err)
	}
	return &Interface{Shape: s, metadata: metadata}
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
	return &Func{Shape: s, metadata: metadata}
}

type Struct struct {
	Shape    *Shape
	metadata *metadata.Type
}

func (s *Struct) Name() string {
	return s.Shape.Name
}

func (s *Struct) Doc() string {
	return s.metadata.Raw.Doc
}

func (s *Struct) Fields() FieldList {
	typ := s.Shape.Type
	comments := s.metadata.FieldComments()
	r := make([]*Field, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		rt := f.Type
		rv := rzero(f.Type)
		shape := s.Shape.e.extract(rt, rv)
		r[i] = &Field{Name: f.Name, Shape: shape, Field: f, Doc: comments[f.Name]}
	}
	return FieldList(r)
}

func (s *Struct) String() string {
	return fmt.Sprintf("&Struct{Name: %-12q, Fields: %v}", s.Name(), s.metadata.Raw.FieldNames)
}

type FieldList []*Field

func (fl FieldList) String() string {
	parts := make([]string, len(fl))
	for i, v := range fl {
		parts[i] = fmt.Sprintf("%+v,", v)
	}
	return fmt.Sprintf("%+v", parts)
}

type Field struct {
	Name  string
	Shape *Shape
	Field reflect.StructField
	Doc   string
}

func (f *Field) String() string {
	doc := f.Doc
	if len(doc) > 10 {
		doc = doc[:10] + "..."
	}
	return fmt.Sprintf("&Field{Name: %-12q, type: %v, Doc:%q}", f.Name, f.Shape.Type, doc)
}

type Interface struct {
	Shape    *Shape
	metadata *metadata.Type
}

func (iface *Interface) Name() string {
	return iface.Shape.Name
}

func (iface *Interface) Doc() string {
	return iface.metadata.Raw.Doc
}

func (iface *Interface) Methods() VarList {
	typ := iface.Shape.Type
	comments := iface.metadata.FieldComments()
	r := make([]*Var, typ.NumMethod())
	for i := 0; i < typ.NumMethod(); i++ {
		f := typ.Method(i)
		rt := f.Type
		rv := rzero(f.Type)
		shape := iface.Shape.e.extract(rt, rv)
		r[i] = &Var{Name: f.Name, Shape: shape, Doc: comments[f.Name]}
	}
	return r
}

func (iface *Interface) String() string {
	return fmt.Sprintf("&Interface{Name: %-12q, Fields: %v}", iface.Name(), iface.metadata.Raw.FieldNames)
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
	return fmt.Sprintf("&Func{Name: %-12q, Args: %v, Returns: %v}", f.Name(), f.metadata.Args(), f.metadata.Returns())
}

type VarList []*Var

func (vl VarList) String() string {
	parts := make([]string, len(vl))
	for i, v := range vl {
		parts[i] = fmt.Sprintf("%+v,", v)
	}
	return fmt.Sprintf("%+v", parts)
}

type Var struct {
	Name  string
	Shape *Shape
	Doc   string
}

func (v *Var) String() string {
	doc := v.Doc
	if len(doc) > 10 {
		doc = doc[:10] + "..."
	}
	return fmt.Sprintf("&Var{Name: %-12q, type: %v, Doc: %v}", v.Name, v.Shape.Type, doc)
}

func rzero(rt reflect.Type) reflect.Value {
	// TODO: fixme
	return reflect.New(rt)
}
