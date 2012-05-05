package text

import (
	"unicode/utf8"
	"github.com/bobappleyard/ts"
)

func init() {
	ts.RegisterExtension("@text", pkg)
}

func pkg(itpr *ts.Interpreter) map[string] *ts.Object {
	reada := itpr.Accessor("readByte")
	writea := itpr.Accessor("writeByte")
	
	readByte := func(s *ts.Object) (byte, bool) {
		b := s.Call(reada)
		if b == ts.False {
			return 0, false
		}
		return byte(b.ToInt()), true
	}
	writeByte := func(s *ts.Object, b byte) {
		s.Call(writea, ts.Wrap(b))
	}
	
	return map[string] *ts.Object {
		"read8": ts.Wrap(func(o, s *ts.Object) *ts.Object {
			bs := make([]byte, 0, 6)
			var b byte
			var m bool
			for b, m = readByte(s); m && b & 0x80 != 0; b, m = readByte(s) {
				bs = append(bs, b)
			}
			if !m {
				return ts.False
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


