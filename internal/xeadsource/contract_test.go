package xeadsource

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

type referenceTranscriptionContract struct {
	ConstructMappings map[string]map[string]json.RawMessage `json:"constructMappings"`
}

func TestClassificationPolicyFromContractCarriesMachineReadableAuthorityLimits(t *testing.T) {
	root := repositoryRootFromSource(t)
	contractBytes, err := os.ReadFile(filepath.Join(root, "reference", "xead-source-transcription-contract.synthetic.json"))
	if err != nil {
		t.Fatal(err)
	}
	policy, err := ClassificationPolicyFromContract(FormatXead, contractBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(policy["TableKey"].Classifications, []string{"unresolved"}) ||
		policy["TableKey"].Extension != "" || len(policy["TableKey"].AuthorityLimits) != 0 {
		t.Fatalf("synthetic contract overstates TableKey semantics: %#v", policy["TableKey"])
	}
}

func TestInspectWithContractBindsExactContractAndNormalizedPolicy(t *testing.T) {
	root := repositoryRootFromSource(t)
	contractBytes, err := os.ReadFile(filepath.Join(root, "reference", "xead-source-transcription-contract.synthetic.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := InspectWithContract(
		FormatXead,
		[]byte(`<System FormatVersion="1.2"><Table><TableKey Type="PK"><TableKeyField FieldID="F1"/></TableKey></Table></System>`),
		contractBytes,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.TranscriptionContractFingerprint == "" {
		t.Fatal("inspection has no exact contract fingerprint")
	}
	if got.ClassificationPolicyFingerprint == "" {
		t.Fatal("inspection has no normalized policy fingerprint")
	}
	var keyCandidate *Candidate
	for i := range got.Candidates {
		if got.Candidates[i].SourceName == "TableKey" {
			keyCandidate = &got.Candidates[i]
			break
		}
	}
	if keyCandidate == nil {
		t.Fatal("TableKey candidate is missing")
	}
	if !reflect.DeepEqual(keyCandidate.Classifications, []string{"unresolved"}) ||
		keyCandidate.Extension != "" || len(keyCandidate.AuthorityLimits) != 0 {
		t.Fatalf("synthetic contract overstated TableKey semantics: %#v", keyCandidate)
	}
}

func assertInventoryCoverage(
	t *testing.T,
	format Format,
	counts map[string]int,
	names map[string]string,
	contract map[string]json.RawMessage,
) {
	t.Helper()
	for inventoryName, count := range counts {
		if count <= 0 {
			continue
		}
		sourceName, ok := names[inventoryName]
		if !ok {
			t.Fatalf("%s observed inventory %q has no XML construct mapping", format, inventoryName)
		}
		if _, ok := known[format][sourceName]; !ok {
			t.Fatalf("%s observed construct %q is not in reader known set", format, sourceName)
		}
		if _, ok := contract[sourceName]; !ok {
			t.Fatalf("%s observed construct %q has no transcription classification", format, sourceName)
		}
	}
}

func repositoryRootFromSource(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestInspectWithContractRejectsUnsupportedSourceProfile(t *testing.T) {
	root := repositoryRootFromSource(t)
	contractBytes, err := os.ReadFile(filepath.Join(root, "reference", "xead-source-transcription-contract.synthetic.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{
		`<System FormatVersion="9.9"><Table/></System>`,
		`<System><Table/></System>`,
		`<OtherRoot FormatVersion="1.2"><Table/></OtherRoot>`,
	} {
		if _, err := InspectWithContract(FormatXead, []byte(source), contractBytes); err == nil {
			t.Fatalf("unsupported source profile was classified: %s", source)
		}
	}
}

func TestInspectWithContractRequiresSourceProfileDeclaration(t *testing.T) {
	contractBytes := []byte(`{
		"constructMappings": {
			"xead": {
				"System": {"classification": ["source-metadata"]}
			}
		}
	}`)
	_, err := InspectWithContract(
		FormatXead,
		[]byte(`<System FormatVersion="1.2"/>`),
		contractBytes,
	)
	if err == nil || !strings.Contains(err.Error(), "no source profile") {
		t.Fatalf("classification without source profile was accepted: %v", err)
	}
}
