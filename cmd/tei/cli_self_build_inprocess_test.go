package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/buildintent"
)

// These tests pin SB-C0 §4.22 migration step 2: the entry can derive the plan
// itself from a caller-named build catalog, and records which data it used.
//
// The child route and this route are mutually exclusive on purpose. §4.16 warned
// that two routes writing the same record make the record's meaning depend on the
// route; refusing both at once keeps one attempt to one derivation.

type selfBuildInProcessResult struct {
	OK        bool `json:"ok"`
	Qualified bool `json:"qualified"`
	// DefinitionBinding is the entry-side reading of §4.36: what the builder said
	// it read, compared with what the plan names.
	DefinitionBinding *struct {
		State          string `json:"state"`
		PlannedDigest  string `json:"plannedDigest"`
		ReportedDigest string `json:"reportedDigest"`
		Matched        bool   `json:"matched"`
		Reason         string `json:"reason"`
	} `json:"definitionBinding"`
	PlanReceipt *struct {
		State             string `json:"state"`
		DerivedBy         string `json:"derivedBy"`
		CatalogPath       string `json:"catalogPath"`
		CatalogDigest     string `json:"catalogDigest"`
		SourceRoot        string `json:"sourceRoot"`
		Authority         string `json:"authority"`
		IntentFingerprint string `json:"intentFingerprint"`
		PlanFingerprint   string `json:"planFingerprint"`
		PlanMatchesExpect bool   `json:"planMatchesExpected"`
		L2Claimed         bool   `json:"l2Claimed"`
		FailureCause      string `json:"failureCause"`
	} `json:"planReceipt"`
	PlanCrossCheck *struct {
		State string `json:"state"`
	} `json:"planCrossCheck"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

const inProcessIntentBody = `{
  "formatVersion": 1,
  "subject": {"kind": "plugin", "id": "tei.plugin.inprocess_probe", "version": "1.0.0", "definitionSource": "definition"},
  "sources": [
    {"kind": "plugin-definition", "id": "definition", "ref": "plugin.tei"},
    {"kind": "go-package", "id": "implementation", "ref": "implementation.go.txt"}
  ],
  "builder": {"contract": "tei.plugin-build", "compatibleMajor": 1},
  "recipe": {"kind": "go-package", "entrypoint": "."},
  "outputKind": "plugin-package",
  "target": "host/linux/amd64",
  "profile": "release"
}
`

const inProcessCatalogBody = `{
  "formatVersion": 1,
  "providers": [
    {
      "plugin": "tei.builder.go",
      "version": "1.2.0",
      "implementation": "go.builder",
      "contract": "tei.plugin-build",
      "contractMajor": 1,
      "authorityScope": "build-tool",
      "artifactDigest": "sha256:1111111111111111111111111111111111111111111111111111111111111111",
      "recipeKinds": ["go-package"],
      "targets": ["host/linux/amd64"],
      "profiles": ["release"]
    }
  ],
  "dependencies": [],
  "targets": [{"target": "host/linux/amd64", "profiles": ["release"]}]
}
`

type inProcessFixture struct {
	root        string
	intentPath  string
	catalogPath string
	sourceRoot  string
	catalogSha  string
}

func writeInProcessFixture(t *testing.T) inProcessFixture {
	t.Helper()
	root := t.TempDir()
	sourceRoot := filepath.Join(root, "src")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	pluginPath := filepath.Join(sourceRoot, "plugin.tei")
	if err := os.WriteFile(pluginPath, []byte(`plugin tei.plugin.inprocess_probe v1.0.0 {
    runtime builtin
    provides deployment-executor runtime.inprocess_probe {
        action application-start
    }
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "implementation.go.txt"), []byte("package impl\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	intentPath := filepath.Join(root, "intent.json")
	if err := os.WriteFile(intentPath, []byte(inProcessIntentBody), 0o600); err != nil {
		t.Fatal(err)
	}
	catalogPath := filepath.Join(root, "catalog.json")
	catalogBytes := []byte(inProcessCatalogBody)
	if err := os.WriteFile(catalogPath, catalogBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	return inProcessFixture{
		root:        root,
		intentPath:  intentPath,
		catalogPath: catalogPath,
		sourceRoot:  sourceRoot,
		catalogSha:  digestOfBytes(catalogBytes),
	}
}

// deriveForTest resolves the same declaration with the same API the entry uses,
// so a test knows the plan fingerprint without guessing it.
func (f inProcessFixture) deriveForTest(t *testing.T) buildintent.ResolvedBuildReceipt {
	t.Helper()
	catalogBytes, err := os.ReadFile(f.catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := buildintent.ParseBuildCatalog(catalogBytes)
	if err != nil {
		t.Fatal(err)
	}
	intentBytes, err := os.ReadFile(f.intentPath)
	if err != nil {
		t.Fatal(err)
	}
	intent, err := buildintent.ParseIntent(intentBytes)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := buildintent.Resolve(intent, buildintent.NewBuildContext(f.sourceRoot, catalog))
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func runInProcessShape(t *testing.T, args ...string) (selfBuildInProcessResult, int, error) {
	t.Helper()
	var out bytes.Buffer
	code, err := runSelfBuildCommand(args, &out)
	var report selfBuildInProcessResult
	if out.Len() > 0 {
		if decodeErr := json.Unmarshal(out.Bytes(), &report); decodeErr != nil {
			t.Fatalf("decode report: %v: %s", decodeErr, out.String())
		}
	}
	return report, code, err
}

// The entry derives the plan itself and records the data it used, so a reader can
// tell an entry derivation from a child receipt.
func TestSelfBuildResolvesInProcessFromANamedCatalog(t *testing.T) {
	fixture := writeInProcessFixture(t)
	child := buildSelfBuildChild(t, fixture.root)
	output := filepath.Join(fixture.root, "artifact")
	derived := fixture.deriveForTest(t)

	report, code, err := runInProcessShape(t,
		"-child", child,
		"-workspace", outerWorkspace(t, fixture.root),
		"-intent", "tei.plugin.inprocess_probe",
		"-intent-file", fixture.intentPath,
		"-build-catalog", fixture.catalogPath,
		"-build-catalog-digest", fixture.catalogSha,
		"-build-source-root", fixture.sourceRoot,
		"-expected-plan-fingerprint", derived.PlanFingerprint,
		"-state", filepath.Join(fixture.root, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output, "-digest", digestOfBytes([]byte("artifact-bytes"))),
	)
	if code != 0 {
		t.Fatalf("code=%d error=%+v runErr=%v", code, report.Error, err)
	}
	receipt := report.PlanReceipt
	if receipt == nil {
		t.Fatal("no plan receipt")
	}
	if receipt.State != intentResolutionResolved {
		t.Fatalf("state=%q cause=%q", receipt.State, receipt.FailureCause)
	}
	if receipt.DerivedBy != derivedByEntryInProcess {
		t.Fatalf("derivedBy=%q, want %q", receipt.DerivedBy, derivedByEntryInProcess)
	}
	if receipt.PlanFingerprint != derived.PlanFingerprint {
		t.Fatalf("planFingerprint=%q, want %q", receipt.PlanFingerprint, derived.PlanFingerprint)
	}
	if receipt.CatalogDigest != fixture.catalogSha || receipt.CatalogPath == "" || receipt.SourceRoot == "" {
		t.Fatalf("the inputs are not recorded: %+v", receipt)
	}
	if receipt.Authority != derivedCacheAuthority {
		t.Fatalf("authority=%q", receipt.Authority)
	}
	if receipt.L2Claimed {
		t.Fatal("in-process derivation claimed L2")
	}
	if report.Qualified {
		t.Fatal("in-process derivation reported the attempt as qualified")
	}
	// The fixture builder reports an artifact digest but no plan fingerprint, so
	// the cross-check has nothing to compare against and says so. That is the
	// honest outcome: an entry-derived plan does not make a builder's silence
	// into agreement.
	if report.PlanCrossCheck == nil || report.PlanCrossCheck.State != planCrossCheckUnperformed {
		t.Fatalf("planCrossCheck=%+v", report.PlanCrossCheck)
	}
}

// A catalog named without its digest cannot be recorded, so it is refused before
// anything runs (SC0-25).
func TestSelfBuildRefusesACatalogWithoutItsDigest(t *testing.T) {
	fixture := writeInProcessFixture(t)
	child := buildSelfBuildChild(t, fixture.root)
	output := filepath.Join(fixture.root, "artifact")

	_, code, err := runInProcessShape(t,
		"-child", child,
		"-workspace", outerWorkspace(t, fixture.root),
		"-intent", "tei.plugin.inprocess_probe",
		"-intent-file", fixture.intentPath,
		"-build-catalog", fixture.catalogPath,
		"-state", filepath.Join(fixture.root, "state"),
		"-output", output,
	)
	if code != 2 {
		t.Fatalf("a catalog without a digest was accepted: code=%d", code)
	}
	if err == nil || !strings.Contains(err.Error(), "must be given together") {
		t.Fatalf("err=%v", err)
	}
	if _, err := os.Stat(output); err == nil {
		t.Fatal("the builder was started despite the refusal")
	}
}

// A digest that does not describe the file means the record would name data the
// run did not use.
func TestSelfBuildRefusesACatalogWhoseDigestDoesNotMatch(t *testing.T) {
	fixture := writeInProcessFixture(t)
	child := buildSelfBuildChild(t, fixture.root)
	output := filepath.Join(fixture.root, "artifact")

	report, code, err := runInProcessShape(t,
		"-child", child,
		"-workspace", outerWorkspace(t, fixture.root),
		"-intent", "tei.plugin.inprocess_probe",
		"-intent-file", fixture.intentPath,
		"-build-catalog", fixture.catalogPath,
		"-build-catalog-digest", "sha256:"+strings.Repeat("0", 64),
		"-build-source-root", fixture.sourceRoot,
		"-state", filepath.Join(fixture.root, "state"),
		"-output", output,
	)
	if code != 2 {
		t.Fatalf("a wrong catalog digest was accepted: code=%d", code)
	}
	if report.Error == nil || !strings.Contains(report.Error.Message, "build catalog digest is") {
		t.Fatalf("error=%+v runErr=%v", report.Error, err)
	}
}

// One attempt gets one derivation. Asking for both routes is refused rather than
// resolved by precedence.
func TestSelfBuildRefusesBothDerivationRoutesAtOnce(t *testing.T) {
	fixture := writeInProcessFixture(t)
	child := buildSelfBuildChild(t, fixture.root)

	_, code, err := runInProcessShape(t,
		"-child", child,
		"-workspace", outerWorkspace(t, fixture.root),
		"-intent", "tei.plugin.inprocess_probe",
		"-intent-file", fixture.intentPath,
		"-intent-resolver", "/nonexistent/resolver",
		"-build-catalog", fixture.catalogPath,
		"-build-catalog-digest", fixture.catalogSha,
		"-build-source-root", fixture.sourceRoot,
		"-state", filepath.Join(fixture.root, "state"),
		"-output", filepath.Join(fixture.root, "artifact"),
	)
	if code != 2 {
		t.Fatalf("both routes were accepted: code=%d", code)
	}
	if err == nil || !strings.Contains(err.Error(), "cannot both be given") {
		t.Fatalf("err=%v", err)
	}
}
