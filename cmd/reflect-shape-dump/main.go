package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	log.SetFlags(0)
	if err := run(); err != nil {
		log.Fatalf("!! %+v", err)
	}
}

func run() error {
	if len(os.Args) <= 1 {
		log.Fatalf("please run <cmd> <fullpath>\n\t e.g. go run <> github.com/podhmo/reflect-shape.Struct\n")
	}
	t := template.Must(template.New("code").Parse(code))
	fullpath := os.Args[1]
	parts := strings.Split(fullpath, ".")
	path := strings.Join(parts[:len(parts)-1], ".")
	name := parts[len(parts)-1]

	log.Printf("check by go list.\tpath=%q\tname=%q", path, name)
	ctx := context.Background()
	if err := exec.CommandContext(ctx, "go", "list", path).Run(); err != nil {
		return err
	}

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
		"Path":      path,
		"Qualifier": "x",
		"Name":      name,
	}); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "go", "run", filepath.Join(d, "main.go"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
