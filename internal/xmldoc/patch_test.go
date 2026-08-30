package xmldoc

import (
	"strings"
	"testing"
)

func TestPatchSetAppliesEditsRegardlessOfOrder(t *testing.T) {
	src := []byte(`<a v="1"/><b v="2"/><c v="3"/>`)

	// Deliberately unsorted: the set must order the edits itself.
	ps := PatchSet{
		{Start: 26, End: 27, New: []byte("300")},
		{Start: 6, End: 7, New: []byte("100")},
		{Start: 16, End: 17, New: []byte("200")},
	}

	got, err := ps.Apply(src)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	want := `<a v="100"/><b v="200"/><c v="300"/>`
	if string(got) != want {
		t.Errorf("Apply = %q, want %q", got, want)
	}
}

func TestPatchSetLeavesSourceUntouched(t *testing.T) {
	src := []byte(`<a v="1"/>`)
	original := string(src)

	if _, err := (PatchSet{{Start: 6, End: 7, New: []byte("9")}}).Apply(src); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if string(src) != original {
		t.Errorf("Apply mutated its input: %q, want %q", src, original)
	}
}

func TestPatchSetInsertsAtAPoint(t *testing.T) {
	src := []byte(`<inv></inv>`)

	got, err := (PatchSet{{Start: 5, End: 5, New: []byte(`<s id="1"/>`)}}).Apply(src)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	want := `<inv><s id="1"/></inv>`
	if string(got) != want {
		t.Errorf("Apply = %q, want %q", got, want)
	}
}

func TestPatchSetRejectsOverlappingEdits(t *testing.T) {
	src := []byte(`<a v="1234"/>`)

	ps := PatchSet{
		{Start: 6, End: 10, New: []byte("x")},
		{Start: 8, End: 11, New: []byte("y")},
	}

	if _, err := ps.Apply(src); err == nil {
		t.Fatal("Apply accepted overlapping edits")
	} else if !strings.Contains(err.Error(), "overlap") {
		t.Errorf("error = %v, want it to mention an overlap", err)
	}
}

func TestPatchSetAllowsTwoInsertsAtTheSameOffset(t *testing.T) {
	src := []byte(`<inv></inv>`)

	ps := PatchSet{
		{Start: 5, End: 5, New: []byte("<a/>")},
		{Start: 5, End: 5, New: []byte("<b/>")},
	}

	got, err := ps.Apply(src)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// Both land; the set preserves the order they were declared in.
	want := `<inv><a/><b/></inv>`
	if string(got) != want {
		t.Errorf("Apply = %q, want %q", got, want)
	}
}

func TestPatchSetRejectsOutOfRangeEdits(t *testing.T) {
	src := []byte(`<a/>`)

	cases := map[string]Patch{
		"negative start":   {Start: -1, End: 1},
		"end past input":   {Start: 0, End: 99},
		"end before start": {Start: 3, End: 1},
	}

	for name, p := range cases {
		if _, err := (PatchSet{p}).Apply(src); err == nil {
			t.Errorf("%s: Apply accepted an invalid range", name)
		}
	}
}

func TestPatchSetEmptyIsIdentity(t *testing.T) {
	src := []byte(`<a v="1"/>`)

	got, err := PatchSet{}.Apply(src)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if string(got) != string(src) {
		t.Errorf("Apply = %q, want %q", got, src)
	}
}
