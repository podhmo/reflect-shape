package reflectshape

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/podhmo/reflect-shape/metadata"
)

var rnil reflect.Value

func init() {
	rnil = reflect.ValueOf(nil)
}

type Extractor struct {
	Seen map[reflect.Type]Shape

	MetadataLookup *metadata.Lookup
	RevisitArglist bool

	OnError func(s Shape, err error, title string)

	c int
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

		if e.RevisitArglist && e.MetadataLookup != nil {
			fixupArglist(e.MetadataLookup, &fn, ob, fullname, false)
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
	case reflect.Pointer:
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
		if e.MetadataLookup != nil && ob != nil {
			fixupArglist(e.MetadataLookup, &s, ob, name, isMethod)
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

func (v *ref) Doc() string { return fmt.Sprintf("?? $ref of %v ??", v.originalRT) }
