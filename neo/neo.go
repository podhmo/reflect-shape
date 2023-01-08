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
			seen:   map[ID]*Shape{},
		}
	}
	return c.extractor.Extract(ob)
}

type Extractor struct {
	Config *Config

	seen     map[ID]*Shape
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

	shape, ok := e.seen[id]
	if ok {
		return shape
	}
	shape = &Shape{ID: id, Type: rt, Value: rv, Number: len(e.seen), e: e}
	e.seen[id] = shape
	return shape
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

	Number int
	e      *Extractor
}

func (s *Shape) Equal(another *Shape) bool {
	return s.ID == another.ID
}

type ID struct {
	rt reflect.Type
	pc uintptr
}
