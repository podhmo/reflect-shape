package reflectshape_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	reflectshape "github.com/podhmo/reflect-shape"
)

type S0 struct{}
type S1 struct{}

func F0()         {}
func F1()         {}
func (s0 S0) M()  {}
func (s1 *S1) M() {}

var cfg = &reflectshape.Config{IncludeGoTestFiles: true}

func TestIdentity(t *testing.T) {
	type testcase struct {
		msg string
		x   any
		y   any
	}

	t.Run("equal", func(t *testing.T) {
		cases := []testcase{
			{msg: "same-struct", x: S0{}, y: S0{}},
			{msg: "same-struct-pointer", x: S0{}, y: &S0{}},
			{msg: "same-function", x: F0, y: F0},
			{msg: "same-method", x: new(S0).M, y: new(S0).M},
			{msg: "same-method-pointer", x: new(S0).M, y: (S0{}).M},
		}

		cfg := &reflectshape.Config{}
		for _, c := range cases {
			t.Run(c.msg, func(t *testing.T) {
				x := cfg.Extract(c.x)
				y := cfg.Extract(c.y)
				if !x.Equal(y) {
					t.Errorf("Shape.ID, must be %v == %v", c.x, c.y)
				}
			})
		}
	})

	t.Run("not-equal", func(t *testing.T) {
		cases := []testcase{
			{msg: "another-struct", x: S0{}, y: S1{}},
			{msg: "another-function", x: F0, y: F1},
			{msg: "another-method", x: new(S1).M, y: new(S0).M},
			{msg: "function-and-method", x: F0, y: new(S0).M},
		}

		for _, c := range cases {
			t.Run(c.msg, func(t *testing.T) {
				x := cfg.Extract(c.x)
				y := cfg.Extract(c.y)
				if x.Equal(y) {
					t.Errorf("Shape.ID, must be %v != %v", c.x, c.y)
				}
			})
		}
	})
}
func TestPointerLevel(t *testing.T) {
	type testcase struct {
		msg   string
		input any
		lv    int
	}

	cases := []testcase{
		{msg: "zero", input: S0{}, lv: 0},
		{msg: "one", input: &S0{}, lv: 1},
		{msg: "two", input: func() **S0 { s := new(S0); return &s }(), lv: 2},
		{msg: "zero-int", input: 0, lv: 0},
		{msg: "zero-slice", input: []S0{}, lv: 0},
		{msg: "one-slice", input: &[]S0{}, lv: 1},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			s := cfg.Extract(c.input)
			if want, got := c.lv, s.Lv; want != got {
				t.Errorf("Shape.Lv, must be want:%v == got:%v", want, got)
			}
		})
	}
}

func TestPackagePath(t *testing.T) {
	cases := []struct {
		msg     string
		input   any
		pkgpath string
	}{
		{msg: "struct", input: S0{}, pkgpath: "github.com/podhmo/reflect-shape_test"},
		{msg: "struct-pointer", input: &S0{}, pkgpath: "github.com/podhmo/reflect-shape_test"},
		{msg: "func", input: F0, pkgpath: "github.com/podhmo/reflect-shape_test"},
		{msg: "slice", input: []S0{}, pkgpath: ""},
		// stdlib
		{msg: "int", input: int(0), pkgpath: ""},
		{msg: "stdlib-func", input: t.Run, pkgpath: "testing"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.msg, func(t *testing.T) {
			shape := cfg.Extract(c.input)
			if want, got := c.pkgpath, shape.Package.Path; want != got {
				t.Errorf("Shape.Package.Path: %#+v != %#+v", want, got)
			}
		})
	}
}

func TestPackageScopeNames(t *testing.T) {
	t.Run("one", func(t *testing.T) {
		want := []string{"F0"}

		cfg := &reflectshape.Config{}
		shape := cfg.Extract(F0)

		if got := shape.Package.Scope().Names(); !reflect.DeepEqual(want, got) {
			t.Errorf("Package.Names(): %#+v != %#+v", want, got)
		}
	})

	t.Run("many", func(t *testing.T) {
		want := []string{"F1", "S0", "S1"}

		cfg := &reflectshape.Config{}

		cfg.Extract(S0{})
		cfg.Extract(&S0{})
		cfg.Extract(&S1{})

		cfg.Extract(new(S0).M) // ignored
		cfg.Extract(new(S1).M) // ignored

		// cfg.Extract(F0) // not seen
		shape := cfg.Extract(F1)

		if got := shape.Package.Scope().Names(); !reflect.DeepEqual(want, got) {
			t.Errorf("Package.Names(): %#+v != %#+v", want, got)
		}
	})
}

// This is Foo.
func Foo(ctx context.Context, name string, nickname *string) error {
	return nil
}

// Foo's alternative that return variables are named.
func FooWithRetNames(ctx context.Context, name string, nickname *string) (err error) {
	return nil
}

// Foo's alternative that arguments are not named.
func FooWithoutArgNames(context.Context, string, *string) error {
	return nil
}

// Foo's alternative that variadic arguments.
func FooWithVariadicArgs(ctx context.Context, name string, nickname *string, args ...any) error {
	return nil
}

func TestFunc(t *testing.T) {
	cases := []struct {
		fn           any
		args         []string
		returns      []string
		isMethod     bool
		isVariadic   bool
		fillArgNames bool
	}{
		{fn: Foo, args: []string{"ctx", "name", "nickname"}, returns: []string{""}},
		{fn: FooWithRetNames, args: []string{"ctx", "name", "nickname"}, returns: []string{"err"}},
		{fn: FooWithoutArgNames, args: []string{"", "", ""}, returns: []string{""}},
		// isVariadic
		{fn: FooWithVariadicArgs, args: []string{"ctx", "name", "nickname", "args"}, returns: []string{""}, isVariadic: true},
		// isMethod
		{fn: new(S0).M, args: nil, returns: nil, isMethod: true},
		// fillArgNames
		{fn: FooWithoutArgNames, args: []string{"ctx", "arg1", "arg2"}, returns: []string{"err"}, fillArgNames: true},
	}

	for i, c := range cases {
		c := c
		cfg := &reflectshape.Config{IncludeGoTestFiles: true, FillArgNames: c.fillArgNames, FillReturnNames: c.fillArgNames}
		t.Run(fmt.Sprintf("case%d", i), func(t *testing.T) {
			fn := cfg.Extract(c.fn).Func()
			t.Logf("%s", fn)

			{
				var got []string
				args := fn.Args()
				for _, v := range args {
					got = append(got, v.Name)
				}

				want := c.args
				type ref struct{ XS []string }
				if diff := cmp.Diff(ref{want}, ref{got}); diff != "" {
					t.Errorf("Shape.Func().Args(): -want, +got: \n%v", diff)
				}
			}

			{
				var got []string
				args := fn.Returns()
				for _, v := range args {
					got = append(got, v.Name)
				}

				want := c.returns
				type ref struct{ XS []string }
				if diff := cmp.Diff(ref{want}, ref{got}); diff != "" {
					t.Errorf("Shape.Func().Returns(): -want, +got: \n%v", diff)
				}
			}

			if want, got := c.isMethod, fn.IsMethod(); want != got {
				t.Errorf("Shape.Func().IsMethod(): want:%v != got:%v", want, got)
			}
			if want, got := c.isVariadic, fn.IsVariadic(); want != got {
				t.Errorf("Shape.Func().IsVariadic(): want:%v != got:%v", want, got)
			}
		})
	}

	t.Run("doc", func(t *testing.T) {
		want := "This is Foo."
		got := cfg.Extract(Foo).Func().Doc()
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Shape.Func().Doc(): -want, +got: \n%v", diff)
		}
	})
	// PANIC (not supported)
	// fmt.Println(cfg.Extract(func(fmt string, args ...any) {}).MustFunc())
}

// Wrap type
type Wrap[T any] struct {
	Value T
}

// Person object
type Person struct {
	Name     string // name of person
	Father   *Person
	Children []*Person
}

func TestStruct(t *testing.T) {
	cases := []struct {
		ob     any
		name   string
		fields []string
		docs   []string
	}{
		{name: "Person", ob: Person{}, fields: []string{"Name", "Father", "Children"}, docs: []string{"name of person", "", ""}},
		{name: "Person", ob: &Person{}, fields: []string{"Name", "Father", "Children"}},
	}

	for i, c := range cases {
		c := c
		t.Run(fmt.Sprintf("case%d", i), func(t *testing.T) {
			s := cfg.Extract(c.ob).Struct()
			t.Logf("%s", s)

			if want, got := c.name, s.Name(); want != got {
				t.Errorf("Shape.Struct().Name():  want:%v != got:%v", want, got)
			}

			{
				var got []string
				fields := s.Fields()
				for _, v := range fields {
					got = append(got, v.Name)
				}
				if want := c.fields; !reflect.DeepEqual(want, got) {
					t.Errorf("Shape.Struct().Fields(): names, want:%#+v != got:%#+v", want, got)
				}
			}

			if c.docs != nil {
				var got []string
				fields := s.Fields()
				for _, v := range fields {
					got = append(got, v.Doc)
				}
				if want := c.docs; !reflect.DeepEqual(want, got) {
					t.Errorf("Shape.Struct().Fields(): docs, want:%#+v != got:%#+v", want, got)
				}
			}
		})
	}

	t.Run("doc-generics", func(t *testing.T) {
		want := "Wrap type"
		got := cfg.Extract(Wrap[int]{Value: 10}).Struct().Doc()
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Shape.Struct().Doc(): -want, +got: \n%v", diff)
		}
	})
}

func UseContext(ctx context.Context) {}

func TestInterface(t *testing.T) {
	cases := []struct {
		input   any
		modify  func(*reflectshape.Shape) *reflectshape.Interface
		name    string
		methods []string
	}{
		{name: "Context", input: UseContext, methods: []string{"Deadline", "Done", "Err", "Value"},
			modify: func(s *reflectshape.Shape) *reflectshape.Interface {
				return s.Func().Args()[0].Shape.Interface()
			}},
	}

	for i, c := range cases {
		c := c
		t.Run(fmt.Sprintf("case%d", i), func(t *testing.T) {
			iface := c.modify(cfg.Extract(c.input))
			t.Logf("%s", iface)

			if want, got := c.name, iface.Name(); want != got {
				t.Errorf("Shape.Interface().Name():  want:%v != got:%v", want, got)
			}

			{
				var got []string
				fields := iface.Methods()
				for _, v := range fields {
					got = append(got, v.Name)
				}
				if want := c.methods; !reflect.DeepEqual(want, got) {
					t.Errorf("Shape.Interface().Methods(): names, want:%#+v != got:%#+v", want, got)
				}
			}
		})
	}
}

// Ordering is desc or asc
type Ordering string

func TestNamed(t *testing.T) {
	cases := []struct {
		input any
		name  string
		doc   string
	}{
		{input: Ordering("desc"), name: "Ordering", doc: "Ordering is desc or asc"},
		{input: &Person{}, name: "Person", doc: "Person object"},
	}

	for i, c := range cases {
		c := c
		t.Run(fmt.Sprintf("case%d", i), func(t *testing.T) {
			got := cfg.Extract(c.input).Named()
			t.Logf("%s", got)

			if want, got := c.name, got.Name(); want != got {
				t.Errorf("Shape.Type().Name():  want:%v != got:%v", want, got)
			}
			if want, got := c.doc, got.Doc(); want != got {
				t.Errorf("Shape.Type().Doc():  want:%v != got:%v", want, got)
			}
		})
	}
}
