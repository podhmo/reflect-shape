package arglist

import (
	"fmt"
	"go/ast"
)

type NameSet struct {
	Name    string
	Recv    string
	Args    []string
	Returns []string
}

func (ns NameSet) IsAnonymous() bool {
	return ns.Name == ""
}

func (ns NameSet) IsMethod() bool {
	return ns.Recv != ""
}

func walkFuncType(typ *ast.FuncType, ns *NameSet) error {
	if typ.Params != nil {
		var names []string
		i := 0
		for _, x := range typ.Params.List {
			if len(x.Names) == 0 {
				names = append(names, fmt.Sprintf("arg%d", i))
				i++
				continue
			}
			if _, ok := x.Type.(*ast.Ellipsis); ok {
				names = append(names, fmt.Sprintf("*%s", x.Names[0].Name))
				continue
			}
			for _, ident := range x.Names {
				names = append(names, ident.Name)
			}
		}
		ns.Args = names
	}
	if typ.Results != nil {
		var names []string
		i := 0
		for _, x := range typ.Results.List {
			if len(x.Names) == 0 {
				names = append(names, fmt.Sprintf("ret%d", i))
				i++
				continue
			}
			for _, ident := range x.Names {
				names = append(names, ident.Name)
			}
		}
		ns.Returns = names
	}
	return nil
}

func InspectFunc(decl *ast.FuncDecl) (NameSet, error) {
	var r NameSet
	r.Name = decl.Name.Name
	if decl.Recv != nil {
		r.Recv = decl.Recv.List[0].Names[0].Name
	}
	if err := walkFuncType(decl.Type, &r); err != nil {
		return r, err
	}
	return r, nil
}

func InspectFuncLit(lit *ast.FuncLit) (NameSet, error) {
	var r NameSet
	r.Name = ""
	if err := walkFuncType(lit.Type, &r); err != nil {
		return r, err
	}
	return r, nil
}
