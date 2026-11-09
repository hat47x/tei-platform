package orderdatarealization

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestChain(t *testing.T) *Chain {
	t.Helper()
	root, err := FindRepoRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewChain(root, filepath.Join(root, "reference", "workspace"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestDeclarationsDecode(t *testing.T) {
	d, digest, err := LoadDeclarations()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(digest, "sha256:") || d.Numbering.ExternalNumber.OnUnavailable != PolicyProvisional {
		t.Fatalf("digest %s policy %s", digest, d.Numbering.ExternalNumber.OnUnavailable)
	}
	if d.Attribute.ExistingRowsValue != nil || d.Migration.OldWriterValues[d.Attribute.Attribute] != nil {
		t.Fatal("existing rows and old writers must leave the attribute NULL")
	}
	bad := strings.Replace(string(declarationsJSON), `"provisional-then-reassign"`, `"guess"`, 1)
	if _, _, err := decodeDeclarations([]byte(bad)); err == nil {
		t.Fatal("unknown policy accepted")
	}
	if _, _, err := decodeDeclarations([]byte(strings.Replace(string(declarationsJSON), `"schema"`, `"schemaX": 1, "schema"`, 1))); err == nil {
		t.Fatal("unknown field accepted")
	}
}

func TestLineEditKeepsLineEndingsAndRequiresOneAnchor(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.tei")
	os.WriteFile(p, []byte("x\r\n  anchor\r\ny\r\n"), 0o644)
	if _, err := applyLineEdit(dir, LineEdit{File: "a.tei", Anchor: "  anchor", Mode: "after", Lines: []string{"  new"}}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p)
	if string(raw) != "x\r\n  anchor\r\n  new\r\ny\r\n" {
		t.Fatalf("got %q", raw)
	}
	if _, err := applyLineEdit(dir, LineEdit{File: "a.tei", Anchor: "missing", Mode: "after"}); err == nil {
		t.Fatal("missing anchor accepted")
	}
	os.WriteFile(p, []byte("a\na\n"), 0o644)
	if _, err := applyLineEdit(dir, LineEdit{File: "a.tei", Anchor: "a", Mode: "replace"}); err == nil {
		t.Fatal("ambiguous anchor accepted")
	}
}

// TestDesignStepsAndRevisionBinding runs steps 1-4 (no database) and then
// shows that a later step stops when the copy changes or another copy is read.
func TestDesignStepsAndRevisionBinding(t *testing.T) {
	if testing.Short() {
		t.Skip("builds two tools")
	}
	c := newTestChain(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := c.RunDesign(ctx); err != nil {
		t.Fatal(err)
	}
	r := c.Rec.Revisions
	if r.R0 == r.R1 || r.R1 == r.R2 || r.R2 == r.R3 || r.VariantNoDecisions != r.R1 {
		t.Fatalf("revisions %+v", r)
	}
	if len(c.Rec.Steps) != 4 {
		t.Fatalf("steps %d", len(c.Rec.Steps))
	}
	for _, s := range c.Rec.Steps[3:] {
		if s.ModelRevision != r.R3 {
			t.Fatalf("step %d bound to %s", s.Step, s.ModelRevision)
		}
	}
	if len(c.missing["create"]) == 0 {
		t.Fatal("missing items of the create candidate not kept")
	}
	// The reference workspace was only read.
	if got, _ := Revision(c.Source); got != r.R0 {
		// R0 is computed on the copy; the copy is byte-identical.
		t.Fatalf("source revision %s != R0 %s", got, r.R0)
	}
	// Another copy (R0) is refused.
	var mm *RevisionMismatch
	if err := c.require(5, c.R0Dir); !errors.As(err, &mm) || mm.Got != r.R0 {
		t.Fatalf("another copy not refused: %v", err)
	}
	// A change to the copy after step 3 is refused.
	f := filepath.Join(c.WorkDir, "semantic", "sales", "order.tei")
	raw, _ := os.ReadFile(f)
	os.WriteFile(f, append(raw, []byte("\n")...), 0o644)
	if err := c.require(5, c.WorkDir); !errors.Is(err, ErrRevisionMismatch) {
		t.Fatalf("changed copy not refused: %v", err)
	}
	os.WriteFile(f, raw, 0o644)
	if err := c.require(5, c.WorkDir); err != nil {
		t.Fatalf("restored copy refused: %v", err)
	}
}
