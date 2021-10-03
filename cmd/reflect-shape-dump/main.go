package main

import (
	"log"
	"os"
	"text/template"
)

const code = `
package main

import (
	"log"
	"os"

	reflectshape "github.com/podhmo/reflect-shape"
	"github.com/podhmo/reflect-shape/arglist"

	{{.Qualifier}} "{{.Path}}"
)


func main() {
	if err := run(); err != nil {
		log.Fatalf("!! %+v", err)
	}
}

func run() error {
	e := reflectshape.NewExtractor()
	e.RevisitArglist = true
	e.ArglistLookup = arglist.NewLookup()

	var target func() {{.Qualifier}}.{{.Name}}
	s := e.Extract(target).(reflectshape.Function)
	return reflectshape.Fdump(os.Stdout, s.Returns.Values[0])
}
`

func main() {
	if err := run(); err != nil {
		log.Fatalf("!! %+v", err)
	}
}

func run() error {
	t := template.Must(template.New("code").Parse(code))
	return t.Execute(os.Stdout, map[string]string{
		"Path":      "github.com/podhmo/reflect-shape",
		"Qualifier": "x",
		"Name":      "Function",
	})
}
