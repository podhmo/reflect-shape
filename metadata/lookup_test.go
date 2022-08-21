package metadata

import (
	"go/token"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Person is person
type Person struct {
}

func TestStruct(t *testing.T) {
	type result struct {
		Name string
		Doc  string
	}

	want := result{
		Name: "Person",
		Doc:  "Person is person",
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
