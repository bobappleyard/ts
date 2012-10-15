package parse

import (
	"fmt"
)

// A Pratt operator precedence parser.
type Parser struct {
	pp map[key] Prefix
	ip map[key] Infix
	el Prefix
}

type key struct {
	k int
	s string
}

// Handles the token at the start of the expression.
type Prefix interface {
	Prefix(p *Parser, l *Lexer, t Token) *Node
}

// Handles tokens after the first in a particular call to Parse().
type Infix interface {
	Precedence(t Token) int
	Infix(p *Parser, l *Lexer, left *Node, t Token) *Node
}

// Handles both prefix and infix.
type Both interface {
	Prefix
	Infix
}

// Simple wrapper round Prefix
type ParserFunc func(p *Parser, l *Lexer, t Token) *Node

func (f ParserFunc) Prefix(p *Parser, l *Lexer, t Token) *Node {
	return f(p, l, t)
}

// Node in the syntax tree.
type Node struct {
	Kind int
	Token Token
	Data interface{}
	Parent *Node
	Child []*Node
}

// Add a child node, setting said node's Parent field appropriately.
func (n *Node) Add(cs... *Node) *Node {
	for _, c := range cs {
		if c != nil {
			c.Parent = n
		}
	}
	n.Child = append(n.Child, cs...)
	return n
}

// Call a function on the node. Recursively scan the node's children if that
// function returns true.
func (n *Node) Scan(f func(*Node) bool) {
	if n != nil && f(n) {
		for _, x := range n.Child {
			x.Scan(f)
		}
	}
}

func (n *Node) String() string {
	if n == nil {
		return "<nil>"
	}
	res := fmt.Sprintf(
		"(%d [%d %s] ",
		n.Kind, n.Token.Kind, n.Token.Text,
	)
	for i, x := range n.Child {
		if i > 0 {
			res += ", "
		}
		res += x.String()
	}
	return res + ")"
}

// Parse an expression at a given precedence level, returning a syntax tree.
func (p *Parser) Parse(l *Lexer, prec int) *Node {
	return p.ParseWith(l, prec, l.Next())
}

// Parse an expression at a given precedence level, given an initial token.
func (p *Parser) ParseWith(l *Lexer, prec int, t Token) *Node {
	pp := p.prefix(t)
	left := pp.Prefix(p, l, t)
	for {
		t = l.Lookahead()
		ip := p.infix(t)
		if ip == nil || ip.Precedence(t) <= prec {
			break
		}
		l.Next()
		left = ip.Infix(p, l, left, t)
	}
	return left
}

func (p *Parser) prefix(t Token) Prefix {
	pp := p.pp[key{t.Kind, t.Text}]
	if pp == nil {
		pp = p.pp[key{t.Kind, ""}]
	}
	if pp == nil {
		if p.el == nil {
			panic(Unexpected(t))
		}
		pp = p.el
	}
	return pp
}

func (p *Parser) infix(t Token) Infix {
	ip := p.ip[key{t.Kind, t.Text}]
	if ip == nil {
		ip = p.ip[key{t.Kind, ""}]
	}
	return ip
}

// Register a prefix handler.
func (p *Parser) RegPrefix(k int, s string, pp Prefix) {
	if p.pp == nil {
		p.pp = map[key] Prefix{}
	}
	p.pp[key{k, s}] = pp
}

// Register an infix handler.
func (p *Parser) RegInfix(k int, s string, ip Infix) {
	if p.ip == nil {
		p.ip = map[key] Infix{}
	}
	p.ip[key{k, s}] = ip
}

// Register a handler for both infix and prefix.
func (p *Parser) RegBoth(k int, s string, bp Both) {
	p.RegPrefix(k, s, bp)
	p.RegInfix(k, s, bp)
}

// Register a handler for a failure to match
func (p *Parser) RegElse(ep Prefix) {
	p.el = ep
}



func TokenError(format string, t Token, args... interface{}) error {
	args = append([]interface{}{t.File, t.Line}, args...)
	return fmt.Errorf("%s(%d): " + format, args...)
}

func Expected(s string, t Token) error {
	return TokenError("expected %s, got %s", t, s, t.Text)
}

func Unexpected(t Token) error {
	return TokenError("unexpected %s", t, t.Text)
}

func UnexpectedEof() error {
	return fmt.Errorf("unexpected EOF")
}

// Panic with an Expected error unless the token's string matches.
func Expect(s string, t Token) {
	if t.Text != s {
		panic(Expected(s, t))
	}
}


