package web

import (
	"io"
	"fmt"
	"strconv"
	"net/http"
	"github.com/bobappleyard/ts"
)

func init() {
	ts.PrimitivePackage("@web", webPkg)
}

func inputVars(r *http.Request) *ts.Object {
	r.ParseForm()
	vars := make(map[interface{}] *ts.Object)
	for k, v := range r.Form {
		vars[k] = ts.Wrap(v)
	}
	return ts.Wrap(vars)
}

func webPkg(itpr *ts.Interpreter) map[string] *ts.Object {
	FileClass := itpr.Import("system").Get(itpr.Accessor("File")).ToClass()

	handle := func(w http.ResponseWriter, r *http.Request, resp *ts.Object) {
		switch resp.Class() {
		case ts.StringClass:
			_, err := io.WriteString(w, resp.ToString())
			if err != nil {
				panic(err)
			}
		case FileClass:
			http.ServeFile(w, r, FileClass.Get(resp, 0).ToString())
		default:
			panic(fmt.Errorf("wrong type: %s", resp))
		}
	}
	
	return map[string] *ts.Object {
		"serve": ts.Wrap(func(o, p, f *ts.Object) *ts.Object {
			port := ":" + strconv.Itoa(p.ToInt())
			hnd := func(w http.ResponseWriter, r *http.Request) {
				defer func() {
					if e := recover(); e != nil {
						fmt.Println("web.serve:", e)
					}
				}()
				resp := f.Call(nil, ts.Wrap(r.URL.Path), inputVars(r))
				handle(w, r, resp)
			}
			http.ListenAndServe(port, http.HandlerFunc(hnd))
			return ts.Nil
		}),
	}
}

