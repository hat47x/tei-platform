package sourceindex

import (
	"math/rand"
	"strings"
	"testing"
)

// positionAtReferenceは、PositionAtの定義そのものである1バイトずつ数える実装。
// 実装が速度のために別の方法へ変わっても、結果がこの定義と一致し続けることを固定する。
func positionAtReference(content []byte, offset int) (Position, bool) {
	if offset < 0 || offset > len(content) {
		return Position{}, false
	}
	line, column := 1, 1
	for i := 0; i < offset; i++ {
		if content[i] == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	return Position{Offset: offset, Line: line, Column: column}, true
}

func positionTestContents() map[string][]byte {
	random := rand.New(rand.NewSource(20261014))
	alphabet := []byte("ab \t\n\n\r\xe3\x81\x82\xf0\x9f\x98\x80")
	noisy := make([]byte, 600)
	for i := range noisy {
		noisy[i] = alphabet[random.Intn(len(alphabet))]
	}
	return map[string][]byte{
		"empty":                  nil,
		"single newline":         []byte("\n"),
		"no newline":             []byte("semantic data x {}"),
		"trailing newline":       []byte("a\nb\n"),
		"leading newlines":       []byte("\n\n\nx"),
		"consecutive newlines":   []byte("a\n\n\nb"),
		"crlf":                   []byte("a\r\nb\r\nc"),
		"multibyte utf-8":        []byte("名前: \xe3\x81\x82\xe3\x81\x84\n値: \xf0\x9f\x98\x80\n"),
		"long single line":       []byte(strings.Repeat("x", 5000)),
		"many short lines":       []byte(strings.Repeat("ab\n", 700)),
		"deterministic noise":    noisy,
		"invalid utf-8 and nuls": {0x00, 0xff, '\n', 0xfe, 0x00, '\n', '\n'},
	}
}

func TestPositionAtMatchesByteWiseDefinitionAtEveryOffset(t *testing.T) {
	for name, content := range positionTestContents() {
		t.Run(name, func(t *testing.T) {
			for offset := 0; offset <= len(content); offset++ {
				want, _ := positionAtReference(content, offset)
				got, err := PositionAt(content, offset)
				if err != nil {
					t.Fatalf("offset %d: %v", offset, err)
				}
				if got != want {
					t.Fatalf("offset %d: got %+v want %+v", offset, got, want)
				}
			}
		})
	}
}

func TestPositionAtAndOffsetAtAreInverseAtEveryOffset(t *testing.T) {
	for name, content := range positionTestContents() {
		t.Run(name, func(t *testing.T) {
			for offset := 0; offset <= len(content); offset++ {
				position, err := PositionAt(content, offset)
				if err != nil {
					t.Fatal(err)
				}
				back, err := OffsetAt(content, position.Line, position.Column)
				if err != nil {
					t.Fatalf("offset %d (%d:%d): %v", offset, position.Line, position.Column, err)
				}
				if back != offset {
					t.Fatalf("offset %d round-tripped to %d via %d:%d", offset, back, position.Line, position.Column)
				}
			}
		})
	}
}

func TestPositionAtRejectsOffsetsOutsideContentWithTheSameMessage(t *testing.T) {
	content := []byte("a\nb")
	for _, offset := range []int{-1, len(content) + 1, 1 << 30} {
		_, err := PositionAt(content, offset)
		if err == nil {
			t.Fatalf("offset %d was accepted", offset)
		}
		if !strings.Contains(err.Error(), "is outside [0,3]") {
			t.Fatalf("offset %d: unexpected error text %q", offset, err)
		}
	}
}

func TestPositionAtDoesNotModifyContent(t *testing.T) {
	content := []byte("a\nb\nc")
	snapshot := string(content)
	for offset := 0; offset <= len(content); offset++ {
		if _, err := PositionAt(content, offset); err != nil {
			t.Fatal(err)
		}
	}
	if string(content) != snapshot {
		t.Fatal("PositionAt modified its input")
	}
}
