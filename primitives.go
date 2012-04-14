package ts

import (
	"fmt"
	"unsafe"
	"strings"
	"strconv"
	"os"
	"regexp"
	"unicode/utf8"
)

/*******************************************************************************

	Main entrypoints

*******************************************************************************/

// The root class. All other classes descend from this one.
var ObjectClass *Class

// Primitive data types. The normal operations on classes (extension, creation)
// do not work on these classes.
var ClassClass, AccessorClass, NilClass, BooleanClass, TrueClass, FalseClass,
    StringClass, NumberClass, IntClass, FltClass, FunctionClass, ArrayClass,
    HashClass, CollectionClass, IteratorClass, ErrorClass,
    StreamClass, frameClass, skeletonClass, boxClass, undefinedClass,
    stringIterator, arrayIterator, hashIterator *Class

// Built in values.
var Nil, True, False *Object

func init() {
	initBaseClasses()
	initSimpleClasses()
	initNumberClasses()
	initCollectionClasses()
	initDataClasses()
}

// Given a bool, int, float64, string or []*Object return an object
// corresponding to that value.
//
// If a function is passed in: This function should take an object argument
// representing the receiver, and then zero to four other object arguments
// representing the arguments to the primitive. Alternatively, an array of
// objects may represent the arguments.
//
// Ignore the receiver in the case of functions that are not methods; its value 
// is undefined.
//
func Wrap(x interface{}) *Object {
	if x == nil {
		return Nil
	}
	switch v := x.(type) {
	case *Object:
		if v == nil {
			return Nil
		}
		return v
	case *Class:
		if v == nil {
			return Nil
		}
		return v.o
	case bool:
		if v {
			return True
		}
		return False
	case int8:
		return new(intObj).init(int(v))
	case uint8:
		return new(intObj).init(int(v))
	case int16:
		return new(intObj).init(int(v))
	case uint16:
		return new(intObj).init(int(v))
	case int32:
		return new(intObj).init(int(v))
	case uint32:
		return new(intObj).init(int(v))
	case int64:
		return new(intObj).init(int(v))
	case uint64:
		return new(intObj).init(int(v))
	case int:
		return new(intObj).init(v)
	case float32:
		return new(fltObj).init(float64(v))
	case float64:
		return new(fltObj).init(v)
	case string:
		return new(strObj).init(v)
	case []*Object:
		return new(arrObj).init(v)
	case []int:
		res := make([]*Object, len(v))
		for i, x := range v {
			res[i] = Wrap(x)
		}
		return Wrap(res)
	case []string:
		res := make([]*Object, len(v))
		for i, x := range v {
			res[i] = Wrap(x)
		}
		return Wrap(res)
	case map[*Object] *Object:
		if v == nil {
			return Nil
		}
		res := make(map[interface{}] *Object)
		for k, v := range v {
			res[keyData(k)] = v
		}
		return new(hashObj).init(res)
	case map[interface{}] *Object:
		if v == nil {
			return Nil
		}
		return new(hashObj).init(v)
	case func(*Object, []*Object) *Object:
		if v == nil {
			return Nil
		}
		return new(funcObj).init(func(p *process) {
			p.b = len(p.s) - p.n
			p.ret(v(p.t, p.args()))
		})
	case func(o *Object) *Object:
		if v == nil {
			return Nil
		}
		return new(funcObj).init(func(p *process) {
			p.b = len(p.s) - p.n
			p.parseArgs()
			p.ret(v(p.t))
		})
	case func(o, a *Object) *Object:
		if v == nil {
			return Nil
		}
		return new(funcObj).init(func(p *process) {
			var a *Object
			p.b = len(p.s) - p.n
			p.parseArgs(&a)
			p.ret(v(p.t, a))
		})
	case func(o, a, b *Object) *Object:
		if v == nil {
			return Nil
		}
		return new(funcObj).init(func(p *process) {
			var a, b *Object
			p.b = len(p.s) - p.n
			p.parseArgs(&a, &b)
			p.ret(v(p.t, a, b))
		})
	case func(o, a, b, c *Object) *Object:
		if v == nil {
			return Nil
		}
		return new(funcObj).init(func(p *process) {
			var a, b, c *Object
			p.b = len(p.s) - p.n
			p.parseArgs(&a, &b, &c)
			p.ret(v(p.t, a, b, c))
		})
	case func(o, a, b, c, d *Object) *Object:
		if v == nil {
			return Nil
		}
		return new(funcObj).init(func(p *process) {
			var a, b, c, d *Object
			p.b = len(p.s) - p.n
			p.parseArgs(&a, &b, &c, &d)
			p.ret(v(p.t, a, b, c, d))
		})
	case error:
		return Wrap(v.Error())
	}
	panic(fmt.Errorf("invalid type: %v", x))
}

// Public field slot.
func FSlot(n string, f interface{}) Slot {
	return Slot{Kind: Field, Vis: Public, Name: n, Value: Wrap(f)}
}

// Private field slot.
func PSlot(n string, f interface{}) Slot {
	return Slot{Kind: Field, Vis: Private, Name: n, Value: Wrap(f)}
}

// Public method slot.
func MSlot(n string, f interface{}) Slot {
	return Slot{Kind: Method, Vis: Public, Name: n, Value: Wrap(f)}
}

// Public property slot.
func PropSlot(n string, g, s interface{}) Slot {
	gv, sv := Wrap(g), Wrap(s)
	return Slot{Kind: Property, Vis: Public, Name: n, Value: gv, Set: sv}
}

// Slot describing a method that descendant classes ought to implement.
func AbstractMethod(n string) Slot {
	return MSlot(n, func(o *Object, args []*Object) *Object {
		panic(fmt.Errorf("abstract method: %s.%s", o.c.n, n))
	})
}

// Retrieve int associated with the object. Panics if there is no such datum.
func (o *Object) ToInt() int {
	o.checkClass(o.c == IntClass)
	return (*intObj)(unsafe.Pointer(o)).d
}

// Retrieve float64 associated with the object. Panics if there is no such
// datum.
func (o *Object) ToFloat() float64 {
	o.checkClass(o.c == FltClass)
	return (*fltObj)(unsafe.Pointer(o)).d
}

// Retrieve string associated with the object. Panics if there is no such datum.
func (o *Object) ToString() string {
	o.checkClass(o.c == StringClass)
	return (*strObj)(unsafe.Pointer(o)).d
}

// Retrieve []*Object associated with the object. Panics if there is no such 
// datum.
func (o *Object) ToArray() []*Object {
	o.checkClass(o.c == ArrayClass)
	return (*arrObj)(unsafe.Pointer(o)).d
}

// Retrieve *Class associated with the object. Panics if there is no such datum.
func (o *Object) ToClass() *Class {
	o.checkClass(o.c == ClassClass)
	return (*clsObj)(unsafe.Pointer(o)).d
}

func (o *Object) UserData() interface{} {
	o.checkClass(o.c.flags & UserData != 0)
	return (*userObj)(unsafe.Pointer(o)).d
}

func (o *Object) SetUserData(x interface{}) {
	o.checkClass(o.c.flags & UserData != 0)
	(*userObj)(unsafe.Pointer(o)).d = x
}

var extensions = map[string] func(*Interpreter) {}

// Inform the system about an extension to the language.
func RegisterExtension(n string, f func(*Interpreter)) {
	extensions[n] = f
}

func PrimitivePackage(n string, f func(*Interpreter) map[string] *Object) {
	RegisterExtension(n, func(itpr *Interpreter) {
		es := []Slot{}
		for k, v := range f(itpr) {
			es = append(es, Slot{Name: k, Kind: Field, Vis: Public, Value: v})
		}
		pkg := itpr.Get("Package").ToClass().Extend(itpr, "Package", 0, es)
		aset := itpr.Accessor("__aset__")
		itpr.Get("packages").Call(aset, Wrap(n), pkg.New(Wrap(n)))
	})
}

// registration function called by New()
func definePrimitives(i *Interpreter) {
	path := i.Accessor("path")
	loaded := map[string] *Object{}
	pmClass := ObjectClass.extend("PackageManager", 0, []Slot {
		FSlot("path", root() + "/pkg"),
		MSlot("copy", func(o *Object) *Object {
			return o;
		}),
		MSlot("unload", func(o, nm *Object) *Object {
			delete(loaded, nm.String())
			return Nil
		}),
		MSlot("__aget__", func(o, nm *Object) *Object {
			name := nm.ToString()
			// already loaded
			if pkg := loaded[name]; pkg != nil {
				return pkg
			}
			// defined as an extension
			if f := extensions[name]; f != nil {
				f(i)
			} else {
			// written in TranScript
				i.Load(o.Get(path).ToString() + "/" +  name + ".pkg")
			}
			if pkg := loaded[name]; pkg != nil {
				return pkg
			}
			panic(fmt.Errorf("package improperly defined: %s", name))
		}),
		MSlot("__aset__", func(o, i, x *Object) *Object {
			loaded[i.ToString()] = x
			return Nil		
		}),	
	})
	var pkgClass *Class
	pkgClass = ObjectClass.extend("Package", 0, []Slot{
		PSlot("name_f", Nil),
		MSlot("create", func(o, n *Object) *Object {
			pkgClass.Set(o, 0, n)
			return Nil
		}),
		PropSlot("name", func(o *Object) *Object {
			return pkgClass.Get(o, 0)
		}, Nil),
	})
	
	AccessorClass.n = ""
	cs := []*Class {
		ObjectClass, ClassClass, FunctionClass, AccessorClass,
		BooleanClass, TrueClass, FalseClass, NilClass,
		NumberClass, IntClass, FltClass,
		IteratorClass, CollectionClass,
		StringClass, ArrayClass,  
		HashClass, pmClass, pkgClass,
		arrayIterator, hashIterator, stringIterator, ErrorClass,
		StreamClass,
	}
	for _, x := range cs {
		x.added = false
		i.addClass(x)
		if x.n != "" {
			i.Define(x.n, x.o)
		}
	}
	AccessorClass.n = "Accessor"
	
	var accClass *Class
	accClass = AccessorClass.Extend(i, "Accessor", Final, []Slot {
		MSlot("__new__", func(o *Object, name *Object) *Object {
			nm := name.ToString()
			if nm == "" {
				panic(fmt.Errorf("bad name"))
			}
			res := new(accObj).init(i.Accessor(nm))
			res.c = accClass
			return res
		}),
	})
	i.Define("Accessor", accClass.o)
		
	i.Define("packages", pmClass.alloc())
	pmClass.flags = Final|Primitive
	
	i.Define("read", Wrap(func(o *Object) *Object {
		return Wrap(readString(os.Stdin, '\n'))
	}))
	
	i.Define("print", Wrap(func(o *Object, args []*Object) *Object {
		as := make([]interface{}, len(args))
		for i, x := range args {
			as[i] = x
		}
		fmt.Println(as...)
		return Nil
	}))
	
	i.Define("exit", Wrap(func(o, n *Object) *Object {
		os.Exit(n.ToInt())
		return Nil
	}))
	
	i.Define("throw", Wrap(func(o, x *Object) *Object {
		panic(x)
	}))
	i.Define("catch", new(funcObj).init(func(p *process) {
		defer func() {
			if e := recover(); e != nil {
				p.ret(p.wrapError(e))
			}
		}()
		var thk *Object
		p.b = len(p.s) - p.n
		p.parseArgs(&thk)
		thk.Call(nil)
		p.ret(False)
	}))
	
}

/*******************************************************************************

	Primitive data

*******************************************************************************/

func (o *Object) checkClass(pass bool) {
	if !pass {
		panic(fmt.Errorf("wrong type: %s", o.c.n))
	}
}

type userObj struct {
	Object
	d interface{}
}

func (o *userObj) init(c *Class, f []*Object) *Object {
	o.c = c
	o.f = f
	return (*Object)(unsafe.Pointer(o))
}

type funcObj struct {
	Object
	d func(*process)
}

func (o *Object) funcData() func(*process) {
	if o.c == FunctionClass {
		return (*funcObj)(unsafe.Pointer(o)).d
	}
	return o.bindMethod(o.c.m[_Object_call]).funcData()
}

func (o *funcObj) init(x func(*process)) *Object {
	o.c = FunctionClass
	o.f = []*Object{False}
	o.d = x
	return (*Object)(unsafe.Pointer(o))
}

type intObj struct {
	Object
	d int
}

func (o *intObj) init(x int) *Object {
	o.c = IntClass
	o.d = x
	return (*Object)(unsafe.Pointer(o))
}

type fltObj struct {
	Object
	d float64
}

func (o *fltObj) init(x float64) *Object {
	o.c = FltClass
	o.d = x
	return (*Object)(unsafe.Pointer(o))
}

type strObj struct {
	Object
	d string
}

func (o *strObj) init(x string) *Object {
	o.c = StringClass
	o.d = x
	return (*Object)(unsafe.Pointer(o))
}

type arrObj struct {
	Object
	d []*Object
}

func (o *Object) setArray(x []*Object) {
	o.checkClass(o.c == ArrayClass)
	(*arrObj)(unsafe.Pointer(o)).d = x
}

func (o *arrObj) init(x []*Object) *Object {
	o.c = ArrayClass
	o.d = x
	return (*Object)(unsafe.Pointer(o))
}

type hashObj struct {
	Object
	d map[interface{}] *Object
}

func (o *Object) hashData() map[interface{}] *Object {
	o.checkClass(o.c == HashClass)
	return (*hashObj)(unsafe.Pointer(o)).d
}

func (o *hashObj) init(m map[interface{}] *Object) *Object {
	o.c = HashClass
	o.d = m
	if o.d == nil {
		o.d = make(map[interface{}] *Object)
	}
	return (*Object)(unsafe.Pointer(o))
}

type skelObj struct {
	Object
	d []Slot
}

func (o *Object) skelData() []Slot {
	o.checkClass(o.c == skeletonClass)
	return (*skelObj)(unsafe.Pointer(o)).d
}

func (o *skelObj) init(x []Slot) *Object {
	o.c = skeletonClass
	o.d = x
	return (*Object)(unsafe.Pointer(o))
}

type clsObj struct {
	Object
	d *Class
}

func (o *clsObj) init(x *Class) *Object {
	o.c = ClassClass
	o.f = []*Object{False}
	o.d = x
	return (*Object)(unsafe.Pointer(o))
}

type boxObj struct {
	Object
	d *Object
}

func (o *Object) boxData() *Object {
	o.checkClass(o.c == boxClass || o.c == undefinedClass)
	return (*boxObj)(unsafe.Pointer(o)).d
}

func (o *Object) setBoxData(x *Object) {
	o.checkClass(o.c == boxClass)
	(*boxObj)(unsafe.Pointer(o)).d = x
}

func (o *boxObj) init(x *Object) *Object {
	o.c = boxClass
	o.d = x
	return (*Object)(unsafe.Pointer(o))
}

type accObj struct {
	Object
	d *Accessor
}

func (o *Object) accessorData() *Accessor {
	o.checkClass(o.c.Is(AccessorClass))
	return (*accObj)(unsafe.Pointer(o)).d
}

func (o *accObj) init(x *Accessor) *Object {
	o.c = AccessorClass
	o.d = x
	return (*Object)(unsafe.Pointer(o))
}

/*******************************************************************************

	Class Specifications

*******************************************************************************/

// These are slots defined on every object.
const (
	_Object_new = iota
	_Object_create
	_Object_eq
	_Object_call
	_Object_getFailed
	_Object_setFailed
	_Object_callFailed
	_Object_toString
	_Object_equals
	_Object_type
)

func initBaseClasses() {
	ObjectClass = &Class{n: "Object"}
	ClassClass = &Class{n: "Class", a: ObjectClass}
	ClassClass.o = new(clsObj).init(ClassClass)
	ObjectClass.o = new(clsObj).init(ObjectClass)

	boxClass = &Class{
		flags: Final|Primitive,
		n: "Box",
		a: ObjectClass,
	}
	undefinedClass = &Class{
		flags: Final|Primitive,
		n: "Undefined",
		a: ObjectClass,
	}
	skeletonClass = &Class{
		flags: Final|Primitive,
		n: "Skeleton",
		a: ObjectClass,
	}

	FunctionClass = &Class{
		flags: Final|Primitive,
		n: "Function",
		a: ObjectClass,
	}
	FunctionClass.o = new(clsObj).init(FunctionClass)
	FunctionClass.e = []Slot {
		FSlot("help", False),
		MSlot("copy", func(o *Object) *Object {
			return o;
		}),
		MSlot("__call__", func(o *Object, args []*Object) *Object {
			return o.Call(nil, args...)
		}),
	}
	
	ObjectClass.e = []Slot {
		MSlot("__new__", func(o *Object, args []*Object) *Object {
			ObjectClass.Call(o, _Object_create, args...)
			return o
		}),
		MSlot("create", func(o *Object) *Object {
			return Nil
		}),
		MSlot("__eq__", func(o, x *Object) *Object {
			return Wrap(o == x)
		}),
		MSlot("__call__", func(o *Object, args []*Object) *Object {
			panic(fmt.Errorf("wrong type: %s", o.c.n))
		}),
		MSlot("__getFailed__", func(o, a *Object) *Object {
			panic(fmt.Errorf("undefined: %s.%s", o.c.n, a.accessorData().n))
		}),
		MSlot("__setFailed__", func(o, a, x *Object) *Object {
			panic(fmt.Errorf("undefined: %s.%s", o.c.n, a.accessorData().n))
		}),
		MSlot("__callFailed__", func(o *Object, args []*Object) *Object {
			if len(args) < 1 {
				 panic(fmt.Errorf("wrong number of arguments %d", len(args)))
			}
			a := args[0].accessorData()
			panic(fmt.Errorf("undefined: %s.%s", o.c.n, a.n))
		}),
		MSlot("toString", func(o *Object) *Object {
			return Wrap(fmt.Sprintf("#<%s>", o.c.n))
		}),
		MSlot("equals", func(o, x *Object) *Object {
			return ObjectClass.Call(o, _Object_eq, x)
		}),
		PropSlot("type", func(o *Object) *Object {
			return o.c.o
		}, Nil),
		MSlot("copy", func(o *Object) *Object {
			f := make([]*Object, len(o.f))
			copy(f, o.f)
			return &Object{o.c, f}
		}),
		MSlot("apply", func(o, args *Object) *Object {
			return o.Call(nil, args.ToArray()...)
		}),
		MSlot("is", func(o, d *Object) *Object {
			c := ObjectClass.Get(o, _Object_type).ToClass()
			return Wrap(c.Is(d.ToClass()))
		}),
		MSlot("__neq__", func(o, x *Object) *Object {
			return Wrap(ObjectClass.Call(o, _Object_eq, x) == False)
		}),
		MSlot("__inv__", func(o *Object) *Object {
			return False
		}),
	}
	
	ClassClass.e = []Slot {
		FSlot("help", False),
		MSlot("is", func(o, x *Object) *Object {
			c := o.ToClass()
			d := o.ToClass()
			return Wrap(c.Is(d))
		}),
		MSlot("copy", func(o *Object) *Object {
			return o;
		}),
		MSlot("name", func(o *Object) *Object {
			return Wrap(o.ToClass().n)
		}),
		MSlot("names", func(o, flags *Object) *Object {
			c := o.ToClass()
			s := flags.ToString()
			hook := strings.Index(s, "+") != -1
			deep := strings.Index(s, "*") != -1
			return Wrap(c.Names(hook, deep))
		}),
		MSlot("values", func(o *Object) *Object {
			c := o.ToClass()
			res := new(hashObj).init(nil)
			m := res.hashData()
			for _, x := range c.e {
				if x.Vis == Public {
					m[x.Name] = x.Value
				}
			}
			return res
		}),
		MSlot("info", func(o *Object) *Object {
			c := o.ToClass()
			fmt.Println(c.n)
			fmt.Println("-----")
			for i, e := range c.e {
				nm := e.Name
				if e.Class != c {
					nm = e.Class.n + "." + nm
				}
				switch e.Kind {
				case Method:
					fmt.Println(i, e.offset, nm + "()")
				case Field:
					fmt.Println(i, e.offset, nm, e.Value)
				case Marker:
					fmt.Println(i, "-->", e.next, nm)
				}
			}
			return Nil
		}),
	}
	ClassClass.flags = Final|Primitive
	
	AccessorClass = ObjectClass.extend("Accessor", Primitive, []Slot {
		MSlot("copy", func(o *Object) *Object {
			return o;
		}),
		MSlot("name", func(a *Object) *Object {
			return Wrap(a.accessorData().n)
		}),
		MSlot("defined", func(a, o *Object) *Object {
			return Wrap(o.Defined(a.accessorData()))
		}),
		MSlot("get", func(a, o *Object) *Object {
			return o.Get(a.accessorData())
		}),
		MSlot("set", func(a, o, x *Object) *Object {
			o.Set(a.accessorData(), x)
			return Nil
		}),
		MSlot("call", func(a *Object, args []*Object) *Object {
			o := args[0]
			args = args[1:]
			return o.Call(a.accessorData(), args...)
		}),
		MSlot("info", func(o *Object) *Object {
			a := o.accessorData()
			fmt.Println(a.n)
			fmt.Println("-----")
			for _, e := range a.e {
				nm := e.Name
				if e.Kind == Method {
					nm += "()"
				}
				fmt.Println(e.offset, e.Class.n, nm)
			}
			return Nil
		}),
	})
	
	ErrorClass = ObjectClass.extend("Error", 0, []Slot {
		FSlot("msg", ""),
		FSlot("file", ""),
		FSlot("line", 0),
		MSlot("toString", func(o *Object) *Object {
			msg := ErrorClass.Get(o, 0)
			file := ErrorClass.Get(o, 1)
			line := ErrorClass.Get(o, 2).ToInt()
			if line == 0 {
				return msg
			}
			return Wrap(fmt.Sprintf("%s(%d): %s", file, line, msg))
		}),
		MSlot("create", func(o, msg *Object) *Object {
			ErrorClass.Set(o, 0, msg)
			return Nil
		}),
	})
}

func (c *Class) Names(hook, deep bool) []string {
	in := map[string] bool{}
	classScanNames(c, in, hook, deep)
	res := []string{}
	for x := range in {
		res = append(res, x)
	}
	return res
}

func classScanNames(c *Class, in map[string] bool, hook, deep bool) {
	if hook {
		for _, x := range c.e {
			if x.Vis == Public {
				in[x.Name] = true
			}
		}
	} else {
		for _, x := range c.e {
			if strings.HasPrefix(x.Name, "__") &&
			   strings.HasSuffix(x.Name, "__") {
			   continue
			}
			if x.Vis == Public {
				in[x.Name] = true
			}
		}
	}	
	if deep && c.a != nil {
		classScanNames(c.a, in, hook, deep)
	}
}

func keyData(o *Object) interface{} {
	switch o.c {
	case StringClass:
		return o.ToString()
	case IntClass:
		return o.ToInt()
	case FltClass:
		return o.ToFloat()
	}
	return o
}

func trimString(args []*Object) string {
	switch len(args) {
	case 0:
		return " \n\t"
	case 1:
		return args[0].ToString()
	}
	panic(fmt.Errorf("wrong number of arguments %d", len(args)))
}

func initSimpleClasses() {
	NilClass = ObjectClass.extend("Nil", Final|Primitive, []Slot {
		MSlot("copy", func(o *Object) *Object {
			return o;
		}),
		MSlot("toString", func(o *Object) *Object {
			return Wrap("nil")
		}),
	})
	Nil = &Object{c: NilClass}

	BooleanClass = ObjectClass.extend("Boolean", 0, []Slot {
		MSlot("copy", func(o *Object) *Object {
			return o;
		}),
	})
	
	TrueClass = BooleanClass.extend("True", 0, []Slot {
		MSlot("toString", func(o *Object) *Object {
			return Wrap("true")
		}),
	})
	FalseClass = BooleanClass.extend("False", 0, []Slot {
		MSlot("toString", func(o *Object) *Object {
			return Wrap("false")
		}),
		MSlot("__inv__", func(o *Object) *Object {
			return True
		}),
	})
	True = &Object{c: TrueClass}
	False = &Object{c: FalseClass}
	BooleanClass.flags = Final|Primitive
	TrueClass.flags = Final|Primitive
	FalseClass.flags = Final|Primitive
}

func initDataClasses() {
	stringIterator = IteratorClass.extend("ArrayIterator", UserData, []Slot {
		PSlot("index", 0),
		PSlot("res", Nil),
		MSlot("create", func(o, s *Object) *Object {
			o.SetUserData([]rune(s.ToString()))
			return Nil
		}),
		MSlot("key", func(o *Object) *Object {
			return stringIterator.Get(o, 0)
		}),
		MSlot("value", func(o *Object) *Object {
			i := stringIterator.Get(o, 0).ToInt()
			a := o.UserData().([]rune)
			return Wrap(string(a[i]))
		}),
		MSlot("next", func(o *Object) *Object {
			i := stringIterator.Get(o, 0).ToInt()
			stringIterator.Set(o, 0, Wrap(i+1))
			return Nil
		}),
		MSlot("done", func(o *Object) *Object {
			i := stringIterator.Get(o, 0).ToInt()
			a := o.UserData().([]rune)
			return Wrap(i >= len(a))
		}),
		MSlot("clear", func(o *Object) *Object {
			stringIterator.Set(o, 1, Wrap([]*Object{}))
			return Nil			
		}),
		MSlot("set", func(o, x *Object) *Object {
			a := stringIterator.Get(o, 1)
			ArrayClass.Call(a, 1, x) 
			return Nil
		}),
		MSlot("result", func(o *Object) *Object {
			a := stringIterator.Get(o, 1)
			return ArrayClass.Call(a, 0)
		}),
	})
	StringClass = CollectionClass.extend("String", Final|Primitive, []Slot {
		MSlot("iterate", func(o *Object) *Object {
			return stringIterator.New(o)
		}),
		MSlot("copy", func(o *Object) *Object {
			return o;
		}),
		MSlot("split", func(o *Object, args []*Object) *Object {
			if len(args) > 1 {
				 panic(fmt.Errorf("wrong number of arguments %d", len(args)))
			}
			sep := ""
			if len(args) == 1 {
				sep = args[0].ToString()
			}
			return Wrap(strings.Split(o.ToString(), sep))
		}),
		MSlot("toString", func(o *Object) *Object {
			return o
		}),
		MSlot("toInt", func(o *Object) *Object {
			i, err := strconv.Atoi(o.ToString())
			if err != nil {
				panic(err)
			}
			return Wrap(i)
		}),
		MSlot("toFloat", func(o *Object) *Object {
			f, err := strconv.ParseFloat(o.ToString(), 64)
			if err != nil {
				panic(err)
			}
			return Wrap(f)
		}),
		MSlot("toNumber", func(o *Object) *Object {
			i, err := strconv.Atoi(o.ToString())
			if err != nil {
				f, err := strconv.ParseFloat(o.ToString(), 64)
				if err != nil {
					panic(err)
				}
				return Wrap(f)
			}
			return Wrap(i)
		}),
		MSlot("startsWith", func(o, s *Object) *Object {
			return Wrap(strings.HasPrefix(o.ToString(), s.ToString()))
		}),
		MSlot("endsWith", func(o, s *Object) *Object {
			return Wrap(strings.HasSuffix(o.ToString(), s.ToString()))
		}),
		MSlot("contains", func(o, s *Object) *Object {
			return Wrap(strings.Index(o.ToString(), s.ToString()) != -1)
		}),
		MSlot("matches", func(o, s *Object) *Object {
			m, err := regexp.MatchString(s.ToString(), o.ToString())
			if err != nil {
				panic(err)
			}
			return Wrap(m)
		}),
		MSlot("subst", func(o *Object, args []*Object) *Object {
			res := ""
			i := 0
			for _, c := range o.ToString() {
				if c == '%' {
					cur := args[i]
					res += cur.String()
					i++
					continue
				}
				res += string(c)
			}
			return Wrap(res)
		}),
		PropSlot("size", func(o *Object) *Object {
			return Wrap(utf8.RuneCountInString(o.ToString()))
		}, Nil),
		MSlot("trim", func(o *Object, args []*Object) *Object {
			return Wrap(strings.Trim(o.ToString(), trimString(args)))
		}),
		MSlot("trimLeft", func(o *Object, args []*Object) *Object {
			return Wrap(strings.TrimLeft(o.ToString(), trimString(args)))
		}),
		MSlot("trimRight", func(o *Object, args []*Object) *Object {
			return Wrap(strings.TrimRight(o.ToString(), trimString(args)))
		}),
		MSlot("quote", func(o *Object) *Object {
			return Wrap(strconv.Quote(o.ToString()))
		}),
		MSlot("unquote", func(o *Object) *Object {
			s, err := strconv.Unquote(o.ToString())
			if err != nil {
				panic(err)
			}
			return Wrap(s)
		}),
		MSlot("__add__", func(o, s *Object) *Object {
			res := Wrap(o.ToString() + s.ToString())
			return res
		}),
		MSlot("__eq__", func(o, s *Object) *Object {
			if s.c != StringClass {
				return False
			}
			return Wrap(o.ToString() == s.ToString())
		}),
	})

	StreamClass = ObjectClass.extend("Stream", 0, []Slot {
		AbstractMethod("readByte"),
		AbstractMethod("writeByte"),
		AbstractMethod("close"),
		MSlot("readChar", func(o *Object) *Object {
			buf := []byte{}
			for {
				x := StreamClass.Call(o, 0)
				if x == False {
					return False
				}
				b := byte(x.ToInt())
				buf = append(buf, b)
				if b & 0x80 == 0 {
					break
				}
			}
			r, _ := utf8.DecodeRune(buf)
			if r == utf8.RuneError {
				panic(fmt.Errorf("bad encoding"))
			}
			return Wrap(string(r))
		}),
		MSlot("writeChar", func(o, r *Object) *Object {
			buf := []byte{0,0,0,0,0,0}
			n := utf8.EncodeRune(buf, rune(r.ToInt()))
			for i := 0; i < n; i++ {
				StreamClass.Call(o, 1, Wrap(buf[i]))
			}
			return Nil
		}),
		MSlot("writeString", func(o, s *Object) *Object {
			buf := ([]byte)(s.ToString())
			for i := range buf {
				StreamClass.Call(o, 1, Wrap(buf[i]))
			}
			return Nil
		}),
	})
}

func numG(fi func(a, b int) *Object,
          ff func(a, b float64) *Object,
          fe func(a, b *Object) *Object) func(o, b *Object) *Object {
	return func(o, bv *Object) *Object {
		av := o
		if av.c == IntClass && bv.c == IntClass {
			a := av.ToInt()
			b := bv.ToInt()
			return fi(a, b)
		}
		var a, b float64
		if av.c == IntClass {
			a = float64(av.ToInt())
		} else if fe == nil || av.c == FltClass {
			a = av.ToFloat()
		} else {
			return fe(av, bv)
		}
		if bv.c == IntClass {
			b = float64(bv.ToInt())
		} else if fe == nil || bv.c == FltClass {
			b = bv.ToFloat()
		} else {
			return fe(av, bv)
		}
		return ff(a, b)
	}
}

func numOp(fi func(a,b int) int,
           ff func(a,b float64) float64) func(o, b *Object) *Object {
	return numG(func(a,b int) *Object {
		if fi == nil {
			return new(fltObj).init(ff(float64(a), float64(b)))
		}
		return new(intObj).init(fi(a, b))
	}, func(a,b float64) *Object {
		return new(fltObj).init(ff(a, b))
	}, nil)
}

func numPred(fi func(a,b int) bool,
             ff func(a,b float64) bool) func(o, b *Object) *Object {
	return numG(func(a,b int) *Object {
		return Wrap(fi(a, b))
	}, func(a,b float64) *Object {
		return Wrap(ff(a, b))
	}, func(a,b *Object) *Object {
		return False
	})
}

func initNumberClasses() {
	NumberClass = ObjectClass.extend("Number", 0, []Slot {
		AbstractMethod("toInt"),
		AbstractMethod("toFloat"),
		MSlot("copy", func(o *Object) *Object {
			return o;
		}),
		MSlot("__add__", numOp(func(a, b int) int {
			return a + b
		}, func(a, b float64) float64 {
			return a + b
		})),
		MSlot("__sub__", numOp(func(a, b int) int {
			return a - b
		}, func(a, b float64) float64 {
			return a - b
		})),
		MSlot("__mul__", numOp(func(a, b int) int {
			return a * b
		}, func(a, b float64) float64 {
			return a * b
		})),
		MSlot("__div__", numOp(nil, func(a, b float64) float64 {
			return a / b
		})),
		MSlot("__eq__", numPred(func(a, b int) bool {
			return a == b
		}, func(a, b float64) bool {
			return a == b
		})),
		MSlot("__lt__", numPred(func(a, b int) bool {
			return a < b
		}, func(a, b float64) bool {
			return a < b
		})),
		MSlot("__lte__", numPred(func(a, b int) bool {
			return a <= b
		}, func(a, b float64) bool {
			return a <= b
		})),
		MSlot("__gt__", numPred(func(a, b int) bool {
			return a > b
		}, func(a, b float64) bool {
			return a > b
		})),
		MSlot("__gte__", numPred(func(a, b int) bool {
			return a >= b
		}, func(a, b float64) bool {
			return a >= b
		})),
	})

	IntClass = NumberClass.extend("Integer", Final|Primitive, []Slot {
		MSlot("toString", func(o *Object) *Object {
			return Wrap(fmt.Sprint(o.ToInt()))
		}),
		MSlot("toInt", func(o *Object) *Object {
			return o
		}),
		MSlot("toFloat", func(o *Object) *Object {
			return Wrap(float64(o.ToInt()))
		}),
		MSlot("__neg__", func(o *Object) *Object {
			return Wrap(-o.ToInt())
		}),
	})
	
	FltClass = NumberClass.extend("Float", Final|Primitive, []Slot {
		MSlot("toString", func(o *Object) *Object {
			return Wrap(fmt.Sprint(o.ToFloat()))
		}),
		MSlot("toInt", func(o *Object) *Object {
			return Wrap(int(o.ToFloat()))
		}),
		MSlot("toFloat", func(o *Object) *Object {
			return o
		}),
		MSlot("__neg__", func(o *Object) *Object {
			return Wrap(-o.ToFloat())
		}),	
	})
	
	NumberClass.flags = Final|Primitive
}

func setOp(a, b []*Object, op int) (ina, inb, inboth []*Object) {
	loop: for _, x := range a {
		for i := 0; i < len(b); i++ {
			if ObjectClass.Call(x, op, b[i]) != False {
				inboth = append(inboth, x)
				b = append(b[:i], b[i+1:]...)
				continue loop
			}
		}
		ina = append(ina, x)
	}
	inb = b
	return
}

func itDone(it *Object) bool {
	return IteratorClass.Call(it, _Iterator_done) != False
}

func iterate(col *Object, f func(*Object)) {
	it := CollectionClass.Call(col, 0)
	for !itDone(it) {
		f(it)
		IteratorClass.Call(it, _Iterator_next)
	}
}

func iterateWrite(col *Object, f func(*Object)) *Object {
	it := CollectionClass.Call(col, 0)
	IteratorClass.Call(it, _Iterator_clear)
	for !itDone(it) {
		f(it)
		IteratorClass.Call(it, _Iterator_next)
	}
	return IteratorClass.Call(it, _Iterator_result)
}

const (
	_Iterator_key = iota
	_Iterator_value
	_Iterator_next
	_Iterator_done
	_Iterator_clear
	_Iterator_set
	_Iterator_result
)

func initCollectionClasses() {
	IteratorClass = ObjectClass.extend("Iterator", 0, []Slot {
		AbstractMethod("key"),
		AbstractMethod("value"),
		AbstractMethod("next"),
		AbstractMethod("done"),
		AbstractMethod("clear"),
		AbstractMethod("set"),
		AbstractMethod("result"),
	})
	
	CollectionClass = ObjectClass.extend("Collection", 0, []Slot {
		AbstractMethod("iterate"),
		AbstractMethod("__aget__"),
		AbstractMethod("__aset__"),
		AbstractMethod("deepEquals"),
		PropSlot("size", Nil, Nil),
		MSlot("copy", func(o *Object) *Object {
			return iterateWrite(o, func(it *Object) {
				v := IteratorClass.Call(it, _Iterator_value)
				IteratorClass.Call(it, _Iterator_set, v)
			})
		}),
		MSlot("filter", func(o, f *Object) *Object {
			return iterateWrite(o, func(it *Object) {
				v := IteratorClass.Call(it, _Iterator_value)
				k := IteratorClass.Call(it, _Iterator_key)
				if f.Call(nil, k, v) != False {
					IteratorClass.Call(it, _Iterator_set, v)
				}
			})
		}),
		MSlot("map", func(o, f *Object) *Object {
			return iterateWrite(o, func(it *Object) {
				v := IteratorClass.Call(it, _Iterator_value)
				IteratorClass.Call(it, _Iterator_set, f.Call(nil, v))
			})
		}),
		MSlot("each", func(o, f *Object) *Object {
			iterate(o, func(it *Object) {
				k := IteratorClass.Call(it, _Iterator_key)
				v := IteratorClass.Call(it, _Iterator_value)
				f.Call(nil, k, v)
			})
			return Nil
		}),
		MSlot("reduce", func(o, f *Object) *Object {
			var res *Object
			iterate(o, func(it *Object) {
				x := IteratorClass.Call(it, _Iterator_value)
				res = f.Call(nil, res, x)
			})
			return res
		}),
	})

	arrayIterator = IteratorClass.extend("ArrayIterator", 0, []Slot {
		PSlot("array", Nil),
		PSlot("index", 0),
		PSlot("res", Nil),
		MSlot("create", func(o, array *Object) *Object {
			arrayIterator.Set(o, 0, array)
			return Nil
		}),
		MSlot("key", func(o *Object) *Object {
			return arrayIterator.Get(o, 1)
		}),
		MSlot("value", func(o *Object) *Object {
			i := arrayIterator.Get(o, 1).ToInt()
			a := arrayIterator.Get(o, 0).ToArray()
			return a[i]
		}),
		MSlot("next", func(o *Object) *Object {
			i := arrayIterator.Get(o, 1).ToInt()
			arrayIterator.Set(o, 1, Wrap(i+1))
			return Nil
		}),
		MSlot("done", func(o *Object) *Object {
			i := arrayIterator.Get(o, 1).ToInt()
			a := arrayIterator.Get(o, 0).ToArray()
			return Wrap(i >= len(a))
		}),
		MSlot("clear", func(o *Object) *Object {
			arrayIterator.Set(o, 2, Wrap([]*Object{}))
			return Nil
		}),
		MSlot("set", func(o, x *Object) *Object {
			a := arrayIterator.Get(o, 2)
			ArrayClass.Call(a, 1, x)
			return Nil
		}),
		MSlot("result", func(o *Object) *Object {
			return arrayIterator.Get(o, 2)
		}),
	})
	ArrayClass = CollectionClass.extend("Array", Final, []Slot {
		MSlot("join", func(o *Object, args []*Object) *Object {
			if len(args) > 1 {
				 panic(fmt.Errorf("wrong number of arguments %d", len(args)))
			}
			sep := ""
			if len(args) == 1 {
				sep = args[0].ToString()
			}
			os := o.ToArray()
			ss := make([]string, len(os))
			for i, x := range os {
				ss[i] = x.String()
			}
			return Wrap(strings.Join(ss, sep))
		}),
		MSlot("add", func(o *Object, args []*Object) *Object {
			o.setArray(append(o.ToArray(), args...))
			return Nil
		}),
		MSlot("iterate", func(o *Object) *Object {
			return arrayIterator.New(o)
		}),
		MSlot("__new__", func(o, c *Object) *Object {
			arr := make([]*Object, c.ToInt())
			for i := range arr {
				arr[i] = Nil
			}
			return Wrap(arr)
		}),
		MSlot("toString", func(o *Object) *Object {
			s := "["
			for _, x := range o.ToArray() {
				t := x.String()
				if s == "[" {
					s += t
				} else {
					s += ", " + t 
				}
			}
			return Wrap(s + "]")
		}),
		PropSlot("size", func(o *Object) *Object {
			return Wrap(len(o.ToArray()))
		}, Nil),
		MSlot("remove", func(o *Object, x *Object) *Object {
			eq := x.c.m[_Object_eq]
			a := o.ToArray()
			for i := 0; i < len(a); i++ {
				if x.callMethod(eq, a[i:i+1]) != False {
					o.setArray(append(a[:i], a[i+1:]...))
					a = o.ToArray()
					i--
				}
			}
			return Nil
		}),
		MSlot("insert", func(o, _i, x *Object) *Object {
			a := o.ToArray()
			i := _i.ToInt()
			o.setArray(append(a[:i], append([]*Object{x}, a[i:]...)...))
			return Nil
		}),
		MSlot("delete", func(o, _i *Object) *Object {
			a := o.ToArray()
			i := _i.ToInt()
			o.setArray(append(a[:i], a[i+1:]...))
			return Nil
		}),
		MSlot("slice", func(o *Object, args []*Object) *Object {
			t := o.ToArray()
			from := 0
			to := len(t)
			switch len(args) {
			case 2:
				to = args[1].ToInt()
				fallthrough
			case 1:
				from = args[0].ToInt()
			case 0:
			default:
				 panic(fmt.Errorf("wrong number of arguments %d", len(args)))
			}
			return Wrap(t[from:to])
		}),
		MSlot("member", func(o, x *Object) *Object {
			eq := x.c.m[_Object_eq]
			args := []*Object{nil}
			for i, y := range o.ToArray() {
				args[0] = y
				if x.callMethod(eq, args) != False {
					return Wrap(i)
				}
			}
			return False
		}),
		MSlot("subset", func(o, x *Object) *Object {
			a, _, _ := setOp(o.ToArray(), x.ToArray(), _Object_eq)
			return Wrap(len(a) == 0)
		}),
		MSlot("union", func(o, x *Object) *Object {
			a, b, both := setOp(o.ToArray(), x.ToArray(), _Object_eq)
			a = append(a, b...)
			a = append(a, both...)
			return Wrap(a)
		}),
		MSlot("difference", func(o, x *Object) *Object {
			a, _, _ := setOp(o.ToArray(), x.ToArray(), _Object_eq)
			return Wrap(a)
		}),
		MSlot("intersection", func(o, x *Object) *Object {
			_, _, both := setOp(o.ToArray(), x.ToArray(), _Object_eq)
			return Wrap(both)
		}),
		MSlot("__aget__", func(o, i *Object) *Object {
			return o.ToArray()[i.ToInt()]
		}),
		MSlot("__aset__", func(o, i, x *Object) *Object {
			o.ToArray()[i.ToInt()] = x
			return Nil
		}),
		MSlot("__add__", func(o, x *Object) *Object {
			a := o.ToArray()
			b := x.ToArray()
			res := make([]*Object, len(a) + len(b))
			copy(res, a)
			copy(res[len(a):], b)
			return Wrap(res)
		}),
	})
	
	hashIterator = IteratorClass.extend("HashIterator", 0, []Slot {
		PSlot("keys", Nil),
		PSlot("index", 0),
		PSlot("hash", Nil),
		PSlot("res", Nil),
		MSlot("create", func(o, hash *Object) *Object {
			keys := HashClass.Call(hash, 0)
			hashIterator.Set(o, 0, keys)
			hashIterator.Set(o, 2, hash)
			return Nil
		}),
		MSlot("key", func(o *Object) *Object {
			a := hashIterator.Get(o, 0).ToArray()
			i := hashIterator.Get(o, 1).ToInt()
			return a[i]
		}),
		MSlot("value", func(o *Object) *Object {
			a := hashIterator.Get(o, 0).ToArray()
			i := hashIterator.Get(o, 1).ToInt()
			h := hashIterator.Get(o, 2).hashData()
			return h[keyData(a[i])]
		}),
		MSlot("next", func(o *Object) *Object {
			i := hashIterator.Get(o, 1).ToInt()
			hashIterator.Set(o, 1, Wrap(i+1))
			return Nil
		}),
		MSlot("done", func(o *Object) *Object {
			a := hashIterator.Get(o, 0).ToArray()
			i := hashIterator.Get(o, 1).ToInt()
			return Wrap(i >= len(a))
		}),
		MSlot("clear", func(o *Object) *Object {
			h := Wrap(map[interface{}] *Object{})
			hashIterator.Set(o, 3, h)
			return Nil
		}),
		MSlot("set", func(o, x *Object) *Object {
			a := hashIterator.Get(o, 0).ToArray()
			i := hashIterator.Get(o, 1).ToInt()
			h := hashIterator.Get(o, 3).hashData()
			h[keyData(a[i])] = x
			return Nil
		}),
		MSlot("result", func(o *Object) *Object {
			return hashIterator.Get(o, 3)
		}),
	})
	HashClass = CollectionClass.extend("Hash", Final, []Slot {
		MSlot("keys", func(o *Object) *Object {
			res := []*Object{}
			for k := range o.hashData() {
				res = append(res, Wrap(k))
			}
			return Wrap(res)
		}),
		MSlot("iterate", func(o *Object) *Object {
			return hashIterator.New(o)
		}),
		MSlot("__new__", func(o *Object) *Object {
			return new(hashObj).init(nil)
		}),
		MSlot("toString", func(o *Object) *Object {
			res := "{"
			start := true
			for k, v := range o.hashData() {
				if !start {
					res += ", "
				}
				start = false
				res += fmt.Sprintf("%v: %v", k, v)
			}
			res += "}"
			return Wrap(res)
		}),
		MSlot("__aget__", func(o, k *Object) *Object {
			res := o.hashData()[keyData(k)]
			if res == nil {
				res = False
			}
			return res
		}),
		MSlot("__aset__", func(o, k, v *Object) *Object {
			o.hashData()[keyData(k)] = v
			return Nil
		}),
		PropSlot("size", func(o *Object) *Object {
			return Wrap(len(o.hashData()))
		}, Nil),
	})
}

