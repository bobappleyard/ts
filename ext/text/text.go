package text

import (
	"unicode/utf8"
	"github.com/bobappleyard/ts"
)

func init() {
	ts.PrimitivePackage("@text", pkg)
}

func pkg(itpr *ts.Interpreter) map[string] *ts.Object {
	reada := itpr.Accessor("readByte")
	writea := itpr.Accessor("writeByte")
	
	readByte := func(s *ts.Object) byte {
		return byte(s.Call(reada).ToInt())
	}
	writeByte := func(s *ts.Object, b byte) {
		s.Call(writea, ts.Wrap(b))
	}
	
	return map[string] *ts.Object {
		"read8": ts.Wrap(func(o, s *ts.Object) *ts.Object {
			bs := make([]byte, 0, 6)
			var b byte
			for b = readByte(s); b & 0x80 != 0; b = readByte(s) {
				bs = append(bs, b)
			}
			bs = append(bs, b)
			r, _ := utf8.DecodeRune(bs)
			if r == utf8.RuneError {
				panic("bad character")
			}
			return ts.Wrap(string(r))
		}),
		"write8": ts.Wrap(func(o, s, c *ts.Object) *ts.Object {
			bs := make([]byte, 6)
			size := utf8.EncodeRune(bs, rune(c.ToInt()))
			for i := 0; i < size; i++ {
				writeByte(s, bs[i])
			}
			return ts.Nil
		}),
	}
}


