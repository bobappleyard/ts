package web

import (
	"fmt"
	"strconv"
	"net/http"
	//"net/url"
	"github.com/bobappleyard/ts"
	_ "github.com/bobappleyard/ts/ext/system"
)

func init() {
	ts.RegisterExtension("web", pkg)
}

func pkg(itpr *ts.Interpreter) map[string] *ts.Object {
	var Request, Response *ts.Class

	Request = ts.ObjectClass.Extend(itpr, "Request", ts.UserData, []ts.Slot {
		ts.PSlot("headMap", ts.Nil),
		ts.PSlot("formMap", ts.Nil),
		ts.PropSlot("method", func(o *ts.Object) *ts.Object {
			return ts.Wrap(o.UserData().(*http.Request).Method)
		}, ts.Nil),
		ts.PropSlot("proto", func(o *ts.Object) *ts.Object {
			return ts.Wrap(o.UserData().(*http.Request).Proto)
		}, ts.Nil),
		ts.PropSlot("path", func(o *ts.Object) *ts.Object {
			return ts.Wrap(o.UserData().(*http.Request).URL.Path)
		}, ts.Nil),
		ts.PropSlot("header", func(o *ts.Object) *ts.Object {
			return Request.Get(o, 0)
		}, ts.Nil),
		ts.PropSlot("form", func(o *ts.Object) *ts.Object {
			rv := o.UserData().(*http.Request)
			if err := rv.ParseMultipartForm(32 << 20); err != nil {
				panic(err)
			}
			fm := make(map[*ts.Object]*ts.Object)
			for k, v := range rv.Form {
				fm[ts.Wrap(k)] = ts.Wrap(v)
			}
			fmw := ts.Wrap(fm)
			Request.Set(o, 1, fmw)
			return fmw
		}, ts.Nil),
	})

	wrapReq := func(r *http.Request) *ts.Object {
		o := Request.New()
		o.SetUserData(r)
		hm := make(map[*ts.Object]*ts.Object)
		for k, v := range r.Header {
			hm[ts.Wrap(k)] = ts.Wrap(v)
		}
		Request.Set(o, 0, ts.Wrap(hm))
		if err := r.ParseForm(); err != nil {
			panic(err)
		}
		return o
	}
	
	Response = ts.ObjectClass.Extend(itpr, "Response", ts.UserData, []ts.Slot {
		ts.MSlot("writeBuffer", func(o, b *ts.Object) *ts.Object {
			bv := b.ToBuffer()
			rw := o.UserData().(http.ResponseWriter)
			if _, err := rw.Write(bv); err != nil {
				panic(err)
			}
			return ts.Nil
		}),
		ts.MSlot("writeHeader", func(o, c, h *ts.Object) *ts.Object {
			/*cv := int(c.ToInt())
			hv := h.ToHash()
			rw := o.UserData().(http.ResponseWriter)
			rwh := rw.Header()
			for k, v := range hv {
				rwh[k.(string)] = []string{v.ToString()}
			}
			rw.WriteHeader(cv)*/
			return ts.Nil
		}),
	})
	
	wrapResp := func(w http.ResponseWriter) *ts.Object {
		o := Response.New()
		o.SetUserData(w)
		return o
	}

	return map[string] *ts.Object {
		"serve": ts.Wrap(func(o, p, f *ts.Object) *ts.Object {
			port := ":" + strconv.Itoa(int(p.ToInt()))
			hnd := func(w http.ResponseWriter, r *http.Request) {
				defer func() {
					if e := recover(); e != nil {
						fmt.Println("web.serve:", e)
					}
				}()
				f.Call(nil, wrapResp(w), wrapReq(r))
			}
			http.ListenAndServe(port, http.HandlerFunc(hnd))
			return ts.Nil
		}),
		"get": ts.Wrap(func(o, u *ts.Object) *ts.Object {
			resp, err := http.Get(u.ToString())
			if err != nil {
				panic(err)
			}
			if resp.StatusCode != 200 {
				panic(fmt.Errorf("web.get: status %d", resp.StatusCode))
			}
			return ts.Wrap(resp.Body)
		}),
		"post": ts.Wrap(func(o, u, form *ts.Object) *ts.Object {
			/*vals := url.Values{}
			for k, v := range form.ToHash() {
				vals.Add(k.(string), v.ToString())
			}
			resp, err := http.PostForm(u.ToString(), vals)
			if err != nil {
				panic(err)
			}
			if resp.StatusCode != 200 {
				panic(fmt.Errorf("web.get: status %d", resp.StatusCode))
			}
			return ts.Wrap(resp.Body)*/
			return ts.Nil
		}),
	}
}

