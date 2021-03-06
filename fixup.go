package reflectshape

import (
	"log"

	"github.com/podhmo/reflect-shape/arglist"
)

func fixupArglist(lookup *arglist.Lookup, fn *Function, ob interface{}, fullname string, isMethod bool) {
	params := fn.Params.Keys
	returns := fn.Returns.Keys

	// fixup names
	nameset, err := lookup.LookupNameSetFromFunc(ob)
	if err != nil {
		log.Printf("function %q, arglist lookup is failed %v", fullname, err)
	}

	d := 0
	if isMethod && nameset.Recv != "" { // is method
		d = 1
	}

	if len(nameset.Args) != len(params)-d {
		log.Printf("the length of arguments is mismatch, got=%d != want=%d", len(nameset.Args), len(params)-d)
	} else {
		if d > 0 {
			nameset.Args = append([]string{nameset.Recv}, nameset.Args...)
		}
		fn.Params.Keys = nameset.Args
	}
	if len(nameset.Returns) != len(returns) {
		log.Printf("the length of returns is mismatch, got=%d != want=%d", len(nameset.Returns), len(returns))
	} else {
		fn.Returns.Keys = nameset.Returns
	}
}
