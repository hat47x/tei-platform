package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"reflect"

	"tei/internal/database"
	"tei/internal/knowledgecontext"
	"tei/internal/subject"
	"tei/internal/workspace"
)

// ISSUE-000076 REF-01B。一件の変更要求（BP-02で order_entry.customer_reference を read-write にしたい）
// について、既存機構だけで因果列を再現できることを固定する。
// 新しいIndex / Knowledge store / CLI は作らない。

const (
	ref01Pattern      = "order_entry.customer_reference"
	ref01Dependency   = "master_data.customer/find_by_id"
	ref01Conflict     = "BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT"
	ref01PatternPath  = "applications/order-entry/patterns/customer-reference.tei"
	ref01ContractPath = "databases/master-data/contracts/customer.tei"
)

func ref01Paths() (workspacePath, fixturePath, probePath string) {
	root := repoPath()
	return filepath.Join(root, "reference", "workspace"),
		filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"),
		filepath.Join(root, "reference", "business-system-design", "order-entry-customer-edit-conflict.json")
}

func ref01TreeDigest(t *testing.T, root string) string {
	t.Helper()
	var files []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	h := sha256.New()
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		rel, _ := filepath.Rel(root, f)
		h.Write([]byte(filepath.ToSlash(rel)))
		h.Write([]byte{0})
		h.Write(content)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func ref01BlockingCodes(r report) []string {
	var out []string
	for _, d := range r.Diagnostics {
		if d.Severity == severitySemanticError || d.Severity == severityArchitecturePolicy {
			out = append(out, d.Code)
		}
	}
	return out
}

func ref01HasEvidence(items []knowledgecontext.QualificationEvidence, source string) bool {
	for _, e := range items {
		if e.Source == source {
			return true
		}
	}
	return false
}

// Step 1 と Step 5。exact subject から Accepted context へ入り、元 source を書き換えない。
func TestBrownfieldREF01EntersAcceptedContextFromExactSubjectReadOnly(t *testing.T) {
	workspacePath, _, _ := ref01Paths()
	before := ref01TreeDigest(t, workspacePath)

	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	sources, err := workspace.BuildSourceIndex(workspacePath, loaded)
	if err != nil {
		t.Fatal(err)
	}
	target := subject.Subject{Kind: "application-pattern", ID: ref01Pattern}
	ctx := knowledgecontext.Build(loaded.World, sources, target)
	result, err := knowledgecontext.QualifyWithSources(ctx, sources, knowledgecontext.QualificationRequirements{
		ExpectedKind: "application-pattern",
		Application:  "order_entry",
		RequiredDependencies: []knowledgecontext.DependencyRequirement{{
			Direction: "outgoing",
			Relation:  "contract-query",
			Kind:      "database-contract-query",
			ID:        ref01Dependency,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != knowledgecontext.QualificationApplicable {
		t.Fatalf("status=%s reasons=%+v", result.Status, result.Reasons)
	}
	if !ref01HasEvidence(result.Evidence, ref01PatternPath) {
		t.Fatalf("candidate Evidence did not return to the Pattern source: %+v", result.Evidence)
	}
	if !ref01HasEvidence(result.DependencyEvidence, ref01ContractPath) {
		t.Fatalf("dependency Evidence did not return to the Contract Query source: %+v", result.DependencyEvidence)
	}
	all := append(append([]knowledgecontext.QualificationEvidence{}, result.Evidence...), result.DependencyEvidence...)
	for _, e := range all {
		if strings.TrimSpace(e.SourceFingerprint) == "" {
			t.Fatalf("Evidence without a source fingerprint: %+v", e)
		}
	}
	wantLimits := []string{"only-explicit-requirements-were-evaluated", "qualification-is-not-acceptance", "qualification-is-not-write-or-runtime-authority"}
	if !reflect.DeepEqual(result.AuthorityLimits, wantLimits) {
		t.Fatalf("authority limits = %v, want %v", result.AuthorityLimits, wantLimits)
	}
	for i := 0; i < reflect.TypeOf(result).NumField(); i++ {
		name := strings.ToLower(reflect.TypeOf(result).Field(i).Name)
		for _, forbidden := range []string{"accept", "write", "authoriz", "grant", "permission", "decision"} {
			if strings.Contains(name, forbidden) && name != "authoritylimits" {
				t.Fatalf("qualification result carries an authority-bearing field %q", name)
			}
		}
	}
	joined := strings.ToLower(strings.Join(result.AuthorityLimits, " "))
	if !strings.Contains(joined, "accept") && !strings.Contains(joined, "write") {
		t.Fatalf("qualification must state that it grants no Acceptance / write authority: %+v", result.AuthorityLimits)
	}

	if after := ref01TreeDigest(t, workspacePath); after != before {
		t.Fatal("read-only investigation changed the authoritative Reference Workspace")
	}
}

// Step 2 と Step 3。既存の run 経路で blocker が一件に絞られ、因果が再構築でき、無関係な Authority を作らない。
func TestBrownfieldREF01ReachesExactlyTheAccessConflictWithoutInventingAuthority(t *testing.T) {
	workspacePath, fixturePath, probePath := ref01Paths()
	before := ref01TreeDigest(t, workspacePath)

	result, err := run(workspacePath, fixturePath, probePath)
	if err != nil {
		t.Fatal(err)
	}
	if got := ref01BlockingCodes(result); len(got) != 1 || got[0] != ref01Conflict {
		t.Fatalf("blocking diagnostics = %v, want exactly [%s]", got, ref01Conflict)
	}
	var conflict diagnostic
	for _, d := range result.Diagnostics {
		if d.Code == ref01Conflict {
			conflict = d
		}
	}
	if conflict.Step != "BP-02" {
		t.Fatalf("conflict step = %q, want BP-02", conflict.Step)
	}
	for _, want := range []string{ref01Pattern, "master", "read-write", "read"} {
		if !strings.Contains(conflict.Message, want) {
			t.Fatalf("conflict message does not carry %q: %s", want, conflict.Message)
		}
	}

	var row *patternMeaningRow
	for i := range result.BusinessActionPatternMeaningRows {
		r := &result.BusinessActionPatternMeaningRows[i]
		if r.StepID == "BP-02" && r.Pattern == ref01Pattern {
			row = r
		}
	}
	if row == nil {
		t.Fatalf("no derived Pattern row for BP-02 / %s: %+v", ref01Pattern, result.BusinessActionPatternMeaningRows)
	}
	if row.Datasource != "master" || row.Relation != "sales.order.customer" {
		t.Fatalf("causal chain Pattern -> datasource -> relation was not reconstructed: %+v", *row)
	}
	if row.Capability != "" {
		t.Fatalf("a write Capability appeared for the read-only reference Pattern: %+v", *row)
	}

	for _, d := range result.Diagnostics {
		if strings.Contains(strings.ToLower(d.Code+" "+d.Message), "authorization") {
			t.Fatalf("an unrelated Authorization concern entered the causal chain: %+v", d)
		}
	}
	for _, r := range result.ExplicitBusinessMeaningRows {
		if r.StepID == "BP-02" && len(r.Capabilities) != 0 {
			t.Fatalf("BP-02 gained a synthetic Capability: %+v", r.Capabilities)
		}
	}

	if after := ref01TreeDigest(t, workspacePath); after != before {
		t.Fatal("running the probe changed the authoritative Reference Workspace")
	}
}

// Step 4。scratch で master を read-write にした場合だけ同じ diagnostic が消える。
// diagnostic が文言への hard-code ではなく、Accepted Application contract との差から導出されることを確かめる。
func TestBrownfieldREF01ConflictClearsOnlyWhenScratchAcceptedAccessChanges(t *testing.T) {
	workspacePath, fixturePath, probePath := ref01Paths()
	originalDigest := ref01TreeDigest(t, workspacePath)

	scratch := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(scratch, 0o755); err != nil {
		t.Fatal(err)
	}
	copyDirectory(t, workspacePath, scratch)

	appPath := filepath.Join(scratch, "applications", "order-entry", "application.tei")
	content, err := os.ReadFile(appPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	start := strings.Index(text, "database master")
	if start < 0 {
		t.Fatal("scratch Application has no master datasource requirement")
	}
	head, tail := text[:start], text[start:]
	if !strings.Contains(tail, "access read") {
		t.Fatalf("master access is not read in the Accepted source: %s", tail)
	}
	tail = strings.Replace(tail, "access read", "access read-write", 1)
	if err := os.WriteFile(appPath, []byte(head+tail), 0o600); err != nil {
		t.Fatal(err)
	}

	accepted, err := run(workspacePath, fixturePath, probePath)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := run(scratch, fixturePath, probePath)
	if err != nil {
		t.Fatal(err)
	}
	if got := ref01BlockingCodes(accepted); len(got) != 1 || got[0] != ref01Conflict {
		t.Fatalf("Accepted workspace should show the conflict, got %v", got)
	}
	if got := ref01BlockingCodes(changed); len(got) != 0 {
		t.Fatalf("conflict should clear when the owning Application access matches the request, got %v", got)
	}
	for _, r := range changed.ExplicitBusinessMeaningRows {
		if r.StepID == "BP-02" && len(r.Capabilities) != 0 {
			t.Fatalf("scratch access change generated a Capability: %+v", r.Capabilities)
		}
	}
	if ref01TreeDigest(t, workspacePath) != originalDigest {
		t.Fatal("the scratch experiment changed the authoritative Reference Workspace")
	}
}

// 強化: scratchでmaster accessを変えても、World上で変わるのはそのaccess一点だけで、
// Capability / Binding / Authorization関連を含む他のどのCanonical要素も増減しない。
func TestBrownfieldREF01ScratchAccessChangeIsTheOnlyCanonicalDifference(t *testing.T) {
	workspacePath, _, _ := ref01Paths()
	scratch := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(scratch, 0o755); err != nil {
		t.Fatal(err)
	}
	copyDirectory(t, workspacePath, scratch)
	appPath := filepath.Join(scratch, "applications", "order-entry", "application.tei")
	content, err := os.ReadFile(appPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	start := strings.Index(text, "database master")
	if start < 0 {
		t.Fatal("no master datasource requirement")
	}
	changed := text[:start] + strings.Replace(text[start:], "access read", "access read-write", 1)
	if changed == text {
		t.Fatal("scratch edit did not change the Application source")
	}
	if err := os.WriteFile(appPath, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}

	accepted, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	experiment, err := workspace.Load(scratch)
	if err != nil {
		t.Fatal(err)
	}

	diffs := 0
	for i := range experiment.World.Applications {
		app := &experiment.World.Applications[i]
		for j := range app.Databases {
			if string(app.Databases[j].Alias) == "master" && app.Databases[j].Access == database.AccessReadWrite {
				app.Databases[j].Access = database.AccessRead
				diffs++
			}
		}
	}
	if diffs != 1 {
		t.Fatalf("expected exactly one master requirement to differ, found %d", diffs)
	}
	if !reflect.DeepEqual(accepted.World, experiment.World) {
		t.Fatal("the scratch access change altered Canonical meaning beyond the one datasource access (Capability / Binding / Authorization / Pattern included)")
	}
}
