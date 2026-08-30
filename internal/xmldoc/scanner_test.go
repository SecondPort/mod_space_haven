package xmldoc

import (
	"strings"
	"testing"
)

func TestScannerEmitsTokensWithExactByteRanges(t *testing.T) {
	src := []byte(`<a x="1"><b/></a>`)
	sc := NewScanner(src)

	want := []struct {
		kind       Kind
		name       string
		start, end int
	}{
		{KindStart, "a", 0, 9},
		{KindSelfClose, "b", 9, 13},
		{KindEnd, "a", 13, 17},
	}

	for i, w := range want {
		tok, ok := sc.Next()
		if !ok {
			t.Fatalf("token %d: scanner stopped early", i)
		}
		if tok.Kind != w.kind {
			t.Errorf("token %d: kind = %v, want %v", i, tok.Kind, w.kind)
		}
		if got := tok.Name(); got != w.name {
			t.Errorf("token %d: name = %q, want %q", i, got, w.name)
		}
		if tok.Start != w.start || tok.End != w.end {
			t.Errorf("token %d: span = [%d,%d), want [%d,%d)", i, tok.Start, tok.End, w.start, w.end)
		}
		if got := string(src[tok.Start:tok.End]); !strings.HasPrefix(got, "<") {
			t.Errorf("token %d: span %q does not start a tag", i, got)
		}
	}

	if _, ok := sc.Next(); ok {
		t.Error("scanner produced a token past the end of input")
	}
}

func TestScannerParsesAttributeValueRanges(t *testing.T) {
	src := []byte(`<s elementaryId="157"  inStorage="42" note='ok'/>`)
	sc := NewScanner(src)

	tok, ok := sc.Next()
	if !ok {
		t.Fatal("no token produced")
	}

	attr, ok := tok.Attr("inStorage")
	if !ok {
		t.Fatal("attribute inStorage not found")
	}
	if attr.Value != "42" {
		t.Errorf("value = %q, want %q", attr.Value, "42")
	}
	if got := string(src[attr.ValueStart:attr.ValueEnd]); got != "42" {
		t.Errorf("value range yields %q, want %q", got, "42")
	}
	if got := string(src[attr.Start:attr.End]); got != `inStorage="42"` {
		t.Errorf("attribute range yields %q", got)
	}

	single, ok := tok.Attr("note")
	if !ok {
		t.Fatal("single-quoted attribute not found")
	}
	if single.Value != "ok" {
		t.Errorf("single-quoted value = %q, want %q", single.Value, "ok")
	}

	if _, ok := tok.Attr("missing"); ok {
		t.Error("Attr reported a missing attribute as present")
	}
}

func TestScannerIntAttr(t *testing.T) {
	sc := NewScanner([]byte(`<a n="17" bad="x"/>`))
	tok, _ := sc.Next()

	if got, ok := tok.IntAttr("n"); !ok || got != 17 {
		t.Errorf("IntAttr(n) = %d, %v; want 17, true", got, ok)
	}
	if _, ok := tok.IntAttr("bad"); ok {
		t.Error("IntAttr accepted a non-numeric value")
	}
	if _, ok := tok.IntAttr("nope"); ok {
		t.Error("IntAttr accepted a missing attribute")
	}
}

func TestScannerSkipsCommentsDirectivesAndCDATA(t *testing.T) {
	src := []byte(`<?xml version="1.0"?><!-- <fake attr="x"/> --><r><![CDATA[<not a tag>]]></r>`)
	sc := NewScanner(src)

	var kinds []Kind
	for {
		tok, ok := sc.Next()
		if !ok {
			break
		}
		kinds = append(kinds, tok.Kind)
	}

	want := []Kind{KindDirective, KindComment, KindStart, KindCDATA, KindEnd}
	if len(kinds) != len(want) {
		t.Fatalf("kinds = %v, want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Errorf("kind %d = %v, want %v", i, kinds[i], want[i])
		}
	}
}

func TestScannerNameIsAvoidsAllocation(t *testing.T) {
	sc := NewScanner([]byte(`<feat id="3"/>`))
	tok, _ := sc.Next()

	if !tok.NameIs("feat") {
		t.Error("NameIs(feat) = false, want true")
	}
	if tok.NameIs("fea") || tok.NameIs("feats") {
		t.Error("NameIs matched a different name")
	}
}

func TestWalkTracksParentStack(t *testing.T) {
	src := []byte(`<root><feat><inv><s id="1"/></inv></feat><feat><prod><inv/></prod></feat></root>`)

	var parents []string
	err := Walk(src, func(tok Token, stack []string) error {
		if tok.Kind == KindStart || tok.Kind == KindSelfClose {
			if tok.NameIs("inv") {
				if len(stack) == 0 {
					parents = append(parents, "")
					return nil
				}
				parents = append(parents, stack[len(stack)-1])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	want := []string{"feat", "prod"}
	if len(parents) != len(want) {
		t.Fatalf("parents = %v, want %v", parents, want)
	}
	for i := range want {
		if parents[i] != want[i] {
			t.Errorf("parent %d = %q, want %q", i, parents[i], want[i])
		}
	}
}

func TestWalkStackExcludesTheTokenItself(t *testing.T) {
	// A self-closing sibling between <feat> and <inv> must not become the parent.
	src := []byte(`<feat><meta v="1"/><inv/></feat>`)

	var parent string
	_ = Walk(src, func(tok Token, stack []string) error {
		if tok.NameIs("inv") && len(stack) > 0 {
			parent = stack[len(stack)-1]
		}
		return nil
	})

	if parent != "feat" {
		t.Errorf("parent of <inv> = %q, want %q (a self-closing sibling leaked into the stack)", parent, "feat")
	}
}

func TestFindElementEndHandlesNestedSameName(t *testing.T) {
	src := []byte(`<ship sid="1"><ship sid="2"></ship><x/></ship><tail/>`)
	sc := NewScanner(src)
	tok, _ := sc.Next()

	end, err := FindElementEnd(src, tok)
	if err != nil {
		t.Fatalf("FindElementEnd: %v", err)
	}
	want := strings.Index(string(src), "<tail/>")
	if end != want {
		t.Errorf("end = %d, want %d; span = %q", end, want, src[tok.Start:end])
	}
}

func TestFindElementEndOnSelfClosingElement(t *testing.T) {
	src := []byte(`<a/><b/>`)
	sc := NewScanner(src)
	tok, _ := sc.Next()

	end, err := FindElementEnd(src, tok)
	if err != nil {
		t.Fatalf("FindElementEnd: %v", err)
	}
	if end != 4 {
		t.Errorf("end = %d, want 4", end)
	}
}

func TestFindElementEndReportsUnclosedElement(t *testing.T) {
	src := []byte(`<a><b></b>`)
	sc := NewScanner(src)
	tok, _ := sc.Next()

	if _, err := FindElementEnd(src, tok); err == nil {
		t.Error("FindElementEnd accepted an unclosed element")
	}
}

func TestWalkStopsOnSentinelError(t *testing.T) {
	src := []byte(`<a/><b/><c/>`)

	var seen int
	err := Walk(src, func(tok Token, stack []string) error {
		seen++
		if tok.NameIs("b") {
			return ErrStopWalk
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk returned the sentinel to the caller: %v", err)
	}
	if seen != 2 {
		t.Errorf("visited %d tokens, want 2", seen)
	}
}

func TestUnescape(t *testing.T) {
	cases := map[string]string{
		"plain":                       "plain",
		"a &amp; b":                   "a & b",
		"&lt;tag&gt;":                 "<tag>",
		"&quot;q&quot; &apos;s&apos;": `"q" 's'`,
		"&#65;&#x42;":                 "AB",
		"&unknown;":                   "&unknown;",
	}
	for in, want := range cases {
		if got := Unescape(in); got != want {
			t.Errorf("Unescape(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEscape(t *testing.T) {
	if got, want := Escape(`a & <b> "c" 'd'`), `a &amp; &lt;b&gt; &quot;c&quot; &apos;d&apos;`; got != want {
		t.Errorf("Escape = %q, want %q", got, want)
	}
}
