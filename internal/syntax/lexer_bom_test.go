package syntax

import "testing"

func TestLexerSkipsOnlyOneLeadingUTF8BOM(t *testing.T) {
	const body = "namespace sales\n"
	src := "\ufeff" + body
	lx := NewLexer(src)
	first, err := lx.Next()
	if err != nil || first.Kind != TokenIdent || first.Text != "namespace" {
		t.Fatalf("BOM付き先頭token: %+v, %v", first, err)
	}
	if first.Start.Offset != 3 || first.Start.Column != 1 {
		t.Fatalf("offsetは元のbyte列に、columnは可視文字に合わせる: %+v", first.Start)
	}
	if got := src[first.Start.Offset:first.End.Offset]; got != "namespace" {
		t.Fatalf("offsetで元のSourceを切れません: %q", got)
	}

	middle := NewLexer("namespace \ufeffsales")
	if _, err := middle.Next(); err != nil {
		t.Fatal(err)
	}
	if _, err := middle.Next(); err == nil {
		t.Fatal("途中のBOMを黙って受け入れています")
	}

	doubled := NewLexer("\ufeff\ufeffnamespace")
	if _, err := doubled.Next(); err == nil {
		t.Fatal("先頭でも2個目のBOMを黙って受け入れています")
	}
}
