package reflectshape_test

import (
	"context"
	"fmt"
	"go/token"
	"io"
	"reflect"
	"regexp"
	"testing"
	"time"

	reflectshape "github.com/podhmo/reflect-shape"
	"github.com/podhmo/reflect-shape/metadata"
)

type EmitFunc func(ctx context.Context, w io.Writer) error

func TestPrimitive(t *testing.T) {
	type MyInt int // new type
	type MyInt2 = int

	i := 0
	cases := []struct {
		msg    string
		input  interface{}
		output string
	}{
		{msg: "int", input: 1, output: "int"},
		{msg: "new type", input: MyInt(1), output: "github.com/podhmo/reflect-shape_test.MyInt"},
		{msg: "type alias", input: MyInt2(1), output: "int"},
		{msg: "pointer", input: &i, output: "*int"},
	}
	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			got := reflectshape.Extract(c.input)
			if _, ok := got.(reflectshape.Primitive); !ok {
				t.Errorf("Extract(), expected type is Primitive, but %T", got)
			}
			// format
			if want, got := c.output, fmt.Sprintf("%v", got); want != got {
				t.Errorf("Extract(), expected string expression is %q but %q", want, got)
			}
		})
	}
}

func TestStruct(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
		Age  int
	}

	cases := []struct {
		msg        string
		input      interface{}
		fieldNames []string
		output     string
	}{
		{msg: "user defined", input: Person{}, fieldNames: []string{"Name", "Age"}, output: "github.com/podhmo/reflect-shape_test.Person"},
		{msg: "stdlib", input: time.Now(), fieldNames: []string{"wall", "ext", "loc"}, output: "time.Time"},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			s := reflectshape.Extract(c.input)
			got, ok := s.(reflectshape.Struct)
			if !ok {
				t.Fatalf("Extract(), expected type is Struct, but %T", s)
			}
			if want, got := c.fieldNames, got.Fields.Keys; !reflect.DeepEqual(want, got) {
				t.Fatalf("Extract(), expected fieldNames is %v, but %v", want, got)
			}

			// format
			if want, got := c.output, fmt.Sprintf("%v", got); want != got {
				t.Errorf("Extract(), expected string expression is %q but %q", want, got)
			}
		})
	}
}

func TestContainer(t *testing.T) {
	type V struct{}

	cases := []struct {
		msg    string
		input  interface{}
		output string
	}{
		{msg: "slice-primitive", input: []int{}, output: "slice[int]"},
		{msg: "slice-primitive2", input: []int{1, 2, 3}, output: "slice[int]"},
		{msg: "slice-struct", input: []V{}, output: "slice[github.com/podhmo/reflect-shape_test.V]"},
		{msg: "map-primitive", input: map[string]int{}, output: "map[string, int]"},
		{msg: "map-primitive2", input: map[string]int{"foo": 20}, output: "map[string, int]"},
		{msg: "map-primitive3", input: map[string]**int{}, output: "map[string, **int]"},
		{msg: "map-struct", input: map[string]V{}, output: "map[string, github.com/podhmo/reflect-shape_test.V]"},
	}
	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			s := reflectshape.Extract(c.input)
			got, ok := s.(reflectshape.Container)
			if !ok {
				t.Errorf("Extract(), expected type is Container, but %T", s)
			}
			// format
			if want, got := c.output, fmt.Sprintf("%v", got); want != got {
				t.Errorf("Extract(), expected string expression is %q but %q", want, got)
			}
		})
	}
}

type ListUserInput struct {
	Query string
	Limit int
}

type Person struct {
	Name string `json:"name"`
	Age  int
}

func ListUser(ctx context.Context, input ListUserInput) ([]Person, error) {
	return nil, nil
}

func TestFunction(t *testing.T) {
	cases := []struct {
		msg    string
		input  interface{}
		output string
	}{
		{
			msg:    "actual-func",
			input:  ListUser,
			output: "github.com/podhmo/reflect-shape_test.ListUser(context.Context, github.com/podhmo/reflect-shape_test.ListUserInput) (slice[github.com/podhmo/reflect-shape_test.Person], error)",
		},
		{
			msg:    "simple",
			input:  func(x, y int) int { return 0 },
			output: "github.com/podhmo/reflect-shape_test.TestFunction.func(int, int) (int)",
		},
		{
			msg:    "with-context",
			input:  func(ctx context.Context, x, y int) int { return 0 },
			output: "github.com/podhmo/reflect-shape_test.TestFunction.func(context.Context, int, int) (int)",
		},
		{
			msg:    "with-error",
			input:  func(x, y int) (int, error) { return 0, nil },
			output: "github.com/podhmo/reflect-shape_test.TestFunction.func(int, int) (int, error)",
		},
		{
			msg:    "without-returns",
			input:  func(s string) {},
			output: "github.com/podhmo/reflect-shape_test.TestFunction.func(string) ()",
		},
		{
			msg:    "without-params",
			input:  func() string { return "" },
			output: "github.com/podhmo/reflect-shape_test.TestFunction.func() (string)",
		},
		{
			msg:    "with-pointer",
			input:  func(*string) string { return "" },
			output: "github.com/podhmo/reflect-shape_test.TestFunction.func(*string) (string)",
		},
		{
			msg: "var",
			input: func() interface{} {
				var handler EmitFunc = func(context.Context, io.Writer) error { return nil }
				return handler
			}(),
			output: "github.com/podhmo/reflect-shape_test.TestFunction.func(context.Context, io.Writer) (error)",
		},
		{
			msg: "var-nil",
			input: func() interface{} {
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
			s := reflectshape.Extract(c.input)
			got, ok := s.(reflectshape.Function)
			if !ok {
				t.Errorf("Extract(), expected type is Function, but %T", s)
			}
			if want, got := c.output, normalize(fmt.Sprintf("%v", got)); want != got {
				t.Errorf("Extract(), expected string expression is %q but %q", want, got)
			}
		})
	}
}

type DB struct {
}

func Foo(db *DB)        {}
func Bar(anotherDB *DB) {}

func TestFunctionArglistOfSameSignature(t *testing.T) {
	lookup := metadata.NewLookup(token.NewFileSet())
	cases := []struct {
		msg            string
		revisitArgList bool
		lookup         *metadata.Lookup
		input0         interface{}
		args0          []string
		input1         interface{}
		args1          []string
	}{
		{msg: "no-lookup", lookup: nil, revisitArgList: false, input0: Foo, args0: []string{"args0"}, input1: Bar, args1: []string{"args0"}},
		{msg: "revisit-disabled", lookup: lookup, revisitArgList: false, input0: Foo, args0: []string{"db"}, input1: Bar, args1: []string{"db"}}, // shared
		{msg: "revisit-enabled", lookup: lookup, revisitArgList: true, input0: Foo, args0: []string{"db"}, input1: Bar, args1: []string{"anotherDB"}},
	}
	for _, c := range cases {
		c := c
		t.Run(c.msg, func(t *testing.T) {
			e := reflectshape.NewExtractor()
			e.MetadataLookup = c.lookup
			e.RevisitArglist = c.revisitArgList
			{
				s := e.Extract(c.input0).(reflectshape.Function)
				if want, got := c.args0, s.Params.Keys; !reflect.DeepEqual(want, got) {
					t.Errorf("Extract(), %s's expected args is %v but got %v", s.Name, want, got)
				}
			}
			{
				s := e.Extract(c.input1).(reflectshape.Function)
				if want, got := c.args1, s.Params.Keys; !reflect.DeepEqual(want, got) {
					t.Errorf("Extract(), %s's expected args is %v but got %v", s.Name, want, got)
				}
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
