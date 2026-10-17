package identity

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"testing"
)

// 次の二つの正規表現が、CanonicalIDとLocalNameの定義である。
// 検証を手書きの走査に置き換えても、受理する文字列の集合がこの定義と一致し続けることを固定する。
var (
	canonicalIDReference = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$`)
	localNameReference   = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

func parseCanonicalIDReference(s string) (CanonicalID, error) {
	s = strings.TrimSpace(s)
	if !canonicalIDReference.MatchString(s) {
		return "", fmt.Errorf("invalid canonical id %q", s)
	}
	return CanonicalID(s), nil
}

func parseLocalNameReference(s string) (LocalName, error) {
	s = strings.TrimSpace(s)
	if !localNameReference.MatchString(s) {
		return "", fmt.Errorf("invalid local name %q", s)
	}
	return LocalName(s), nil
}

func sameResult[T comparable](t *testing.T, input string, got T, gotErr error, want T, wantErr error) {
	t.Helper()
	if got != want {
		t.Fatalf("%q: value %v, want %v", input, got, want)
	}
	if (gotErr == nil) != (wantErr == nil) {
		t.Fatalf("%q: accepted=%v, want accepted=%v (err=%v)", input, gotErr == nil, wantErr == nil, gotErr)
	}
	if gotErr != nil && gotErr.Error() != wantErr.Error() {
		t.Fatalf("%q: error %q, want %q", input, gotErr, wantErr)
	}
}

func checkAgainstReference(t *testing.T, s string) {
	t.Helper()
	gotID, gotIDErr := ParseCanonicalID(s)
	wantID, wantIDErr := parseCanonicalIDReference(s)
	sameResult(t, s, gotID, gotIDErr, wantID, wantIDErr)

	gotName, gotNameErr := ParseLocalName(s)
	wantName, wantNameErr := parseLocalNameReference(s)
	sameResult(t, s, gotName, gotNameErr, wantName, wantNameErr)
}

// 英小文字、数字、"_"、"."、先頭になれない文字、大文字、空白、多バイト文字を含む
// 記号の組合せを、長さ6まで全て列挙して定義と照合する。
func TestIdentityParsingMatchesRegexpDefinitionExhaustively(t *testing.T) {
	alphabet := []string{"a", "z", "0", "9", "_", ".", "A", " ", "é"}
	var walk func(prefix string, depth int)
	walk = func(prefix string, depth int) {
		checkAgainstReference(t, prefix)
		if depth == 0 {
			return
		}
		for _, symbol := range alphabet {
			walk(prefix+symbol, depth-1)
		}
	}
	walk("", 6)
}

func TestIdentityParsingMatchesRegexpDefinitionOnRandomStrings(t *testing.T) {
	random := rand.New(rand.NewSource(20261017))
	alphabet := []string{
		"a", "b", "z", "m", "0", "5", "9", "_", "_", ".", ".", "-", "/", ":", "A", "Z", " ", "\t", "\n", "\r",
		" ", "　", "é", "名", "ａ", "\xff", "\x00", "$", "^", "\\",
	}
	for i := 0; i < 200000; i++ {
		var b strings.Builder
		for n := random.Intn(24); n >= 0; n-- {
			b.WriteString(alphabet[random.Intn(len(alphabet))])
		}
		checkAgainstReference(t, b.String())
	}
}

// 周囲の空白（Unicodeの空白を含む）は、従来どおり取り除いてから判定する。
func TestIdentityParsingTrimsSurroundingWhitespaceLikeBefore(t *testing.T) {
	for _, input := range []string{
		" a ", "\ta.b\n", " a ", "　a.b　", "a b", " a. ", "\na\n.b", "a.\nb", "\n\n",
	} {
		checkAgainstReference(t, input)
	}
	if got, err := ParseCanonicalID(" sales.order \n"); err != nil || got != "sales.order" {
		t.Fatalf("surrounding whitespace was not trimmed: %q %v", got, err)
	}
}

func TestSymbolToSegmentKeepsRejectingSymbolsWithoutAValidLeadingLetter(t *testing.T) {
	for _, tc := range []struct {
		symbol  string
		want    string
		wantErr bool
	}{
		{"OrderStatus", "order_status", false},
		{"orderId", "order_id", false},
		{"HTTPServer", "http_server", false},
		{"9abc", "", true}, // 先頭が数字
		{"_x", "x", false}, // 先頭の"_"は取り除かれる
		{"a-b c", "a_b_c", false},
		{"名前", "", true}, // 英小文字・数字以外
		{"  ", "", true},
	} {
		got, err := SymbolToSegment(tc.symbol)
		if (err != nil) != tc.wantErr || got != tc.want {
			t.Fatalf("SymbolToSegment(%q) = %q, %v; want %q (error=%v)", tc.symbol, got, err, tc.want, tc.wantErr)
		}
	}
}

var benchmarkIDs = []string{
	"sales.order", "sales.order_line.unit_price", "std.deployment.executor", "hub.hub_id",
	"a", "a.b.c.d.e.f.g.h", "attribute_000123.field_0004",
}

var benchmarkSink CanonicalID

func BenchmarkParseCanonicalID(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		id, err := ParseCanonicalID(benchmarkIDs[i%len(benchmarkIDs)])
		if err != nil {
			b.Fatal(err)
		}
		benchmarkSink = id
	}
}

func BenchmarkParseCanonicalIDRegexpDefinition(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		id, err := parseCanonicalIDReference(benchmarkIDs[i%len(benchmarkIDs)])
		if err != nil {
			b.Fatal(err)
		}
		benchmarkSink = id
	}
}
