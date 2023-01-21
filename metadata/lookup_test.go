package metadata

import (
	"context"
	"go/token"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Person is person
type Person struct {

	// name is the name of person
	Name string
}

func TestType(t *testing.T) {
	type result struct {
		Name          string
		Doc           string
		FieldComments map[string]string
	}

	want := result{
		Name: "Person",
		Doc:  "Person is person",
		FieldComments: map[string]string{
			"Name": "name is the name of person",
		},
	}

	fset := token.NewFileSet()
	l := NewLookup(fset)
	l.IncludeGoTestFiles = true // for test

	metadata, err := l.LookupFromType(Person{})
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	got := result{
		Name:          metadata.Name(),
		Doc:           metadata.Doc(),
		FieldComments: metadata.FieldComments(),
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("LookupFromType() mismatch (-want +got):\n%s", diff)
	}
}

// Hello is function returns greeting message
func Hello(name string) string {
	return "Hello " + name
}

func TestFunc(t *testing.T) {
	type result struct {
		Name string
		Doc  string
		Args []string
	}

	want := result{
		Name: "Hello",
		Doc:  "Hello is function returns greeting message",
		Args: []string{"name"},
	}

	fset := token.NewFileSet()
	l := NewLookup(fset)
	l.IncludeGoTestFiles = true

	metadata, err := l.LookupFromFunc(Hello)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	var args []string
	for _, p := range metadata.Args() {
		args = append(args, p.Name)
	}
	got := result{
		Name: metadata.Name(),
		Doc:  metadata.Doc(),
		Args: args,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("LookupFromFunc() mismatch (-want +got):\n%s", diff)
	}
}

type S struct{}

// Method1 is one of S
func (s *S) Method1(name string) error { return nil }

// Method2 is one of S
func (s S) Method2(ctx context.Context, name string) (result int, err error) { return 0, nil }

func TestMethod(t *testing.T) {
	type result struct {
		Name    string
		Doc     string
		Args    []string
		Returns []string
	}

	cases := []struct {
		msg    string
		want   result
		target interface{}
	}{
		{
			msg: "pointer",
			want: result{
				Name:    "Method1",
				Doc:     "Method1 is one of S",
				Args:    []string{"name"},
				Returns: []string{""},
			},
			target: (&S{}).Method1,
		},
		{
			msg: "value",
			want: result{
				Name:    "Method2",
				Doc:     "Method2 is one of S",
				Args:    []string{"ctx", "name"},
				Returns: []string{"result", "err"},
			},
			target: (S{}).Method2,
		},
	}

	fset := token.NewFileSet()
	l := NewLookup(fset)
	l.IncludeGoTestFiles = true

	for _, c := range cases {
		c := c
		t.Run(c.msg, func(t *testing.T) {
			metadata, err := l.LookupFromFunc(c.target)
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}

			var args []string
			for _, p := range metadata.Args() {
				args = append(args, p.Name)
			}
			var returns []string
			for _, p := range metadata.Returns() {
				returns = append(returns, p.Name)
			}

			got := result{
				Name:    metadata.Name(),
				Doc:     metadata.Doc(),
				Args:    args,
				Returns: returns,
			}

			if diff := cmp.Diff(c.want, got); diff != "" {
				t.Errorf("LookupFromFunc() mismatch (-want +got):\n%s", diff)
			}

		})
	}
}

// I is I
type I interface {
	// Foo is Foo
	Foo()
}

func TestInterface(t *testing.T) {
	type result struct {
		Name string
		Doc  string
	}

	var iface I
	cases := []struct {
		msg    string
		want   result
		target interface{}
	}{
		{
			msg: "value",
			want: result{
				Name: "I",
				Doc:  "I is I",
			},
			target: iface,
		},
	}

	fset := token.NewFileSet()
	l := NewLookup(fset)
	l.IncludeGoTestFiles = true

	for _, c := range cases {
		c := c
		t.Run(c.msg, func(t *testing.T) {
			metadata, err := l.LookupFromTypeForReflectType(reflect.TypeOf(func() I { return nil }).Out(0))
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			got := result{
				Name: metadata.Name(),
				Doc:  metadata.Doc(),
			}

			if diff := cmp.Diff(c.want, got); diff != "" {
				t.Errorf("LookupFromType() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
