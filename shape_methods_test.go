package reflectshape_test

import (
	"context"
	"reflect"
	"testing"

	reflectshape "github.com/podhmo/reflect-shape"
	"github.com/podhmo/reflect-shape/arglist"
)

type something struct{}

func (s *something) ExportedMethod(ctx context.Context) string        { return "" }
func (s *something) unexportedMethod(ctx context.Context) string      { return "" } //nolint
func (s *something) AnotherExportedMethod(ctx context.Context) string { return "" }

func TestMethod(t *testing.T) {
	e := reflectshape.NewExtractor()
	e.ArglistLookup = arglist.NewLookup()

	target := &something{}

	s := e.Extract(target).(reflectshape.Struct)
	mmap := s.Methods()

	t.Run("exported method", func(t *testing.T) {
		if got := len(mmap.Keys); got != 2 {
			t.Errorf("unexpected number of methods found, %d", got)
		}
	})
	t.Run("unexported method", func(t *testing.T) {
		for _, name := range s.Fields.Keys {
			if name == "unexportedMethod" {
				t.Errorf("unexported method is not extracting target")
			}
		}
	})
	t.Run("types", func(t *testing.T) {
		for i, m := range mmap.Values {
			name := mmap.Keys[i]
			mt, ok := reflect.TypeOf(target).MethodByName(name)
			if !ok {
				t.Fatalf("method %s is not found", name)
			}
			if want, got := mt, m.GetReflectType(); !reflect.DeepEqual(want, got) {
				t.Errorf("want method %s type:\n\t%v\nbut got:\n\t%v", name, want, got)
			}
		}
	})
}
