package xmldoc

import (
	"fmt"
	"sort"
)

// Patch replaces src[Start:End) with New. A patch where Start equals End is an
// insertion at that offset.
type Patch struct {
	Start, End int
	New        []byte
}

// PatchSet is a batch of edits against one source buffer. Every offset in a set
// refers to the original buffer, so callers can collect edits in any order
// without tracking how earlier ones shifted the text — the classic source of
// off-by-N corruption when splicing a save file by hand.
type PatchSet []Patch

// SetAttr returns a patch that rewrites one attribute's value in place,
// leaving the attribute's name, quoting and position untouched.
func SetAttr(a Attr, value string) Patch {
	return Patch{Start: a.ValueStart, End: a.ValueEnd, New: []byte(Escape(value))}
}

// SetAttrRaw is SetAttr for values that are already known to be safe, such as
// numbers. It skips escaping so the written bytes are exactly value.
func SetAttrRaw(a Attr, value string) Patch {
	return Patch{Start: a.ValueStart, End: a.ValueEnd, New: []byte(value)}
}

// InsertAt returns a patch that inserts text at offset without removing anything.
func InsertAt(offset int, text string) Patch {
	return Patch{Start: offset, End: offset, New: []byte(text)}
}

// Delete returns a patch that removes src[start:end).
func Delete(start, end int) Patch {
	return Patch{Start: start, End: end}
}

// Apply returns a new buffer with every patch applied. The source is never
// modified. Applying an empty set copies the source.
//
// It fails if any patch falls outside the source or if two patches overlap,
// because an overlap means two edits disagree about the same bytes and
// silently letting one win is how a save file gets quietly corrupted.
func (ps PatchSet) Apply(src []byte) ([]byte, error) {
	if len(ps) == 0 {
		out := make([]byte, len(src))
		copy(out, src)
		return out, nil
	}

	ordered := make(PatchSet, len(ps))
	copy(ordered, ps)
	// Stable, so several insertions at one offset keep their declared order.
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Start < ordered[j].Start
	})

	grown := 0
	for i, p := range ordered {
		if p.Start < 0 || p.End < p.Start || p.End > len(src) {
			return nil, fmt.Errorf("xmldoc: patch %d has range [%d,%d) outside a %d-byte document", i, p.Start, p.End, len(src))
		}
		if i > 0 && p.Start < ordered[i-1].End {
			return nil, fmt.Errorf("xmldoc: patches overlap at [%d,%d) and [%d,%d)",
				ordered[i-1].Start, ordered[i-1].End, p.Start, p.End)
		}
		grown += len(p.New) - (p.End - p.Start)
	}

	out := make([]byte, 0, len(src)+grown)
	cursor := 0
	for _, p := range ordered {
		out = append(out, src[cursor:p.Start]...)
		out = append(out, p.New...)
		cursor = p.End
	}
	return append(out, src[cursor:]...), nil
}
