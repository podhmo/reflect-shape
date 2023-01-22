package reflectshape_test

import (
	"reflect"
	"testing"

	reflectshape "github.com/podhmo/reflect-shape"
)

func TestIsZeroRecursive(t *testing.T) {
	type S struct {
		Name string
		Age  int
	}

	type W struct {
		Name string
		S    S
	}

	tests := []struct {
		name string
		v    interface{}
		want bool
	}{
		{"zero-int", 0, true},
		{"int", 1, false},
		{"zero-string", "", true},
		{"string", "x", false},
		// {"nil", nil, true}, // panic
		{"nil-slice", func() []int { return nil }(), true},
		{"empty-slice", func() []int { return []int{} }(), false},
		{"nil-map", func() map[int]int { return nil }(), true},
		{"empty-slice", func() map[int]int { return map[int]int{} }(), false},
		// struct
		{"zero-struct", S{}, true},
		{"zero-struct-pointer", &S{}, true},
		{"not-zero-struct1", S{Name: "foo"}, false},
		{"not-zero-struct2", S{Age: 20}, false},
		// recursive
		{"zero-rec-struct", W{}, true},
		{"zero-rec-struct2", W{S: S{}}, true},
		{"not-zero-rec-struct", W{S: S{Age: 20}}, false},
	}
	for _, tt := range tests {
		rt := reflect.TypeOf(tt.v)
		rv := reflect.ValueOf(tt.v)
		t.Run(tt.name, func(t *testing.T) {
			if got := reflectshape.IsZeroRecursive(rt, rv); got != tt.want {
				t.Errorf("IsZeroRecursive() = %v, want %v", got, tt.want)
			}
		})
	}
}
