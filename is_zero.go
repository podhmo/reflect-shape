package reflectshape

import (
	"reflect"
)

func IsZeroRecursive(rt reflect.Type, rv reflect.Value) bool {
	switch rt.Kind() {
	case reflect.Bool:
		return rv.IsZero()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.IsZero()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.IsZero()
	case reflect.Float32, reflect.Float64:
		return rv.IsZero()
	case reflect.Complex64, reflect.Complex128:
		return rv.IsZero()
	case reflect.String:
		return rv.IsZero()
	case reflect.Struct:
		for i := 0; i < rt.NumField(); i++ {
			ft := rt.Field(i)
			fv := rv.Field(i)
			if !IsZeroRecursive(ft.Type, fv) {
				return false
			}
		}
		return true
	case reflect.Pointer:
		if rv.IsNil() {
			return true
		}
		return IsZeroRecursive(rt.Elem(), rv.Elem())
	// case reflect.Invalid:
	// case reflect.Uintptr, reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
	default:
		return !rv.IsValid() || rv.IsNil()
	}
}
