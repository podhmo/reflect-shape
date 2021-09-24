package arglist

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"
)

func inspectFuncFromFile(f *ast.File, name string) (NameSet, error) {
	ob := f.Scope.Lookup(name)
	if ob == nil {
		return NameSet{}, fmt.Errorf("not found %q", name)
	}
	decl, ok := ob.Decl.(*ast.FuncDecl)
	if !ok {
		return NameSet{}, fmt.Errorf("unexpected decl %T", ob)
	}
	return InspectFunc(decl, true)
}

func TestInspectFunc(t *testing.T) {
	const code = `package foo
func Sum(x int, y,z int) int {
	return x + y + z
}
func Sum2(xs ...int) int {
	return 0
}
func Sprintf(ctx context.Context, fmt string, vs ...interface{}) (string, error) {
	return fmt.Sprintf(fmt, vs...), nil
}
func Sprintf2(ctx context.Context, fmt string, vs ...interface{}) (s string, err error) {
	return fmt.Sprintf(fmt, vs...), nil
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "foo.go", code, parser.ParseComments)
	if err != nil {
		t.Fatalf("something wrong in parse-file %v", err)
	}

	cases := []struct {
		name string
		want NameSet
	}{
		{
			name: "Sum",
			want: NameSet{
				Name:    "Sum",
				Args:    []string{"x", "y", "z"},
				Returns: []string{"ret0"},
			},
		},
		{
			name: "Sum2",
			want: NameSet{
				Name:    "Sum2",
				Args:    []string{"*xs"},
				Returns: []string{"ret0"},
			},
		},
		{
			name: "Sprintf",
			want: NameSet{
				Name:    "Sprintf",
				Args:    []string{"ctx", "fmt", "*vs"},
				Returns: []string{"ret0", "ret1"},
			},
		},
		{
			name: "Sprintf2",
			want: NameSet{
				Name:    "Sprintf2",
				Args:    []string{"ctx", "fmt", "*vs"},
				Returns: []string{"s", "err"},
			},
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got, err := inspectFuncFromFile(f, c.name)
			if err != nil {
				t.Fatalf("unexpected error %+v", err)
			}
			if !reflect.DeepEqual(c.want, got) {
				t.Errorf("want:\n\t%q\nbut got:\n\t%q\n", c.want, got)
			}
		})
	}
}

func TestInspectAnonymousFunc(t *testing.T) {
	l := NewLookup()

	inner0 := func(x string) (string, error) { return "", nil }
	var outer0inner func(string) string
	outer0 := func(x string) (string, error) {
		outer0inner := func(y string) string { return "" }
		return outer0inner(x), nil
	}
	_ = outer0

	cases := []struct {
		msg    string
		fn     interface{}
		want   NameSet
		hasErr bool
	}{
		{
			msg: "simple",
			fn:  inner0,
			want: NameSet{
				Name:    "",
				Args:    []string{"x"},
				Returns: []string{"ret0", "ret1"},
			},
		},
		{
			msg: "nested",
			fn:  outer0inner,
			want: NameSet{
				Name:    "",
				Args:    []string{"y"},
				Returns: []string{"ret0"},
			},
			hasErr: true, // not supported yet
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.msg, func(t *testing.T) {
			if c.fn == nil {
				t.Fatalf("unexpected input %+v", c.fn)
			}

			ns, err := l.LookupNameSetFromFunc(c.fn)
			if c.hasErr {
				if err == nil {
					t.Errorf("error is expected, but not error is occured")
				}
				if len(ns.Args) != 0 {
					t.Errorf("len(ns.Args) == 0 is expected, but got %d", len(ns.Args))
				}
				if len(ns.Returns) != 0 {
					t.Errorf("len(ns.Returns) == 0 is expected, but got %d", len(ns.Returns))
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error %+v", err)
			}
			if want, got := c.want, ns; !reflect.DeepEqual(want, got) {
				t.Errorf("want:\n\t%#+v\nbut got:\n\t%#+v", want, got)
			}
		})
	}
}

type Foo struct{}

func (f *Foo) Message(name string, age int) string {
	return ""
}
func (f *Foo) Message2(
	name string,
	age int,
) string {
	return ""
}

func TestInspectMethod(t *testing.T) {
	l := NewLookup()

	cases := []struct {
		msg    string
		name   string
		fn     interface{}
		want   NameSet
		hasErr bool
	}{
		{
			msg: "single-line",
			fn:  (&Foo{}).Message,
			want: NameSet{
				Name:    "Message",
				Recv:    "f",
				Args:    []string{"name", "age"},
				Returns: []string{"ret0"},
			},
		},
		{
			msg: "multi-line",
			fn:  (&Foo{}).Message2,
			want: NameSet{
				Name:    "Message2",
				Recv:    "f",
				Args:    []string{"name", "age"},
				Returns: []string{"ret0"},
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.msg, func(t *testing.T) {
			if c.fn == nil {
				t.Fatalf("unexpected input %+v", c.fn)
			}

			ns, err := l.LookupNameSetFromFunc(c.fn)
			if c.hasErr {
				if err == nil {
					t.Errorf("error is expected, but not error is occured")
				}
				if len(ns.Args) != 0 {
					t.Errorf("len(ns.Args) == 0 is expected, but got %d", len(ns.Args))
				}
				if len(ns.Returns) != 0 {
					t.Errorf("len(ns.Returns) == 0 is expected, but got %d", len(ns.Returns))
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error %+v", err)
			}
			if want, got := c.want, ns; !reflect.DeepEqual(want, got) {
				t.Errorf("want:\n\t%#+v\nbut got:\n\t%#+v", want, got)
			}
		})
	}
}
