package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	d, err := ioutil.TempDir(".", ".reflect-shape")
	if err != nil {
		return err
	}
	defer func() {
		log.Println("remove all", d)
		if err := os.RemoveAll(d); err != nil {
			log.Println("!? something wrong in os.RemoveAll(),", err)
		}
	}()

	f, err := os.Create(filepath.Join(d, "main.go"))
	if err != nil {
		return err
	}
	defer f.Close()

	log.Println("create", filepath.Join(d, "main.go"))
	if err := t.Execute(f, map[string]string{
		"Path":      "github.com/podhmo/reflect-shape",
		"Qualifier": "x",
		"Name":      "Function",
	}); err != nil {
		return err
	}

	cmd := exec.CommandContext(context.Background(), "go", "run", filepath.Join(d, "main.go"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
