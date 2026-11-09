package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func hardlinkedFiles(t *testing.T) (source, alias string) {
	t.Helper()
	dir := t.TempDir()
	source = filepath.Join(dir, "input.json")
	alias = filepath.Join(dir, "output.json")
	if err := os.WriteFile(source, []byte("original-input"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(source, alias); err != nil {
		t.Skipf("hard links are unavailable: %v", err)
	}
	if referencePathText(source) == referencePathText(alias) {
		t.Fatal("test setup requires distinct path spellings")
	}
	return source, alias
}

func TestReferenceOutputCollisionIncludesHardLinks(t *testing.T) {
	source, alias := hardlinkedFiles(t)
	collision, err := outputDestinationsReferToSameFile(source, alias)
	if err != nil || !collision {
		t.Fatalf("hardlinked output not detected: collision=%t err=%v", collision, err)
	}
	other := filepath.Join(t.TempDir(), "independent.json")
	if err := os.WriteFile(other, []byte("other"), 0o600); err != nil {
		t.Fatal(err)
	}
	if collision, err := outputDestinationsReferToSameFile(source, other); err != nil || collision {
		t.Fatalf("unrelated files incorrectly collide: collision=%t err=%v", collision, err)
	}
	newFile := filepath.Join(t.TempDir(), "new.json")
	if collision, err := outputDestinationsReferToSameFile(source, newFile); err != nil || collision {
		t.Fatalf("nonexistent new output incorrectly collides: collision=%t err=%v", collision, err)
	}
}

func TestReferenceCLIDoesNotOverwriteHardlinkedInputs(t *testing.T) {
	for _, tc := range []struct {
		command string
		input   string
		want    string
	}{
		{command: "preview", input: "-candidate", want: "入力ファイル"},
		{command: "review", input: "-candidate", want: "入力ファイル"},
		{command: "reject", input: "-candidate", want: "入力ファイル"},
		{command: "confirm-check", input: "-input", want: "入力ファイル"},
		{command: "confirm-record", input: "-input", want: "入力ファイル"},
		{command: "accept", input: "-candidate", want: "candidate"},
		{command: "attach-qualification", input: "-qualification", want: "-qualification"},
		{command: "propose-qualified-reference-pattern", input: "-reference-reuse", want: "-reference-reuse"},
		{command: "attach-semantic-evaluation", input: "-semantic-evaluation", want: "-semantic-evaluation"},
		{command: "attach-rmt-transcription", input: "-rmt-transcription", want: "-rmt-transcription"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			source, alias := hardlinkedFiles(t)
			candidate := filepath.Join(t.TempDir(), "candidate.json")
			if err := os.WriteFile(candidate, []byte("invalid-candidate"), 0o600); err != nil {
				t.Fatal(err)
			}
			args := []string{tc.command, "-workspace", t.TempDir(), "-candidate", candidate,
				"-expected-review-fingerprint", "synthetic-review",
				"-input", candidate,
				"-qualification", candidate, "-reference-reuse", candidate,
				"-semantic-evaluation", candidate, "-rmt-transcription", candidate}
			// Only the selected input may be hardlinked to the output.
			args = append(args, tc.input, source, "-out", alias, "-allow-reference-baseline")
			var stdout, stderr bytes.Buffer
			if code := runCLI(args, &stdout, &stderr); code != 2 ||
				!strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("hardlink input overwrite not rejected: code=%d out=%s err=%s",
					code, stdout.String(), stderr.String())
			}
			content, err := os.ReadFile(source)
			if err != nil {
				t.Fatal(err)
			}
			if string(content) != "original-input" {
				t.Fatalf("hardlinked input was modified: %q", content)
			}
		})
	}
}

func TestReferenceAcceptRejectsHardlinkedOutputFlags(t *testing.T) {
	derived, out := hardlinkedFiles(t)
	candidate := filepath.Join(t.TempDir(), "candidate.json")
	if err := os.WriteFile(candidate, []byte("invalid-candidate"), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"accept", "-workspace", t.TempDir(), "-candidate", candidate,
		"-expected-review-fingerprint", "synthetic-review",
		"-derived-out", derived, "-out", out, "-allow-reference-baseline"}
	var stdout, stderr bytes.Buffer
	if code := runCLI(args, &stdout, &stderr); code != 2 ||
		!strings.Contains(stderr.String(), "-derived-out") {
		t.Fatalf("hardlinked accept outputs not rejected: code=%d out=%s err=%s",
			code, stdout.String(), stderr.String())
	}
	content, err := os.ReadFile(derived)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "original-input" {
		t.Fatalf("derived destination was modified: %q", content)
	}
}
