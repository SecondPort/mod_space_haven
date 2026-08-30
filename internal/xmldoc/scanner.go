// Package xmldoc provides an offset-preserving reader for XML documents.
//
// Space Haven save files must survive an edit byte for byte: the game is
// sensitive to attribute order, indentation and self-closing style, and a
// decode/encode round trip through encoding/xml would silently rewrite all
// three. So this package never re-serializes a document. It scans the raw
// bytes, reports where every tag and attribute value lives, and edits are
// expressed as byte-range replacements (see PatchSet) applied to the original
// buffer.
//
// The scanner is tag-oriented: it reports tags, comments, directives and CDATA
// sections, and skips character data, which carries no information in this
// format. It allocates nothing per token — names and values are materialized
// only when asked for.
package xmldoc

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Kind classifies a scanned token.
type Kind uint8

const (
	// KindStart is an opening tag: <name ...>
	KindStart Kind = iota
	// KindEnd is a closing tag: </name>
	KindEnd
	// KindSelfClose is a self-closing tag: <name .../>
	KindSelfClose
	// KindComment is a comment: <!-- ... -->
	KindComment
	// KindDirective is a processing instruction or declaration: <?...?>, <!...>
	KindDirective
	// KindCDATA is a character-data section: <![CDATA[...]]>
	KindCDATA
)

func (k Kind) String() string {
	switch k {
	case KindStart:
		return "start"
	case KindEnd:
		return "end"
	case KindSelfClose:
		return "self-close"
	case KindComment:
		return "comment"
	case KindDirective:
		return "directive"
	case KindCDATA:
		return "cdata"
	default:
		return "unknown"
	}
}

// Token is a single scanned tag. Start and End delimit the token in the source
// buffer, so src[tok.Start:tok.End] always reproduces it exactly.
type Token struct {
	Kind       Kind
	Start, End int

	src                []byte
	nameStart, nameEnd int
	attrStart, attrEnd int
}

// Name returns the element name. It allocates; prefer NameIs for comparisons.
func (t Token) Name() string {
	return string(t.src[t.nameStart:t.nameEnd])
}

// NameIs reports whether the element name equals want, without allocating.
func (t Token) NameIs(want string) bool {
	n := t.nameEnd - t.nameStart
	if n != len(want) {
		return false
	}
	for i := range want {
		if t.src[t.nameStart+i] != want[i] {
			return false
		}
	}
	return true
}

// Attr is a single attribute of a token. Start and End delimit the whole
// name="value" pair; ValueStart and ValueEnd delimit the value inside the
// quotes, which is the range an edit replaces.
type Attr struct {
	Name       string
	Value      string
	Start, End int
	ValueStart int
	ValueEnd   int
}

// Attr looks up one attribute by name.
func (t Token) Attr(name string) (Attr, bool) {
	var found Attr
	ok := false
	t.eachAttr(func(a Attr) bool {
		if a.Name == name {
			found, ok = a, true
			return false
		}
		return true
	})
	return found, ok
}

// Attrs returns every attribute in source order.
func (t Token) Attrs() []Attr {
	var out []Attr
	t.eachAttr(func(a Attr) bool {
		out = append(out, a)
		return true
	})
	return out
}

// IntAttr looks up an attribute and parses it as a base-10 integer.
func (t Token) IntAttr(name string) (int, bool) {
	a, ok := t.Attr(name)
	if !ok {
		return 0, false
	}
	v, err := strconv.Atoi(a.Value)
	if err != nil {
		return 0, false
	}
	return v, true
}

func (t Token) eachAttr(visit func(Attr) bool) {
	src, i, end := t.src, t.attrStart, t.attrEnd
	for i < end {
		for i < end && isSpace(src[i]) {
			i++
		}
		nameStart := i
		for i < end && isNameByte(src[i]) {
			i++
		}
		if i == nameStart {
			i++ // not an attribute name; step over it
			continue
		}
		nameEnd := i
		for i < end && isSpace(src[i]) {
			i++
		}
		if i >= end || src[i] != '=' {
			continue // valueless attribute
		}
		i++
		for i < end && isSpace(src[i]) {
			i++
		}
		if i >= end {
			return
		}
		quote := src[i]
		if quote != '"' && quote != '\'' {
			continue
		}
		i++
		valueStart := i
		for i < end && src[i] != quote {
			i++
		}
		valueEnd := i
		if i < end {
			i++ // closing quote
		}
		if !visit(Attr{
			Name:       string(src[nameStart:nameEnd]),
			Value:      string(src[valueStart:valueEnd]),
			Start:      nameStart,
			End:        i,
			ValueStart: valueStart,
			ValueEnd:   valueEnd,
		}) {
			return
		}
	}
}

// Scanner walks the tags of an XML buffer in document order.
type Scanner struct {
	src []byte
	pos int
}

// NewScanner returns a scanner positioned at the start of src.
func NewScanner(src []byte) *Scanner { return &Scanner{src: src} }

// NewScannerAt returns a scanner positioned at pos.
func NewScannerAt(src []byte, pos int) *Scanner { return &Scanner{src: src, pos: pos} }

// Pos reports the scanner's current offset.
func (s *Scanner) Pos() int { return s.pos }

// Next returns the next token, or false once the buffer is exhausted.
func (s *Scanner) Next() (Token, bool) {
	src := s.src
	for {
		lt := indexByteFrom(src, s.pos, '<')
		if lt < 0 {
			s.pos = len(src)
			return Token{}, false
		}
		tok, next, ok := scanTag(src, lt)
		if !ok {
			// Not a well-formed tag; treat the '<' as text and keep going.
			s.pos = lt + 1
			continue
		}
		s.pos = next
		return tok, true
	}
}

func scanTag(src []byte, start int) (Token, int, bool) {
	if start+1 >= len(src) {
		return Token{}, 0, false
	}

	switch {
	case hasPrefixAt(src, start, "<!--"):
		end := indexFrom(src, start+4, "-->")
		if end < 0 {
			return Token{}, 0, false
		}
		end += len("-->")
		return Token{Kind: KindComment, Start: start, End: end, src: src}, end, true

	case hasPrefixAt(src, start, "<![CDATA["):
		end := indexFrom(src, start+9, "]]>")
		if end < 0 {
			return Token{}, 0, false
		}
		end += len("]]>")
		return Token{Kind: KindCDATA, Start: start, End: end, src: src}, end, true

	case src[start+1] == '?' || src[start+1] == '!':
		end := indexByteFrom(src, start+2, '>')
		if end < 0 {
			return Token{}, 0, false
		}
		end++
		return Token{Kind: KindDirective, Start: start, End: end, src: src}, end, true
	}

	i := start + 1
	kind := KindStart
	if src[i] == '/' {
		kind = KindEnd
		i++
	}

	nameStart := i
	for i < len(src) && isNameByte(src[i]) {
		i++
	}
	nameEnd := i
	if nameEnd == nameStart {
		return Token{}, 0, false
	}

	attrStart := i
	for i < len(src) {
		c := src[i]
		if c == '"' || c == '\'' {
			i++
			for i < len(src) && src[i] != c {
				i++
			}
			if i < len(src) {
				i++
			}
			continue
		}
		if c == '>' {
			break
		}
		i++
	}
	if i >= len(src) {
		return Token{}, 0, false
	}

	attrEnd := i
	if kind == KindStart && attrEnd > attrStart && src[attrEnd-1] == '/' {
		kind = KindSelfClose
		attrEnd--
	}

	return Token{
		Kind:      kind,
		Start:     start,
		End:       i + 1,
		src:       src,
		nameStart: nameStart,
		nameEnd:   nameEnd,
		attrStart: attrStart,
		attrEnd:   attrEnd,
	}, i + 1, true
}

// ErrStopWalk ends a Walk early without reporting an error to the caller.
var ErrStopWalk = errors.New("xmldoc: stop walk")

// Walk visits every token in src, passing the names of the elements currently
// open around it. The stack never contains the visited token itself, so a
// token's parent is always stack[len(stack)-1].
//
// Returning ErrStopWalk from visit ends the walk and returns nil; any other
// error is returned to the caller.
func Walk(src []byte, visit func(tok Token, stack []string) error) error {
	sc := NewScanner(src)
	stack := make([]string, 0, 32)

	for {
		tok, ok := sc.Next()
		if !ok {
			return nil
		}

		if tok.Kind == KindEnd {
			if n := len(stack); n > 0 {
				stack = stack[:n-1]
			}
			if err := visit(tok, stack); err != nil {
				if errors.Is(err, ErrStopWalk) {
					return nil
				}
				return err
			}
			continue
		}

		if err := visit(tok, stack); err != nil {
			if errors.Is(err, ErrStopWalk) {
				return nil
			}
			return err
		}
		if tok.Kind == KindStart {
			stack = append(stack, tok.Name())
		}
	}
}

// FindElementEnd returns the offset just past the element opened by tok,
// matching nested elements of the same name. For a self-closing tag it is
// tok.End.
func FindElementEnd(src []byte, tok Token) (int, error) {
	switch tok.Kind {
	case KindSelfClose:
		return tok.End, nil
	case KindStart:
	default:
		return 0, fmt.Errorf("xmldoc: cannot take the span of a %s token", tok.Kind)
	}

	name := tok.Name()
	depth := 1
	sc := NewScannerAt(src, tok.End)
	for {
		next, ok := sc.Next()
		if !ok {
			return 0, fmt.Errorf("xmldoc: element <%s> at offset %d is never closed", name, tok.Start)
		}
		if !next.NameIs(name) {
			continue
		}
		switch next.Kind {
		case KindStart:
			depth++
		case KindEnd:
			depth--
			if depth == 0 {
				return next.End, nil
			}
		}
	}
}

// Unescape resolves the XML entities that can appear in an attribute value.
func Unescape(s string) string {
	if !strings.ContainsRune(s, '&') {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] != '&' {
			b.WriteByte(s[i])
			i++
			continue
		}
		semi := strings.IndexByte(s[i:], ';')
		if semi < 0 {
			b.WriteString(s[i:])
			break
		}
		entity := s[i : i+semi+1]
		switch entity {
		case "&amp;":
			b.WriteByte('&')
		case "&lt;":
			b.WriteByte('<')
		case "&gt;":
			b.WriteByte('>')
		case "&quot;":
			b.WriteByte('"')
		case "&apos;":
			b.WriteByte('\'')
		default:
			if r, ok := numericEntity(entity); ok {
				b.WriteRune(r)
			} else {
				b.WriteString(entity)
			}
		}
		i += semi + 1
	}
	return b.String()
}

func numericEntity(entity string) (rune, bool) {
	body := strings.TrimSuffix(strings.TrimPrefix(entity, "&"), ";")
	if !strings.HasPrefix(body, "#") {
		return 0, false
	}
	body = body[1:]
	base := 10
	if len(body) > 0 && (body[0] == 'x' || body[0] == 'X') {
		base, body = 16, body[1:]
	}
	n, err := strconv.ParseInt(body, base, 32)
	if err != nil || n < 0 {
		return 0, false
	}
	return rune(n), true
}

var attrEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
	"'", "&apos;",
)

// Escape encodes a string for use as an XML attribute value.
func Escape(s string) string { return attrEscaper.Replace(s) }

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isNameByte(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z',
		c >= 'A' && c <= 'Z',
		c >= '0' && c <= '9',
		c == '_', c == '-', c == '.', c == ':':
		return true
	default:
		return c >= 0x80 // permit UTF-8 element names
	}
}

func indexByteFrom(src []byte, from int, c byte) int {
	if from < 0 {
		from = 0
	}
	if from >= len(src) {
		return -1
	}
	idx := bytes.IndexByte(src[from:], c)
	if idx < 0 {
		return -1
	}
	return from + idx
}

func indexFrom(src []byte, from int, sub string) int {
	if from < 0 {
		from = 0
	}
	if from >= len(src) {
		return -1
	}
	idx := bytes.Index(src[from:], []byte(sub))
	if idx < 0 {
		return -1
	}
	return from + idx
}

func hasPrefixAt(src []byte, at int, prefix string) bool {
	if at < 0 || at+len(prefix) > len(src) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if src[at+i] != prefix[i] {
			return false
		}
	}
	return true
}
