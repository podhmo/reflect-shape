package unsaferuntime

import (
	"fmt"
	"runtime"
	"strings"
	"unsafe"
)

type Accessor struct {
	// TODO: cache
	// fmPCtoPC map[uintptr]uintptr
}

func New() *Accessor {
	return &Accessor{}
}

func (a *Accessor) FuncForPC(pc uintptr) *runtime.Func {
	rfunc := runtime.FuncForPC(pc)
	if !strings.HasSuffix(rfunc.Name(), "-fm") {
		return rfunc
	}
	target := strings.TrimSuffix(rfunc.Name(), "-fm")

	var prevs []*moduledata
	findDepth := 1
	for datap := &runtime_firstmoduledata; datap != nil; datap = datap.next {
		if datap.minpc <= pc && pc < datap.maxpc {
			m := datap
			for _, functab := range m.ftab {
				//	fmt.Printf("functab: %x, %x\n", functab.entryoff, functab.funcoff)
				funcoff := functab.funcoff
				rfunc := (*runtime.Func)(unsafe.Pointer(&m.pclntable[funcoff]))
				if rfunc.Name() == target {
					return rfunc
				}
			}
			// find prev
			if len(prevs) > 0 {
				m := prevs[len(prevs)-findDepth]
				for _, functab := range m.ftab {
					//	fmt.Printf("functab: %x, %x\n", functab.entryoff, functab.funcoff)
					funcoff := functab.funcoff
					rfunc := (*runtime.Func)(unsafe.Pointer(&m.pclntable[funcoff]))
					if rfunc.Name() == target {
						return rfunc
					}
				}
				findDepth++
			}
			// find next
			continue
		}
		prevs = append(prevs, datap)
	}
	return nil
}

func Print(pc uintptr, pkg string) error {
	prefix := strings.TrimSuffix(pkg, ".") + "."

	for datap := &runtime_firstmoduledata; datap != nil; datap = datap.next {
		if datap.minpc <= pc && pc < datap.maxpc {
			m := datap
			for _, functab := range m.ftab {
				//	fmt.Printf("functab: %x, %x\n", functab.entryoff, functab.funcoff)
				funcoff := functab.funcoff
				rfunc := (*runtime.Func)(unsafe.Pointer(&m.pclntable[funcoff]))

				if strings.Contains(rfunc.Name(), prefix) {
					filename, lineno := rfunc.FileLine(rfunc.Entry())
					fmt.Printf("* %s\t%v:%v\n", rfunc.Name(), filename, lineno)
				}
			}
		}
	}

	return nil
}
