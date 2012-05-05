package system

import (
	"io"
	"os"
	"strings"
	"github.com/bobappleyard/ts"
	_ "github.com/bobappleyard/ts/ext/text"
)

func init() {
	ts.RegisterExtension("system", pkg)
}

func pkg(itpr *ts.Interpreter) map[string] *ts.Object {
	var File, Stream *ts.Class

	data := itpr.Import("data")
	text := itpr.Import("text")
	Mixin := data.Get(itpr.Accessor("Mixin"))
	utf8 := text.Get(itpr.Accessor("utf8"))
	readAll := text.Get(itpr.Accessor("readAll"))
	
	newStream := func(x interface{}) *ts.Object {
		s := Stream.New()
		s.SetUserData(x)
		return s
	}
	
	File = ts.ObjectClass.Extend(itpr, "File", 0, []ts.Slot {
		ts.FSlot("path", ts.Nil),
		ts.MSlot("read", func(o, f *ts.Object) *ts.Object {
			fl, err := os.Open(File.Get(o, 0).ToString())
			if err != nil {
				panic(err)
			}
			defer fl.Close()
			return f.Call(nil, newStream(fl))
		}),
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
		ts.MSlot("text", func(o *ts.Object) *ts.Object {
			return File.Call(o, 1, readAll)
		}),
		ts.MSlot("write", func(o, f *ts.Object) *ts.Object {
			fl, err := os.Create(File.Get(o, 0).ToString())
			if err != nil {
				panic(err)
			}
			defer fl.Close()
			return f.Call(nil, newStream(fl))
		}),
		ts.MSlot("append", func(o, f *ts.Object) *ts.Object {
			path := File.Get(o, 0).ToString()
			flags := os.O_WRONLY | os.O_APPEND | os.O_CREATE
			fl, err := os.OpenFile(path, flags, 0666)
			if err != nil {
				panic(err)
			}
			defer fl.Close()
			return f.Call(nil, newStream(fl))
		}),
	})
	
	Stream = ts.ObjectClass.Extend(itpr, "Stream", ts.UserData, []ts.Slot {
		ts.MSlot("readBuffer", func(o, b *ts.Object) *ts.Object {
			buf := b.ToBuffer()
			r := o.UserData().(io.Reader)
			n, err := r.Read(buf)
			if err == io.EOF {
				if n == 0 {
					return ts.False
				}
				return ts.Wrap(n)
			}
			if err != nil {
				panic(err)
			}
			return ts.Wrap(n)
		}),
		ts.MSlot("writeBuffer", func(o, b *ts.Object) *ts.Object {
			buf := b.ToBuffer()
			w := o.UserData().(io.Writer)
			n, err := w.Write(buf)
			if err != nil {
				panic(err)
			}
			return ts.Wrap(n)
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
	Stream = Mixin.Call(nil, utf8, Stream.Object()).ToClass()
	Stream.SetFlag(ts.Final)

	env := map[*ts.Object]*ts.Object {}
	for _, x := range os.Environ() {
		ss := strings.Split(x, "=")
		env[ts.Wrap(ss[0])] = ts.Wrap(ss[1])
	}
	
	return map[string] *ts.Object {
		"input": newStream(os.Stdin),
		"output": newStream(os.Stdout),
		"File": File.Object(),
		"args": ts.Wrap(os.Args),
		"env": ts.Wrap(env),
		"eval": ts.Wrap(func(o, expr *ts.Object) *ts.Object {
			return itpr.Eval(expr.ToString())
		}),
	}
}


