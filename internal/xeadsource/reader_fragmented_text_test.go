package xeadsource

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseWithLimitsPreservesFragmentedText(t *testing.T) {
	const fragments = 40000
	source := []byte("<System>" + strings.Repeat("x<!--split-->", fragments) + "</System>")
	doc, err := ParseWithLimits(FormatXead, source, ParseLimits{
		MaxBytes: int64(len(source)), MaxDepth: 1, MaxElements: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Root.Text != strings.Repeat("x", fragments) || len(doc.Root.Children) != 0 {
		t.Fatal("fragmented character data was not preserved")
	}
}

func TestParseKeepsMixedTextSeparateFromChildText(t *testing.T) {
	source := []byte(`<System> before<!--split--><Table ID="T1"> inner<!--split-->text </Table>after </System>`)
	doc, err := Parse(FormatXead, source)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Root.Text != "beforeafter" || len(doc.Root.Children) != 1 || doc.Root.Children[0].Text != "innertext" {
		t.Fatal("text order, trimming or child boundaries changed")
	}
}

func BenchmarkParseFragmentedText(b *testing.B) {
	for _, fragments := range []int{1000, 10000} {
		b.Run(fmt.Sprint(fragments), func(b *testing.B) {
			source := []byte("<System>" + strings.Repeat("x<!--split-->", fragments) + "</System>")
			limits := ParseLimits{MaxBytes: int64(len(source)), MaxDepth: 1, MaxElements: 1}
			b.SetBytes(int64(len(source)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := ParseWithLimits(FormatXead, source, limits); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
