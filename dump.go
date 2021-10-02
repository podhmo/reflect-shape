package reflectshape

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func Dump(s Shape) error {
	return Fdump(os.Stdout, s)
}
func Fdump(w io.Writer, s Shape) error {
	dumper := &dumper{
		W:       w,
		Shape:   s,
		counter: map[Identity]int{s.GetIdentity(): 0},
		seen:   map[Identity]bool{},
	}
	if err := dumper.dump(s, 0); err != nil {
		return err
	}
	for len(dumper.q) > 0 {
		fmt.Fprintln(w, "----------------------------------------")
		item := dumper.q[0]
		dumper.q = dumper.q[1:]
		if err := dumper.dump(item, 0); err != nil {
			return err
		}
	}
	return nil
}

type dumper struct {
	W       io.Writer
	counter map[Identity]int
	seen    map[Identity]bool
	q       []Shape
	Shape   Shape
}

func (d *dumper) typeStrOf(s Shape) string {
	switch s := s.(type) {
	case Function, Interface, Struct:
		k := s.GetIdentity()
		i, ok := d.counter[k]
		if !ok {
			i = len(d.counter)
			d.counter[k] = i
			d.q = append(d.q, s)
		}
		return fmt.Sprintf("%q=#%d\t%s", s.GetName(), i, s.GetReflectType())
	default:
		return fmt.Sprintf("%q\t%s", s.GetFullName(), s.GetReflectType())
	}
}

func (d *dumper) dump(s Shape, lv int) error {
	w := d.W
	indent := strings.Repeat("  ", lv)

	if lv == 0 {
		fmt.Fprintf(w, "%02d:%s%T\ttype=%s\n", lv, indent, s, d.typeStrOf(s))
	}

	k := s.GetIdentity()
	if _, ok := d.seen[k]; ok {
		return nil
	}
	d.seen[k] = true

	switch s := s.(type) {
	case nil:
	case Primitive:
	case Struct:
		indent := "  " + indent
		for i, name := range s.Fields.Keys {
			f := s.Fields.Values[i]
			fmt.Fprintf(w, "%02d:%sfield=%q\t%T\ttype=%s\n", lv+1, indent, name, f, d.typeStrOf(f))
			if err := d.dump(f, lv+1); err != nil {
				return err
			}
		}
	case Interface:
		indent := "  " + indent
		for i, name := range s.Methods.Keys {
			f := s.Methods.Values[i]
			fmt.Fprintf(w, "%02d:%smethod=%q\t%T\ttype=%s\n", lv+1, indent, name, f, d.typeStrOf(f))
			if err := d.dump(f, lv+1); err != nil {
				return err
			}
		}
	case Container:
		indent := "  " + indent
		for i, arg := range s.Args {
			fmt.Fprintf(w, "%02d:%s[%d]\t%T\ttype=%s\n", lv+1, indent, i, arg, d.typeStrOf(arg))
			if err := d.dump(arg, lv+1); err != nil {
				return err
			}
		}
	case Function:
		indent := "  " + indent
		for i, name := range s.Params.Keys {
			x := s.Params.Values[i]
			fmt.Fprintf(w, "%02d:%sarg[%d]=%q\t%T\ttype=%s\n", lv+1, indent, i, name, x, d.typeStrOf(x))
			if err := d.dump(x, lv+1); err != nil {
				return err
			}
		}
		for i, name := range s.Returns.Keys {
			x := s.Returns.Values[i]
			fmt.Fprintf(w, "%02d:%sret[%d]=%q\t%T\ttype=%s\n", lv+1, indent, i, name, x, d.typeStrOf(x))
			if err := d.dump(x, lv+1); err != nil {
				return err
			}
		}
	case Unknown:
	default:
		return fmt.Errorf("unexpected type %T %+v", s, s)
	}
	return nil
}
