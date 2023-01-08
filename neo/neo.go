package neo

import "reflect"

type Config struct {
	IncludeComments bool
	IncludeArgNames bool

	extractor *Extractor
}

func (c *Config) Extract(ob interface{}) *Shape {
	if c.extractor == nil {
		c.extractor = &Extractor{
			Config: c,
		}
	}
	return c.extractor.Extract(ob)
}

type Extractor struct {
	Config *Config

	seen     map[reflect.Type]*Shape
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

	// todo: cache
	return &Shape{ID: id, Type: rt, Value: rv, e: e}
}

type Package struct {
	ID   string
	Name string
	Path string

	shapes map[ID]*Shape
}

type Shape struct {
	ID    ID
	Type  reflect.Type
	Value reflect.Value

	e *Extractor
}

func (s *Shape) Equal(another *Shape) bool {
	return s.ID == another.ID
}

type ID struct {
	rt reflect.Type
	pc uintptr
}

func (id *ID) Kind() reflect.Kind {
	return id.rt.Kind()
}
