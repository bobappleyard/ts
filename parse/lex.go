package parse

import (
	"io"
	"unicode/utf8"
)

// Out of band representation for EOF as a character.
const Eof rune = -1

// A lexical analyser where each state is a function.
type Lexer struct {
	src *Source
	start State
	t []Token
}

type Source struct {
	src io.Reader
	lex *Lexer
	eof bool
	buf []byte
	file string
	s, p, l, line, nline int // start, position, last position
}

// A state in the lexical analysis. The programmer should provide these to the
// Lexer to allow it to process text.
type State func(l *Source) State

// A token represents a section of a source text.
type Token struct {
	Kind, Line int
	File, Text string
}

// Initialise a Lexer with a source and a start state before beginning
// processing.
func (l *Lexer) Init(src io.Reader, nm string, start State) *Lexer {
	l.src = new(Source).init(src, l, nm)
	l.start = start
	return l
}

// Parse a token from the source.
//
// Matching process:
//
//	1. Begin with the start state passed into Init().
//	2. Call the current state, passing in the Lexer.
//	3. This provides the next state.
//	4. If the next state is nil, halt.
//	5. Otherwise go back to step 2 with the current state as the next state.
//
// States should issue calls to Read(), Peek(), Clear() and Save() as required 
// to move through the source and match tokens. These methods should not be 
// called outside the dynamic extent of Next().
//
// A given run through the matching process may produce no tokens (in which case 
// the matching process is repeated), one token (in which case it is returned) 
// or more tokens (in which case the first token match is returned, and the 
// matching process is foregone for retrieving those extra tokens).
//
func (l *Lexer) Next() Token {
	t := l.Lookahead()
	l.t = l.t[1:]
	return t
}

// Parse a token from the source as Next(), but remember that token so that it 
// is returned from the next call to Parse() or Lookahead().
func (l *Lexer) Lookahead() Token {
	for len(l.t) == 0 {
		l.src.Clear()
		for s := l.start; s != nil; s = s(l.src) {}
	}
	return l.t[0]
}

func (s *Source) init(src io.Reader, l *Lexer, nm string) *Source {
	s.src = src
	s.lex = l
	s.file = nm
	s.line = 1
	s.nline = 1
	return s
}

// Read a character from the source. Will return Eof when at the end of the 
// source and panic for any other errors.
func (s *Source) Read() rune {
	if s.p >= len(s.buf) {
		if s.eof {
			return Eof
		}
		s.expand()
		if s.eof {
			return Eof
		}
	}
	s.l = s.p
	r, n := utf8.DecodeRune(s.buf[s.p:])
	s.p += n
	if r == '\n' {
		s.nline++
	}
	return r
}

// Read a character from the source as in Read(), but remember that character so 
// that it is returned from the next call to Read() or Peek().
func (s *Source) Peek() rune {
	r := s.Read()
	s.p = s.l
	if r == '\n' {
		s.nline--
	}
	return r
}

// Ignore the portion of text currently under consideration.
func (s *Source) Clear() {
	s.s = s.p
	s.line = s.nline
}

// Save the portion of text currently under consideration with a provided Kind.
func (s *Source) Save(k int) {
	t := Token{k, s.line, s.file, string(s.buf[s.s:s.p])}
	s.lex.t = append(s.lex.t, t)
}

func (s *Source) Pos() (int, int) {
	return s.p, s.nline
}

func (s *Source) SetPos(i, n int) {
	s.p, s.nline = i, n
}

func (s *Source) expand() {
	nextk := make([]byte, 1024)
	n, e := s.src.Read(nextk)
	if e == io.EOF {
		s.eof = true
	} else if e != nil {
		panic(e)
	}
	s.buf = append(s.buf, nextk[:n]...)
}

