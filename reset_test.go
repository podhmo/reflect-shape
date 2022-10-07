package reflectshape_test

import (
	"testing"

	reflectshape "github.com/podhmo/reflect-shape"
)

func TestResetName(t *testing.T) {
	type A struct {
	}
	e := reflectshape.NewExtractor()
	s := e.Extract(A{})
	if want, got := "A", s.GetName(); want != got {
		t.Errorf("ResetName(), before reset, expected type name is %s, but got %s", want, got)
	}
	reflectshape.ResetName(s, "B")
	if want, got := "B", s.GetName(); want != got {
		t.Errorf("ResetName(), after reset, expected type name is %s, but got %s", want, got)
	}

	s2 := e.Extract(A{})
	if want, got := "B", s2.GetName(); want != got {
		t.Errorf("ResetName(), after reset, expected type name is %s, but got %s", want, got)
	}
}
