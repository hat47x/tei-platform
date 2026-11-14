package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

const (
	workbench     = "../../reference/tei-studio-workbench"
	transformFile = workbench + "/builder-sources/inspection-reader-v1.transform.json"
	contractFile  = workbench + "/inspection-reader.contract.json"
	committedPart = workbench + "/inspection-reader.generated.ts"
)

func read(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// A Windows checkout may materialize CRLF; the Source of record is LF.
	return bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
}

// edit returns the contract with one string replaced, failing if the original
// text is absent so a renamed field cannot turn a negative test into a no-op.
func edit(t *testing.T, contract []byte, old, replacement string) []byte {
	t.Helper()
	if !bytes.Contains(contract, []byte(old)) {
		t.Fatalf("contract does not contain %q", old)
	}
	return bytes.Replace(contract, []byte(old), []byte(replacement), 1)
}

// The committed part is a derived file. It must be exactly what the Generator
// Source produces from the contract Source that is on disk now; otherwise the
// Workbench would be built from a stale or hand-edited part.
func TestCommittedPartIsTheCurrentGeneration(t *testing.T) {
	artifact, result, err := generate(read(t, transformFile), read(t, contractFile))
	if err != nil {
		t.Fatal(err)
	}
	if committed := read(t, committedPart); !bytes.Equal(artifact, committed) {
		t.Fatalf("%s is not the current generation (generated %s, committed %s); regenerate it",
			committedPart, result.ArtifactDigest, digest(committed))
	}
	if !bytes.Contains(artifact, []byte(`contractDigest: "`+result.ContractCanonicalDigest+`"`)) {
		t.Fatalf("part does not carry the independently computed contract digest %s", result.ContractCanonicalDigest)
	}
}

func TestGenerationIsRepeatable(t *testing.T) {
	first, firstReport, err := generate(read(t, transformFile), read(t, contractFile))
	if err != nil {
		t.Fatal(err)
	}
	second, secondReport, err := generate(read(t, transformFile), read(t, contractFile))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) || firstReport != secondReport {
		t.Fatal("two generations from the same Sources differ")
	}
}

// A Source revision must change both the part and the identity it carries, so
// a part built from the previous revision cannot pass as the current one.
func TestSourceRevisionChangesPartAndIdentity(t *testing.T) {
	transformRaw, contract := read(t, transformFile), read(t, contractFile)
	current, currentReport, err := generate(transformRaw, contract)
	if err != nil {
		t.Fatal(err)
	}
	for name, revised := range map[string][]byte{
		"version":       edit(t, contract, `"version": "1.0.0"`, `"version": "1.0.1"`),
		"failure code":  edit(t, contract, `"STALE_MODEL_PROJECTION"`, `"STALE_PROJECTION"`),
		"response name": edit(t, contract, `"canonical": "canonical"`, `"canonical": "projection"`),
	} {
		next, nextReport, err := generate(transformRaw, revised)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if bytes.Equal(current, next) {
			t.Fatalf("%s: revised contract produced the same part", name)
		}
		if nextReport.ContractCanonicalDigest == currentReport.ContractCanonicalDigest ||
			nextReport.ArtifactDigest == currentReport.ArtifactDigest {
			t.Fatalf("%s: revised contract kept an old identity", name)
		}
		if bytes.Contains(next, []byte(currentReport.ContractCanonicalDigest)) {
			t.Fatalf("%s: revised part still names the previous contract digest", name)
		}
		if nextReport.TransformSourceFingerprint != currentReport.TransformSourceFingerprint {
			t.Fatalf("%s: Generator Source identity changed with the contract", name)
		}
	}
	// Formatting is not a revision: the carried identity is over the canonical form.
	var compact bytes.Buffer
	if err := json.Compact(&compact, contract); err != nil {
		t.Fatal(err)
	}
	same, sameReport, err := generate(transformRaw, compact.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(current, same) || sameReport.ContractBytesDigest == currentReport.ContractBytesDigest {
		t.Fatal("reformatted contract must give the same part and a different bytes digest")
	}
}

func TestRejectedSources(t *testing.T) {
	transformRaw, contract := read(t, transformFile), read(t, contractFile)
	cases := map[string]struct {
		transform, contract []byte
		want                string
	}{
		"another part id": {transformRaw,
			edit(t, contract, `"tei.studio.inspection_reader"`, `"tei.studio.catalog_reader"`), "assert-equal"},
		"another format": {transformRaw, edit(t, contract, `"formatVersion": 1`, `"formatVersion": 2`), "assert-equal"},
		"missing field": {transformRaw,
			edit(t, contract, `"sourceStatus": "sourceStatus"`, `"status": "sourceStatus"`), "sourceStatus"},
		"string leaves its literal": {transformRaw,
			edit(t, contract, `"read-only"`, `"read-only\" || true || \""`), "code-safe"},
		"blank value":         {transformRaw, edit(t, contract, `"unknown"`, `""`), "code-safe"},
		"array in contract":   {transformRaw, edit(t, contract, `"unknown"`, `["unknown"]`), "unsupported kind"},
		"contract not object": {transformRaw, []byte(`["tei.studio.inspection_reader"]`), "one JSON object"},
		"duplicate key":       {transformRaw, edit(t, contract, `"formatVersion": 1,`, `"formatVersion": 1, "formatVersion": 1,`), "contract Source JSON"},
		"generator with another input": {
			edit(t, transformRaw, `"name": "contract",`, `"name": "other",`), contract, "not declared"},
		"generator is not a Source": {[]byte(`{}`), contract, "transform"},
	}
	for name, c := range cases {
		artifact, _, err := generate(c.transform, c.contract)
		if err == nil {
			t.Fatalf("%s: accepted", name)
		}
		if artifact != nil {
			t.Fatalf("%s: returned a part with an error", name)
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%s: error %q does not name %q", name, err, c.want)
		}
	}
}

func TestChildWritesOnceAndReportsWhatItWrote(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "part.ts")
	args := []string{"-transform", transformFile, "-contract", contractFile, "-artifact-out", out, "-state", dir}
	var stdout bytes.Buffer
	if err := run(args, &stdout); err != nil {
		t.Fatal(err)
	}
	var got report
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != reportSchema || got.ArtifactDigest != digest(written) || got.StateDir != dir {
		t.Fatalf("report does not describe the written artifact: %+v", got)
	}
	// A second run must not replace the file: a stale part left in place would
	// otherwise be indistinguishable from this run's output.
	if err := run(args, &bytes.Buffer{}); err == nil {
		t.Fatal("existing artifact was overwritten")
	}
	// A rejected Source writes nothing.
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{"formatVersion": 2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing.ts")
	if err := run([]string{"-transform", transformFile, "-contract", bad, "-artifact-out", missing}, &bytes.Buffer{}); err == nil {
		t.Fatal("rejected contract was accepted")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("rejected contract left an artifact")
	}
}

// The contract is not a second definition of the Operation. Every response
// field it names must be a field the real InspectSubject Operation returns for
// a real Workspace, with the effect the contract expects.
func TestContractNamesWhatTheRealOperationReturns(t *testing.T) {
	var contract struct {
		Operation struct{ Effect string }
		Response  map[string]string
		Catalog   map[string]string
	}
	if err := json.Unmarshal(read(t, contractFile), &contract); err != nil {
		t.Fatal(err)
	}
	const root = "../../reference/workspace"
	wanted := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	inspection, err := operations.InspectSubject(root, wanted)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(inspection)
	if err != nil {
		t.Fatal(err)
	}
	var response map[string]json.RawMessage
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatal(err)
	}
	for role, name := range contract.Response {
		if role == "subjectKind" || role == "subjectId" {
			continue
		}
		if _, ok := response[name]; !ok {
			t.Fatalf("contract response.%s names %q, which InspectSubject does not return", role, name)
		}
	}
	var effect string
	if err := json.Unmarshal(response[contract.Response["effect"]], &effect); err != nil || effect != contract.Operation.Effect {
		t.Fatalf("InspectSubject effect %q is not the contract effect %q", effect, contract.Operation.Effect)
	}
	var subject map[string]string
	if err := json.Unmarshal(response[contract.Response["subject"]], &subject); err != nil {
		t.Fatal(err)
	}
	if subject[contract.Response["subjectKind"]] != wanted.Kind || subject[contract.Response["subjectId"]] != wanted.ID {
		t.Fatalf("contract subject fields do not read the inspected Subject: %v", subject)
	}
	catalog, err := operations.CatalogWorkspace(root, "")
	if err != nil {
		t.Fatal(err)
	}
	raw, err = json.Marshal(catalog)
	if err != nil {
		t.Fatal(err)
	}
	var catalogFields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &catalogFields); err != nil {
		t.Fatal(err)
	}
	for role, name := range contract.Catalog {
		// The reader compares these two; they must be the same observation.
		if string(catalogFields[name]) != string(response[contract.Response[role]]) {
			t.Fatalf("catalog.%s (%s) and the inspection disagree for one unchanged Workspace", role, name)
		}
	}
}
