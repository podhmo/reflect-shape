package foo

type Foo struct {
}

func (f *Foo) Hello(name string) string {
	return "Hello " + name
}
