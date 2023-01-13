package neo_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/podhmo/reflect-shape/neo"
)

type S0 struct{}
type S1 struct{}

func F0()         {}
func F1()         {}
func (s0 S0) M()  {}
func (s1 *S1) M() {}

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

		cfg := &neo.Config{}
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

		cfg := &neo.Config{}
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

func TestPackageNames(t *testing.T) {
	t.Run("one", func(t *testing.T) {
		want := []string{"F0"}

		cfg := &neo.Config{}
		shape := cfg.Extract(F0)

		if got := shape.Package.Scope().Names(); !reflect.DeepEqual(want, got) {
			t.Errorf("Package.Names(): %#+v != %#+v", want, got)
		}
	})

	t.Run("many", func(t *testing.T) {
		want := []string{"F1", "S0", "S1"}

		cfg := &neo.Config{}

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

func TestFunc(t *testing.T) {
	cases := []struct {
		fn       any
		name     string
		args     []string
		returns  []string
		isMethod bool
	}{
		{fn: Foo, args: []string{"ctx", "name", "nickname"}, returns: []string{""}},
		{fn: FooWithRetNames, args: []string{"ctx", "name", "nickname"}, returns: []string{"err"}},
		{fn: FooWithoutArgNames, args: []string{"", "", ""}, returns: []string{""}},
		{fn: new(S0).M, args: nil, returns: nil, isMethod: true},
	}

	cfg := &neo.Config{}
	for i, c := range cases {
		c := c

		t.Run(fmt.Sprintf("case:%d", i), func(t *testing.T) {
			fn := cfg.Extract(c.fn).MustFunc()
			t.Logf("%s", fn)

			{
				var got []string
				args := fn.Args()
				for _, v := range args {
					got = append(got, v.Name)
				}
				if want := c.args; !reflect.DeepEqual(want, got) {
					t.Errorf("Shape.Func().Args(): want:%#+v != got:%#+v", want, got)
				}
			}

			{
				var got []string
				args := fn.Returns()
				for _, v := range args {
					got = append(got, v.Name)
				}
				if want := c.returns; !reflect.DeepEqual(want, got) {
					t.Errorf("Shape.Func().Returns(): want:%#+v != got:%#+v", want, got)
				}
			}

			if want, got := c.isMethod, fn.IsMethod(); want != got {
				t.Errorf("Shape.Func().IsMethod(): want:%v != got:%v", want, got)
			}
		})
	}

	t.Run("doc", func(t *testing.T) {
		want := "This is Foo."
		got := cfg.Extract(Foo).MustFunc().Doc()
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Shape.Func().Doc(): -want, +got: \n%v", diff)
		}
	})

	// PANIC (not supported)
	// fmt.Println(cfg.Extract(func(fmt string, args ...any) {}).MustFunc())
}
