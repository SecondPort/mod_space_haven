// Package savegame is the domain core of the editor: it reads and edits a
// Space Haven save without ever re-serializing it.
//
// Every write is a byte-range replacement against the original document (see
// internal/xmldoc), so an edited save differs from the original only in the
// attribute values that were actually changed. Nothing here touches the
// filesystem — loading, backing up and writing live in internal/library, which
// keeps this package testable against fixtures alone.
package savegame

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/SecondPort/mod_space_haven/internal/xmldoc"
)

// Errors the editor is expected to handle rather than crash on.
var (
	// ErrPlayerShipNotFound means the save has no settlement flagged as the
	// player's, so there is no cargo hold to edit.
	ErrPlayerShipNotFound = errors.New("savegame: no player ship in this save")
	// ErrCharacterNotFound means no character carries the requested entity id.
	ErrCharacterNotFound = errors.New("savegame: character not found")
	// ErrTechNotFound means the save has no research entry for a technology.
	ErrTechNotFound = errors.New("savegame: technology not found in this save")
)

// Save is a parsed save file held in memory.
type Save struct {
	data  []byte
	dirty bool
}

// Parse takes ownership of a save file's bytes.
func Parse(data []byte) (*Save, error) {
	if len(data) == 0 {
		return nil, errors.New("savegame: the save file is empty")
	}
	sc := xmldoc.NewScanner(data)
	if _, ok := sc.Next(); !ok {
		return nil, errors.New("savegame: the file contains no XML")
	}
	buf := make([]byte, len(data))
	copy(buf, data)
	return &Save{data: buf}, nil
}

// Bytes returns a copy of the current document, ready to be written to disk.
func (s *Save) Bytes() []byte {
	out := make([]byte, len(s.data))
	copy(out, s.data)
	return out
}

// Size reports the document length in bytes.
func (s *Save) Size() int { return len(s.data) }

// Dirty reports whether the save holds edits that are not on disk yet.
func (s *Save) Dirty() bool { return s.dirty }

// MarkSaved records that the current document has been written out.
func (s *Save) MarkSaved() { s.dirty = false }

// apply commits a batch of edits.
//
// Patches that would write back the bytes already there are dropped first, so
// re-entering a value the save already holds leaves the document alone and does
// not raise the unsaved-changes flag.
func (s *Save) apply(ps xmldoc.PatchSet) error {
	effective := make(xmldoc.PatchSet, 0, len(ps))
	for _, p := range ps {
		if p.Start >= 0 && p.End <= len(s.data) && p.Start <= p.End &&
			bytes.Equal(s.data[p.Start:p.End], p.New) {
			continue
		}
		effective = append(effective, p)
	}
	if len(effective) == 0 {
		return nil
	}

	out, err := effective.Apply(s.data)
	if err != nil {
		return err
	}
	s.data = out
	s.dirty = true
	return nil
}

// region is a slice of the document plus the offset it starts at, so patches
// built against the slice can be translated back to document coordinates.
type region struct {
	base int
	data []byte
}

func (s *Save) region(start, end int) region {
	return region{base: start, data: s.data[start:end]}
}

func (r region) shift(p xmldoc.Patch) xmldoc.Patch {
	p.Start += r.base
	p.End += r.base
	return p
}

// firstToken returns the first token in r that satisfies match.
func (r region) firstToken(match func(xmldoc.Token) bool) (xmldoc.Token, bool) {
	sc := xmldoc.NewScanner(r.data)
	for {
		tok, ok := sc.Next()
		if !ok {
			return xmldoc.Token{}, false
		}
		if match(tok) {
			return tok, true
		}
	}
}

// eachToken visits every token in r.
func (r region) eachToken(visit func(xmldoc.Token) bool) {
	sc := xmldoc.NewScanner(r.data)
	for {
		tok, ok := sc.Next()
		if !ok {
			return
		}
		if !visit(tok) {
			return
		}
	}
}

// isElement reports whether a token opens an element (either form).
func isElement(tok xmldoc.Token) bool {
	return tok.Kind == xmldoc.KindStart || tok.Kind == xmldoc.KindSelfClose
}

// setIntAttr builds a patch that writes an integer into an attribute, and
// reports an error naming the element when the attribute is missing.
func setIntAttr(tok xmldoc.Token, name string, value int) (xmldoc.Patch, error) {
	a, ok := tok.Attr(name)
	if !ok {
		return xmldoc.Patch{}, fmt.Errorf("savegame: <%s> has no %s attribute", tok.Name(), name)
	}
	return xmldoc.SetAttrRaw(a, fmt.Sprint(value)), nil
}

// leadingWhitespace returns the run of whitespace immediately before offset,
// which is how new elements inherit their siblings' indentation.
func leadingWhitespace(data []byte, offset int) string {
	i := offset
	for i > 0 && isSpaceByte(data[i-1]) {
		i--
	}
	return string(data[i:offset])
}

func isSpaceByte(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
