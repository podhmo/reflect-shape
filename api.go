package reflectshape

import "reflect"

func Extract(ob interface{}) Shape {
	e := &Extractor{
		Seen: map[reflect.Type]Shape{},
	}
	return e.Extract(ob)
}

func NewExtractor() *Extractor {
	return &Extractor{
		Seen: map[reflect.Type]Shape{},
	}
}

// unsafe
func ResetName(s Shape, name string) {
	v := s.info()
	v.Name = name
}

func ResetPackage(s Shape, name string) {
	v := s.info()
	v.Package = name
}
func ResetReflectType(s Shape, rt reflect.Type) {
	v := s.info()
	v.reflectType = rt
	v.identity = ""
}
