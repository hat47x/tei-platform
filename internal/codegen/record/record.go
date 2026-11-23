// Package record holds the versioned records a code generator writes beside
// its output (#5773): one TraceLine per generated symbol, and a Manifest of the
// files written. Each JSON form is described by a JSON Schema (2020-12)
// embedded under schema/, together with the schema of diag.Record.
//
// A record names its schema in its "schema" field. The version is draft.1:
// the shape may still change, and a change gives a new version string rather
// than a silent edit.
//
// Validate checks the same rules the schemas state, so callers need no JSON
// Schema library.
package record

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"

	"tei/internal/core"
)

// Schema identifiers. Each is the "$id" of the matching schema file.
const (
	DiagnosticSchema = "tei.codegen.diagnostic/draft.1"
	TraceSchema      = "tei.codegen.trace/draft.1"
	ManifestSchema   = "tei.codegen.manifest/draft.1"
)

//go:embed schema/*.schema.json
var schemas embed.FS

// SchemaFile returns the JSON Schema of the named record: "diagnostic",
// "trace" or "manifest".
func SchemaFile(name string) ([]byte, error) {
	return schemas.ReadFile("schema/" + name + ".schema.json")
}

// TraceLine ties one generated symbol to the design element it came from.
type TraceLine struct {
	Schema string `json:"schema"`
	// File is the generated file, as written in the Manifest.
	File string `json:"file"`
	// Lines is the first and last line of the symbol in File, 1-based and
	// inclusive. The first line is that of its doc comment, when it has one.
	Lines [2]int `json:"lines"`
	// Symbol is the name of the generated declaration.
	Symbol  string           `json:"symbol"`
	Element core.CanonicalID `json:"element"`
	// Role says what the symbol does for Element, such as "decision" or
	// "check".
	Role string `json:"role"`
	// Provider names the generator that wrote the symbol.
	Provider string `json:"provider,omitempty"`
}

var rolePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Validate reports the first way t breaks the trace schema, or nil.
func (t TraceLine) Validate() error {
	switch {
	case t.Schema != TraceSchema:
		return fmt.Errorf("trace schema %q, want %q", t.Schema, TraceSchema)
	case t.File == "":
		return fmt.Errorf("trace of %s has no file", t.Symbol)
	case t.Lines[0] < 1 || t.Lines[1] < t.Lines[0]:
		return fmt.Errorf("trace of %s: lines %v must satisfy 1 <= first <= last", t.Symbol, t.Lines)
	case t.Symbol == "":
		return fmt.Errorf("trace in %s has no symbol", t.File)
	case t.Element == "":
		return fmt.Errorf("trace of %s has no element", t.Symbol)
	case !rolePattern.MatchString(t.Role):
		return fmt.Errorf("trace of %s: role %q does not match %s", t.Symbol, t.Role, rolePattern)
	}
	return nil
}

// Manifest lists the files one generation wrote.
type Manifest struct {
	Schema string `json:"schema"`
	// Input is the SHA-256 of the input record the files were generated
	// from, in lower-case hex. It is empty until that record exists.
	Input string         `json:"input,omitempty"`
	Files []ManifestFile `json:"files"`
}

// ManifestFile is one written file.
type ManifestFile struct {
	Path string `json:"path"`
	// SHA256 is the digest of the file content, in lower-case hex.
	SHA256 string `json:"sha256"`
	// Writer names the generator that wrote the file.
	Writer string `json:"writer"`
	// Hook names the hook point that changed the file, when one did.
	Hook string `json:"hook,omitempty"`
}

// FileContent is a file to list in a Manifest.
type FileContent struct {
	Path    string
	Content []byte
}

// NewManifest lists files sorted by path, each with the SHA-256 of its
// content and the given writer and hook.
func NewManifest(input string, files []FileContent, writer, hook string) Manifest {
	m := Manifest{Schema: ManifestSchema, Input: input, Files: make([]ManifestFile, 0, len(files))}
	for _, file := range files {
		sum := sha256.Sum256(file.Content)
		m.Files = append(m.Files, ManifestFile{
			Path: file.Path, SHA256: hex.EncodeToString(sum[:]), Writer: writer, Hook: hook,
		})
	}
	sort.SliceStable(m.Files, func(i, j int) bool { return m.Files[i].Path < m.Files[j].Path })
	return m
}

var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Validate reports the first way m breaks the manifest schema, or nil. Files
// must be sorted by path with no path twice.
func (m Manifest) Validate() error {
	if m.Schema != ManifestSchema {
		return fmt.Errorf("manifest schema %q, want %q", m.Schema, ManifestSchema)
	}
	if m.Input != "" && !digestPattern.MatchString(m.Input) {
		return fmt.Errorf("manifest input %q is not a lower-case hex SHA-256", m.Input)
	}
	if m.Files == nil {
		return fmt.Errorf("manifest has no files list")
	}
	for i, file := range m.Files {
		switch {
		case file.Path == "":
			return fmt.Errorf("manifest file %d has no path", i)
		case !digestPattern.MatchString(file.SHA256):
			return fmt.Errorf("manifest file %s: sha256 %q is not a lower-case hex SHA-256", file.Path, file.SHA256)
		case file.Writer == "":
			return fmt.Errorf("manifest file %s has no writer", file.Path)
		case i > 0 && m.Files[i-1].Path >= file.Path:
			return fmt.Errorf("manifest files are not sorted by path with no repeats: %s after %s", file.Path, m.Files[i-1].Path)
		}
	}
	return nil
}
