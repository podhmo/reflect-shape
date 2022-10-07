package main

import (
	"fmt"
	"go/token"

	reflectshape "github.com/podhmo/reflect-shape"
	"github.com/podhmo/reflect-shape/metadata"
)

type Person struct {
	Name   string
	Father *Person
}

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
	fmt.Println("----------------------------------------")

	// shape's fields are [Name, Father]
	fmt.Printf("shape's fields are %v\n", shape.Fields.Keys)

	// shape's methods are [Hello]
	fmt.Printf("shape's methods are %v\n", shape.Methods().Names)

	fmt.Println("----------------------------------------")
	// shape's verbose output is *main.Person{Name, Father}
	fmt.Printf("shape's verbose output is %+v\n", shape)
}
