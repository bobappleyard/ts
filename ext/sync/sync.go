package sync

import (
	"sync"
	"github.com/bobappleyard/ts"
)

func init() {
	ts.RegisterExtension("sync", pkg)
}

func pkg(itpr *ts.Interpreter) map[string] *ts.Object {
	MutexClass := ts.ObjectClass.Extend(itpr, "Mutex", ts.UserData, []ts.Slot {
		ts.MSlot("create", func(o *ts.Object) *ts.Object {
			o.SetUserData(new(sync.Mutex))
			return ts.Nil
		}),
		ts.MSlot("lock", func(o *ts.Object) *ts.Object {
			o.UserData().(*sync.Mutex).Lock()
			return ts.Nil
		}),
		ts.MSlot("unlock", func(o *ts.Object) *ts.Object {
			o.UserData().(*sync.Mutex).Unlock()
			return ts.Nil
		}),
		ts.MSlot("with", func(o, f *ts.Object) *ts.Object {
			m := o.UserData().(*sync.Mutex)
			m.Lock()
			defer m.Unlock()
			return f.Call(nil)
		}),
	})
	
	ChanClass := ts.ObjectClass.Extend(itpr, "Channel", ts.UserData, []ts.Slot {
		ts.MSlot("create", func(o *ts.Object) *ts.Object {
			o.SetUserData(make(chan *ts.Object))
			return ts.Nil
		}),
		ts.MSlot("send", func(o, x *ts.Object) *ts.Object {
			o.UserData().(chan *ts.Object) <- x
			return ts.Nil
		}),
		ts.MSlot("receive", func(o *ts.Object) *ts.Object {
			return <- o.UserData().(chan *ts.Object)
		}),
	})
	
	return map[string] *ts.Object {
		"spawn": ts.Wrap(func(o, f *ts.Object) *ts.Object {
			go f.Call(nil)
			return ts.Nil
		}),
		"Mutex": MutexClass.Object(),
		"Channel": ChanClass.Object(),
	}
}



