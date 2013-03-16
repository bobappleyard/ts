package re

import (
	"io"
	"regexp"
	"unicode/utf8"
	"github.com/bobappleyard/ts"
)

func init() {
	ts.RegisterExtension("re", pkg)
}

type runeReader struct {
	read *ts.Accessor
	inner *ts.Object
}

func (r runeReader) ReadRune() (res rune, n int, err error) {
	inp := r.inner.Call(r.read)
	if inp == ts.Done {
		return 0, 0, io.EOF
	}
	c := inp.ToString()
	res, _ = utf8.DecodeRuneInString(c)
	return res, 1, nil
}

func pkg(it *ts.Interpreter) map[string] *ts.Object {
	var Regex *ts.Class
	
	read := it.Accessor("readChar")
	
	Regex = ts.ObjectClass.Extend(it, "Regex", ts.UserData, []ts.Slot {
		ts.MSlot("create", func(o, expr *ts.Object) *ts.Object {
			re := regexp.MustCompile(expr.ToString())
			o.SetUserData(re)
			return ts.Nil
		}),
		ts.MSlot("match", func(o, src *ts.Object) *ts.Object {
			re := o.UserData().(*regexp.Regexp)
			r := runeReader{read, src}
			return ts.Wrap(re.FindReaderSubmatchIndex(r))
		}),
	})

	return map[string] *ts.Object {
		"Regex": Regex.Object(),
	}
}

