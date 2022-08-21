package metadata

import (
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
	l.IncludeGoTestFiles = true

	metadata, err := l.LookupFromStruct(Person{})
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	got := result{
		Name: metadata.Name(),
		Doc:  metadata.Doc(),
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
