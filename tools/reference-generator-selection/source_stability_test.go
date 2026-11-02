package generatorselection

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManifest(t *testing.T, dir, id, source string, sourceBytes []byte) {
	t.Helper()
	manifest := CandidateManifest{
		Schema: CandidateSchema, ID: id, Origin: "handwritten", Language: LanguagePython,
		Source: source, SourceDigest: digest(sourceBytes),
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "candidate.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCandidateSourceMustBeInsideCandidateDirectory(t *testing.T) {
	workspaceDir, contract, fixtures, _ := paths(t)
	parent := t.TempDir()
	external := []byte("print('external')\n")
	if err := os.WriteFile(filepath.Join(parent, "external.py"), external, 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{name: "parent traversal", source: "../external.py", want: "escapes the candidate directory"},
		{name: "absolute path", source: filepath.Join(parent, "external.py"), want: "is absolute"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			candidateDir := filepath.Join(t.TempDir(), "candidate")
			if err := os.MkdirAll(candidateDir, 0o755); err != nil {
				t.Fatal(err)
			}
			writeManifest(t, candidateDir, "outside", tc.source, external)
			receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, filepath.Dir(candidateDir))
			if err != nil {
				t.Fatal(err)
			}
			if len(receipt.Candidates) != 1 {
				t.Fatalf("candidates=%+v", receipt.Candidates)
			}
			candidate := receipt.Candidates[0]
			if candidate.Passed || !strings.Contains(candidate.RejectReason, tc.want) {
				t.Fatalf("candidate=%+v want %q", candidate, tc.want)
			}
		})
	}

	t.Run("symlink escape", func(t *testing.T) {
		candidateDir := filepath.Join(t.TempDir(), "candidate")
		if err := os.MkdirAll(candidateDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(parent, "external.py"), filepath.Join(candidateDir, "alias.py")); err != nil {
			t.Skipf("symlink is unavailable: %v", err)
		}
		writeManifest(t, candidateDir, "alias", "alias.py", external)
		receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, filepath.Dir(candidateDir))
		if err != nil {
			t.Fatal(err)
		}
		candidate := receipt.Candidates[0]
		if candidate.Passed || !strings.Contains(candidate.RejectReason, "resolves outside the candidate directory") {
			t.Fatalf("candidate=%+v", candidate)
		}
	})
}

// A script that changes its own bytes between cases must not be qualified:
// the case outputs from two different source versions are not one generator's
// result.
func TestStaticScriptSourceDigestIsStablePerCase(t *testing.T) {
	if _, runtime := ResolveScriptRuntime(LanguagePython); !runtime.Available {
		t.Skip("python3 is unavailable")
	}
	workspaceDir, contract, fixtures, _ := paths(t)
	out := filepath.Join(t.TempDir(), "candidate")
	if _, err := Generate(contract, "python-template", out); err != nil {
		t.Fatal(err)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, filepath.Dir(out))
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Candidates) != 1 {
		t.Fatalf("candidates=%+v", receipt.Candidates)
	}
	candidate := receipt.Candidates[0]
	if !candidate.Passed {
		t.Fatalf("static Python candidate did not pass: %+v", candidate)
	}
	for _, result := range candidate.Results {
		if result.Execution == nil || result.Execution.SourceDigest != candidate.SourceDigest {
			t.Fatalf("case %s did not run the declared source digest: %+v", result.CaseID, result.Execution)
		}
	}
}

func TestJavaScriptSourceChangeBetweenCasesIsRejected(t *testing.T) {
	if _, runtime := ResolveScriptRuntime(LanguageJavaScript); !runtime.Available {
		t.Skip("node is unavailable")
	}
	workspaceDir, contract, fixtures, _ := paths(t)
	script := []byte(`const fs = require('fs');
let body = '';
process.stdin.on('data', (chunk) => { body += chunk; });
process.stdin.on('end', () => {
  const input = JSON.parse(body);
  const amount = (Number(input.quantity) * Number(input.unitPrice)).toFixed(2);
  console.log(JSON.stringify({ state: 'present', amount, currency: input.currency }));
  fs.appendFileSync(__filename, '\n// changed during execution\n');
});
`)
	candidatesDir := t.TempDir()
	candidateDir := filepath.Join(candidatesDir, "self-mutating-js")
	if err := os.MkdirAll(candidateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(candidateDir, "main.js"), script, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := CandidateManifest{
		Schema: CandidateSchema, ID: "self-mutating-js", Origin: "handwritten", Language: LanguageJavaScript,
		Source: "main.js", SourceDigest: digest(script),
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(candidateDir, "candidate.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, candidatesDir)
	if err != nil {
		t.Fatal(err)
	}
	candidate := receipt.Candidates[0]
	if candidate.Passed || !strings.Contains(candidate.RejectReason, "source changed during execution") {
		t.Fatalf("candidate=%+v", candidate)
	}
	if len(candidate.Results) == 0 || !candidate.Results[0].Passed {
		t.Fatalf("the first JavaScript case did not run against the original source: %+v", candidate.Results)
	}
	if candidate.Results[0].Execution == nil || candidate.Results[0].Execution.SourceDigest != candidate.SourceDigest {
		t.Fatalf("the JavaScript case did not record the executed source digest: %+v", candidate.Results[0])
	}
}

func TestScriptSourceChangeBetweenCasesIsRejected(t *testing.T) {
	if _, runtime := ResolveScriptRuntime(LanguagePython); !runtime.Available {
		t.Skip("python3 is unavailable")
	}
	workspaceDir, contract, fixtures, _ := paths(t)
	script := []byte(`import json, sys
from decimal import Decimal, ROUND_HALF_UP
data = json.load(sys.stdin)
amount = (Decimal(data["quantity"]) * Decimal(data["unitPrice"])).quantize(Decimal("0.01"), rounding=ROUND_HALF_UP)
print(json.dumps({"state": "present", "amount": str(amount), "currency": data["currency"]}))
with open(__file__, "a", encoding="utf-8") as f:
    f.write("\n# changed during execution\n")
`)
	candidatesDir := t.TempDir()
	candidateDir := filepath.Join(candidatesDir, "self-mutating")
	if err := os.MkdirAll(candidateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(candidateDir, "main.py"), script, 0o600); err != nil {
		t.Fatal(err)
	}
	writeManifest(t, candidateDir, "self-mutating", "main.py", script)

	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, candidatesDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Candidates) != 1 {
		t.Fatalf("candidates=%+v", receipt.Candidates)
	}
	candidate := receipt.Candidates[0]
	if candidate.Passed || !strings.Contains(candidate.RejectReason, "source changed during execution") {
		t.Fatalf("candidate=%+v", candidate)
	}
	if len(candidate.Results) == 0 || !candidate.Results[0].Passed {
		t.Fatalf("the first case did not run against the original source: %+v", candidate.Results)
	}
	if candidate.Results[0].Execution == nil || candidate.Results[0].Execution.SourceDigest != candidate.SourceDigest {
		t.Fatalf("the case did not record the executed source digest: %+v", candidate.Results[0])
	}
}
