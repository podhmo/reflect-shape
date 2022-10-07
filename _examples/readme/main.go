package main

// run with `go run ./_examples/readme`

import (
	"fmt"
	"go/token"

	reflectshape "github.com/podhmo/reflect-shape"
	"github.com/podhmo/reflect-shape/metadata"
)

// Person is person object
type Person struct {
	Name   string
	Father *Person
}

// Hello returns a greeting message.
func (p *Person) Hello() string {
	return fmt.Sprintf("%s: hello", p.Name)
}

func main() {
	extractor := reflectshape.NewExtractor()
	extractor.RevisitArglist = true
	extractor.MetadataLookup = metadata.NewLookup(token.NewFileSet())

	shape := extractor.Extract(&Person{}).(reflectshape.Struct) // or reflectshape.Container or reflectshape.Function or reflectshape.Primitive

	// shape is main.Person
	fmt.Printf("shape is %v\n", shape)
	// shape's doc is "Person is person object"
	fmt.Printf("shape's doc is %q\n", shape.Doc())
	fmt.Println("----------------------------------------")

	// shape's fields are [Name, Father]
	fmt.Printf("shape's fields are %v\n", shape.Fields.Keys)

	// shape's methods are [Hello]
	fmt.Printf("shape's methods are %v\n", shape.Methods().Names)
	// shape.Hello() 's doc is "Hello returns a greeting message."
	fmt.Printf("shape.Hello() 's doc is %q\n", shape.Methods().Functions["Hello"].Doc())

	fmt.Println("----------------------------------------")
	// shape's verbose output is *main.Person{Name, Father}
	fmt.Printf("shape's verbose output is %+v\n", shape)
}
