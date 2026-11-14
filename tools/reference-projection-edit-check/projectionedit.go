// Package projectionedit is a read-only front half for editing Source through a
// Property/Table projection (#2497, TASK 7 条件).
//
// It does not write Source. A row carries the Source identity and the span of one
// value; Judge decides, from the current Source, whether a change to that value
// is applicable; Apply hands the resulting candidate to the existing
// reference-shared-acceptance CLI (preview, review, accept) and nothing else.
// There is no second parser, writer or acceptance path here.
package projectionedit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"tei/internal/syntax"
	"tei/internal/workspace"
)

// States. Only StateApplied means Source changed; every other state leaves every
// byte of the workspace as it was.
const (
	StateEligible    = "ELIGIBLE"    // Judge only: the change would be handed to acceptance
	StateApplied     = "APPLIED"     // acceptance wrote Source
	StateNoop        = "NOOP"        // the new value equals the projected value
	StateStale       = "STALE"       // Source changed since the projection was read
	StateAmbiguous   = "AMBIGUOUS"   // more than one Source definition for the subject
	StateUnsupported = "UNSUPPORTED" // no editable Source definition for this edit
	StateWithheld    = "WITHHELD"    // the caller holds no write grant
	StateRejected    = "REJECTED"    // existing acceptance refused the candidate
)

const (
	propertyType     = "type"
	opSetAttrType    = "set-attribute-type"
	candidateSchema  = "tei.reference.change-candidate/v0"
	attributeSubject = "semantic-attribute"
)

// Row is one Property/Table row for one attribute's type. Revision is the model
// revision (the acceptance tool's own definition) the row was read at.
type Row struct {
	SubjectKind string `json:"subjectKind"`
	SubjectID   string `json:"subjectId"`
	Property    string `json:"property"`
	Value       string `json:"value"`
	Artifact    string `json:"artifact"`
	Start       int    `json:"start"`
	End         int    `json:"end"`
	Revision    string `json:"revision"`
	// Definitions is how many Source definitions the subject resolved to. Only a
	// row with exactly one is ever editable.
	Definitions int `json:"definitions"`
}

// Edit is a requested change to one projected value.
type Edit struct {
	Op       string
	NewValue string
}

// Context is what the caller asserts about itself. WriteGrant is an input to this
// check, not an authority the check can verify.
type Context struct {
	WriteGrant bool
}

// Verdict is the outcome of Judge or Apply.
type Verdict struct {
	State       string   `json:"state"`
	Reason      string   `json:"reason,omitempty"`
	Diagnostics []string `json:"diagnostics,omitempty"`
	Candidate   []byte   `json:"-"`
}

// Acceptance runs the existing reference-shared-acceptance binary.
type Acceptance struct {
	Binary string
}

func (a Acceptance) run(args ...string) ([]byte, int, error) {
	cmd := exec.Command(a.Binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		exit, ok := err.(*exec.ExitError)
		if !ok {
			return nil, -1, err
		}
		code = exit.ExitCode()
	}
	out := stdout.Bytes()
	if code != 0 && len(out) == 0 {
		out = stderr.Bytes()
	}
	return out, code, nil
}

// Revision returns the workspace's current model revision.
func (a Acceptance) Revision(root string) (string, error) {
	out, code, err := a.run("revision", "-workspace", root)
	if err != nil || code != 0 {
		return "", fmt.Errorf("revision: code=%d err=%v out=%s", code, err, out)
	}
	var r struct {
		ModelRevision string `json:"modelRevision"`
	}
	if err := json.Unmarshal(out, &r); err != nil || r.ModelRevision == "" {
		return "", fmt.Errorf("revision output unreadable: %v: %s", err, out)
	}
	return r.ModelRevision, nil
}

// Project reads one row per attribute from the current Source. The span comes
// from the existing syntax anchors; nothing is scanned out of the text.
func (a Acceptance) Project(root string) ([]Row, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return nil, err
	}
	revision, err := a.Revision(root)
	if err != nil {
		return nil, err
	}
	count := map[string]int{}
	rows := map[string]Row{}
	var order []string
	// Several Data can live in one file; parse each file once so a definition is
	// counted once per place it is written, not once per Data in the file.
	parsedFiles := map[string]bool{}
	for _, ns := range loaded.World.Semantics {
		for _, d := range ns.Data {
			location, ok := loaded.Sources["semantic-data:"+string(d.ID)]
			if !ok || parsedFiles[location.File] {
				continue
			}
			parsedFiles[location.File] = true
			content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(location.File)))
			if err != nil {
				return nil, err
			}
			parsed, err := syntax.ParseSemanticFileAnchored(string(content))
			if err != nil {
				return nil, err
			}
			for _, anchor := range parsed.Anchors {
				if anchor.Kind != "semantic-attribute-type" {
					continue
				}
				id := anchor.ID
				count[id]++
				if _, seen := rows[id]; seen {
					continue
				}
				rows[id] = Row{
					SubjectKind: attributeSubject, SubjectID: id, Property: propertyType,
					Value:    string(content[anchor.Anchor.Start.Offset:anchor.Anchor.End.Offset]),
					Artifact: location.File, Start: anchor.Anchor.Start.Offset, End: anchor.Anchor.End.Offset,
					Revision: revision,
				}
				order = append(order, id)
			}
		}
	}
	out := make([]Row, 0, len(order))
	for _, id := range order {
		row := rows[id]
		row.Definitions = count[id]
		out = append(out, row)
	}
	return out, nil
}

// Find returns the row for a subject, or false when it has none.
func Find(rows []Row, subject string) (Row, bool) {
	for _, r := range rows {
		if r.SubjectID == subject {
			return r, true
		}
	}
	return Row{}, false
}

// Judge decides from the current Source whether the edit may be handed to
// acceptance. It writes nothing, and runs the existing preview to learn whether
// acceptance itself would refuse.
func (a Acceptance) Judge(root string, row Row, edit Edit, ctx Context) (Verdict, error) {
	if edit.Op != opSetAttrType || row.Property != propertyType || row.SubjectKind != attributeSubject {
		return Verdict{State: StateUnsupported, Reason: "no operation is defined for this subject and property"}, nil
	}
	if !ctx.WriteGrant {
		return Verdict{State: StateWithheld, Reason: "the caller holds no write grant"}, nil
	}
	current, err := a.Project(root)
	if err != nil {
		return Verdict{}, err
	}
	now, ok := Find(current, row.SubjectID)
	switch {
	case !ok:
		return Verdict{State: StateUnsupported, Reason: "the subject has no Source definition now"}, nil
	case now.Definitions != 1:
		return Verdict{State: StateAmbiguous, Reason: fmt.Sprintf("the subject has %d Source definitions", now.Definitions)}, nil
	case row.Definitions != 1:
		return Verdict{State: StateAmbiguous, Reason: "the projected row did not resolve to exactly one definition"}, nil
	case now.Revision != row.Revision:
		return Verdict{State: StateStale, Reason: "the model revision changed since the row was read"}, nil
	case now.Value != row.Value || now.Start != row.Start || now.End != row.End || now.Artifact != row.Artifact:
		return Verdict{State: StateStale, Reason: "the value or its span differs from the projected row"}, nil
	case edit.NewValue == row.Value:
		return Verdict{State: StateNoop, Reason: "the new value equals the current value"}, nil
	}
	candidate, err := json.MarshalIndent(map[string]any{
		"schema":            candidateSchema,
		"id":                "projection-edit-" + row.SubjectID,
		"baseModelRevision": row.Revision,
		"origin":            map[string]string{"kind": "studio", "actor": "projection-edit-check"},
		"intent":            "projection row edit: " + row.SubjectID + " " + row.Property + " -> " + edit.NewValue,
		"evidence":          []map[string]string{{"source": row.Artifact, "note": "the Source file the projected row was read from"}},
		"delta":             []map[string]string{{"op": opSetAttrType, "id": row.SubjectID, "attributeType": edit.NewValue}},
	}, "", "  ")
	if err != nil {
		return Verdict{}, err
	}
	dir, err := os.MkdirTemp("", "projection-edit-")
	if err != nil {
		return Verdict{}, err
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "candidate.json")
	if err := os.WriteFile(path, candidate, 0o600); err != nil {
		return Verdict{}, err
	}
	out, code, err := a.run("preview", "-workspace", root, "-candidate", path)
	if err != nil {
		return Verdict{}, err
	}
	if code != 0 {
		return Verdict{State: StateRejected, Reason: "the existing acceptance preview refused the candidate",
			Diagnostics: diagnosticCodes(out), Candidate: candidate}, nil
	}
	return Verdict{State: StateEligible, Candidate: candidate}, nil
}

// Apply judges, then lets the existing acceptance (review fingerprint, accept)
// write Source. Nothing else writes.
func (a Acceptance) Apply(root string, row Row, edit Edit, ctx Context) (Verdict, error) {
	v, err := a.Judge(root, row, edit, ctx)
	if err != nil || v.State != StateEligible {
		return v, err
	}
	dir, err := os.MkdirTemp("", "projection-edit-")
	if err != nil {
		return Verdict{}, err
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "candidate.json")
	if err := os.WriteFile(path, v.Candidate, 0o600); err != nil {
		return Verdict{}, err
	}
	review, code, err := a.run("review", "-workspace", root, "-candidate", path, "-format", "json")
	if err != nil || code != 0 {
		return Verdict{State: StateRejected, Reason: "review failed", Diagnostics: diagnosticCodes(review), Candidate: v.Candidate}, err
	}
	var projection struct {
		ReviewFingerprint string `json:"reviewFingerprint"`
	}
	if err := json.Unmarshal(review, &projection); err != nil || projection.ReviewFingerprint == "" {
		return Verdict{}, fmt.Errorf("review output has no reviewFingerprint: %v", err)
	}
	out, code, err := a.run("accept", "-workspace", root, "-candidate", path,
		"-expected-review-fingerprint", projection.ReviewFingerprint)
	if err != nil {
		return Verdict{}, err
	}
	if code != 0 {
		return Verdict{State: StateRejected, Reason: "acceptance refused", Diagnostics: diagnosticCodes(out), Candidate: v.Candidate}, nil
	}
	return Verdict{State: StateApplied, Candidate: v.Candidate}, nil
}

// diagnosticCodes pulls every "code" string out of acceptance's JSON output.
func diagnosticCodes(out []byte) []string {
	var found []string
	var walk func(v any)
	walk = func(v any) {
		switch t := v.(type) {
		case map[string]any:
			if c, ok := t["code"].(string); ok && c != "" {
				found = append(found, c)
			}
			for _, x := range t {
				walk(x)
			}
		case []any:
			for _, x := range t {
				walk(x)
			}
		}
	}
	var doc any
	if json.Unmarshal(out, &doc) == nil {
		walk(doc)
	} else if s := strings.TrimSpace(string(out)); s != "" {
		found = append(found, s)
	}
	return found
}
