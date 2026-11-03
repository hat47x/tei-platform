package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseSemanticFileAcceptsOptionalFormatVersion(t *testing.T) {
	src := "formatVersion 1\nnamespace sales\n\ntype OrderId from std.identifier\n"
	ns, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if ns.FormatVersion != 1 {
		t.Fatalf("formatVersion=%d, want 1", ns.FormatVersion)
	}
	if string(ns.ID) != "sales" {
		t.Fatalf("namespace=%q", ns.ID)
	}
	if len(ns.Types) != 1 {
		t.Fatalf("types=%d", len(ns.Types))
	}
}

func TestParseSemanticFileWithoutFormatVersionReportsZero(t *testing.T) {
	ns, err := syntax.NewParser("namespace sales\n").ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if ns.FormatVersion != 0 {
		t.Fatalf("formatVersion=%d, want 0 for an undeclared header", ns.FormatVersion)
	}
}

func TestParseSemanticFileRejectsInvalidFormatVersion(t *testing.T) {
	for _, tc := range []struct{ name, src string }{
		{name: "zero", src: "formatVersion 0\nnamespace sales\n"},
		{name: "negative", src: "formatVersion -1\nnamespace sales\n"},
		{name: "not a number", src: "formatVersion one\nnamespace sales\n"},
		{name: "missing value", src: "formatVersion\nnamespace sales\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := syntax.NewParser(tc.src).ParseSemanticFile()
			if err == nil {
				t.Fatal("invalid formatVersion was accepted")
			}
		})
	}
}

func TestParseSemanticFileAnchoredSkipsFormatVersionHeader(t *testing.T) {
	src := "formatVersion 1\nnamespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n}\n"
	anchored, err := syntax.ParseSemanticFileAnchored(src)
	if err != nil {
		t.Fatal(err)
	}
	if anchored.Value.FormatVersion != 1 {
		t.Fatalf("formatVersion=%d", anchored.Value.FormatVersion)
	}
	if len(anchored.Anchors) == 0 {
		t.Fatal("no anchors were produced")
	}
	if anchored.Anchors[0].Kind != "semantic" || anchored.Anchors[0].ID != "sales" {
		t.Fatalf("first anchor=%+v", anchored.Anchors[0])
	}
	// The anchor for the namespace must point at the namespace identifier, not
	// the format header.
	if got := src[anchored.Anchors[0].Anchor.Start.Offset:anchored.Anchors[0].Anchor.End.Offset]; got != "sales" {
		t.Fatalf("namespace anchor range=%q, want %q", got, "sales")
	}
}
