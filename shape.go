package reflectshape

import (
	"context"
	"fmt"
	"go/token"
	"log"
	"reflect"
	"regexp"

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
	Lv      int // pointer level. v is 0, *v is 1.
	Package *Package
	e       *Extractor
}

func (s *Shape) Equal(another *Shape) bool {
	return s.ID == another.ID
}

func (s *Shape) FullName() string {
	return fmt.Sprintf("%s.%s", s.Package.Path, s.Name)
}

func (s *Shape) String() string {
	return fmt.Sprintf("&Shape#%d{Name: %q, Kind: %v, Type: %v, Package: %v}", s.Number, s.Name, s.Kind, s.Type, s.Package.Name)
}

func (s *Shape) Struct() *Struct {
	if s.Kind != reflect.Struct {
		panic(fmt.Sprintf("shape %v is not Struct kind, %s", s, s.Kind))
	}
	lookup := s.e.Lookup
	if lookup == nil || s.Name == "" {
		return &Struct{Shape: s}
	}

	metadata, err := lookup.LookupFromTypeForReflectType(s.Type)
	if err != nil {
		log.Printf("MustStruct(): %+v", err)
		return &Struct{Shape: s}
	}
	return &Struct{Shape: s, metadata: metadata}
}

func (s *Shape) Interface() *Interface {
	if s.Kind != reflect.Interface {
		panic(fmt.Sprintf("shape %v is not Interface kind, %s", s, s.Kind))
	}
	lookup := s.e.Lookup
	if lookup == nil || s.Name == "" {
		return &Interface{Shape: s}
	}

	metadata, err := lookup.LookupFromTypeForReflectType(s.Type)
	if err != nil {
		log.Printf("MustInterface(): %+v", err)
		return &Interface{Shape: s}
	}
	return &Interface{Shape: s, metadata: metadata}
}

var anonymousFuncNameRegex = regexp.MustCompile(`func\d+$`)

func (s *Shape) Func() *Func {
	if s.Kind != reflect.Func && s.ID.pc == 0 {
		panic(fmt.Sprintf("shape %v is not func kind, %s", s, s.Kind))
	}
	lookup := s.e.Lookup
	if lookup == nil || s.Name == "" || (s.Package.Path == "" && anonymousFuncNameRegex.MatchString(s.Name)) {
		return &Func{Shape: s}
	}

	metadata, err := lookup.LookupFromFuncForPC(s.ID.pc)
	if err != nil {
		log.Printf("MustFunc(): %+v", err)
		return &Func{Shape: s}
	}
	return &Func{Shape: s, metadata: metadata}
}

func (s *Shape) Named() *Named {
	// TODO: check
	lookup := s.e.Lookup
	if lookup == nil || s.Name == "" {
		return &Named{Shape: s}
	}

	metadata, err := lookup.LookupFromTypeForReflectType(s.Type)
	if err != nil {
		log.Printf("MustType(): %+v", err)
		return &Named{Shape: s}
	}
	return &Named{Shape: s, metadata: metadata}
}

type Named struct {
	Shape    *Shape
	metadata *metadata.Type
}

func (t *Named) Name() string {
	return t.Shape.Name
}

func (t *Named) Pos() token.Pos {
	return t.metadata.Raw.Pos
}

func (t *Named) Doc() string {
	if t.metadata == nil {
		return ""
	}
	return t.metadata.Doc()
}

func (t *Named) String() string {
	doc := t.Doc()
	tsize := t.Shape.e.Config.DocTruncationSize
	if len(doc) > tsize {
		doc = doc[:tsize] + "..."
	}
	return fmt.Sprintf("&Type{Name: %q, kind: %s, type: %v, Doc: %q}", t.Name(), t.Shape.Kind, t.Shape.Type, doc)
}

type Struct struct {
	Shape    *Shape
	metadata *metadata.Type
}

func (s *Struct) Name() string {
	return s.Shape.Name
}

func (s *Struct) Pos() token.Pos {
	return s.metadata.Raw.Pos
}

func (s *Struct) Doc() string {
	if s.metadata == nil {
		return ""
	}
	return s.metadata.Doc()
}

func (s *Struct) Fields() FieldList {
	typ := s.Shape.Type
	var comments map[string]string
	if s.metadata != nil {
		comments = s.metadata.FieldComments()
	} else {
		comments = map[string]string{}
	}

	r := make([]*Field, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		rt := f.Type
		rv := rzero(f.Type)
		shape := s.Shape.e.extract(rt, rv)
		r[i] = &Field{StructField: f, Shape: shape, Doc: comments[f.Name]}
	}
	return FieldList(r)
}

func (s *Struct) String() string {
	doc := s.Doc()
	tsize := s.Shape.e.Config.DocTruncationSize
	if len(doc) > tsize {
		doc = doc[:tsize] + "..."
	}

	fields := s.Fields()
	fieldNames := make([]string, len(fields))
	for i, f := range fields {
		fieldNames[i] = f.Name
	}
	return fmt.Sprintf("&Struct{Name: %q, Fields: %v, Doc: %q}", s.Name(), fieldNames, doc)
}

type FieldList []*Field

func (fl FieldList) Len() int {
	return len(fl)
}

func (fl FieldList) String() string {
	parts := make([]string, len(fl))
	for i, v := range fl {
		parts[i] = fmt.Sprintf("%+v,", v)
	}
	return fmt.Sprintf("%+v", parts)
}

type Field struct {
	reflect.StructField
	Shape *Shape
	Doc   string
}

func (f *Field) String() string {
	doc := f.Doc
	tsize := f.Shape.e.Config.DocTruncationSize
	if len(doc) > tsize {
		doc = doc[:tsize] + "..."
	}
	return fmt.Sprintf("&Field{Name: %q, type: %v, Doc:%q}", f.Name, f.Shape.Type, doc)
}

type Interface struct {
	Shape    *Shape
	metadata *metadata.Type
}

func (iface *Interface) Name() string {
	return iface.Shape.Name
}

func (iface *Interface) Pos() token.Pos {
	return iface.metadata.Raw.Pos
}

func (iface *Interface) Doc() string {
	if iface.metadata == nil {
		return ""
	}
	return iface.metadata.Doc()
}

func (iface *Interface) Methods() VarList {
	typ := iface.Shape.Type
	var comments map[string]string
	if iface.metadata != nil {
		comments = iface.metadata.FieldComments()
	} else {
		comments = map[string]string{}
	}

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
	doc := iface.Doc()
	tsize := iface.Shape.e.Config.DocTruncationSize
	if len(doc) > tsize {
		doc = doc[:tsize] + "..."
	}

	methods := iface.Methods()
	methodNames := make([]string, len(methods))
	for i, m := range methods {
		methodNames[i] = m.Name
	}
	return fmt.Sprintf("&Interface{Name: %q, Methods: %v, Doc: %q}", iface.Name(), methodNames, doc)
}

type Func struct {
	Shape    *Shape
	metadata *metadata.Func
}

func (f *Func) Name() string {
	return f.Shape.Name
}

func (f *Func) Pos() token.Pos {
	return f.metadata.Raw.Pos
}

func (f *Func) IsMethod() bool {
	return f.Shape.IsMethod
}
func (f *Func) IsVariadic() bool {
	return f.Shape.Type.IsVariadic()
}

func (f *Func) Args() VarList {
	typ := f.Shape.Type
	var args []metadata.Var
	if f.metadata != nil {
		args = f.metadata.Args()
	} else {
		args = make([]metadata.Var, typ.NumIn())
	}

	r := make([]*Var, typ.NumIn())
	needFillNames := f.Shape.e.Config.FillArgNames
	for i := 0; i < typ.NumIn(); i++ {
		rt := typ.In(i)
		rv := rzero(rt)
		shape := f.Shape.e.extract(rt, rv)
		p := args[i]
		name := p.Name
		if name == "" && needFillNames {
			switch {
			case rcontextType == rt:
				name = "ctx"
			default:
				name = fmt.Sprintf("arg%d", i)
			}
		}
		r[i] = &Var{Name: name, Shape: shape, Doc: p.Doc}
	}
	return VarList(r)
}

func (f *Func) Returns() VarList {
	typ := f.Shape.Type
	var args []metadata.Var
	if f.metadata != nil {
		args = f.metadata.Returns()
	} else {
		args = make([]metadata.Var, typ.NumOut())
	}

	needFillNames := f.Shape.e.Config.FillReturnNames
	errUsed := false
	r := make([]*Var, typ.NumOut())
	for i := 0; i < typ.NumOut(); i++ {
		rt := typ.Out(i)
		rv := rzero(rt)
		shape := f.Shape.e.extract(rt, rv)
		p := args[i]
		name := p.Name
		if name == "" && needFillNames {
			switch {
			case rerrType == rt && errUsed:
				name = fmt.Sprintf("err%d", i)
			case rerrType == rt:
				name = "err"
				errUsed = true
			default:
				name = fmt.Sprintf("ret%d", i)
			}
		}
		r[i] = &Var{Name: name, Shape: shape, Doc: p.Doc}
	}
	return VarList(r)
}

func (f *Func) Doc() string {
	if f.metadata == nil {
		return ""
	}
	return f.metadata.Doc()
}
func (f *Func) Recv() string {
	if f.metadata == nil {
		if f.Shape.IsMethod {
			return "i"
		}
		return ""
	}
	return f.metadata.Recv
}

func (f *Func) String() string {
	doc := f.Doc()
	tsize := f.Shape.e.Config.DocTruncationSize
	if len(doc) > tsize {
		doc = doc[:tsize] + "..."
	}

	args := f.Args()
	argNames := make([]string, len(args))
	for i, m := range args {
		argNames[i] = m.Name
	}

	returns := f.Returns()
	returnNames := make([]string, len(returns))
	for i, m := range returns {
		returnNames[i] = m.Name
	}
	return fmt.Sprintf("&Func{Name: %q, Args: %v, Returns: %v, Doc: %q}", f.Name(), argNames, returnNames, doc)
}

type VarList []*Var

func (vl VarList) Len() int {
	return len(vl)
}

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
	tsize := v.Shape.e.Config.DocTruncationSize
	if len(doc) > tsize {
		doc = doc[:tsize] + "..."
	}
	return fmt.Sprintf("&Var{Name: %q, type: %v, Doc: %q}", v.Name, v.Shape.Type, doc)
}

func rzero(rt reflect.Type) reflect.Value {
	// TODO: fixme
	return reflect.New(rt).Elem()
}

var (
	rcontextType = reflect.TypeOf(func(context.Context) {}).In(0)
	rerrType     = reflect.TypeOf(func(error) {}).In(0)
)
