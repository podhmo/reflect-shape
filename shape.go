package reflectshape

import (
	"fmt"
	"go/token"
	"reflect"
	"strings"
)

type Identity string
type Kind reflect.Kind

func (k Kind) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`%q`, k.String())), nil
}
func (k Kind) String() string {
	return reflect.Kind(k).String()
}

type Shape interface {
	// GetName returns name something like `<name>`
	GetName() string
	// GetFullName returns fullname something like `<package path>.<name>`
	GetFullName() string
	// GetPackage returns package path like `<package path>`
	GetPackage() string
	// GetLV returns pointer's level (e.g. int is 0, *int is 1)
	GetLv() int

	GetReflectKind() reflect.Kind
	GetReflectType() reflect.Type
	GetReflectValue() reflect.Value

	// GetIdentity returns id of the shape.
	GetIdentity() Identity

	Clone() Shape
	deref(seen map[reflect.Type]Shape) Shape
	info() *Info
}
type ShapeList []Shape

type ShapeMap struct {
	Keys   []string `json:"keys"`
	Values []Shape  `json:"values"`
}

func (m *ShapeMap) Len() int {
	return len(m.Keys)
}

func (m *ShapeMap) Get(k string) (Shape, bool) {
	for i, name := range m.Keys {
		if name == k {
			return m.Values[i], true
		}
	}
	return nil, false
}

type FunctionSet struct {
	Names     []string            `json:"names"`
	Functions map[string]Function `json:"values"`
}

func (m *FunctionSet) Len() int {
	return len(m.Names)
}

func (m *FunctionSet) Get(k string) (Function, bool) {
	v, ok := m.Functions[k]
	return v, ok
}

type Info struct {
	Kind    Kind   `json:"kind"`
	Name    string `json:"name"`
	Lv      int    `json:"lv"` // v is 0, *v is 1
	Package string `json:"package"`

	completed    bool // complete means that shape does not have any refs
	reflectType  reflect.Type
	reflectValue reflect.Value
	identity     Identity
	extractor    *Extractor
}

func (v *Info) info() *Info {
	return v
}

func (v *Info) GetName() string {
	return v.Name
}
func (v *Info) GetFullName() string {
	return strings.TrimPrefix(v.Package+"."+v.Name, ".")
}
func (v *Info) GetLv() int {
	return v.Lv
}
func (v *Info) GetPackage() string {
	return v.Package
}
func (v *Info) GetReflectKind() reflect.Kind {
	return reflect.Kind(v.Kind)
}
func (v *Info) GetReflectType() reflect.Type {
	return v.reflectType
}
func (v *Info) GetReflectValue() reflect.Value {
	return v.reflectValue
}
func (v *Info) GetIdentity() Identity {
	if v.identity != "" {
		return v.identity
	}
	rt := v.GetReflectType()
	v.identity = Identity(fmt.Sprintf("%s:%s@%d", v.GetFullName(), rt, rt.Size()))
	return v.identity
}
func (v *Info) Clone() *Info {
	return &Info{
		Kind:         v.Kind,
		Name:         v.Name,
		Lv:           v.Lv,
		Package:      v.Package,
		reflectType:  v.reflectType,
		reflectValue: v.reflectValue,
		completed:    v.completed,
		extractor:    v.extractor,
	}
}

func (v *Info) doc(s Shape) string {
	e := v.extractor
	if e.MetadataLookup == nil {
		return ""
	}
	mob, err := e.MetadataLookup.LookupFromStructForReflectType(v.reflectType)
	if err != nil {
		fmt.Println("@@", err)
		if e.OnError != nil {
			e.OnError(s, err, "Doc")
		}
		return ""
	}
	return mob.Doc()
}

type Primitive struct {
	*Info
}

func (v Primitive) Clone() Shape {
	var new Primitive
	new.Info = v.Info.Clone()
	return new
}

func (v Primitive) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%s%s",
		strings.Repeat("*", v.Lv),
		v.GetFullName(),
	)
}
func (v Primitive) deref(seen map[reflect.Type]Shape) Shape {
	return v
}
func (v Primitive) Doc() string {
	return v.Info.doc(v)
}

type FieldMetadata struct {
	Anonymous bool // embedded?
	FieldName string
	Required  bool
}

type Struct struct {
	*Info
	Fields   ShapeMap `json:"fields"`
	Tags     []reflect.StructTag
	Metadata []FieldMetadata
}

func (v *Struct) FieldName(i int) string {
	name := v.Metadata[i].FieldName
	if name != "" {
		return name
	}

	if val, ok := v.Tags[i].Lookup("json"); ok {
		name = strings.SplitN(val, ",", 2)[0] // todo: omitempty, inline
		v.Metadata[i].FieldName = name        // cache
		return name
	}
	if val, ok := v.Tags[i].Lookup("form"); ok {
		name = strings.SplitN(val, ",", 2)[0]
		v.Metadata[i].FieldName = name // cache
		return name
	}

	return v.Fields.Keys[i]
}

func (v Struct) Format(f fmt.State, c rune) {
	if c == 'v' && f.Flag('+') {
		fmt.Fprintf(f, "%s%s{%s}",
			strings.Repeat("*", v.Lv),
			v.GetFullName(),
			strings.Join(v.Fields.Keys, ", "),
		)
		return
	}
	fmt.Fprintf(f, "%s%s",
		strings.Repeat("*", v.Lv),
		v.GetFullName(),
	)
}
func (s Struct) Clone() Shape {
	var new Struct
	new.Info = s.Info.Clone()
	new.Fields = s.Fields
	new.Tags = s.Tags
	new.Metadata = s.Metadata
	return new
}

func (v Struct) deref(seen map[reflect.Type]Shape) Shape {
	if v.Info.completed {
		return v
	}

	v.Info.completed = true
	for i, e := range v.Fields.Values {
		v.Fields.Values[i] = e.deref(seen)
	}
	return v
}

func (v *Struct) Methods() FunctionSet {
	methodMap := FunctionSet{Functions: map[string]Function{}}
	rt := v.reflectType

	seen := map[string]bool{}
	candidates := []reflect.Type{rt, reflect.PtrTo(rt)}

	for _, rt := range candidates {
		n := rt.NumMethod()
		path := []string{rt.Name()}
		rts := []reflect.Type{rt}
		rvs := []reflect.Value{reflect.ValueOf(nil)} // xxx
		for i := 0; i < n; i++ {
			method := rt.Method(i)
			name := method.Name
			if !token.IsExported(name) {
				continue
			}

			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = true

			shape := v.extractor.extract(
				append(path, name),
				append(rts, method.Type),
				append(rvs, method.Func),
				method,
			)
			shape = shape.deref(v.extractor.Seen)
			fn := shape.(Function)
			if v.extractor.MetadataLookup != nil { // always revisit
				fixupArglist(v.extractor.MetadataLookup, &fn, method.Func.Interface(), name, true)
			}
			methodMap.Names = append(methodMap.Names, name)
			methodMap.Functions[name] = fn
		}
	}
	return methodMap
}
func (v Struct) Doc() string {
	return v.Info.doc(v)
}

type Interface struct {
	*Info
	Methods ShapeMap `json:"methods"`
}

func (v Interface) Format(f fmt.State, c rune) {
	if c == 'v' && f.Flag('+') {
		fmt.Fprintf(f, "%s%s{%s}",
			strings.Repeat("*", v.Lv),
			v.GetFullName(),
			strings.Join(v.Methods.Keys, "(), "),
		)
		return
	}
	fmt.Fprintf(f, "%s%s",
		strings.Repeat("*", v.Lv),
		v.GetFullName(),
	)
}
func (s Interface) Clone() Shape {
	var new Interface
	new.Info = s.Info.Clone()
	new.Methods = s.Methods
	return new
}

func (v Interface) deref(seen map[reflect.Type]Shape) Shape {
	if v.Info.completed {
		return v
	}

	v.Info.completed = true
	for i, e := range v.Methods.Values {
		v.Methods.Values[i] = e.deref(seen)
	}
	return v
}
func (v Interface) Doc() string {
	return v.Info.doc(v)
}

// for generics
type Container struct {
	*Info
	Args ShapeList `json:"args"`
}

func (v Container) Format(f fmt.State, c rune) {
	expr := "%v"
	if c == 'v' && f.Flag('+') {
		expr = "%+v"
	}
	args := make([]string, len(v.Args))
	for i := range v.Args {
		args[i] = fmt.Sprintf(expr, v.Args[i])
	}

	fmt.Fprintf(f, "%s%s[%s]",
		strings.Repeat("*", v.Lv),
		v.GetFullName(),
		strings.Join(args, ", "),
	)
}
func (s Container) Clone() Shape {
	var new Container
	new.Info = s.Info.Clone()
	new.Args = s.Args
	return new
}
func (v Container) deref(seen map[reflect.Type]Shape) Shape {
	if v.Info.completed {
		return v
	}

	v.Info.completed = true
	for i, e := range v.Args {
		v.Args[i] = e.deref(seen)
	}
	return v
}
func (v Container) Doc() string {
	return v.Info.doc(v)
}

type Function struct {
	*Info
	Params  ShapeMap `json:"params"`  // for function's In
	Returns ShapeMap `json:"returns"` // for function's Out
}

func (v Function) Doc() string {
	e := v.Info.extractor
	if e.MetadataLookup == nil {
		return ""
	}
	mfunc, err := e.MetadataLookup.LookupFromFuncForPC(v.reflectValue.Pointer())
	if err != nil {
		if e.OnError != nil {
			e.OnError(v, err, "Doc")
		}
		return ""
	}
	return mfunc.Doc()
}

func (v Function) Format(f fmt.State, c rune) {
	expr := "%v"
	if c == 'v' && f.Flag('+') {
		expr = "%+v"
	}

	params := make([]string, len(v.Params.Keys))
	for i, val := range v.Params.Values {
		params[i] = fmt.Sprintf(expr, val)
	}
	returns := make([]string, len(v.Returns.Keys))
	for i, val := range v.Returns.Values {
		returns[i] = fmt.Sprintf(expr, val)
	}
	fmt.Fprintf(f, "%s%s(%s) (%s)",
		strings.Repeat("*", v.Lv),
		v.GetFullName(),
		strings.Join(params, ", "),
		strings.Join(returns, ", "),
	)
}
func (s Function) Clone() Shape {
	var new Function
	new.Info = s.Info.Clone()
	new.Params = s.Params
	new.Returns = s.Returns
	return new
}
func (v Function) deref(seen map[reflect.Type]Shape) Shape {
	if v.Info.completed {
		return v
	}

	v.Info.completed = true
	for i, e := range v.Params.Values {
		v.Params.Values[i] = e.deref(seen)
	}
	for i, e := range v.Returns.Values {
		v.Returns.Values[i] = e.deref(seen)
	}
	return v
}

type Unknown struct {
	*Info
}

func (v Unknown) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "UNKNOWN[%v]", v.Info.GetReflectValue())
}
func (s Unknown) Clone() Shape {
	var new Unknown
	new.Info = s.Info.Clone()
	return new
}
func (v Unknown) deref(seen map[reflect.Type]Shape) Shape {
	if v.Info.completed {
		return v
	}

	v.Info.completed = true
	return v
}
