package main

import (
	"fmt"
	"net/http"
	"reflect"

	reflectshape "github.com/podhmo/reflect-shape"
)

func main() {
	motivation1()
	fmt.Println("\n========================================\n")
	motivation2()
	fmt.Println("\n========================================\n")
	motivation3()
}

func motivation1() {
	// with reflect
	{
		fmt.Println("reflect.TypeOf() pkgpath:", reflect.TypeOf(http.ListenAndServe).PkgPath())
		// Output: reflect.TypeOf() pkgpath:

		fmt.Println("reflect.TypeOf() pkgpath:", reflect.TypeOf(http.Client{}).PkgPath())
		// Output: reflect.TypeOf() pkgpath: net/http
		fmt.Println("reflect.TypeOf() pkgpath:", reflect.TypeOf(&http.Client{}).PkgPath())
		// Output: reflect.TypeOf() pkgpath:
	}
	fmt.Println("----------------------------------------")

	// with reflect-shape
	{
		fmt.Println("reflect-shape pkgpath:", cfg.Extract(http.ListenAndServe).Package.Path)
		// Output: reflect-shape pkgpath: net/http

		fmt.Println("reflect-shape pkgpath:", cfg.Extract(&http.Client{}).Package.Path)
		// Output: reflect-shape pkgpath: net/http
	}
}

func motivation2() {
	foo := func(x, y int) {}
	bar := func(x, y int) {}

	// with reflect
	{
		fmt.Println("reflect.TypeOf() id: foo == foo?", reflect.TypeOf(foo) == reflect.TypeOf(foo))
		// Output: reflect.TypeOf() id: foo == foo? true
		fmt.Println("reflect.TypeOf() id: foo == bar?", reflect.TypeOf(foo) == reflect.TypeOf(bar))
		// Output: reflect.TypeOf() id: foo == bar? true
	}
	fmt.Println("----------------------------------------")

	// with reflect-shape
	{
		fmt.Println("reflect-shape id: foo == foo?", cfg.Extract(foo).ID == cfg.Extract(foo).ID)
		// Output: reflect-shape id: foo == foo? true
		fmt.Println("reflect-shape id: foo == bar?", cfg.Extract(foo).ID == cfg.Extract(bar).ID)
		// Output: reflect-shape id: foo == bar? false

		// or cfg.Extract(foo).Equal(cfg.Extract(bar))
	}
}

// This is Hello
func Hello(
	name string, // name of target
) {
	fmt.Println("hello", name)
}

// This is Bar
type Bar struct {
	Name string // name of Bar
}

func motivation3() {
	{
		shape := cfg.Extract(Hello)
		fmt.Println("Name", shape.Name, "kind", shape.Kind, "Doc", shape.Func().Doc())
		for _, a := range shape.Func().Args() {
			fmt.Println("--", "Arg", a.Name, "kind", a.Shape.Kind, "Doc", a.Doc)
		}
		// Output: Name Hello kind func Doc This is Hello
		// -- Arg name kind string Doc name of target
	}
	{
		shape := cfg.Extract(&Bar{})
		fmt.Println("Name", shape.Name, "kind", shape.Kind, "Doc", shape.Struct().Doc())
		for _, f := range shape.Struct().Fields() {
			fmt.Println("--", "Field", f.Name, "kind", f.Shape.Kind, "Doc", f.Doc)
		}
		// Output: Name Bar kind struct Doc This is Bar
		// -- Field Name kind string Doc name of Bar
	}
}

var cfg = &reflectshape.Config{}
