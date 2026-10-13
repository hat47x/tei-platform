package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/xeadsource"
)

func accountingOracleFixture(t *testing.T) ([]byte, []byte, commandOutput) {
	t.Helper()
	source := []byte(`<System FormatVersion="1.2">
  <Table ID="private-customer-table"/>
  <Function ID="private-function">
    <IOTable ID="private-io" TableID="private-customer-table" OpR="+"/>
  </Function>
  <Task ID="private-task"><TaskAction ID="">
    <TaskFunctionIO FunctionID="private-function" IOID="0"/>
  </TaskAction></Task>
  <Task ID="private-empty-task"/>
</System>`)
	contract := commandContractFixture()
	output, err := buildCommandOutput(xeadsource.FormatXead, source, contract, false)
	if err != nil {
		t.Fatal(err)
	}
	return source, contract, output
}

func TestPublicAccountingOracleRequiresExactSourceAndCompleteAggregate(t *testing.T) {
	source, contract, original := accountingOracleFixture(t)
	expected, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyPublicAccountingBaseline(original.Accounting, expected); err != nil {
		t.Fatal(err)
	}
	changedSource, err := buildCommandOutput(xeadsource.FormatXead, append(append([]byte(nil), source...), ' '), contract, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyPublicAccountingBaseline(changedSource.Accounting, expected); err == nil {
		t.Fatal("source change with equal counts was accepted")
	}
	changedContract, err := buildCommandOutput(xeadsource.FormatXead, source, append(append([]byte(nil), contract...), ' '), false)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyPublicAccountingBaseline(changedContract.Accounting, expected); err == nil {
		t.Fatal("contract change with equal counts was accepted")
	}
}

func TestPublicAccountingOracleRejectsFalseAndAmbiguousBaselines(t *testing.T) {
	_, _, original := accountingOracleFixture(t)
	encode := func(v commandOutput) []byte {
		t.Helper()
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	copyOutput := func() commandOutput {
		t.Helper()
		var copied commandOutput
		if err := json.Unmarshal(encode(original), &copied); err != nil {
			t.Fatal(err)
		}
		return copied
	}
	falseCount := copyOutput()
	falseCount.Accounting.TaskCount++
	falsePolicy := copyOutput()
	falsePolicy.Accounting.DerivationPolicyRevision = "different"
	falseAuthority := copyOutput()
	falseAuthority.Accounting.AuthorityLimits = append(falseAuthority.Accounting.AuthorityLimits, "accepted-semantic-equivalence")
	localOutput := copyOutput()
	localOutput.LocalOutputNotice = "private data must not be an oracle"
	valid := encode(original)
	nestedDuplicate := strings.Replace(string(valid), `"taskCount":2`, `"taskCount":2,"TaskCount":2`, 1)
	if nestedDuplicate == string(valid) {
		t.Fatal("fixture did not contain expected task count")
	}
	topDuplicate := append(append([]byte(nil), valid[:len(valid)-1]...), []byte(`,"Accounting":{}`)...)
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"changed task count", encode(falseCount)},
		{"changed policy revision", encode(falsePolicy)},
		{"unearned authority", encode(falseAuthority)},
		{"private local output notice", encode(localOutput)},
		{"missing aggregate", []byte(`{}`)},
		{"unknown field", []byte(strings.Replace(string(valid), `"accounting":`, `"futureMeaning":true,"accounting":`, 1))},
		{"case-alias duplicate at root", topDuplicate},
		{"case-alias duplicate count", []byte(nestedDuplicate)},
		{"trailing JSON", append(append([]byte(nil), valid...), []byte(" {}")...)},
		{"broken JSON", []byte(`{"accounting":`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := verifyPublicAccountingBaseline(original.Accounting, tc.data); err == nil {
				t.Fatal("unverified or ambiguous public accounting baseline was accepted")
			}
		})
	}
}

func TestCLIExpectedAccountingChecksLocalReprojectionAndRedaction(t *testing.T) {
	source, contract, _ := accountingOracleFixture(t)
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "source.xead")
	contractPath := filepath.Join(dir, "contract.json")
	baselinePath := filepath.Join(dir, "baseline.json")
	projectionPath := filepath.Join(dir, "local-projection.json")
	for filename, content := range map[string][]byte{
		inputPath:    source,
		contractPath: contract,
	} {
		if err := os.WriteFile(filename, content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	bin := filepath.Join(dir, "reference-xead-cross-view")
	if output, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("building CLI failed: %v\n%s", err, output)
	}
	run := func(flags ...string) ([]byte, error) {
		t.Helper()
		args := []string{"-input", inputPath, "-contract", contractPath}
		return exec.Command(bin, append(args, flags...)...).CombinedOutput()
	}
	baseline, err := run()
	if err != nil {
		t.Fatalf("capture public baseline: %v\n%s", err, baseline)
	}
	if err := os.WriteFile(baselinePath, baseline, 0o600); err != nil {
		t.Fatal(err)
	}
	local, err := run("-local-projection")
	if err != nil {
		t.Fatalf("capture local projection: %v\n%s", err, local)
	}
	if err := os.WriteFile(projectionPath, local, 0o600); err != nil {
		t.Fatal(err)
	}
	verified, err := run("-verify-projection", projectionPath, "-expect-accounting", baselinePath)
	if err != nil {
		t.Fatalf("matching source and independent public baseline rejected: %v\n%s", err, verified)
	}
	for _, private := range []string{"private-task", "private-function", "private-customer-table", "sourceTokens", "localProjection"} {
		if strings.Contains(string(verified), private) {
			t.Fatalf("verified public output disclosed %q", private)
		}
	}
	var saved commandOutput
	if err := json.Unmarshal(baseline, &saved); err != nil {
		t.Fatal(err)
	}
	saved.Accounting.TaskFunctionIOCount++
	tampered, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(baselinePath, tampered, 0o600); err != nil {
		t.Fatal(err)
	}
	if output, err := run("-verify-projection", projectionPath, "-expect-accounting", baselinePath); err == nil {
		t.Fatalf("altered public conformance baseline was accepted: %s", output)
	}
	if err := os.WriteFile(baselinePath, baseline, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputPath, append(source, ' '), 0o600); err != nil {
		t.Fatal(err)
	}
	if output, err := run("-expect-accounting", baselinePath); err == nil {
		t.Fatalf("same-count altered source passed the source-bound oracle: %s", output)
	}
}
