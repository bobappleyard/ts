package ts

import (
	"unicode"
	"fmt"
	"strconv"
	. "github.com/bobappleyard/ts/parse"
)


/*******************************************************************************

	Lexer

*******************************************************************************/

const (
	invalid = iota
	eof
	str
	inum
	fnum
	id
	op
	literal
)

func start(l *Source) State {
	r := l.Read()
	switch r {
	case Eof:
		l.Save(eof)
		return nil
	case ' ', '\t', '\n':
		return nil
	case '[', ']', '(', ')', '{', '}', ';', ':', ',', '.':
		l.Save(literal)
		return nil
	case '/':
		return inCmt
	case '"':
		return inStr
	case '_':
		return inId
	case '!', '$', '%', '^', '&', '*', '-', '=', 
	     '+', '~', '?', '@', '<', '>', '|':
		return inOp
	}
	if unicode.IsLetter(r) {
		return inId
	}
	if unicode.IsDigit(r) {
		return inNum
	}
	panic(fmt.Errorf("illegal character: %c [%d]", r, r))
}

func inCmt(l *Source) State {
	switch l.Peek() {
	case '/':
		l.Read()
		return inCmtl
	case '*':
		l.Read()
		return inCmtb
	}
	return inOp
}

func inCmtl(l *Source) State {
	switch l.Read() {
	case Eof, '\n':
		return nil
	}
	return inCmtl
}

func inCmtb(l *Source) State {
	switch l.Read() {
	case Eof:
		panic(UnexpectedEof())
	case '*':
		return inCmtbe
	}
	return inCmtb
}

func inCmtbe(l *Source) State {
	switch l.Read() {
	case Eof:
		panic(UnexpectedEof())
	case '/':
		return nil
	}
	return inCmtb
}

func inStr(l *Source) State {
	switch l.Read() {
	case Eof:
		panic(UnexpectedEof())
	case '"':
		l.Save(str)
		return nil
	case '\\':
		l.Read()
	}
	return inStr
}

func inId(l *Source) State {
	r := l.Peek()
	if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
		l.Read()
		return inId
	}
	l.Save(id)
	return nil
}

func inOp(l *Source) State {
	switch l.Peek() {
	case '!', '$', '%', '^', '&', '*', '-', '=', 
	     '+', '~', '?', '@', '<', '>', '|':
		l.Read()
		return inOp
	}
	l.Save(op)
	return nil
}

func inNum(l *Source) State {
	c := l.Peek()
	if c == '.' {
		p, line := l.Pos()
		l.Read()
		if unicode.IsDigit(l.Peek()) {
			return inFnum
		}
		l.SetPos(p, line)
	} else if unicode.IsDigit(c) {
		l.Read()
		return inNum
	}
	l.Save(inum)
	return nil
}

func inFnum(l *Source) State {
	c := l.Peek()
	if unicode.IsDigit(c) {
		l.Read()
		return inFnum
	}
	l.Save(fnum)
	return nil
}

/*******************************************************************************

	Parser

*******************************************************************************/

func tNode(k int, t string) *Node {
	return &Node{Kind: k, Token: Token{Text: t}}
}

func kNode(k int) *Node {
	return &Node{Kind: k}
}

func vNode(x interface{}) *Node {
	return &Node{Kind: valNode, Data: Wrap(x)}
}

var keywords = []string {
	"if", "then", "else", "elif",
	"true", "false", "nil",
	"def",
	"fn", "return", "end", 
	"class", "this", "super",
	"private", "public",
	"package", "export", "import",
}

func checkKeyword(t Token) {
	if lookup(t.Text, keywords) != -1 {
		panic(Unexpected(t))
	}
}

func parseList(l *Lexer, n *Node, e string, m func() *Node) {
	t := l.Lookahead()
	if t.Text == e {
		return
	}
	for {
		c := m()
		n.Add(c)
		t = l.Lookahead()
		if listEnd(t, e) {
			return
		}
		l.Next()
	}
}

func listEnd(t Token, e string) bool {
	s := t.Text
	if s == "," {
		return false
	}
	if s == e {
		return true
	}
	panic(Expected(", or " + e, t))
}

func parseName(l *Lexer) *Node {
	t := l.Next()
	if t.Kind != id {
		panic(Expected("identifier", t))
	}
	return &Node{Token: t}
}

func parseBlock(l *Lexer, n *Node) {
	for {
		if l.Lookahead().Text == "end" {
			l.Next()
			return
		}
		n.Add(stmt.Parse(l, 0))
		Expect(";", l.Next())
	}
}

// eof
func parseEof(p *Parser, l *Lexer, t Token) *Node {
	e := ErrorClass.New(Wrap("unexpected eof"))
	ErrorClass.Set(e, 1, Wrap(t.File))
	ErrorClass.Set(e, 2, Wrap(t.Line))
	panic(e)
}

// operators
type prefixOp struct {
	p int
	m string
}

func (q prefixOp) Prefix(p *Parser, l *Lexer, t Token) *Node {
	n := &Node{Kind: callNode, Token: t}
	return n.Add(tNode(lookNode, q.m).Add(p.Parse(l, q.p)))
}

type leftOp struct {
	p int
	m string
}

func (q leftOp) Precedence(t Token) int {
	return q.p
}

func (q leftOp) Infix(p *Parser, l *Lexer, left *Node, t Token) *Node {
	right := p.Parse(l, q.p)
	n := &Node{Kind: callNode, Token: t}
	return n.Add(tNode(lookNode, q.m).Add(left), right)
}

type rightOp struct {
	p int
	m string
}

func (q rightOp) Precedence(t Token) int {
	return q.p
}

func (q rightOp) Infix(p *Parser, l *Lexer, left *Node, t Token) *Node {
	right := p.Parse(l, q.p-1)
	n := &Node{Kind: callNode, Token: t}
	return n.Add(tNode(lookNode, q.m).Add(left), right)
}

type logOp struct {
	p int
}

func (q logOp) Precedence(t Token) int {
	return q.p
}

func (q logOp) Infix(p *Parser, l *Lexer, left *Node, t Token) *Node {
	right := p.Parse(l, q.p-1)
	return tNode(logNode, t.Text).Add(left, right)
}

// grouping
func parseGroup(p *Parser, l *Lexer, t Token) *Node {
	n := p.Parse(l, 0)
	Expect(")", l.Next())
	return n
}

// value parsers
func parseStr(p *Parser, l *Lexer, t Token) *Node {
	s, err := strconv.Unquote(t.Text)
	if err != nil {
		panic(err)
	}
	return &Node{Kind: valNode, Token: t, Data: Wrap(s)}
}

func parseInum(p *Parser, l *Lexer, t Token) *Node {
	i, err := strconv.Atoi(t.Text)
	if err != nil {
		panic(err)
	}
	return &Node{Kind: valNode, Token: t, Data: Wrap(i)}
}

func parseFnum(p *Parser, l *Lexer, t Token) *Node {
	f, err := strconv.ParseFloat(t.Text, 64)
	if err != nil {
		panic(err)
	}
	return &Node{Kind: valNode, Token: t, Data: Wrap(f)}
}

func parseBuiltin(p *Parser, l *Lexer, t Token) *Node {
	var v interface{}
	switch t.Text {
	case "true":
		v = True
	case "false":
		v = False
	case "nil":
		v = Nil
	case "this":
		return &Node{Kind: thisNode, Token: t}
	case "super":
		return  &Node{Kind: superNode, Token: t}
	}
	return &Node{Kind: valNode, Token: t, Data: v}
}

// variables
func parseId(p *Parser, l *Lexer, t Token) *Node {
	checkKeyword(t)
	return &Node{Kind: varNode, Token: t}
}

func parseDef(p *Parser, l *Lexer, t Token) *Node {
	n := parseDefv(l, Public)
	n.Token = t
	return n
}

func parseDefv(l *Lexer, v SlotVis) *Node {
	n := &Node{Kind: defNode}
	loop: for {
		c := &Node{Kind: varNode, Data: v}
		nm := parseName(l)
		checkKeyword(nm.Token)
		c.Add(nm)
		n.Add(c)
		switch l.Lookahead().Text {
		case "=":
			l.Next()
			c.Add(expr.Parse(l, 0))
		case "(":
			l.Next()
			c.Kind = fnNode
			c.Add(parseFn(l))
		case "get", "set":
			c.Kind = propNode
			c.Add(parseProp(l))
		case ",":
			l.Next()
			c.Add(nil)
			continue
		default:
			c.Add(nil)
			break loop
		}
		// have to check the seperators again if extra stuff is provided
		switch l.Lookahead().Text {
		case ",":
			l.Next()
		default:
			break loop
		}
	}
	return n
}

func parseProp(l *Lexer) *Node {
	var g, s *Node
	loop: for {
		t := l.Lookahead()
		switch t.Text {
		case "get":
			if g != nil {
				panic(Unexpected(t))
			}
			l.Next()
			Expect("(", l.Next())
			g = parseFn(l)
		case "set":
			if s != nil {
				panic(Unexpected(t))
			}
			l.Next()
			Expect("(", l.Next())
			s = parseFn(l)
		default:
			break loop
		}
	}
	return new(Node).Add(g, s)
}

// functions
type funcParser struct { p int }

func (q funcParser) Prefix(p *Parser, l *Lexer, t Token) *Node {
	// defining
	Expect("(", l.Next())
	return parseFn(l)
}

func (q funcParser) Precedence(t Token) int {
	return q.p
}

func (q funcParser) Infix(p *Parser, l *Lexer, left *Node, t Token) *Node {
	// calling
	n := &Node{Kind: callNode, Token: t}
	n.Add(left)
	parseList(l, n, ")", func() *Node {
		return p.Parse(l, 0)
	})
	Expect(")", l.Next())
	return n
}

func parseFn(l *Lexer) *Node {
	args := new(Node)
	fn := kNode(fnNode).Add(args)
	inOpt := false
	parseList(l, args, ")", func() *Node {
		if args.Data == true {
			panic(fmt.Errorf("bad function syntax"))
		}
		n := parseName(l)
		switch l.Lookahead().Text {
		case "*":
			l.Next()
			args.Data = true
		case "?":
			l.Next()
			args.Kind++
			inOpt = true
		default:
			if inOpt {
				panic(fmt.Errorf("bad function syntax"))
			}
		}
		return n
	})
	Expect(")", l.Next())
	t := l.Lookahead()
	if t.Text == "=" {
		l.Next()
		fn.Add((&Node{Kind: retNode, Token: t}).Add(expr.Parse(l, 0)))
	} else {
		parseBlock(l, fn)
	}
	return fn
}

// arrays
type arrParser struct { p int }

func (q arrParser) Prefix(p *Parser, l *Lexer, t Token) *Node {
	avar := tNode(varNode, "@tmp")
	nn := &Node{Kind: callNode, Token: t}
	def := kNode(defNode).Add(kNode(varNode).Add(
		avar,
		nn.Add(tNode(varNode, "Array"), vNode(0)),
	))
	add := &Node{Kind: callNode, Token: t}
	add.Add(tNode(lookNode, "add").Add(avar))
	parseList(l, add, "]", func() *Node {
		return p.Parse(l, 0)
	})
	Expect("]", l.Next())
	n := &Node{Kind: callNode, Token: t}
	return n.Add(kNode(fnNode).Add(new(Node),def,add,kNode(retNode).Add(avar)))
}

func (q arrParser) Precedence(t Token) int {
	return q.p
}

func (q arrParser) Infix(p *Parser, l *Lexer, left *Node, t Token) *Node {
	n := kNode(alookNode).Add(left)
	n.Token = t
	parseList(l, n, "]", func() *Node {
		return p.Parse(l, 0)
	})
	Expect("]", l.Next())
	return n
}

// hashes
func parseHash(p *Parser, l *Lexer, t Token) *Node {
	n := &Node{Token:t}
	parseList(l, n, "}", func() *Node {
		k := expr.Parse(l, 0)
		Expect(":", l.Next())
		v := expr.Parse(l, 0)
		return new(Node).Add(k, v)
	})
	Expect("}", l.Next())
	return transHash(n)
}

func transHash(n *Node) *Node {
	avar := tNode(varNode, "@tmp")
	fn := kNode(fnNode).Add(new(Node))
	fn.Add(kNode(defNode).Add(kNode(varNode).Add(
		avar,
		kNode(callNode).Add(tNode(varNode, "Hash")),
	)))
	for _, x := range n.Child {
		fn.Add(kNode(mutNode).Add(
			kNode(alookNode).Add(avar, x.Child[0]),
			x.Child[1],
		))
	}
	fn.Add(kNode(retNode).Add(avar))
	return (&Node{Kind: callNode, Token: n.Token}).Add(fn)
}

// objects
type objParser struct { p int }

func (q objParser) Precedence(t Token) int {
	return q.p
}

func (q objParser) Infix(p *Parser, l *Lexer, left *Node, t Token) *Node {
	n := parseName(l)
	checkKeyword(n.Token)
	n.Kind = lookNode
	return n.Add(left)
}

// classes
func parseClass(l *Lexer, nm, gl *Node) *Node {
	n := kNode(classNode)
	n.Add(nm, gl)
	Expect("(", l.Next())
	if l.Lookahead().Text == ")" {
		n.Add(nil)
	} else {
		n.Add(expr.Parse(l, 0))
	}
	Expect(")", l.Next())
	v := Public
	loop: for {
		t := l.Next()
		switch t.Text {
		case "private":
			v = Private
		case "public":
			v = Public
		case "def":
			n.Add(parseDefv(l, v))
			Expect(";", l.Next())
		case "end":
			break loop
		default:
			panic(Expected("def", t))
		}
	}
	return n
}

func parseAnonClass(p *Parser, l *Lexer, t Token) *Node {
	return parseClass(l, nil, nil)
}

func parseInnerClass(p *Parser, l *Lexer, t Token) *Node {
	nm := parseName(l)
	c := kNode(varNode).Add(nm, parseClass(l, nm, nil))
	return kNode(defNode).Add(c)
}

func parseReturn(p *Parser, l *Lexer, t Token) *Node {
	n := &Node{Kind: retNode, Token: t}
	if l.Lookahead().Text == ";" {
		n.Add(vNode(Nil))
	} else {
		n.Add(expr.Parse(l, 0))
	}
	return n
}

func parseStmt(p *Parser, l *Lexer, t Token) *Node {
	n := expr.ParseWith(l, 0, t)
	if l.Lookahead().Text == "=" {
		l.Next()
		loc := n
		n = &Node{Kind: mutNode, Token: t}
		n.Add(loc)
		n.Add(expr.Parse(l, 0))
	}
	return n
}
		
func parseIf(p *Parser, l *Lexer, t Token) *Node {
	n := &Node{Kind: ifNode, Token: t}
	cn := expr.Parse(l, 0)
	Expect("then", l.Next())
	tn := new(Node)
	en := new(Node)
	loop: for {
		t = l.Lookahead()
		switch t.Text {
		case "end":
			l.Next()
			break loop
		case "else":
			l.Next()
			parseBlock(l, en)
			break loop
		case "elif":
			l.Next()
			en.Add(parseIf(p, l, t))
			break loop
		}
		tn.Add(stmt.Parse(l, 0))
		Expect(";", l.Next())
	}
	n.Add(cn)
	n.Add(tn)
	n.Add(en)
	return n
}

// packages
func parsePkg(l *Lexer) *Node {
	n := new(Node)
	en := new(Node)
	n.Add(parseDotted(l), en)
	loop: for {
		switch l.Lookahead().Text {
		case "export":
			l.Next()
			parseList(l, en, ";", func() *Node {
				x := parseName(l)
				x.Kind = varNode
				return x
			})
		case "end":
			l.Next()
			break loop
		default:
			n.Add(stmt.Parse(l, 0))
		}
		Expect(";", l.Next())
	}
	return transPkg(n)
}

func transPkg(n *Node) *Node {
	// the package location
	nm, loc := transDotted(n.Child[0])
	pl := kNode(alookNode).Add(tNode(varNode, "packages"), vNode(loc))
	// export as instance of class
	ds := kNode(defNode)
	pc := kNode(classNode).Add(
		tNode(varNode, nm), nil, 
		tNode(varNode, "Package"), ds,
	)
	for _, x := range n.Child[1].Child {
		// use properties to thread access
		d := &Node{Kind: propNode, Data: Public}
		p := tNode(varNode, "@x")
		g := kNode(fnNode).Add(new(Node), kNode(retNode).Add(x))
		s := kNode(fnNode).Add(new(Node).Add(p), kNode(mutNode).Add(x, p))
		ds.Add(d.Add(x, new(Node).Add(g, s)))
	}
	// package internal runs inside a block
	fn := kNode(fnNode).Add(new(Node))
	fn.Add(n.Child[2:]...)
	fn.Add(kNode(retNode).Add(kNode(callNode).Add(pc)))
	return kNode(mutNode).Add(pl, kNode(callNode).Add(fn))
}

func parseDotted(l *Lexer) *Node {
	nm := parseName(l)
	for l.Lookahead().Text == "." {
		l.Next()
		nm.Add(parseName(l))
	}
	return nm
}

func transDotted(x *Node) (string, string) {
	nm, loc := x.Token.Text, x.Token.Text
	for _, y := range x.Child {
		nm = y.Token.Text
		loc += "." + nm
	}
	return nm, loc
}

func transImp(n, d *Node) {
	for _, x := range n.Child {
		impn, impl := transDotted(x)
		lo := &Node{Kind:alookNode, Token: x.Token}
		lo.Add(tNode(varNode, "packages"), vNode(impl))
		d.Add(kNode(defNode).Add(tNode(0, impn), lo))
	}
}

func parseImport(p *Parser, l *Lexer, t Token) *Node {
	m := new(Node)
	parseList(l, m, ";", func() *Node {
		return parseDotted(l)
	})
	n := &Node{Kind: defNode}
	transImp(m, n)
	return n
}

// tying it all together
var expr, stmt = new(Parser), new(Parser)

func parseToplevel(l *Lexer) *Node {
	t := l.Lookahead()
	if t.Kind == eof {
		return nil
	}
	var n *Node
	switch t.Text {
	case "class":
		l.Next()
		nm := parseName(l)
		n = parseClass(l, nm, nm)
	case "package":
		l.Next()
		n = parsePkg(l)
	default:
		n = stmt.Parse(l, 0)
	}
	Expect(";", l.Next())
	return n
}

func init() {
	expr.RegPrefix(eof, "", ParserFunc(parseEof))
	
	expr.RegPrefix(str, "", ParserFunc(parseStr))
	expr.RegPrefix(inum, "", ParserFunc(parseInum))
	expr.RegPrefix(fnum, "", ParserFunc(parseFnum))
	expr.RegPrefix(id, "", ParserFunc(parseId))

	bip := ParserFunc(parseBuiltin)
	expr.RegPrefix(id, "true", bip)
	expr.RegPrefix(id, "false", bip)
	expr.RegPrefix(id, "nil", bip)
	expr.RegPrefix(id, "this", bip)
	expr.RegPrefix(id, "super", bip)
	
	objs := objParser{120}
	expr.RegInfix(literal, ".", objs)
	expr.RegPrefix(id, "class", ParserFunc(parseAnonClass))

	expr.RegBoth(literal, "[", arrParser{110})
	expr.RegPrefix(literal, "{", ParserFunc(parseHash))

	funcs := funcParser{100}
	expr.RegPrefix(id, "fn", funcs)
	expr.RegInfix(literal, "(", funcs)

	expr.RegPrefix(literal, "(", ParserFunc(parseGroup))
	
	expr.RegPrefix(op, "!", prefixOp{60, "__inv__"})
	expr.RegPrefix(op, "-", prefixOp{60, "__neg__"})

	expr.RegInfix(op, "*", leftOp{60, "__mul__"})
	expr.RegInfix(op, "/", leftOp{60, "__div__"})

	expr.RegInfix(op, "+", leftOp{50, "__add__"})
	expr.RegInfix(op, "-", leftOp{50, "__sub__"})
	
	expr.RegInfix(op, "<", leftOp{40, "__lt__"})
	expr.RegInfix(op, "<=", leftOp{40, "__lte__"})
	expr.RegInfix(op, ">", leftOp{40, "__gt__"})
	expr.RegInfix(op, ">=", leftOp{40, "__gte__"})
	
	expr.RegInfix(op, "==", leftOp{30, "__eq__"})
	expr.RegInfix(op, "!=", leftOp{30, "__neq__"})
	
	expr.RegInfix(op, "||", logOp{20})
	expr.RegInfix(op, "&&", logOp{20})
	
	
	stmt.RegPrefix(id, "def", ParserFunc(parseDef))
	stmt.RegPrefix(id, "class", ParserFunc(parseInnerClass))
	stmt.RegPrefix(id, "if", ParserFunc(parseIf))
	stmt.RegPrefix(id, "return", ParserFunc(parseReturn))
//	stmt.RegPrefix(id, "loop", ParserFunc(parseLoop))
//	stmt.RegPrefix(id, "for", ParserFunc(parseFor))
	stmt.RegPrefix(id, "import", ParserFunc(parseImport))
	stmt.RegElse(ParserFunc(parseStmt))
}

