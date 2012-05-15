package ts

import (
	"io"
	"fmt"
	"strings"
	. "github.com/bobappleyard/ts/parse"
	. "github.com/bobappleyard/ts/bytecode"
)

/*******************************************************************************

	Compiler interface

*******************************************************************************/

 // Compile a TranScript source file. The results of the compilation can then be
// saved to a file or executed.
func (u *Unit) Compile(in io.Reader, f string) {
	l := NewScanner(in, f)
	for u.CompileStmt(l) {}
}

// Compile a single toplevel statement.
func (u *Unit) CompileStmt(l *Lexer) bool {
	if len(u.b) == 0 {
		u.b = [][]uint16{nil}
	}
	if len(u.v) == 0 {
		u.v = []*Object{Nil, True, False}
	}
	if l.Lookahead().Kind == eof {
		return false
	}
	u.compileTopLevel(parseToplevel(l))
	return true
}

// Shorthand wrapper around Compile().
func (u *Unit) CompileStr(s string) {
	u.Compile(strings.NewReader(s), "unknown")
}

// Create a lexer for a source file.
func NewScanner(in io.Reader, f string) *Lexer {
	return new(Lexer).Init(in, f, start)
}

/*******************************************************************************

	Implementation

*******************************************************************************/

const (
	invalidNode = iota
	valNode
	// variables and operators
	defNode
	varNode
	mutNode
	// control features
	ifNode
	logNode
	retNode
	// arrays
	alookNode
	// functions
	fnNode
	callNode
	// classes and objects
	classNode
	propNode
	lookNode
	thisNode
	superNode
	nodeCount
)

var nodeNames = []string {
	"?",
	"val",
	"def",
	"var",
	"mut",
	"if",
	"log",
	"ret",
	"alook",
	"fn",
	"call",
	"class",
	"prop",
	"look",
	"this",
	"super",
}

type compilerCtx struct {
	bound, free, boxed, class []string
	block *[]uint16
	offset int
}

type compilerSym int

func (e compilerCtx) write(op uint16, args... int) compilerSym {
	s := compilerSym(len(*e.block))
	*e.block = append(*e.block, op)
	for _, x := range args {
		*e.block = append(*e.block, uint16(x))
	}
	return s
}

func (s compilerSym) place(e compilerCtx) {
	b := *e.block
	b[s+1] = uint16(len(b) + e.offset)
}

func (s compilerSym) pos(e compilerCtx) int {
	return int(s) + e.offset
}

func (e compilerCtx) isBoxed(n string) bool {
	return lookup(n, e.boxed) != -1 || 
	      (lookup(n, e.bound) == -1 && lookup(n, e.free) == -1)
}

func (e compilerCtx) static(n string) int {
	return lookup(n, e.class)
}

func (u *Unit) compileTopLevel(n *Node) {
	if n == nil {
		return
	}
	e := compilerCtx{nil, nil, nil, nil, new([]uint16), len(u.b[0])}
	u.compileNode(n, e)
	u.b[0] = append(u.b[0], *e.block...)
}

func (u *Unit) writeSrc(n *Node, e compilerCtx) {
	t := n.Token
	if t.Line != 0 {
		u.file = t.File
		u.line = t.Line
		e.write(SOURCE, u.getVal(Wrap(t.File)), u.line)
	}
}

func (u *Unit) compileSrc(n *Node, e compilerCtx) {
	t := n.Token
	if t.Line != u.line {
		u.writeSrc(n, e)
	}
}

func (u *Unit) compileNode(n *Node, e compilerCtx) {
	u.compileSrc(n, e)
	switch n.Kind {
	case defNode:
		u.compileDef(n, e)
	case varNode:
		u.compileVar(n, e)
	case valNode:
		u.compileVal(n, e)
	case mutNode:
		u.compileMutation(n, e)
	case ifNode:
		u.compileIf(n, e)
	case logNode:
		u.compileLog(n, e)
	case retNode:
		u.compileRet(n, e)
	case alookNode:
		u.compileMethod(n, "__aget__", e)
	case fnNode:
		u.compileFn(n, e)
	case callNode:
		u.compileCall(n, nil, e)
	case classNode:
		u.compileClass(n, e)
	case lookNode:
		u.compileLook(n, e)
	case thisNode:
		if !isInside(n, classNode) {
			panic(Unexpected(n.Token))
		}
		e.write(THIS)
	case superNode:
		panic(Unexpected(n.Token))
	default:
		panic(fmt.Errorf("unknown AST kind: %d", n.Kind))
	}
}

func printNode(n *Node) {
	if n == nil {
		fmt.Print("<nil>")
		return
	}
	fmt.Print("(")
	if n.Kind > invalidNode && n.Kind < nodeCount {
		fmt.Print(nodeNames[n.Kind])
	} else {
		fmt.Print(nodeNames[invalidNode])
	}
	if n.Token.Text != "" {
		fmt.Printf(" %#v", n.Token.Text)
	}
	for _, x := range n.Child {
		fmt.Print(" ")
		printNode(x)
	}
	fmt.Print(")")
}

func debug(xs... interface{}) {
	for _, x := range xs {
		if n, is := x.(*Node); is {
			printNode(n)
			continue
		}
		fmt.Printf("%#v ", x)
	}
	fmt.Println()
}

func lookup(n string, s []string) int {
	for i, x := range s {
		if x == n {
			return i
		}
	}
	return -1
}

func merge(a, b []string) []string {
	inc := map[string] bool {}
	for _, x := range a {
		inc[x] = true
	}
	for _, x := range b {
		inc[x] = true
	}
	res := []string{}
	for x, _ := range inc {
		res = append(res, x)
	}
	return res
}

func checkUniq(vs []string) {
	inc := map[string] bool {}
	for _, x := range vs {
		if inc[x] {
			panic(fmt.Errorf("%s defined twice in the same context", x))
		}
		inc[x] = true
	}
}

func nodeStrs(ns []*Node) []string {
	res := make([]string, len(ns))
	for i, x := range ns {
		res[i] = x.Token.Text
	}
	return res
}

func closedVars(body []*Node, e compilerCtx) []*Node {
	m := map[string] *Node{}
	for _, n := range body {
		if n == nil {
			continue
		}
		n.Scan(func(n *Node) bool {
			switch n.Kind {
			case defNode:
				for _, x := range n.Child {
					vs := closedVars([]*Node{x.Child[1]}, e)
					for _, v := range vs {
						m[v.Token.Text] = v
					}
				}
				return false
			case varNode:
				s := n.Token.Text
				if lookup(s, e.bound) != -1 || lookup(s, e.free) != -1 {
					m[n.Token.Text] = n
				}
			}
			return true
		})
	}
	res := []*Node{}
	for _, x := range m {
		res = append(res, x)
	}
	return res
}

func isBoxed(n *Node, b []string) bool {
	return n.Kind == mutNode &&
	       n.Child[0].Kind == varNode &&
	       lookup(n.Child[0].Token.Text, b) != -1
}

func boxedVars(n []*Node, b []string, e compilerCtx) []string {
	in := map[string] bool{}
	for _, x := range e.boxed {
		in[x] = true
	}
	for _, x := range b {
		in[x] = false
	}
	for _, x := range n {
		x.Scan(func(n *Node) bool {
			if isBoxed(n, b) {
				in[n.Child[0].Token.Text] = true
			}
			return true;
		})
	}
	res := []string{}
	for x, p := range in {
		if p {
	 		res = append(res, x)
	 	}
	}
	return res
}

func isTail(n *Node) bool {
	if n.Parent == nil {
		return false
	}
	if n.Parent.Kind == retNode {
		return true
	}
	return false
}

func containingNode(n *Node, k int) *Node {
	for cur := n; cur != nil; cur = cur.Parent {
		if cur.Kind == k {
			return cur
		}
	}
	return nil
}

func isInside(n *Node, k int) bool {
	return containingNode(n, k) != nil
}

func (u *Unit) compileArgs(s []*Node, e compilerCtx) {
	for _, x := range s {
		u.compileNode(x, e)
		e.write(PUSH)
	}
}

func (u *Unit) getGlobal(n string) int {
	if i := lookup(n, u.gn); i != -1 {
		return i
	}
	i := len(u.gn)
	u.g = append(u.g, nil)
	u.gn = append(u.gn, n)
	return i
}

func (u *Unit) getAccessor(n string) int {
	if n != "" {
		if i := lookup(n, u.an); i != -1 {
			return i
		}
	}
	i := len(u.an)
	u.a = append(u.a, nil)
	u.an = append(u.an, n)
	return i
}

func (u *Unit) getVal(v *Object) int {
	for i, x := range u.v {
		if x.c.m == nil {
			continue
		}
		if x.callMethod(x.c.m[_Object_eq], []*Object{v}) != False {
			return i
		}
	}
	p := len(u.v)
	u.v = append(u.v, v)
	return p
}

func (u *Unit) compileDef(n *Node, e compilerCtx) {
	for _, x := range n.Child {
		if x.Kind == propNode {
			panic(fmt.Errorf("property definition outside a class"))
		}
		t := x.Child[0].Token
		if x.Child[1] == nil {
			e.write(VALUE, 0)
		} else {
			u.compileNode(x.Child[1], e)
		}
		e.write(PUSH)
		if n.Parent == nil {
			e.write(GLOBAL, u.getGlobal(t.Text))
		} else {
			e.write(BOUND, lookup(t.Text, e.bound))
		}
		e.write(DEFINE)
		e.write(UPDATE)
	}
}

func (u *Unit) compileLookup(n *Node, e compilerCtx) {
	var p uint16
	var l int
	s := n.Token.Text
	if i := lookup(s, e.bound); i != -1 {
		p = BOUND
		l = i
	} else if i := lookup(s, e.free); i != -1 {
		p = FREE
		l = i
	} else {
		p = GLOBAL
		l = u.getGlobal(s)
	}
	e.write(p, l)
}

func (u *Unit) compileVar(n *Node, e compilerCtx) {
	u.compileLookup(n, e)
	if e.isBoxed(n.Token.Text) {
		e.write(UNBOX)
	}
}

func (u *Unit) compileMutation(n *Node, e compilerCtx) {
	switch n.Child[0].Kind {
	case alookNode:
		ns := make([]*Node, len(n.Child[0].Child) + 1)
		copy(ns, n.Child[0].Child)
		ns[len(ns)-1] = n.Child[1]
		m := &Node { Child: ns }
		u.compileMethod(m, "__aset__", e)
	case lookNode:
		u.compileNode(n.Child[1], e)
		e.write(PUSH)
		u.compileNode(n.Child[0].Child[0], e)
		nm := n.Child[0].Token.Text
		e.write(SET, u.getAccessor(nm), e.static(nm))
	case varNode:
		u.compileNode(n.Child[1], e)
		e.write(PUSH)
		u.compileLookup(n.Child[0], e)
		e.write(UPDATE)
	default:
		file := n.Token.File
		line := n.Token.Line
		panic(fmt.Errorf("%s(%d): invalid location for writing", file, line))
	}
}

func (u *Unit) compileLog(n *Node, e compilerCtx) {
	op := n.Token.Text
	u.compileNode(n.Child[0], e)
	bpos := e.write(BRANCH, 0)
	if op == "&&" {
		u.compileNode(n.Child[1], e)
	}
	jpos := e.write(JUMP, 0)
	bpos.place(e)
	if op == "||" {
		u.compileNode(n.Child[1], e)		
	}
	jpos.place(e)
}

func (u *Unit) compileIf(n *Node, e compilerCtx) {
	// if <expr>
	u.compileNode(n.Child[0], e)
	bpos := e.write(BRANCH, 0)
	// then <block>
	u.compileBlock(n.Child[1].Child, e)
	e.write(VALUE, 0)
	jpos := e.write(JUMP, 0)
	// else <block>
	bpos.place(e)
	u.compileBlock(n.Child[2].Child, e)
	e.write(VALUE, 0)
	jpos.place(e)
}

func (u *Unit) compileVal(n *Node, e compilerCtx) {
	e.write(VALUE, u.getVal(n.Data.(*Object)))
}

func (u *Unit) compileFn(n *Node, e compilerCtx) {
	args := n.Child[0]
	body := n.Child[1:]
	// prepare the environment
	bound := nodeStrs(args.Child)
	checkUniq(bound)
	freeNodes := closedVars(body, e)
	free := nodeStrs(freeNodes)
	boxed := boxedVars(body, bound, e)
	f := compilerCtx{bound, free, boxed, e.class, new([]uint16), 0}
	// emit function prologue
	if args.Data == true {
		f.write(PROLOG_REST, len(bound)-args.Kind-1, len(bound)-1)
	} else if args.Kind != 0 {
		f.write(PROLOG_OPT, len(bound)-args.Kind, len(bound))
	} else {
		f.write(PROLOG, len(bound))
	}
	for i, x := range bound {
		if f.isBoxed(x) {
			f.write(BOX, i)
		}
	}
	if len(body) != 0 {
		u.writeSrc(body[0], f)
	}
	// compile the function body
	u.compileBlock(body, f)
	f.write(VALUE, 0)
	f.write(RETURN)
	// store the block
	ix := len(u.b)
	u.b = append(u.b, *f.block)
	// emit closure code
	for _, x := range freeNodes {
		u.compileLookup(x, e)
		e.write(PUSH)
	}
	e.write(CLOSE, ix, len(free))
}

func (u *Unit) compileBlock(n []*Node, e compilerCtx) {
	bound := []string{}
	l := len(e.bound)
	outer := make([]string, l)
	copy(outer, e.bound)
	for _, x := range n {
		if x.Kind == defNode {
			for _, y := range x.Child {
				name := y.Child[0].Token.Text
				p := lookup(name, outer)
				if p != -1 {
					outer[p] = ""
				}
				bound = append(bound, name)
			}
		}
	}
	checkUniq(bound)
	for i, x := range bound {
		u.compileVal(&Node{Data: Wrap(x)}, e)
		e.write(PUSH)
		e.write(UNDEFINE, l+i)
	}
	e.bound = append(outer, bound...)
	e.boxed = append(e.boxed, bound...)
	for _, x := range n {
		u.compileNode(x, e)
	}
	e.write(RETRACT, len(bound))
}

func (u *Unit) compileCall(n *Node, loc func(), e compilerCtx) {
	t := isTail(n)
	var fpos compilerSym
	if !t {
		fpos = e.write(FRAME, 0)
	}
	as := n.Child[1:]
	u.compileArgs(as, e)
	if loc != nil {
		loc()
	} else {
		m := n.Child[0]
		switch m.Kind {
		case lookNode:
			if m.Child[0].Kind == superNode {
				t := m.Token
				e.write(SUPER, e.static(t.Text))
			} else {
				u.compileNode(m.Child[0], e)
				t := m.Token
				e.write(LTHIS)
				e.write(GETM, u.getAccessor(t.Text), e.static(t.Text))
			}
		default:
			u.compileNode(m, e)
		}
	}
	if t {
		e.write(SHUFFLE, len(as))
	}
	e.write(CALL, len(as))
	if !t {
		fpos.place(e)
	}
}

func (u *Unit) compileMethod(n *Node, m string, e compilerCtx) {
	u.compileCall(n, func() {
		u.compileNode(n.Child[0], e)
		e.write(LTHIS)
		e.write(GETM, u.getAccessor(m), e.static(m))
	}, e)
}

func (u *Unit) compileRet(n *Node, e compilerCtx) {
	if !isInside(n, fnNode) {
		panic(Unexpected(n.Token))
	}
	u.compileNode(n.Child[0], e)
	e.write(RETURN)
}

func (u *Unit) compileClass(n *Node, e compilerCtx) {
	ns, es, spec := u.compileSpec(n, e)
	e.class = ns
	u.compileVal(&Node{Data: new(skelObj).init(es)}, e)
	e.write(PUSH)
	if n.Child[2] == nil {
		e.write(GLOBAL, u.getGlobal("Object"))
		e.write(UNBOX)
	} else {
		u.compileNode(n.Child[2], e)
	}
	if n.Child[1] == nil {
		e.write(EXTENDA, u.getAccessor(""))
	} else {
		e.write(EXTEND)
	}
	i, l := 1, 0
	for _, x := range spec {
		if es[i].Flags.Kind() == Property {
			if x.Child[0] == nil {
				e.write(VALUE, 0)
			} else {
				u.compileNode(x.Child[0], e)
				(*e.block)[len(*e.block)-3] = CLOSEM
			}
			e.write(PUSH)
			x = x.Child[1]
			l++
		}
		if x == nil {
			e.write(VALUE, 0)
		} else {
			u.compileNode(x, e)
			if k := es[i].Flags.Kind(); k == Method || k == Property {
				(*e.block)[len(*e.block)-3] = CLOSEM
			}
		}
		e.write(PUSH)
		i++
		l++
	}
	e.write(FINISH, l)
	if n.Child[1] != nil {
		e.write(PUSH)
		g := u.getGlobal(n.Child[1].Token.Text)
		e.write(GLOBAL, g)
		e.write(DEFINE)
		e.write(UPDATE)
	}
}

func (u *Unit) compileSpec(n *Node, e compilerCtx) ([]string, []Slot, []*Node) {
	name := ""
	if n.Child[0] != nil {
		name = n.Child[0].Token.Text
	}
	names := []string{}
	spec := []*Node{}
	es := []Slot{{Name: name}}
	for _, d := range n.Child[3:] {
		for _, x := range d.Child {
			names = append(names, x.Child[0].Token.Text)
			k := Field
			switch x.Kind {
			case fnNode:
				k = Method
			case propNode:
				k = Property
			}
			t := x.Child[0].Token
			es = append(es, Slot {
				Flags: Flags(k, x.Data.(SlotVis)),
				access: uint16(u.getAccessor(t.Text)),
				next: uint16(e.static(t.Text)),
			})
			spec = append(spec, x.Child[1])
		}
	}
	checkUniq(names)
	for i, x := range e.class {
		es = append(es, Slot{
			Flags: Flags(Marker, Private),
			access: uint16(u.getAccessor(x)),
			next: uint16(i),
		})
		names = append(names, x)
	}
	return names, es, spec
}

func (u *Unit) compileLook(n *Node, e compilerCtx) {
	if n.Child[0].Kind == superNode {
		panic(Unexpected(n.Token))
	}
	u.compileNode(n.Child[0], e)
	t := n.Token
	e.write(GET, u.getAccessor(t.Text), e.static(t.Text))
}


