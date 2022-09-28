package metadata

import (
	"context"
	"go/token"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Person is person
type Person struct {

	// name is the name of person
	Name string
}

func TestStruct(t *testing.T) {
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

	metadata, err := l.LookupFromStruct(Person{})
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	got := result{
		Name:          metadata.Name(),
		Doc:           metadata.Doc(),
		FieldComments: metadata.FieldComments(),
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("LookupFromStruct() mismatch (-want +got):\n%s", diff)
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
	got := result{
		Name: metadata.Name(),
		Doc:  metadata.Doc(),
		Args: metadata.Args(),
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
		Name string
		Doc  string
		Args []string
	}

	want := result{
		Name: "Method1",
		Doc:  "Method1 is one of S",
		Args: []string{"name"},
	}

	fset := token.NewFileSet()
	l := NewLookup(fset)
	l.IncludeGoTestFiles = true

	metadata, err := l.LookupFromFunc((&S{}).Method1)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	got := result{
		Name: metadata.Name(),
		Doc:  metadata.Doc(),
		Args: metadata.Args(),
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("LookupFromFunc() mismatch (-want +got):\n%s", diff)
	}
}
