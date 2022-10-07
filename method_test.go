package reflectshape_test

import (
	"context"
	"go/token"
	"reflect"
	"testing"

	reflectshape "github.com/podhmo/reflect-shape"
	"github.com/podhmo/reflect-shape/metadata"
)

type something struct{}

func (s *something) ExportedMethod(ctx context.Context, foo string) string            { return "" }
func (s *something) unexportedMethod(ctx context.Context) string                      { return "" } //nolint
func (s *something) AnotherExportedMethod(ctx context.Context, another string) string { return "" }

func TestMethod(t *testing.T) {
	fset := token.NewFileSet()

	e := reflectshape.NewExtractor()
	e.MetadataLookup = metadata.NewLookup(fset)
	e.MetadataLookup.IncludeGoTestFiles = true
	e.MetadataLookup.IncludeUnexported = true

	target := &something{}

	s := e.Extract(target).(reflectshape.Struct)
	mmap := s.Methods()

	t.Run("exported method", func(t *testing.T) {
		if got := len(mmap.Names); got != 2 {
			t.Errorf("unexpected number of methods found, %d", got)
		}
	})
	t.Run("unexported method", func(t *testing.T) {
		for _, name := range mmap.Names {
			if !token.IsExported(name) {
				t.Errorf("unexported method is found, %s", name)
			}
		}
	})

	t.Run("args", func(t *testing.T) {
		{
			name := "ExportedMethod"
			args := []string{"something", "ctx", "foo"}
			t.Run(name, func(t *testing.T) {
				m := mmap.Functions[name]
				if want, got := args, m.Params.Keys; !reflect.DeepEqual(want, got) {
					t.Errorf("want %v but got %v", want, got)
				}
			})
		}
		{
			name := "AnotherExportedMethod"
			args := []string{"something", "ctx", "another"}
			t.Run(name, func(t *testing.T) {
				m := mmap.Functions[name]
				if want, got := args, m.Params.Keys; !reflect.DeepEqual(want, got) {
					t.Errorf("want %v but got %v", want, got)
				}
			})
		}
	})
}
