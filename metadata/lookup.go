package metadata

import (
	"fmt"
	"go/parser"
	"go/token"
	"reflect"
	"runtime"
	"strings"

	"github.com/podhmo/commentof"
	"github.com/podhmo/commentof/collect"
)

type Lookup struct {
	Fset *token.FileSet
}

func NewLookup(fset *token.FileSet) *Lookup {
	return &Lookup{
		Fset: fset,
	}
}

type Func struct {
	pc  uintptr
	Raw *collect.Func
}

func (m *Func) Fullname() string {
	return runtime.FuncForPC(m.pc).Name()
}

func (m *Func) Name() string {
	return m.Raw.Name
}

func (m *Func) Doc() string {
	return strings.TrimSpace(m.Raw.Doc) // todo: handling comment
}

func (m *Func) Args() []string {
	names := make([]string, len(m.Raw.ParamNames))
	for i, id := range m.Raw.ParamNames {
		names[i] = m.Raw.Params[id].Name
	}
	return names
}

func (m *Func) Returns() []string {
	names := make([]string, len(m.Raw.ReturnNames))
	for i, id := range m.Raw.ReturnNames {
		names[i] = m.Raw.Returns[id].Name
	}
	return names
}

func (l *Lookup) LookupFromFunc(fn interface{}) (*Func, error) {
	pc := reflect.ValueOf(fn).Pointer()
	rfunc := runtime.FuncForPC(pc)
	if rfunc == nil {
		return nil, fmt.Errorf("cannot find runtime.Func")
	}

	filename, _ := rfunc.FileLine(rfunc.Entry())
	funcname := rfunc.Name()
	if strings.Contains(funcname, ".") {
		parts := strings.Split(funcname, ".")
		funcname = parts[len(parts)-1]
	}

	f, err := parser.ParseFile(l.Fset, filename, nil, parser.ParseComments)
	if f == nil {
		return nil, err
	}

	// TODO: package cache
	p, err := commentof.File(l.Fset, f)
	if err != nil {
		return nil, err
	}
	result, ok := p.Functions[funcname]
	if !ok {
		return nil, fmt.Errorf("function not found,")
	}
	return &Func{pc: pc, Raw: result}, nil
}
