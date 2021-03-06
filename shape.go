package reflectshape

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/podhmo/reflect-shape/arglist"
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
	Shape() string

	GetName() string
	GetFullName() string
	GetPackage() string
	GetLv() int

	ResetName(string)
	ResetPackage(string)
	ResetReflectType(reflect.Type)

	GetReflectKind() reflect.Kind
	GetReflectType() reflect.Type
	GetReflectValue() reflect.Value

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

func (v *Info) Shape() string {
	return v.Kind.String()
}
func (v *Info) GetName() string {
	return v.Name
}
func (v *Info) ResetName(name string) {
	v.Name = name
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
func (v *Info) ResetPackage(name string) {
	v.Package = name
}
func (v *Info) GetReflectKind() reflect.Kind {
	return reflect.Kind(v.Kind)
}
func (v *Info) GetReflectType() reflect.Type {
	return v.reflectType
}
func (v *Info) ResetReflectType(rt reflect.Type) {
	v.reflectType = rt
	v.identity = ""
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

	for i := v.GetLv(); i == 0; i-- {
		rt = rt.Elem()
	}

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
			if strings.ToUpper(name[:1]) != name[:1] {
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
			if v.extractor.ArglistLookup != nil { // always revisit
				fixupArglist(v.extractor.ArglistLookup, &fn, method.Func.Interface(), name, true)
			}
			methodMap.Names = append(methodMap.Names, name)
			methodMap.Functions[name] = fn
		}
	}
	return methodMap
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

type Function struct {
	*Info
	Params  ShapeMap `json:"params"`  // for function's In
	Returns ShapeMap `json:"returns"` // for function's Out
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

type ref struct {
	*Info
	originalRT reflect.Type
}

func (v *ref) Clone() Shape {
	return &ref{
		Info:       v.Info.Clone(),
		originalRT: v.originalRT,
	}
}
func (v *ref) deref(seen map[reflect.Type]Shape) Shape {
	if v.Info.completed {
		return seen[v.GetReflectType()]
	}

	v.Info.completed = true
	original := seen[v.originalRT]
	if !original.info().completed {
		original = original.deref(seen)
		seen[v.originalRT] = original
	}
	r := original.Clone()
	info := r.info()
	info.Lv += v.Info.Lv
	seen[v.GetReflectType()] = r
	return r
}

type Extractor struct {
	Seen map[reflect.Type]Shape

	ArglistLookup  *arglist.Lookup
	RevisitArglist bool

	c int
}

var rnil reflect.Value

func init() {
	rnil = reflect.ValueOf(nil)
}

func (e *Extractor) Extract(ob interface{}) Shape {
	rt := reflect.TypeOf(ob)
	if s, ok := e.Seen[rt]; ok {
		if rt.Kind() != reflect.Func {
			return s
		}
		fn := s.Clone().(Function)
		// TODO: cache

		fullname := runtime.FuncForPC(reflect.ValueOf(ob).Pointer()).Name()
		parts := strings.Split(fullname, ".")
		pkgPath := strings.Join(parts[:len(parts)-1], ".")
		fn.Info.Name = parts[len(parts)-1]
		fn.Info.Package = pkgPath

		if e.RevisitArglist && e.ArglistLookup != nil {
			fixupArglist(e.ArglistLookup, &fn, ob, fullname, false)
		}
		return fn
	}
	path := []string{""}
	rts := []reflect.Type{rt}                   // history
	rvs := []reflect.Value{reflect.ValueOf(ob)} // history
	s := e.extract(path, rts, rvs, ob)
	return s.deref(e.Seen)
}

func (e *Extractor) save(rt reflect.Type, s Shape) Shape {
	if _, ok := e.Seen[rt].(*ref); ok {
		e.Seen[rt] = s
	}
	return s
}

func (e *Extractor) extract(
	path []string,
	rts []reflect.Type,
	rvs []reflect.Value,
	ob interface{},
) Shape {
	rt := rts[len(rts)-1]
	rv := rvs[len(rvs)-1]

	// fmt.Fprintln(os.Stderr, path, rts)
	if len(path) > 30 {
		panic("x")
	}
	// fmt.Fprintln(os.Stderr, strings.Repeat("  ", len(rts)), path[len(path)-1], "::", rt)

	if s, ok := e.Seen[rt]; ok {
		return s
	}
	name := rt.Name()
	kind := rt.Kind()
	pkgPath := rt.PkgPath()

	info := &Info{
		Name:         name,
		Kind:         Kind(kind),
		Package:      pkgPath,
		reflectType:  rt,
		reflectValue: rv,
		extractor:    e,
	}
	ref := &ref{Info: info, originalRT: rt}
	e.Seen[rt] = ref

	var inner reflect.Value

	// todo: switch
	switch kind {
	case reflect.Ptr:
		if rv != rnil {
			inner = rv.Elem()
		}
		e.extract(
			append(path, "*"),
			append(rts, rt.Elem()),
			append(rvs, inner),
			nil)
		ref.originalRT = rt.Elem()
		ref.Info.Lv++
		return e.save(rt, ref)
	case reflect.Slice, reflect.Array:
		info.Name = kind.String() // slice

		if rv != rnil && rv.Len() > 0 {
			inner = rv.Index(0)
		}
		args := []Shape{
			e.extract(
				append(path, "slice[0]"),
				append(rts, rt.Elem()),
				append(rvs, inner),
				nil,
			),
		}
		s := Container{
			Args: args,
			Info: info,
		}
		return e.save(rt, s)
	case reflect.Map:
		info.Name = kind.String() // map

		var innerKey reflect.Value
		if rv != rnil && rv.Len() > 0 {
			it := rv.MapRange()
			it.Next()
			innerKey = it.Key()
			inner = it.Value()
		}
		args := []Shape{
			e.extract(
				append(path, "map[0]"),
				append(rts, rt.Key()),
				append(rvs, innerKey),
				nil,
			),
			e.extract(
				append(path, "map[1]"),
				append(rts, rt.Elem()),
				append(rvs, inner),
				nil,
			),
		}
		s := Container{
			Args: args,
			Info: info,
		}
		return e.save(rt, s)
	case reflect.Chan:
		// TODO: if STRICT=1, panic?
		// panic(fmt.Sprintf("not implemented yet or impossible: (%+v,%+v)", rt, rv))
		return Unknown{
			Info: info,
		}
	case reflect.Struct:
		n := rt.NumField()
		names := make([]string, n)
		fields := make([]Shape, n)
		tags := make([]reflect.StructTag, n)
		metadata := make([]FieldMetadata, n)

		if rv == rnil {
			rv = reflect.Zero(rt)
		}
		for i := 0; i < n; i++ {
			f := rt.Field(i)
			names[i] = f.Name
			fields[i] = e.extract(
				append(path, "struct."+f.Name),
				append(rts, f.Type),
				append(rvs, rv.Field(i)),
				nil,
			)
			tags[i] = f.Tag
			metadata[i] = FieldMetadata{
				Anonymous: f.Anonymous,
			}
			// todo: anonymous
		}
		s := Struct{
			Fields: ShapeMap{
				Keys:   names,
				Values: fields,
			},
			Tags:     tags,
			Metadata: metadata,
			Info:     info,
		}
		return e.save(rt, s)
	case reflect.Func:
		name := info.Name
		isMethod := false
		if ob != nil {
			if m, ok := ob.(reflect.Method); ok {
				ob = m.Func.Interface()
				pkgPath = m.PkgPath
				name = m.Name
				isMethod = true
			} else {
				fullname := runtime.FuncForPC(reflect.ValueOf(ob).Pointer()).Name()
				parts := strings.Split(fullname, ".")
				pkgPath = strings.Join(parts[:len(parts)-1], ".")
				name = parts[len(parts)-1]
			}
		}

		pnames := make([]string, rt.NumIn())
		params := make([]Shape, rt.NumIn())
		for i := 0; i < len(params); i++ {
			v := rt.In(i)
			arg := e.extract(
				append(path, "func.p["+strconv.Itoa(i)+"]"),
				append(rts, v),
				append(rvs, rnil),
				nil)
			argname := "args" + strconv.Itoa(i) //
			if v.Kind() == reflect.Func {
				argname = arg.GetName()
			}
			pnames[i] = argname
			params[i] = arg
		}
		rnames := make([]string, rt.NumOut())
		returns := make([]Shape, rt.NumOut())
		for i := 0; i < len(returns); i++ {
			v := rt.Out(i)
			rnames[i] = "ret" + strconv.Itoa(i) //
			returns[i] = e.extract(
				append(path, "func.r["+strconv.Itoa(i)+"]"),
				append(rts, v),
				append(rvs, rnil),
				nil)
		}

		s := Function{
			Params:  ShapeMap{Keys: pnames, Values: params},
			Returns: ShapeMap{Keys: rnames, Values: returns},
			Info: &Info{
				Name:         name,
				Kind:         Kind(kind),
				Package:      pkgPath,
				reflectType:  rt,
				reflectValue: rv,
				extractor:    e,
			},
		}
		// fixup names
		if e.ArglistLookup != nil && ob != nil {
			fixupArglist(e.ArglistLookup, &s, ob, name, isMethod)
		}
		if s.Name == "" {
			s.Name = fmt.Sprintf("func%d", e.c)
			e.c++
		}

		return e.save(rt, s)
	case reflect.Interface:
		names := make([]string, rt.NumMethod())
		methods := make([]Shape, rt.NumMethod())
		for i := 0; i < len(methods); i++ {
			f := rt.Method(i)
			names[i] = f.Name
			methods[i] = e.extract(
				append(path, "interface."+f.Name),
				append(rts, f.Type),
				append(rvs, rnil),
				nil,
			)
		}
		s := Interface{
			Methods: ShapeMap{
				Keys:   names,
				Values: methods,
			},
			Info: info,
		}
		return e.save(rt, s)
	default:
		// fmt.Fprintln(os.Stderr, "\t\t", kind.String())
		s := Primitive{
			Info: info,
		}
		return e.save(rt, s)
	}
}
