package reflectshape

import (
	"log"

	"github.com/podhmo/reflect-shape/metadata"
)

func fixupArglist(lookup *metadata.Lookup, fn *Function, ob interface{}, fullname string, isMethod bool) {
	params := fn.Params.Keys
	returns := fn.Returns.Keys

	// fixup names
	mfunc, err := lookup.LookupFromFunc(ob)
	if err != nil {
		log.Printf("function %q, arglist lookup is failed: %+v", fullname, err)
		return
	}

	d := 0
	if isMethod && mfunc.Recv != "" { // is method
		d = 1
	}

	margs := mfunc.Args()
	if len(margs) != len(params)-d {
		log.Printf("the length of arguments is mismatch, got=%d != want=%d", len(margs), len(params)-d)
	} else {
		if d > 0 {
			margs = append([]string{mfunc.Recv}, margs...)
		}
		fn.Params.Keys = margs
	}

	mreturns := mfunc.Returns()
	if len(mreturns) != len(returns) {
		log.Printf("the length of returns is mismatch, got=%d != want=%d", len(mreturns), len(returns))
	} else {
		fn.Returns.Keys = mreturns
	}
}
