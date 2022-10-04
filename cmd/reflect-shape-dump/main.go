package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

const code = `
package main

import (
	"log"
	"os"
	"go/token"

	reflectshape "github.com/podhmo/reflect-shape"
	"github.com/podhmo/reflect-shape/metadata"

	{{.Qualifier}} "{{.Path}}"
)


func main() {
	log.SetFlags(0)
	if err := run(); err != nil {
		log.Fatalf("!! %+v", err)
	}
}

func run() error {
	e := reflectshape.NewExtractor()
	e.RevisitArglist = true
	fset := token.NewFileSet()
	e.MetadataLookup = metadata.NewLookup(fset)

{{ if eq .Kind "Func" }}
	s := e.Extract({{.Qualifier}}.{{.Name}})
	return reflectshape.Fdump(os.Stdout, s)
{{ else if eq .Kind "Method" }}
	ob := &{{.Qualifier}}.{{.Recv}}{}
	s := e.Extract(ob.{{.Name}})
	return reflectshape.Fdump(os.Stdout, s)
{{ else }}
	var target func() {{.Qualifier}}.{{.Name}}
	s := e.Extract(target).(reflectshape.Function)
	return reflectshape.Fdump(os.Stdout, s.Returns.Values[0])
{{ end }}
}
`

var keep bool

type Kind string

const (
	KindOther  Kind = ""
	KindMethod Kind = "Method"
	KindFunc   Kind = "Func"
)

func init() {
	if v, err := strconv.ParseBool(os.Getenv("KEEP")); err == nil {
		keep = v
	}
}

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
	if strings.LastIndex(path, "/") < strings.LastIndex(path, ".") {
		path = strings.Join(parts[:len(parts)-2], ".")
		name = strings.Join(parts[len(parts)-2:], ".")
	}

	log.Printf("check by go list.\tpath=%q\tname=%q", path, name)
	ctx := context.Background()
	{
		cmd := exec.CommandContext(ctx, "go", "list", path)
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	log.Println("check by go doc.")
	buf := new(bytes.Buffer)
	{
		cmd := exec.CommandContext(ctx, "go", "doc", "-short", fullpath)
		cmd.Stdout = buf
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	kind := KindOther
	if strings.HasPrefix(strings.TrimSpace(buf.String()), "func") {
		if strings.HasPrefix(strings.TrimSpace(buf.String()), "func (") {
			kind = KindMethod
		} else {
			kind = KindFunc
		}
	}

	d, err := ioutil.TempDir(".", ".reflect-shape")
	if err != nil {
		return err
	}
	defer func() {
		if keep {
			return
		}

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
	recvAndName := strings.SplitN(name, ".", 2)
	recv := ""
	if len(recvAndName) > 1 {
		recv = recvAndName[0]
		name = recvAndName[1]
	}
	if err := t.Execute(f, map[string]interface{}{
		"Path":      path,
		"Qualifier": "x",
		"Recv":      recv,
		"Name":      name,
		"Kind":      kind,
	}); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "go", "run", filepath.Join(d, "main.go"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	return cmd.Run()
}
