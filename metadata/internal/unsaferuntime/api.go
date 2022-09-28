package unsaferuntime

import (
	"fmt"
	"runtime"
	"strings"
	"unsafe"
)

func Print(pc uintptr) error {
	m := runtime_findmoduledatap(pc)
	for _, functab := range m.ftab {
		//	fmt.Printf("functab: %x, %x\n", functab.entryoff, functab.funcoff)
		funcoff := functab.funcoff
		rfunc := (*runtime.Func)(unsafe.Pointer(&m.pclntable[funcoff]))

		if strings.Contains(rfunc.Name(), "main.") {
			filename, lineno := rfunc.FileLine(rfunc.Entry())
			fmt.Printf("* %s\t%v:%v\n", rfunc.Name(), filename, lineno)
		}
	}
	return nil
}
