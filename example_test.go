package reflectshape_test

import (
	"fmt"

	reflectshape "github.com/podhmo/reflect-shape"
)

// User is the object for User.
type User struct {
	// name of User.
	Name string
	Age  int // age of User.
}

// Hello function.
func Hello(user *User /* greeting target */) {
	fmt.Println("Hello", user.Name)
}

func ExampleConfig() {
	cfg := reflectshape.Config{IncludeGoTestFiles: true}
	shape := cfg.Extract(User{})

	fmt.Printf("%s %s %s %q\n", shape.Name, shape.Kind, shape.Package.Path, shape.Struct().Doc())
	for _, f := range shape.Struct().Fields() {
		fmt.Printf("-- %s %q\n", f.Name, f.Doc)
	}

	shape2 := cfg.Extract(Hello)
	fmt.Printf("%s %s %s %q\n", shape2.Name, shape2.Kind, shape2.Package.Path, shape2.Func().Doc())
	for _, a := range shape2.Func().Args() {
		fmt.Printf("-- %s %q\n", a.Name, a.Doc)
	}

	// Output:
	// User struct github.com/podhmo/reflect-shape_test "User is the object for User."
	// -- Name "name of User."
	// -- Age "age of User."
	// Hello func github.com/podhmo/reflect-shape_test "Hello function."
	// -- user "greeting target"
}
