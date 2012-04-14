package system

import (
	"io"
	"os"
	"strings"
	"github.com/bobappleyard/ts"
)

func init() {
	ts.PrimitivePackage("system", bsPkg)
}

func bsPkg(itpr *ts.Interpreter) map[string] *ts.Object {
	var File, Stream *ts.Class
	
	newStream := func(x interface{}) *ts.Object {
		s := Stream.New()
		s.SetUserData(x)
		return s
	}
	
	File = ts.ObjectClass.Extend(itpr, "File", 0, []ts.Slot {
		ts.FSlot("path", ts.Nil),
		ts.MSlot("create", func(o, p *ts.Object) *ts.Object {
			File.Set(o, 0, p)
			return ts.Nil
		}),
		ts.MSlot("exists", func(o *ts.Object) *ts.Object {
			_, err := os.Stat(File.Get(o, 0).ToString())
			return ts.Wrap(err == nil)
		}),
		ts.MSlot("open", func(o, f, p *ts.Object) *ts.Object {
			path := File.Get(o, 0).ToString()
			flags, perm := f.ToInt(), os.FileMode(p.ToInt())
			fl, err := os.OpenFile(path, flags, perm)
			if err != nil {
				panic(err)
			}
			return newStream(fl)
		}),
		ts.MSlot("read", func(o, f *ts.Object) *ts.Object {
			fl, err := os.Open(File.Get(o, 0).ToString())
			if err != nil {
				panic(err)
			}
			defer fl.Close()
			f.Call(nil, newStream(fl))
			return ts.Nil
		}),
		ts.MSlot("write", func(o, f *ts.Object) *ts.Object {
			fl, err := os.Create(File.Get(o, 0).ToString())
			if err != nil {
				panic(err)
			}
			defer fl.Close()
			f.Call(nil, newStream(fl))
			return ts.Nil
		}),
		ts.MSlot("append", func(o, f *ts.Object) *ts.Object {
			path := File.Get(o, 0).ToString()
			flags := os.O_WRONLY | os.O_APPEND | os.O_CREATE
			fl, err := os.OpenFile(path, flags, 0666)
			if err != nil {
				panic(err)
			}
			defer fl.Close()
			f.Call(nil, newStream(fl))
			return ts.Nil
		}),
	})
	
	strflags := ts.UserData | ts.Final
	Stream = ts.StreamClass.Extend(itpr, "Stream", strflags, []ts.Slot {
		ts.MSlot("readByte", func(o *ts.Object) *ts.Object {
			buf := []byte{0}
			r := o.UserData().(io.Reader)
			n, err := r.Read(buf)
			if err == io.EOF {
				if n == 0 {
					return ts.False
				}
				return ts.Wrap(buf[0])
			}
			if err != nil {
				panic(err)
			}
			return ts.Wrap(buf[0])
		}),
		ts.MSlot("writeByte", func(o, x *ts.Object) *ts.Object {
			buf := []byte{byte(x.ToInt())}
			w := o.UserData().(io.Writer)
			_, err := w.Write(buf)
			if err != nil {
				panic(err)
			}
			return ts.Nil
		}),
		ts.MSlot("close", func(o *ts.Object) *ts.Object {
			c := o.UserData().(io.Closer)
			err := c.Close()
			if err != nil {
				panic(err)
			}
			return ts.Nil
		}),
	})

	env := map[*ts.Object]*ts.Object {}
	for _, x := range os.Environ() {
		ss := strings.Split(x, "=")
		env[ts.Wrap(ss[0])] = ts.Wrap(ss[1])
	}

	return map[string] *ts.Object {
		"File": File.Object(),
		"args": ts.Wrap(os.Args),
		"env": ts.Wrap(env),
		"eval": ts.Wrap(func(o, expr *ts.Object) *ts.Object {
			return itpr.Eval(expr.ToString())
		}),
		"load": ts.Wrap(func(o, p *ts.Object) *ts.Object {
			itpr.Load(p.ToString())
			return ts.Nil
		}),
	}
}

