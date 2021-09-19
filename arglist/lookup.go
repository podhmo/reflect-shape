package arglist

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"reflect"
	"runtime"
	"strings"
)

// TODO: merge with ../comment

type Lookup struct {
	fset *token.FileSet

	fileCache map[string]*ast.File
	declCache map[*ast.File]map[string][]*ast.FuncDecl
}

// NewLookup is the factory function creating Lookup
func NewLookup() *Lookup {
	return &Lookup{
		fset:      token.NewFileSet(),
		fileCache: map[string]*ast.File{},
		declCache: map[*ast.File]map[string][]*ast.FuncDecl{},
	}
}

func (l *Lookup) LookupAST(filename string) (*ast.File, error) {
	if f, ok := l.fileCache[filename]; ok {
		return f, nil
	}
	mode := parser.ParseComments
	code, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	f, err := parser.ParseFile(l.fset, filename, code, mode)
	if err != nil {
		return nil, err
	}
	l.fileCache[filename] = f
	return f, nil
}

// TODO: remove
func (l *Lookup) LookupFuncDecl(filename string, targetName string) (*ast.FuncDecl, error) {
	f, err := l.LookupAST(filename)
	if err != nil {
		return nil, err
	}
	ob := f.Scope.Lookup(targetName)
	if ob == nil {
		return nil, fmt.Errorf("not found %q in %q", targetName, filename)
	}
	decl, ok := ob.Decl.(*ast.FuncDecl)
	if !ok {
		return nil, fmt.Errorf("%q is unexpected type %T", targetName, ob)
	}
	return decl, nil
}

func (l *Lookup) LookupNameSetFromFunc(fn interface{}) (NameSet, error) {
	if fn == nil {
		return NameSet{}, fmt.Errorf("fn is nil")
	}
	rfunc := runtime.FuncForPC(reflect.ValueOf(fn).Pointer())
	if rfunc == nil {
		return NameSet{}, fmt.Errorf("cannot find runtime.Func")
	}

	filename, lineno := rfunc.FileLine(rfunc.Entry())
	funcname := rfunc.Name()
	if strings.Contains(funcname, ".") {
		parts := strings.Split(funcname, ".")
		funcname = parts[len(parts)-1]
	}

	f, err := l.LookupAST(filename)
	if err != nil {
		return NameSet{Name: funcname}, err
	}

	if ob := f.Scope.Lookup(funcname); ob != nil {
		if decl, ok := ob.Decl.(*ast.FuncDecl); ok {
			r, err := InspectFunc(decl)
			if err != nil {
				return NameSet{Name: funcname}, err
			}
			return r, nil
		} else if ob.Kind == ast.Fun {
			return NameSet{Name: funcname}, fmt.Errorf("invalid value, %q is not function in %q", funcname, filename)
		}
	}

	tkFile := l.fset.File(f.Pos())
	pos := tkFile.LineStart(lineno)

	for _, decl := range f.Decls {
		// fmt.Printf("%T (%d, %d) -> ok=%v\n", decl, decl.Pos(), decl.End(), (decl.Pos() <= pos && pos <= decl.End()))
		isInner := decl.Pos() <= pos && pos <= decl.End()
		if !isInner {
			continue
		}

		var retErr error
		decl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		var innerMost struct {
			Lit  *ast.FuncLit
			Cost int
		}
		ast.Inspect(decl.Body, func(node ast.Node) bool {
			if node == nil {
				return false
			}

			nodeStartLine := tkFile.Line(node.Pos())
			nodeEndLine := tkFile.Line(node.End())

			isInner := nodeStartLine <= lineno && lineno <= nodeEndLine
			if !isInner {
				// fmt.Printf("ignore %T %d (%d, %d) -> ok=%v\n", node, pos, node.Pos(), node.End(), isInner)
				return false
			}

			if target, ok := node.(*ast.FuncLit); ok {
				cost := (lineno - nodeStartLine) + (nodeEndLine - lineno) // xxx
				if innerMost.Lit == nil {
					innerMost.Lit = target
					innerMost.Cost = cost
					// fmt.Printf("got %T %d (%d, %d) -> ok=%v\n", node, pos, node.Pos(), node.End(), isInner)
				} else if innerMost.Cost > cost {
					innerMost.Lit = target
					innerMost.Cost = cost
					// fmt.Printf("update %T %d (%d, %d) -> ok=%v\n", node, pos, node.Pos(), node.End(), isInner)
					// printer.Fprint(os.Stdout, l.fset, node)
				}
			}
			return true
		})

		if retErr != nil {
			return NameSet{Name: funcname}, retErr
		}

		if innerMost.Lit == nil {
			ns, err := InspectFunc(decl) // maybe method
			if err != nil {
				return NameSet{Name: funcname}, fmt.Errorf("inspect func: %w", err)
			}
			return ns, nil
		} else {
			ns, err := InspectFuncLit(innerMost.Lit)
			if err != nil {
				return NameSet{Name: funcname}, fmt.Errorf("inspect funclit: %w", err)
			}
			return ns, nil
		}
	}
	return NameSet{Name: funcname}, fmt.Errorf("not found %q in %q", funcname, filename)
}
