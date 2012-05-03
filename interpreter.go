// A simple dynamic object-oriented programming language.
//
package ts

import (
	"fmt"
	"io"
	"os"
	"sync"
	"strings"
	"encoding/binary"
	. "github.com/bobappleyard/ts/bytecode"
	"github.com/bobappleyard/readline"
)

/*******************************************************************************

	Main types

*******************************************************************************/


// Everything that the runtime sees is encoded as an object. An object has a
// set of fields and methods, specified by a class. The fields may vary in 
// value between different instances of an object and the methods are fixed for
// all instances of a given class.
//
// Note that in some cases objects use extra space on the end to store
// primitive data. Therefore always refer to Objects as pointers to structs.
type Object struct {
	c *Class
	f []*Object
}

// Classes describe objects. They each have a name and an ancestor, accompanied
// by a series of entries describing fields and methods.
type Class struct {
	added bool
	flags int
	o *Object
	n string
	a, p *Class
	e []Slot
	m, f []*Object
}

const (
	Final = 1 << iota
	Primitive
	UserData
)

// Accessors refer to names that can be looked up on objects.
type Accessor struct {
	n string
	e []Slot
}

type SlotKind byte

const (
	Field SlotKind = iota
	Method
	Property
	Marker
)

type SlotVis byte

const (
	Private SlotVis = iota
	Public
)

// Slots describe class members.
//
// When constructing a primitive class, fill in the first four fields. The 
// runtime will fill in the rest.
type Slot struct {
	Name string
	Kind SlotKind
	Vis SlotVis
	Value, Set *Object
	Class *Class
	offset, access, next uint16
}

// An interpreter provides a global environment and some methods to control
// the general flow of execution.
type Interpreter struct {
	o map[string] *Object
	a map[string] *Accessor
	c []*Class
}

// A unit represents some compiled code. 
type Unit struct {
	v, g []*Object
	a []*Accessor
	gn, an []string
	b [][]uint16
	file string
	line int
}

// Dymamic scope record.
type frame struct {
	t *Object
	sc *Class
	e []*Object
	c []uint16
	p, n, b int
	u *Unit
}

// A running computation.
type process struct {
	frame
	v, file *Object
	line int
	s []*Object
	frames []frame
}

/*******************************************************************************

	Toplevel API

*******************************************************************************/

// Create a new interpreter with the default environment.
func New() *Interpreter {
	i := new(Interpreter)
	definePrimitives(i)
	i.Load(root() + "/prelude")
	return i
}

// Start a prompt that reads expressions from stdin and prints them to stdout.
// Swallows and prints all errors.
func (i *Interpreter) Repl() {
	readline.Completer = func(query, ctx string) []string {
		src := i.ListDefined()
		res := []string{}
		for _, x := range src {
			if strings.HasPrefix(x, query) {
				res = append(res, x)
			}
		}
		return res
	}
	for {
		if func() bool {
			defer func() {
				if e := recover(); e != nil {
					fmt.Printf("\033[1;31m%s\033[0m\n", e)
				}
			}()/**/
			u := new(Unit)
			r := readline.Reader()
			if _, e := r.Read(nil); e == io.EOF {
				return true
			}
			l := NewScanner(r, "stdin")
			if u.CompileStmt(l) {
				readline.AddHistory(l.Scanned())
				x := i.Exec(u)
				if x != Nil {
					fmt.Println(x)
				}
			}
			return false
		}() {
			break
		}
	}
	fmt.Println()
}

// Evaluate an expression, returning its value. Panics on error.
func (i *Interpreter) Eval(s string) *Object {
	u := new(Unit)
	u.CompileStr(s + ";")
	return i.Exec(u)
}

// Load a code file into the interpreter. May be in source or compiled form.
// Panics on error.
func (i *Interpreter) Load(p string) {
	u := new(Unit)
	f, err := os.Open(p)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if !u.Load(f) {
		f.Seek(0, 0)
		u.Compile(f, p)
	}
	i.Exec(u)
}

// Import a package and return it.
func (i *Interpreter) Import(n string) *Object {
	aget := i.Accessor("__aget__")
	return i.Get("packages").Call(aget, Wrap(n))
}

// Run some compiled code. Panics on error.
func (i *Interpreter) Exec(u *Unit) *Object {
	u.link(i)
	p := &process{frame: frame{c: u.b[0], u: u}}
	p.run()
	return p.v
}

// Check whether a global variable is defined.
func (i *Interpreter) Defined(n string) bool {
	b := i.lookup(n)
	return b.c == boxClass
}

func (i *Interpreter) ListDefined() []string {
	res := []string{}
	for n, v := range i.o {
		if v.c == boxClass {
			res = append(res, n)
		}
	}
	return res
}

// Look up a global variable and return its value. Panics if the variable does 
// not exist.
func (i *Interpreter) Get(n string) *Object {
	o := i.lookup(n)
	if o.c == undefinedClass {
		panic(fmt.Errorf("undefined variable: %s", n))
	}
	return o.boxData()
}

// Update the value of a  global variable. Panics if the variable does not 
// exist.
func (i *Interpreter) Set(n string, v *Object) {
	o := i.lookup(n)
	if o.c == undefinedClass {
		panic(fmt.Errorf("undefined variable: %s", n))
	}
	o.setBoxData(v)
}

// Define a new global variable.
func (i *Interpreter) Define(n string, v *Object) {
	b := i.lookup(n)
	b.c = boxClass
	b.setBoxData(v)
}

/*******************************************************************************

	Objects

*******************************************************************************/

// Create a new object instance. Panics if such an instance cannot be created.
func (c *Class) New(args... *Object) *Object {
	return c.alloc().callMethod(c.m[_Object_new], args)
}

// Make objects printable from Go.
func (o *Object) String() string {
	return o.callMethod(o.c.m[_Object_toString], nil).ToString()
}

// Check to see whether a member is defined on an object.
func (o *Object) Defined(a *Accessor) bool {
	return a.lookup(o) != nil
}

// Get the slot to which the Accessor corresponds from the object. Panics if the 
// slot does not exist.
func (o *Object) Get(a *Accessor) *Object {
	return o.get(a, nil)
}

// Set the corresponding slot on o to x. Panics if the slot does not exist
// or cannot be written to.
func (o *Object) Set(a *Accessor, x *Object) {
	o.set(a, nil, x)
}

// Call a function or method. If a is nil, o is the function to be called.
// Otherwise a is an accessor for a method to be called on o. Pass args to the
// function or method and return what the function or method returns. Panics
// if the object is not a function or the method does not exist, or if the
// function or method panics.
func (o *Object) Call(a *Accessor, args... *Object) *Object {
	f := o
	if a != nil {
		f = o.getMethod(a, nil)
	}
	return o.callMethod(f, args)
}

func (c *Class) Get(o *Object, i int) *Object {
	o.checkClass(o.Is(c))
	return o.get(nil, &c.e[i])
}

func (c *Class) Set(o *Object, i int, x *Object) {
	o.checkClass(o.Is(c))
	o.set(nil, &c.e[i], x)
}

func (c *Class) Call(o *Object, i int, args... *Object) *Object {
	o.checkClass(o.Is(c))
	return o.callMethod(o.getMethod(nil, &c.e[i]), args)
}

func (o *Object) callMethod(f *Object, args []*Object) *Object {
	p := new(process)
	p.pushFrame(0)
	for _, x := range args {
		p.push(x)
	}
	p.n = len(args)
	p.t = o
	f.funcData()(p)
	p.run()
	return p.v
}

func (o *Object) bindMethod(m *Object) *Object {
	f := m.funcData()
	return new(funcObj).init(func(p *process) {
		p.t = o
		f(p)
	})
}

// Internal get(): may have static class info provided for private access.
func (o *Object) get(a *Accessor, e *Slot) *Object {
	if e == nil {
		e = a.lookup(o)
	}
	if e == nil {
		ao := new(accObj).init(a)
		return o.callMethod(o.c.m[_Object_getFailed], []*Object{ao})
	}
	switch e.Kind {
	case Field:
		return o.f[e.offset]
	case Method:
		return o.bindMethod(o.c.m[e.offset])
	case Property:
		return o.getProperty(e)
	}
	panic(fmt.Errorf("invalid location for reading"))
}

// Internal set(): may have static class info provided for private access.
func (o *Object) set(a *Accessor, e *Slot, x *Object) {
	if e == nil {
		e = a.lookup(o)
	}
	if e == nil {
		ao := new(accObj).init(a)
		o.callMethod(o.c.m[_Object_setFailed], []*Object{ao, x})
		return
	}
	switch e.Kind {
	case Field:
		o.f[e.offset] = x
	case Property:
		o.setProperty(e, x)
	default:
		panic(fmt.Errorf("invalid location for writing"))
	}
}

// Internal get(): may have static class info provided for private access. 
// Assumes slot is a method (and so has "this" provided to it).
func (o *Object) getMethod(a *Accessor, e *Slot) *Object {
	if e == nil {
		e = a.lookup(o)
	}
	if e == nil {
		return o.methodMissing(a)
	}
	switch e.Kind {
	case Field:
		return o.f[e.offset]
	case Method:
		return o.c.m[e.offset]
	case Property:
		return o.getProperty(e)
	}
	panic(fmt.Errorf("invalid location for calling"))
}

func (o *Object) methodMissing(a *Accessor) *Object {
	ao := new(accObj).init(a)
	return Wrap(func(o *Object, args []*Object) *Object {
		args = append([]*Object{ao}, args...)
		return o.callMethod(o.c.m[_Object_callFailed], args)
	})
}

func (o *Object) getProperty(e *Slot) *Object {
	m := o.c.m[e.offset]
	if m == Nil {
		panic(fmt.Errorf("invalid location for reading"))
	}
	return o.callMethod(m, nil)
}

func (o *Object) setProperty(e *Slot, x *Object) {
	m := o.c.m[e.offset+1]
	if m == Nil {
		panic(fmt.Errorf("invalid location for writing"))
	}
	o.callMethod(m, []*Object{x})
}

/*******************************************************************************

	Classes

*******************************************************************************/
 
// Get the object's class.
func (o *Object) Class() *Class {
	return o.c
}

// Get the class' object.
func (c *Class) Object() *Object {
	return c.o
}

// Get the class' name.
func (c *Class) Name() string {
	return c.n
}

// Get the class' ancestor.
func (c *Class) Ancestor() *Class {
	return c.a
}

// Check if the object is an instance of c (or one of its descendants).
func (o *Object) Is(c *Class) bool {
	return o.c.Is(c)
}

// Check if c is the same class as d, or one of d's descendants.
func (c *Class) Is(d *Class) bool {
	for cur := c; cur != nil; cur = cur.a {
		if cur == d {
			return true
		}
	}
	return false
}

// Create a descendant of c with the given name and a series of entries
// describing the class' members. Panics if the class cannot be extended.
func (c *Class) Extend(i *Interpreter, n string, flags int, e []Slot) *Class {
	d := c.extend(n, flags, e)
	i.addClass(d)
	return d
}

func (c *Class) SlotCount() int {
	return len(c.e)
}

func (c *Class) Slot(i int) Slot {
	return c.e[i]
}

// Within the interpreter class extension is divided up into three phases:
//
// 1. The structure of the class is created to be filled in.
//
// 2. The class slots are evaluated.
//
// 3. The values are fed into the class structure and the class is registered 
//    with the interpreter.
func (c *Class) extend(n string, flags int, e []Slot) *Class {
	if c.flags & Final != 0 {
		panic(fmt.Errorf("class is final: %s", c.n))
	}
	if c.flags & UserData != 0 {
		flags |= UserData
	}
	res := &Class{flags: flags, a:c, n:n, e:e}
	res.o = new(clsObj).init(res)
	return res
}

// This is the part that runs after the slots have been evaluated.
func (i *Interpreter) addClass(c *Class) {
	u := new(Unit)
	for i, x := range c.e {
		c.e[i].access = uint16(u.getAccessor(x.Name))
	}
	u.link(i)
	u.addClass(c)
}

var classLock = new(sync.Mutex)

func (u *Unit) addClass(c *Class) {
	classLock.Lock()
	defer classLock.Unlock()
	u.clInner(c)
}

func (u *Unit) clInner(c *Class) {
	if c.added {
		return
	}
	c.added = true
	if c.a != nil {
		u.clInner(c.a)
		c.m = make([]*Object, len(c.a.m))
		c.f = make([]*Object, len(c.a.f))
		copy(c.m, c.a.m)
		copy(c.f, c.a.f)
	}
	for i := range c.e {
		u.addSlot(c, &c.e[i])
	}
}

func (u *Unit) addSlot(c *Class, e *Slot) {
	if e.Kind == Marker {
		e.Class = c.p
		return
	}
	a := u.a[e.access]
	t := c.m
	if e.Kind == Field {
		t = c.f
	}
	for i := range a.e {
		f := &a.e[i]
		if c.Is(f.Class) {
			// shadowing is where a name is defined that has already been
			// defined in an ancestor class, and this definition is 
			// incompatible
			if e.Kind != f.Kind || e.Vis == Private {
				panic(fmt.Errorf("cannot shadow %s.%s", f.Class.n, e.Name))
			}
			// overriding causes update in place
			t[f.offset] = e.Value
			if e.Kind == Property {
				t[f.offset+1] = e.Set
			}
			e.offset = f.offset
			e.Class = f.Class
			return
		}
	}
	// if no previous defitions to override, add a new one
	e.offset = uint16(len(t))
	e.Class = c
	t = append(t, e.Value)
	if e.Kind == Property {
		t = append(t, e.Set)
	}
	if e.Kind == Field {
		c.f = t
	} else {
		c.m = t
	}
	// only public definitions go in the accessor
	if e.Vis == Public {
		a.e = append(a.e, *e)
	}
}

// create a new uninitialised object
func (c *Class) alloc() *Object {
	if c.flags & Primitive != 0 {
		panic(fmt.Errorf("class is primitive: %s", c.n))
	}
	f := make([]*Object, len(c.f))
	copy(f, c.f)
	if c.flags & UserData != 0 {
		return new(userObj).init(c, f)
	}
	return &Object{c: c, f: f}
}

/*******************************************************************************

	Accessors

*******************************************************************************/

// Retrieve the named accessor.
func (i *Interpreter) Accessor(n string) *Accessor {
	if n == "" {
		return new(Accessor)
	}
	if i.a == nil {
		i.a = make(map[string] *Accessor)
	}
	a := i.a[n]
	if a == nil {
		a = &Accessor{n: n}
		i.a[n] = a
	}
	return a
}

// The accessor's name.
func (a *Accessor) Name() string {
	return a.n
}

// Find the entry for the corresponding static and dynamic classes. Returns that
// entry or nil if no entry can be found.
func (a *Accessor) lookup(o *Object) *Slot {
	c := o.c
	for i := range a.e {
		e := &a.e[i]
		if c.Is(e.Class) {
			return e
		}
	}
	return nil
}

// For anonymous classes: lookup the appropriate skeleton class.
func (a *Accessor) lookupa(c *Class) *Class {
	for _, f := range a.e {
		if c == f.Class {
			return f.Value.ToClass()
		}
	}
	return nil
}

const slotUnknown = 0xffff

// For accessing when some information about the slot is statically known.
func (p *process) lookups(m int) *Slot {
	for cur := p.sc; cur != nil; cur = cur.p {
		if m == slotUnknown {
			break
		}
		e := &cur.e[m]
		if e.Kind != Marker && p.v.Is(cur) {
			return e
		}
		m = int(cur.e[m].next)
	}
	return nil
}

/*******************************************************************************

	Evaluation

*******************************************************************************/

func (p *process) run() {
	for int(p.p) < len(p.c) {
		p.step()
	}
}

// Find a global variable. If one doesn't yet exist with that name, create a
// global that has been marked as undefined.
func (i *Interpreter) lookup(n string) *Object {
	if i.o == nil {
		i.o = make(map[string] *Object)
	}
	o := i.o[n]
	if o == nil {
		o = new(boxObj).init(Wrap(n))
		o.c = undefinedClass
		i.o[n] = o
	}
	return o
}

func (p *process) parseArgs(vars... **Object) {
	if p.n != len(vars) {
		 panic(fmt.Errorf("wrong number of arguments %d", p.n))
	}
	for i, x := range p.s[p.b:] {
		*vars[i] = x
	}
}

func (p *process) args() []*Object {
	return p.s[p.b:]
}

func (p *process) push(x *Object) {
	p.s = append(p.s, x)
}

func (p  *process) pop() *Object {
	l := len(p.s)-1
	res := p.s[l]
	p.s = p.s[:l]
	return res
}

func (p *process) pushFrame(k int) {
	f := p.frame
	f.p = k
	p.frames = append(p.frames, f)
}

func (p *process) popFrame() {
	l := len(p.frames)-1
	p.frame = p.frames[l]
	p.frames = p.frames[:l]
}

func (p *process) shuffle(n int) {
	l := len(p.s)
	copy(p.s[p.b:], p.s[l-n:])
	p.s = p.s[:p.b+n]
}

func (p *process) ret(x *Object) {
	p.s = p.s[:p.b]
	p.popFrame()
	p.v = x
}

func (p *process) next() int {
	res := p.c[p.p]
	p.p++
	return int(res)
}

func (p *process) close(block, n int, capture bool) {
	u := p.u
	t := p.t
	sc := p.sc
	b := u.b[block]
	c := len(p.s) - n
	e := make([]*Object, n)
	copy(e, p.s[c:])
	p.s = p.s[:c]
	p.v = new(funcObj).init(func(p *process) {
		p.c = b
		p.p = 0 
		p.u = u
		if capture {
			p.t = t
		}
		p.e = e
		p.sc = sc
	})
}

func (p *process) extend(a *Accessor) {
	s := p.pop()
	e := s.skelData()
	d := p.v.ToClass()
	if a != nil {
		f := a.lookupa(d)
		if f == nil {
			f = d.extend(d.n, 0, e[1:])
			a.e = append(a.e, Slot{Class: d, Value: f.o})
		}
		d = f
	}
	n := e[0].Name
	if n == "" {
		n = d.n
	}
	c := d.extend(n, 0, e[1:])
	c.p = p.sc
	p.sc = c
}

func (p *process) finish(n int) {
	l := len(p.s) - n
	c := p.sc
	e := c.e
	spec := p.s[l:]
	p.s = p.s[:l]
	j := 0
	for i := range e {
		e[i].Name = p.u.a[e[i].access].n
		switch e[i].Kind {
		case Marker:
		case Property:
			e[i].Value = spec[j]
			e[i].Set = spec[j+1]
			j += 2
		default:
			e[i].Value = spec[j]
			j++
		}
	}
	p.u.addClass(c)
	p.v = c.o
	p.sc = c.p
}

func (p *process) prolog(n, m int, rest bool) {
	if p.n < n {
		 panic(fmt.Errorf("wrong number of arguments %d", p.n))
	}
	if !rest && p.n > m {
		 panic(fmt.Errorf("wrong number of arguments %d", p.n))
	}
	for i := p.n; i < m; i++ {
		p.push(False)
	}
	if rest {
		rc := p.n-m
		if rc <= 0 {
			p.push(Wrap([]*Object{}))
		} else {
			sp := len(p.s)-rc
			ra := make([]*Object, rc)
			copy(ra, p.s[sp:])
			p.s = p.s[:sp]
			p.push(Wrap(ra))
		}
		m++
	}
	p.b = len(p.s)-m
}

func (p *process) wrapError(err interface{}) *Object {
	if p.line == 0 {
		return Wrap(err)
	}
	if o, ok := err.(*Object); ok && o.c == ErrorClass {
		return o
	}
	e := ErrorClass.New(Wrap(err))
	ErrorClass.Set(e, 1, p.file)
	ErrorClass.Set(e, 2, Wrap(p.line))
	return e
}

func (p *process) step() {
	defer func() {
		if e := recover(); e != nil {
			panic(p.wrapError(e))
		}
	}()
	op := p.next()
	switch op {
	case NOP:
	
	case JUMP:
		n := p.next()
		p.p = n
		
	case BRANCH:
		n := p.next()
		if p.v == False {
			p.p = n
		}
		
	case VALUE:
		n := p.next()
		p.v = p.u.v[n]
		
	case BOUND:
		n := p.next()
		p.v = p.s[int(p.b + n)]
		
	case FREE:
		n := p.next()
		p.v = p.e[n]
		
	case GLOBAL:
		n := p.next()
		p.v = p.u.g[n]
		
	case BOX:
		n := p.next()
		l := int(p.b + n)
		p.s[l] = new(boxObj).init(p.s[l])

	case UNDEFINE:
		n := p.next()
		l := int(p.b + n)
		b := new(boxObj).init(p.s[l])
		b.c = undefinedClass
		p.s[l] = b 
		
	case UNBOX:
		if p.v.c == undefinedClass {
			s := p.v.boxData().ToString()
			panic(fmt.Errorf("undefined variable: %s", s))
		}
		p.v = p.v.boxData()
		
	case UPDATE:
		if p.v.c == undefinedClass {
			s := p.v.boxData().ToString()
			panic(fmt.Errorf("undefined variable: %s", s))
		}
		p.v.setBoxData(p.pop())
		p.v.c = boxClass
		p.v = Nil
		
	case DEFINE:
		p.v.c = boxClass
		
	case PUSH:
		p.push(p.v)
		
	case FRAME:
		n := p.next()
		p.pushFrame(n)
		
	case SHUFFLE:
		n := p.next()
		p.shuffle(n)
		
	case RETURN:
		p.ret(p.v)
	
	case RETRACT:
		n := p.next()
		p.s = p.s[:len(p.s)-int(n)]
		
	case CALL:
		p.n = p.next()
		p.v.funcData()(p)
		
	case CLOSE:
		b := p.next()
		n := p.next()
		p.close(int(b), int(n), true)
		
	case CLOSEM:
		b := p.next()
		n := p.next()
		p.close(int(b), int(n), false)
	
	case PROLOG:
		n := p.next()
		if p.n != n {
			 panic(fmt.Errorf("wrong number of arguments %d", p.n))
		}
		p.b = len(p.s) - p.n
		
	case PROLOG_OPT:
		n, m := p.next(), p.next()
		p.prolog(n, m, false)
		
	case PROLOG_REST:
		n, m := p.next(), p.next()
		p.prolog(n, m, true)
		
	case EXTEND:
		p.extend(nil)
	
	case EXTENDA:
		n := p.next()
		p.extend(p.u.a[n])
		
	case FINISH:
		n := p.next()
		p.finish(n)
		
	case NEW:
		c := p.v.ToClass()
		p.t = c.alloc()
		p.v = c.m[_Object_new]
		
	case GET:
		n, m := p.next(), p.next()
		a := p.u.a[n]
		p.v = p.v.get(a, p.lookups(m))
		
	case GETM:
		n, m := p.next(), p.next()
		a := p.u.a[n]
		p.v = p.v.getMethod(a, p.lookups(m))
	
	case SET:
		n, m := p.next(), p.next()
		a := p.u.a[n]
		p.v.set(a, p.lookups(m), p.pop())
		p.v = Nil
		
	case THIS:
		p.v = p.t
		
	case LTHIS:
		p.t = p.v
		
	case SUPER:
		n := p.next()
		if n == slotUnknown {
			panic(fmt.Errorf("only use super with methods you have overridden"))
		}
		e := p.sc.e[n]
		if e.offset >= uint16(len(p.sc.a.m)) {
			panic(fmt.Errorf("not present on ancestor: %s.%s", p.sc.n, e.Name))
		}
		p.v = p.sc.a.m[e.offset]
	
	case SOURCE:
		n, m := p.next(), p.next()
		if n != 0 {
			p.file = p.u.v[n]
		}
		p.line = m
//		fmt.Println(p.file, p.line)
	
	default:
		panic(fmt.Errorf("unrecognised opcode: %d", op))
	}
}

/*******************************************************************************

	Compiled code

*******************************************************************************/

// Copy the unit. You should use copies if you are trying to run a unit multiple
// times.
func (u *Unit) Copy() *Unit {
	res := *u
	res.g = make([]*Object, len(u.g))
	res.a = make([]*Accessor, len(u.a))
	return &res
}

func (u *Unit) link(i *Interpreter) {
	for j, x := range u.gn {
		u.g[j] = i.lookup(x)
	}
	for j, x := range u.an {
		u.a[j] = i.Accessor(x)
	}
}

const (
	magic1 = 0x4200
	magic2 = 0x4353
	version = 0
)

const (
	hMagic1 = iota
	hMagic2
	hVersion
	hGlobals
	hAccessors
	hBlocks
	hCode
	hStrings
	hInts
	hFloats
	hSkeletons
	hSkeletonSize
	hSize
)

func read(r io.Reader, x interface{}) {
	err := binary.Read(r, binary.LittleEndian, x)
	if err != nil {
		panic(err)
	}	
}

func write(w io.Writer, x interface{}) {
	err := binary.Write(w, binary.LittleEndian, x)
	if err != nil {
		panic(err)
	}	
}

func readBlock(r io.Reader, c int) []uint16 {
	buf := make([]uint16, c)
	read(r, buf)
	return buf
}

func readString(r io.Reader, stop byte) string {
	var b byte
	var res string
	for {
		read(r, &b)
		if b == stop {
			break
		}
		res = res + string(b)
	}
	return res
}

func writeString(w io.Writer, s string) {
	_, err := io.WriteString(w, s)
	if err != nil {
		panic(err)
	}
	write(w, byte(0))
}

func readHeader(r io.Reader) (header []uint16, ok bool) {
	defer func() {
		recover();
		ok = false
	}()
	header = readBlock(r, hSize)
	ok = header[hMagic1] == magic1 && 
	     header[hMagic2] == magic2 &&
	     header[hVersion] == version
	return
}

// Load a compiled file. Panics on error. Returns whether or not the unit is in
// a valid format.
func (u *Unit) Load(r io.Reader) bool {
	// header
	header, ok := readHeader(r)
	if !ok {
		return false
	}
	globals := int(header[hGlobals])
	Accessors := int(header[hAccessors])
	blocks := int(header[hBlocks])
	code := int(header[hCode])
	strings := int(header[hStrings])
	ints := int(header[hInts])
	floats := int(header[hFloats])
	skeletons := int(header[hSkeletons])
	skeletonSize := int(header[hSkeletonSize])
	values := strings + ints  + floats + skeletons
	
	// globals, Accessors
	u.g = make([]*Object, globals)
	u.gn = make([]string, globals)
	u.a = make([]*Accessor, Accessors)
	u.an = make([]string, Accessors)
	for i := range u.gn {
		u.gn[i] = readString(r, 0)
	}
	for i := range u.an {
		u.an[i] = readString(r, 0)
	}

	// blocks
	u.b = make([][]uint16, blocks)
	cbuf := readBlock(r, code)
	clens := readBlock(r, blocks)
	p := 0
	for i, x := range clens {
		n := p + int(x)
		u.b[i] = cbuf[p:n]
		p = n
	}

	// data
	u.v = make([]*Object, values + 3)
	u.v[0] = Nil
	u.v[1] = True
	u.v[2] = False
	vlocs := readBlock(r, values)
	p = 0
	for i := 0; i < strings; i++ {
		u.v[int(vlocs[i])] = Wrap(readString(r, 0))
	}
	p += strings
	for i := 0; i < ints; i++ {
		var ival int32
		read(r, &ival)
		u.v[int(vlocs[i+p])] = Wrap(int(ival))
	}
	p += ints
	for i := 0; i < floats; i++ {
		var fval float64
		read(r, &fval)
		u.v[int(vlocs[i+p])] = Wrap(fval)
	}
	p += floats
	
	// skeletons
	sbuf := readBlock(r, skeletonSize)
	slens := readBlock(r, skeletons)
	for i := 0; i < skeletons; i++ {
		l := int(slens[i])
		es := make([]Slot, l+1)
		es[0].Name = readString(r, 0)
		for j := 0; j < l; j++ {
			desc := sbuf[3*j]
			es[j+1].Kind = SlotKind(desc & 0xff)
			es[j+1].Vis = SlotVis(desc >> 8)
			es[j+1].access = sbuf[3*j+1]
			es[j+1].next = sbuf[3*j+2]
		}
		u.v[int(vlocs[i+p])] = new(skelObj).init(es)
		sbuf = sbuf[3*l:]
	}
	return true
}

// Save a compiled file. Panics on error.
func (u *Unit) Save(w io.Writer) {
	// header 
	header := make([]uint16, hSize)
	header[hMagic1] = magic1
	header[hMagic2] = magic2
	header[hVersion] = version
	header[hGlobals] =  uint16(len(u.g))
	header[hAccessors] = uint16(len(u.a))
	
	// blocks
	var cbuf []uint16
	clens := make([]uint16, len(u.b))
	for i, x := range u.b {
		clens[i] = uint16(len(x))
		cbuf = append(cbuf, x...)
	}
	header[hBlocks] = uint16(len(u.b))
	header[hCode] = uint16(len(cbuf))
	
	// values
	var stringPs, intPs, floatPs, skeletonPs []uint16
	var strings, ints, floats, skeletons []interface{}
	skeletonSize := 0
	for i, x := range u.v[3:] {
		switch x.c {
		case StringClass:
			stringPs = append(stringPs, uint16(i+3))
			strings = append(strings, x.ToString())
		case IntClass:
			intPs = append(intPs, uint16(i+3))
			ints = append(ints, x.ToInt())
		case FltClass:
			floatPs = append(floatPs, uint16(i+3))
			floats = append(floats, x.ToFloat())
		case skeletonClass:
			sk := x.skelData()
			skeletonPs = append(skeletonPs, uint16(i+3))
			skeletons = append(skeletons, sk)
			skeletonSize += (len(sk)-1)*3
		}
	}
	skeletonCount := len(skeletons)
	header[hStrings] = uint16(len(strings))
	header[hInts] = uint16(len(ints))
	header[hFloats] = uint16(len(floats))
	header[hSkeletons] = uint16(skeletonCount)
	header[hSkeletonSize] = uint16(skeletonSize)

	// write the header
	write(w, header)
	
	// globals and Accessors
	for _, x := range u.gn {
		writeString(w, x)
	}
	for _, x := range u.an {
		writeString(w, x)
	}
	
	// write blocks
	write(w, cbuf)
	write(w, clens)
	
	// write values
	write(w, stringPs)
	write(w, intPs)
	write(w, floatPs)
	write(w, skeletonPs)
	for _, x := range strings {
		writeString(w, x.(string))
	}
	for _, x := range ints {
		write(w, int32(x.(int)))
	}
	for _, x := range floats {
		write(w, x.(float64))
	}
	
	// skeletons
	ns := make([]string, skeletonCount)
	sbuf := make([]uint16, 0, skeletonSize)
	slens := make([]uint16, skeletonCount)
	for i, x := range skeletons {
		es := x.([]Slot)
		ns[i] = es[0].Name
		l := len(es) - 1
		slens[i] = uint16(l)
		sk := make([]uint16, 3*l)
		for j := 0; j < l; j++ {
			y := es[j+1]
			desc := (uint16(y.Vis) << 8) + uint16(y.Kind)
			sk[3*j] = desc
			sk[3*j+1] = y.access
			sk[3*j+2] = y.next
		}
		sbuf = append(sbuf, sk...)
	}
	write(w, sbuf)
	write(w, slens)
	for _, x := range ns {
		writeString(w, x)
	}
}




