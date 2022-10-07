package reflectshape_test

import (
	"go/token"
	"testing"

	reflectshape "github.com/podhmo/reflect-shape"
	"github.com/podhmo/reflect-shape/metadata"
)

type DB struct {
}

func Foo(db *DB)        {}
func Bar(anotherDB *DB) {}

func TestArglist(t *testing.T) {
	fset := token.NewFileSet()
	lookup := metadata.NewLookup(fset)

	t.Run("without-lookup", func(t *testing.T) {
		e := reflectshape.NewExtractor()
		e.MetadataLookup = nil

		{
			s := e.Extract(Foo).(reflectshape.Function)
			want := "args0"
			got := s.Params.Keys[0]
			if s.Params.Len() != 1 {
				t.Errorf("%s: invalid arg list, len(args) == %d", s.GetName(), s.Params.Len())
			}
			if want != got {
				t.Errorf("%s: args[0] name, want %q but got %q", s.GetName(), want, got)
			}
		}
		{
			s := e.Extract(Bar).(reflectshape.Function)
			want := "args0"
			got := s.Params.Keys[0]
			if s.Params.Len() != 1 {
				t.Errorf("%s: invalid arg list, len(args) == %d", s.GetName(), s.Params.Len())
			}
			if want != got {
				t.Errorf("%s: args[0] name, want %q but got %q", s.GetName(), want, got)
			}
		}
	})

	t.Run("disable", func(t *testing.T) {
		e := reflectshape.NewExtractor()
		e.MetadataLookup = lookup

		{
			s := e.Extract(Foo).(reflectshape.Function)
			want := "db"
			got := s.Params.Keys[0]
			if s.Params.Len() != 1 {
				t.Errorf("%s: invalid arg list, len(args) == %d", s.GetName(), s.Params.Len())
			}
			if want != got {
				t.Errorf("%s: args[0] name, want %q but got %q", s.GetName(), want, got)
			}
		}
		{
			s := e.Extract(Bar).(reflectshape.Function)
			want := "db"
			got := s.Params.Keys[0]
			if s.Params.Len() != 1 {
				t.Errorf("%s: invalid arg list, len(args) == %d", s.GetName(), s.Params.Len())
			}
			if want != got {
				t.Errorf("%s: args[0] name, want %q but got %q", s.GetName(), want, got)
			}
		}
	})

	t.Run("enable", func(t *testing.T) {
		e := reflectshape.NewExtractor()
		e.MetadataLookup = lookup
		e.RevisitArglist = true

		{
			s := e.Extract(Foo).(reflectshape.Function)
			want := "db"
			got := s.Params.Keys[0]
			if s.Params.Len() != 1 {
				t.Errorf("%s: invalid arg list, len(args) == %d", s.GetName(), s.Params.Len())
			}
			if want != got {
				t.Errorf("%s: args[0] name, want %q but got %q", s.GetName(), want, got)
			}
		}
		{
			s := e.Extract(Bar).(reflectshape.Function)
			want := "anotherDB"
			got := s.Params.Keys[0]
			if s.Params.Len() != 1 {
				t.Errorf("%s: invalid arg list, len(args) == %d", s.GetName(), s.Params.Len())
			}
			if want != got {
				t.Errorf("%s: args[0] name, want %q but got %q", s.GetName(), want, got)
			}
		}
	})
}
