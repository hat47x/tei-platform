package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/reacandidate"
	"tei/internal/sourceindex"
	"tei/internal/threeelement"
)

// repository-plan-check's TestSchemaVersionLiteralsHaveATest requires every
// schema identifier a non-test source writes to be named by a test. The
// three-element CLI writes several, and an earlier version named none of them,
// so a rename of any envelope would have gone unnoticed here while the
// repository check failed. This test pins the identifiers AND the result
// shapes they describe.
func TestThreeElementCLIResponseShapes(t *testing.T) {
	root := t.TempDir()
	modelPath := filepath.Join(root, "model.json")
	if err := os.WriteFile(modelPath, []byte(fixtureBytes), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name         string
		args         []string
		schema       string
		resultSchema string
	}{
		{"validate", []string{"validate", "-model", modelPath}, "tei.cli.three-element.validate/v0", "tei.three-element.model/v0"},
		{"crud", []string{"crud", "-model", modelPath}, "tei.cli.three-element.crud/v0", "tei.three-element.crud/v0"},
		{"cross-reference", []string{"cross-reference", "-model", modelPath}, "tei.cli.three-element.cross-reference/v0", "tei.three-element.cross-reference/v0"},
		{"diff", []string{"diff", "-before", modelPath, "-after", modelPath}, "tei.cli.three-element.diff/v0", "tei.three-element.diff/v0"},
		{"rea-intake", []string{"rea-intake", "-evidence", writeThreeElementEvidence(t, root)}, "tei.cli.three-element.rea-intake/v0", "tei.rea.candidate-review/v0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			code, err := runThreeElementCommand(tc.args, &out)
			if err != nil {
				t.Fatalf("command error: %v", err)
			}
			if code != 0 && tc.name != "rea-intake" {
				t.Fatalf("exit = %d: %s", code, out.String())
			}
			var envelope struct {
				SchemaVersion string          `json:"schemaVersion"`
				OK            bool            `json:"ok"`
				Result        json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal([]byte(out.String()), &envelope); err != nil {
				t.Fatalf("invalid envelope: %v\n%s", err, out.String())
			}
			if envelope.SchemaVersion != tc.schema {
				t.Fatalf("schemaVersion = %q, want %q", envelope.SchemaVersion, tc.schema)
			}
			var result struct {
				SchemaVersion string `json:"schemaVersion"`
			}
			if err := json.Unmarshal(envelope.Result, &result); err != nil {
				t.Fatalf("result is not an object: %v", err)
			}
			if tc.resultSchema != "" && result.SchemaVersion != tc.resultSchema {
				t.Fatalf("result schemaVersion = %q, want %q", result.SchemaVersion, tc.resultSchema)
			}
		})
	}
}

// The error envelope names its own schema, so a failure is machine-readable
// without reading prose.
func TestThreeElementCLIErrorEnvelopeHasItsOwnSchema(t *testing.T) {
	var out strings.Builder
	code, err := runThreeElementCommand([]string{"unknown-subcommand"}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	var envelope struct {
		SchemaVersion string `json:"schemaVersion"`
		OK            bool   `json:"ok"`
	}
	if err := json.Unmarshal([]byte(out.String()), &envelope); err != nil {
		t.Fatalf("invalid envelope: %v\n%s", err, out.String())
	}
	if envelope.SchemaVersion != "tei.cli.three-element/v0" {
		t.Fatalf("error schemaVersion = %q", envelope.SchemaVersion)
	}
	if envelope.OK {
		t.Fatal("an error envelope reported ok")
	}
}

// The intake's own section identifiers are pinned too, because a caller that
// stores a review packet branches on them.
func TestReaCandidateSectionIdentifiers(t *testing.T) {
	evidence, err := reacandidate.DecodeAttemptEvidence(readThreeElementEvidence(t))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	set, err := reacandidate.GenerateCandidates(evidence)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if set.SchemaVersion != "tei.rea.candidate-set/v0" {
		t.Fatalf("candidate set schemaVersion = %q", set.SchemaVersion)
	}
	binding, err := reacandidate.BindCandidates(emptyReaPort{}, set, nil)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	if binding.SchemaVersion != "tei.rea.candidate-binding/v0" {
		t.Fatalf("binding schemaVersion = %q", binding.SchemaVersion)
	}
	preview, err := reacandidate.PreviewAdoption(set, model())
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.SchemaVersion != "tei.rea.adoption-preview/v0" {
		t.Fatalf("adoption preview schemaVersion = %q", preview.SchemaVersion)
	}
	comparison := reacandidate.CompareCandidateSets(set, set)
	if comparison.SchemaVersion != "tei.rea.candidate-set-comparison/v0" {
		t.Fatalf("comparison schemaVersion = %q", comparison.SchemaVersion)
	}
	packet := reacandidate.BuildReviewPacket(set, &binding, &preview, &comparison)
	if packet.SchemaVersion != "tei.rea.candidate-review/v0" {
		t.Fatalf("review schemaVersion = %q", packet.SchemaVersion)
	}
	if packet.Carrier.SchemaVersion != "tei.rea.candidate-carrier-audit/v0" {
		t.Fatalf("carrier schemaVersion = %q", packet.Carrier.SchemaVersion)
	}
	if packet.ThreeElement.SchemaVersion != "tei.rea.three-element-coverage/v0" {
		t.Fatalf("coverage schemaVersion = %q", packet.ThreeElement.SchemaVersion)
	}
}

// emptyReaPort is an in-memory read-only port: no workspace, no write.
type emptyReaPort struct{}

func (emptyReaPort) Inspect(subject sourceindex.Subject) (reacandidate.SubjectOutcome, error) {
	return reacandidate.SubjectOutcome{Status: "not-found"}, nil
}

func (emptyReaPort) ModelLinks() ([]reacandidate.ModelMapLink, error) { return nil, nil }

func (emptyReaPort) SourceIndex() (reacandidate.SourceIndexView, error) {
	return reacandidate.SourceIndexView{}, nil
}

func (emptyReaPort) WorkspaceIdentity() (string, error) { return "probe", nil }

func model() threeelement.ThreeElementModel {
	return threeelement.NewThreeElementModel("probe-model", "Probe model")
}

func readThreeElementEvidence(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(writeThreeElementEvidence(t, t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func writeThreeElementEvidence(t *testing.T, root string) string {
	t.Helper()
	const digest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	document := map[string]any{
		"ok": true,
		"data": map[string]any{
			"operation":   "analyze_javascript_application",
			"evidence_id": "ev-schema-probe",
			"provider":    map[string]any{"id": "rea-javascript-application", "version": "6.8.0"},
			"subject":     map[string]any{"digest": map[string]any{"sha256": digest}},
			"normalized_result": map[string]any{
				"root_artifact_sha256": digest,
				"graph": map[string]any{
					"schema":   "JavaScriptApplicationGraph",
					"coverage": map[string]any{"status": "complete", "truncated": false},
					"nodes": []any{map[string]any{
						"node_id": "node-probe", "kind": "module",
						"observations": []any{map[string]any{
							"observation_id": "obs-probe",
							"evidence": map[string]any{
								"state": "observed", "authority": "derived",
								"artifact": map[string]any{"sha256": digest},
								"location": map[string]any{"available": true, "value": map[string]any{
									"kind": "artifact-path", "path": "probe.js",
								}},
							},
						}},
					}},
					"edges": []any{},
				},
				"semantic_graph": map[string]any{"unknowns": []any{}},
			},
		},
	}
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "evidence.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// The three-element model schema identifier is asserted through the loader,
// so a change in the const would fail here rather than at a distance.
func TestThreeElementModelSchemaIdentifier(t *testing.T) {
	model := threeelement.NewThreeElementModel("probe", "Probe")
	if model.SchemaVersion != "tei.three-element.model/v0" {
		t.Fatalf("model schemaVersion = %q", model.SchemaVersion)
	}
}
