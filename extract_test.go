package reflectshape_test

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"testing"
	"time"

	reflectshape "github.com/podhmo/reflect-shape"
)

type EmitFunc func(ctx context.Context, w io.Writer) error

type Person struct {
	Name string `json:"name"`
	Age  int
}

func TestPrimitive(t *testing.T) {
	type MyInt int // new type
	type MyInt2 = int

	cases := []struct {
		msg    string
		input  interface{}
		output string
	}{
		{msg: "int", input: 1, output: "int"},
		{msg: "new type", input: MyInt(1), output: "github.com/podhmo/reflect-shape_test.MyInt"},
		{msg: "type alias", input: MyInt2(1), output: "int"},
	}
	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			got := reflectshape.Extract(c.input)
			if _, ok := got.(reflectshape.Primitive); !ok {
				t.Errorf("expected Primitive, but %T", got)
			}
			// format
			if want, got := c.output, fmt.Sprintf("%v", got); want != got {
				t.Errorf("expected string expression is %q but %q", want, got)
			}
		})
	}
}

func TestStruct(t *testing.T) {
	t.Run("user defined", func(t *testing.T) {
		got := reflectshape.Extract(Person{})
		v, ok := got.(reflectshape.Struct)
		if !ok {
			t.Errorf("expected Struct, but %T", got)
		}

		if len(v.Fields.Values) != 2 {
			t.Errorf("expected the number of Person's fields is 1, but %v", len(v.Fields.Values))
		}

		if got := v.FieldName(0); got != "name" {
			t.Errorf("expected field name with json tag is %q, but %q", "name", got)
		}
		if got := v.FieldName(1); got != "Age" {
			t.Errorf("expected field name without json tag is %q, but %q", "name", got)
		}

		// format
		if got, want := fmt.Sprintf("%v", got), "github.com/podhmo/reflect-shape_test.Person"; want != got {
			t.Errorf("expected string expression is %q but %q", want, got)
		}
	})

	t.Run("time.Time", func(t *testing.T) {
		var z time.Time
		got := reflectshape.Extract(z)
		if _, ok := got.(reflectshape.Struct); !ok {
			t.Errorf("expected Struct, but %T", got)
		}

		// format
		if got := fmt.Sprintf("%v", got); got != "time.Time" {
			t.Errorf("expected string expression is %q but %q", "int", got)
		}
	})
}

func TestContainer(t *testing.T) {
	t.Run("slice", func(t *testing.T) {
		t.Run("primitive", func(t *testing.T) {
			got := reflectshape.Extract([]int{})
			v, ok := got.(reflectshape.Container)
			if !ok {
				t.Errorf("expected Container, but %T", got)
			}
			if got := len(v.Args); got != 1 {
				t.Errorf("expected the length of slices's args is %v, but %v", 1, got)
			}

			if got, want := fmt.Sprintf("%v", got), "slice[int]"; want != got {
				t.Errorf("expected string expression is %q but %q", want, got)
			}
		})
		t.Run("primitive has len", func(t *testing.T) {
			got := reflectshape.Extract([]int{1, 2, 3})
			v, ok := got.(reflectshape.Container)
			if !ok {
				t.Errorf("expected Container, but %T", got)
			}
			if got := len(v.Args); got != 1 {
				t.Errorf("expected the length of slices's args is %v, but %v", 1, got)
			}

			if got, want := fmt.Sprintf("%v", got), "slice[int]"; want != got {
				t.Errorf("expected string expression is %q but %q", want, got)
			}
		})
		t.Run("struct", func(t *testing.T) {
			got := reflectshape.Extract([]Person{})
			v, ok := got.(reflectshape.Container)
			if !ok {
				t.Errorf("expected Container, but %T", got)
			}
			if got := len(v.Args); got != 1 {
				t.Errorf("expected the length of slices's args is %v, but %v", 1, got)
			}

			// format
			if got, want := fmt.Sprintf("%v", got), "slice[github.com/podhmo/reflect-shape_test.Person]"; want != got {
				t.Errorf("expected string expression is %q but %q", want, got)
			}
		})
	})

	t.Run("map", func(t *testing.T) {
		t.Run("primitive", func(t *testing.T) {
			got := reflectshape.Extract(map[string]int{})
			v, ok := got.(reflectshape.Container)
			if !ok {
				t.Errorf("expected Container, but %T", got)
			}
			if got := len(v.Args); got != 2 {
				t.Errorf("expected the length of slices's args is %v, but %v", 1, got)
			}

			// format
			if got, want := fmt.Sprintf("%v", got), "map[string, int]"; want != got {
				t.Errorf("expected string expression is %q but %q", want, got)
			}
		})
		t.Run("primitive has len", func(t *testing.T) {
			got := reflectshape.Extract(map[string]int{"foo": 20})
			v, ok := got.(reflectshape.Container)
			if !ok {
				t.Errorf("expected Container, but %T", got)
			}
			if got := len(v.Args); got != 2 {
				t.Errorf("expected the length of slices's args is %v, but %v", 1, got)
			}

			// format
			if got, want := fmt.Sprintf("%v", got), "map[string, int]"; want != got {
				t.Errorf("expected string expression is %q but %q", want, got)
			}
		})
		t.Run("struct", func(t *testing.T) {
			got := reflectshape.Extract(map[string][]Person{})
			v, ok := got.(reflectshape.Container)
			if !ok {
				t.Errorf("expected Container, but %T", got)
			}
			if got := len(v.Args); got != 2 {
				t.Errorf("expected the length of slices's args is %v, but %v", 1, got)
			}

			// format
			if got, want := fmt.Sprintf("%v", got), "map[string, slice[github.com/podhmo/reflect-shape_test.Person]]"; want != got {
				t.Errorf("expected string expression is %q but %q", want, got)
			}
		})
	})
}

type ListUserInput struct {
	Query string
	Limit int
}

func ListUser(ctx context.Context, input ListUserInput) ([]Person, error) {
	return nil, nil
}

func TestFunction(t *testing.T) {
	cases := []struct {
		msg    string
		fn     interface{}
		output string
	}{
		{
			msg:    "actual-func",
			fn:     ListUser,
			output: "github.com/podhmo/reflect-shape_test.ListUser(context.Context, github.com/podhmo/reflect-shape_test.ListUserInput) (slice[github.com/podhmo/reflect-shape_test.Person], error)",
		},
		{
			msg:    "simple",
			fn:     func(x, y int) int { return 0 },
			output: "github.com/podhmo/reflect-shape_test.TestFunction.func(int, int) (int)",
		},
		{
			msg:    "with-context",
			fn:     func(ctx context.Context, x, y int) int { return 0 },
			output: "github.com/podhmo/reflect-shape_test.TestFunction.func(context.Context, int, int) (int)",
		},
		{
			msg:    "with-error",
			fn:     func(x, y int) (int, error) { return 0, nil },
			output: "github.com/podhmo/reflect-shape_test.TestFunction.func(int, int) (int, error)",
		},
		{
			msg:    "without-returns",
			fn:     func(s string) {},
			output: "github.com/podhmo/reflect-shape_test.TestFunction.func(string) ()",
		},
		{
			msg:    "without-params",
			fn:     func() string { return "" },
			output: "github.com/podhmo/reflect-shape_test.TestFunction.func() (string)",
		},
		{
			msg:    "with-pointer",
			fn:     func(*string) string { return "" },
			output: "github.com/podhmo/reflect-shape_test.TestFunction.func(*string) (string)",
		},
		{
			msg: "var",
			fn: func() interface{} {
				var handler EmitFunc = func(context.Context, io.Writer) error { return nil }
				return handler
			}(),
			output: "github.com/podhmo/reflect-shape_test.TestFunction.func(context.Context, io.Writer) (error)",
		},
		{
			msg: "var-nil",
			fn: func() interface{} {
				var handler EmitFunc
				return handler
			}(),
			output: "func(context.Context, io.Writer) (error)",
		},
	}

	rx := regexp.MustCompile(`func\d+(\.\d+)?\(`) // closure name is func<N> or func<M>.<N>
	normalize := func(s string) string { return rx.ReplaceAllString(s, "func(") }

	for _, c := range cases {
		c := c
		t.Run(c.msg, func(t *testing.T) {
			got := reflectshape.Extract(c.fn)
			_, ok := got.(reflectshape.Function)
			if !ok {
				t.Errorf("expected Container, but %T", got)
			}
			if got, want := normalize(fmt.Sprintf("%v", got)), c.output; want != got {
				t.Errorf("expected string expression is %q but %q", want, got)
			}
		})
	}
}

func TestRecursion(t *testing.T) {
	type Person struct {
		Name      *string     `json:"name"`
		Age       int         `json:"age"`
		CreatedAt time.Time   `json:"createdAt"`
		ExpiredAt **time.Time `json:"expiredAt"`
		Father    ********Person
		Mother    *Person
		Children  []Person
	}
	cases := []struct {
		name   string
		output string
	}{
		{
			name:   "Name",
			output: "*string",
		},
		{
			name:   "Age",
			output: "int",
		},
		{
			name:   "CreatedAt",
			output: "time.Time",
		},
		{
			name:   "ExpiredAt",
			output: "**time.Time",
		},
		{
			name:   "Father",
			output: "********github.com/podhmo/reflect-shape_test.Person",
		},
		{
			name:   "Mother",
			output: "*github.com/podhmo/reflect-shape_test.Person",
		},
		{
			name:   "Children",
			output: "slice[github.com/podhmo/reflect-shape_test.Person]",
		},
	}

	e := &reflectshape.Extractor{Seen: map[reflect.Type]reflectshape.Shape{}}
	ob := Person{}
	e.Extract(ob)

	rv := reflect.ValueOf(ob)
	rt := rv.Type()

	for _, c := range cases {
		t.Run(c.output, func(t *testing.T) {
			f, ok := rt.FieldByName(c.name)
			if !ok {
				t.Fatalf("missing field %v", c.name)
			}
			got := e.Seen[f.Type]

			if got, want := fmt.Sprintf("%v", got), c.output; want != got {
				t.Errorf("expected string expression is %q but %q", want, got)
			}
		})
	}
}

func TestDeref(t *testing.T) {
	type Person struct {
		Name *string `json:"name"`
	}

	ob := &Person{}
	s := reflectshape.Extract(ob)

	got := s.(reflectshape.Struct).Fields.Values[0]
	want := reflectshape.Primitive{}
	t.Logf("%T %T\n", got, want)

	if got, want := reflect.TypeOf(got), reflect.TypeOf(want); !got.AssignableTo(want) {
		t.Errorf("unexpected type is found. expected %s, but %s", got, want)
	}
}

func TestPool(t *testing.T) {
	e := &reflectshape.Extractor{Seen: map[reflect.Type]reflectshape.Shape{}}

	f := func() {}
	g := func() {}

	t.Run("f,g", func(t *testing.T) {
		s0 := e.Extract(f)
		s1 := e.Extract(g)
		t.Logf("s0=%v s1=%v", s0.GetIdentity(), s1.GetIdentity())

		if reflect.DeepEqual(s0, s1) {
			t.Errorf("unexpected equal %v %v", s0, s1)
		}
	})
	t.Run("f,f", func(t *testing.T) {
		s0 := e.Extract(f)
		s1 := e.Extract(f)
		t.Logf("s0=%v s1=%v", s0.GetIdentity(), s1.GetIdentity())

		if !reflect.DeepEqual(s0, s1) {
			t.Errorf("unexpected not equal %v %v", s0, s1)
		}
	})
}
