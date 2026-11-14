package directexecution_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/directexecution"
)

func TestDirectWorker(t *testing.T) {
	if os.Getenv("TEI_DIRECT_TEST_WORKER") != "1" {
		return
	}
	switch os.Getenv("TEI_DIRECT_TEST_MODE") {
	case "fail":
		_ = os.WriteFile(os.Getenv("TEI_DIRECT_TEST_OUTPUT"), []byte("partial"), 0600)
		os.Exit(7)
	case "no-output":
		os.Exit(0)
	}
	if err := os.WriteFile(os.Getenv("TEI_DIRECT_TEST_OUTPUT"), []byte(`{"result":"ok"}`), 0600); err != nil {
		os.Exit(8)
	}
	os.Exit(0)
}

func workerOptions(t *testing.T, mode string) directexecution.Options {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "output.json")
	return directexecution.Options{Command: exe, Args: []string{"-test.run=^TestDirectWorker$"},
		OutputPath: output, Env: append(os.Environ(), "TEI_DIRECT_TEST_WORKER=1", "TEI_DIRECT_TEST_MODE="+mode, "TEI_DIRECT_TEST_OUTPUT="+output)}
}

func TestDirectExecutionProducesEvidence(t *testing.T) {
	opts := workerOptions(t, "write")
	sum := sha256.Sum256([]byte(`{"result":"ok"}`))
	opts.ExpectedSHA256 = hex.EncodeToString(sum[:])
	evidence, err := directexecution.Run(opts)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Schema != directexecution.EvidenceSchema || !evidence.ExecutionRequestAccepted.Accepted || !evidence.WorkerExecuted.Executed {
		t.Fatalf("missing execution observation: %+v", evidence)
	}
	if !evidence.ActualOutputVerified.Verified || evidence.ActualOutputVerified.SHA256Hex != opts.ExpectedSHA256 {
		t.Fatalf("digest not verified: %+v", evidence.ActualOutputVerified)
	}
	if evidence.ExecutionEligibility.Eligible || evidence.PlacementAuthority.Granted || evidence.ResultMetadataObtained.Obtained {
		t.Fatal("unobserved authority, eligibility, or metadata was asserted")
	}
	encoded, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	var decoded directexecution.Evidence
	if err := json.Unmarshal(encoded, &decoded); err != nil || decoded.Schema != evidence.Schema {
		t.Fatalf("JSON round trip: %v", err)
	}
}

func TestUnpinnedDigestIsNotVerified(t *testing.T) {
	e, err := directexecution.Run(workerOptions(t, "write"))
	if err != nil {
		t.Fatal(err)
	}
	if e.ActualOutputVerified.Verified || e.ActualOutputVerified.SHA256Hex == "" {
		t.Fatalf("unexpected output claim: %+v", e.ActualOutputVerified)
	}
}
