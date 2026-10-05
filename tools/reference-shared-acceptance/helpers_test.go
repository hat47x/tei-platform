package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source pathを取得できません")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// scratchWorkspaceは、Gate B / Gate Cが固定したreference/workspaceを変更せず、複製に対して受入境界を試すためのcopyを返す。
func scratchWorkspace(t *testing.T) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), "workspace")
	if err := copyTree(filepath.Join(repositoryRoot(t), "reference", "workspace"), dst); err != nil {
		t.Fatalf("scratch workspaceを作れません: %v", err)
	}
	return dst
}

// secondApplicationEnvironmentBindingsは、S7で受理したEnvironment bindingの置き場である。
var secondApplicationEnvironmentBindings = []string{
	filepath.Join("environments", "development", "environment.tei"),
	filepath.Join("environments", "production", "environment.tei"),
}

// removeSecondApplicationBindingsは、`repair_intake`のEnvironment bindingを取り除く。
// Applicationを消してbindingだけ残すと、存在しないApplicationへのbindingという別の不整合になる。
func removeSecondApplicationBindings(t *testing.T, dir string) {
	t.Helper()
	for _, rel := range secondApplicationEnvironmentBindings {
		path := filepath.Join(dir, rel)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var kept []string
		removed := 0
		for _, line := range strings.Split(string(content), "\n") {
			if strings.Contains(line, "bind repair_intake.customer_master") {
				removed++
				continue
			}
			kept = append(kept, line)
		}
		if removed != 1 {
			t.Fatalf("%sのbindingが1件ではありません: %d件", rel, removed)
		}
		if err := os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// scratchWorkspaceBeforeEnvironmentBindingは、S7 Environment bindingを受理する前の状態のcopyを返す。
// `repair_intake` Application自体はS5で受理済みなので残る。
func scratchWorkspaceBeforeEnvironmentBinding(t *testing.T) string {
	t.Helper()
	dir := scratchWorkspace(t)
	removeSecondApplicationBindings(t, dir)
	return dir
}

// scratchWorkspaceBeforeSecondApplicationは、第二Application `repair_intake` を受理する前の状態の
// copyを返す。受理後のReferenceにはそのsourceと受理の記録、S7のEnvironment bindingが含まれるため、
// 「第二Applicationをこれから受け入れる」ことや「まだ記録が無い」ことを前提にするtestはこちらを使う。
func scratchWorkspaceBeforeSecondApplication(t *testing.T) string {
	t.Helper()
	dir := scratchWorkspace(t)
	removeSecondApplicationBindings(t, dir)
	for _, rel := range []string{
		filepath.Join("semantic", "repair-intake"),
		filepath.Join("applications", "repair-intake"),
		filepath.Join(".tei", "decisions"),
	} {
		if err := os.RemoveAll(filepath.Join(dir, rel)); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func revisionOfDir(t *testing.T, dir string) string {
	t.Helper()
	s, err := takeSnapshot(dir)
	if err != nil {
		t.Fatalf("workspaceを読み込めません: %v", err)
	}
	return s.revision
}

func reviewFingerprintViaCLI(t *testing.T, dir, candidatePath string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"review", "-workspace", dir, "-candidate", candidatePath, "-format", "json"},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("review code=%d err=%s out=%s", code, stderr.String(), stdout.String())
	}
	var review ReviewProjection
	if err := json.Unmarshal(stdout.Bytes(), &review); err != nil {
		t.Fatalf("review JSONを読めません: %v: %s", err, stdout.String())
	}
	if review.ReviewFingerprint == "" {
		t.Fatal("review CLI returned no reviewFingerprint")
	}
	return review.ReviewFingerprint
}

func unitPriceCandidate(revision string) Candidate {
	return Candidate{
		Schema:            candidateSchema,
		ID:                "cand-unit-price-positive",
		BaseModelRevision: revision,
		Origin:            Origin{Kind: "cli", Actor: "test"},
		Intent:            "order lineの単価が正であることを業務ruleとして明示する（数量の既存ruleと対）",
		Evidence: []Evidence{
			{Subject: "sales.order_line.quantity_positive", Note: "同じ形のruleが数量には既にある"},
			{Source: "semantic/sales/validation.tei"},
		},
		Delta: []Operation{{
			Op: opAddValidation, Namespace: "sales", ID: "sales.order_line.unit_price_positive",
			Kind: "positive", Attribute: "sales.order_line.unit_price",
		}},
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// treeDigestはdirectory配下の全fileの内容digestを返し、sourceが一切変わっていないことの比較に使う。
func treeDigest(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		sum := sha256.Sum256(content)
		out[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func assertTreeUnchanged(t *testing.T, before, after map[string]string) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("file数が変わりました: before=%d after=%d", len(before), len(after))
	}
	for path, digest := range before {
		if after[path] != digest {
			t.Fatalf("authoritative sourceが変更されています: %s", path)
		}
	}
}

func stageStatus(t *testing.T, p Preview, name string) Stage {
	t.Helper()
	for _, s := range p.Stages {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("stage %s がありません: %+v", name, p.Stages)
	return Stage{}
}

func diagnosticCodes(p Preview) []string {
	var out []string
	for _, d := range allDiagnostics(p) {
		out = append(out, d.Code)
	}
	return out
}

func hasCode(codes []string, want string) bool {
	for _, c := range codes {
		if c == want {
			return true
		}
	}
	return false
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

type numberSource map[core.CanonicalID]*big.Rat

func (s numberSource) Number(attribute core.CanonicalID) (*big.Rat, error) {
	value, ok := s[attribute]
	if !ok {
		return nil, fmt.Errorf("attribute %s の値がありません", attribute)
	}
	return value, nil
}

func (s numberSource) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {
	return nil, fmt.Errorf("このtestでは使いません")
}

func findRule(rules []semantic.Validation, id core.CanonicalID) (semantic.Validation, bool) {
	for _, rule := range rules {
		if rule.ID == id {
			return rule, true
		}
	}
	return semantic.Validation{}, false
}
