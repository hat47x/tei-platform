package orderruledesign

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const handoverDir = "reference/order-rule-design-handover"

// Receipt is the hand-over receipt of stage 4. It is not a qualification
// receipt: it fixes the commit, tree and main that the rebuild saw, and it
// is valid for that commit only.
type Receipt struct {
	Schema          string            `json:"schema"`
	CandidateDigest string            `json:"candidateDigest"`
	Fingerprint     string            `json:"candidateFingerprint"`
	MainCommit      string            `json:"mainCommit"`
	TaskHead        string            `json:"taskHead"`
	TaskTree        string            `json:"taskTree"`
	IntegrationHead string            `json:"integrationHead"`
	IntegrationTree string            `json:"integrationTree"`
	ContainsMain    bool              `json:"integrationContainsMain"`
	RebuildValidate bool              `json:"rebuildValidate"`
	RebuildAccept   bool              `json:"rebuildPreviewAcceptable"`
	RevisionAfter   string            `json:"rebuildRevisionAfter"`
	RunSh           map[string]string `json:"toolNeutralRun"`
	Qualification   string            `json:"qualification"`
}

type stage4Out struct {
	Origin, Task, Integ string
	Receipt             Receipt
	Carried             Carried
}

// ReverifyReceipt recomputes what the receipt fixed, from the repositories
// as they are now. It returns the problems; none means the receipt still
// holds for exactly this commit and this main.
func (s *Sample) ReverifyReceipt(ctx context.Context, r Receipt, origin, integ string) []string {
	var out []string
	add := func(f string, a ...any) { out = append(out, fmt.Sprintf(f, a...)) }
	git := func(dir string, args ...string) string {
		v, err := s.Tools.git(ctx, dir, append([]string{"--no-replace-objects"}, args...)...)
		if err != nil {
			add("%v", err)
		}
		return v
	}
	if main := git(origin, "rev-parse", "main"); main != r.MainCommit {
		add("main advanced: receipt %s, now %s", r.MainCommit, main)
	}
	if reps := git(integ, "for-each-ref", "refs/replace"); reps != "" {
		add("refs/replace present in the integration checkout")
	}
	if fileExists(filepath.Join(integ, ".git", "info", "grafts")) {
		add("info/grafts present in the integration checkout")
	}
	if head := git(integ, "rev-parse", "HEAD"); head != r.IntegrationHead {
		add("integration head %s differs from the receipt %s", head, r.IntegrationHead)
	}
	if tree := git(integ, "rev-parse", "HEAD^{tree}"); tree != r.IntegrationTree {
		add("integration tree %s differs from the receipt %s", tree, r.IntegrationTree)
	}
	if _, err := s.Tools.git(ctx, integ, "--no-replace-objects", "merge-base", "--is-ancestor", r.MainCommit, r.IntegrationHead); err != nil {
		add("the integration head does not contain the main the receipt names")
	}
	return out
}

func (s *Sample) stage4(ctx context.Context) error {
	dir := filepath.Join(s.Work, "stage4")
	origin, task, integ := filepath.Join(dir, "origin"), filepath.Join(dir, "task"), filepath.Join(dir, "integ")
	g := func(d string, a ...string) (string, error) { return s.Tools.git(ctx, d, a...) }
	if _, err := g(dir0(dir), "clone", "-q", s.Repo, origin); err != nil {
		return err
	}
	if _, err := g(origin, "checkout", "-q", "-B", "main"); err != nil {
		return err
	}
	mainCommit, _ := g(origin, "rev-parse", "HEAD")
	// Task side: the candidate goes onto an ordinary commit, bytes unchanged.
	if _, err := g(dir0(dir), "clone", "-q", origin, task); err != nil {
		return err
	}
	if _, err := g(task, "checkout", "-q", "-b", "task/order-rule-design"); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(task, handoverDir, "candidate.json"), s.s3.Candidate); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(task, handoverDir, "attachment.json"), s.s3.Attachment); err != nil {
		return err
	}
	if _, err := g(task, "add", handoverDir); err != nil {
		return err
	}
	if _, err := g(task, "commit", "-q", "-m", "order-rule-design: unit price positive candidate"); err != nil {
		return err
	}
	taskHead, _ := g(task, "rev-parse", "HEAD")
	taskTree, _ := g(task, "rev-parse", "HEAD^{tree}")
	changed, _ := g(task, "diff", "--name-only", mainCommit, taskHead)
	// Integration side: another clone from the same base takes the commit by
	// an ordinary fetch and fast-forward.
	if _, err := g(dir0(dir), "clone", "-q", origin, integ); err != nil {
		return err
	}
	if _, err := g(integ, "fetch", "-q", task, "task/order-rule-design"); err != nil {
		return err
	}
	if _, err := g(integ, "merge", "-q", "--ff-only", "FETCH_HEAD"); err != nil {
		return err
	}
	integHead, _ := g(integ, "rev-parse", "HEAD")
	integTree, _ := g(integ, "rev-parse", "HEAD^{tree}")

	// Rebuild in the integration checkout with its own sources.
	bin := filepath.Join(dir, "integ-bin")
	build := func(name, pkg string) (string, error) {
		out := filepath.Join(bin, name)
		_, err := runCLI(ctx, s.Tools.Env, s.Tools.GoExe, integ, "build", "-o", out, pkg)
		if err != nil {
			return "", err
		}
		if !fileExists(out) {
			return "", fmt.Errorf("go build %s produced nothing", pkg)
		}
		return out, nil
	}
	tei, err := build("tei", "./cmd/tei")
	if err != nil {
		return err
	}
	sa, err := build("sa", "./tools/reference-shared-acceptance")
	if err != nil {
		return err
	}
	_, verr := s.Tools.mustRun(ctx, tei, integ, "validate", "-workspace", "reference/workspace")
	candInteg, err := os.ReadFile(filepath.Join(integ, handoverDir, "candidate.json"))
	if err != nil {
		return err
	}
	if digestBytes(candInteg) != s.s3.Carried.CandidateDigest {
		return fmt.Errorf("the candidate bytes in the integration checkout differ from stage 3")
	}
	scratch := filepath.Join(dir, "integ-scratch")
	if err := copyTree(filepath.Join(integ, "reference", "workspace"), scratch); err != nil {
		return err
	}
	prevRaw, err := s.Tools.mustRun(ctx, sa, integ, "preview", "-workspace", scratch, "-candidate", filepath.Join(integ, handoverDir, "candidate.json"))
	if err != nil {
		return err
	}
	var prev struct {
		Acceptable           bool   `json:"acceptable"`
		CandidateFingerprint string `json:"candidateFingerprint"`
		ModelRevisionAfter   string `json:"modelRevisionAfter"`
	}
	_ = json.Unmarshal(prevRaw, &prev)
	if prev.CandidateFingerprint != s.s3.Fingerprint {
		return fmt.Errorf("rebuilt candidate fingerprint %s differs from stage 3 %s", prev.CandidateFingerprint, s.s3.Fingerprint)
	}
	agree, directRev := s.directEditAgrees(ctx, sa, integ, dir, prev.ModelRevisionAfter)

	// The existing tool-neutral development scenario, unchanged. It needs a
	// clean committed checkout; the qualification receipt is not called
	// because it requires the full repository health check.
	runOut, err := s.Tools.mustRun(ctx, "bash", s.Repo, "tools/reference-tool-neutral-development/run.sh")
	if err != nil {
		return fmt.Errorf("tool-neutral development scenario: %w", err)
	}
	runKV := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(runOut))
	for sc.Scan() {
		if k, v, ok := strings.Cut(sc.Text(), "="); ok && !strings.Contains(k, " ") {
			runKV[k] = v
		}
	}
	if !strings.Contains(string(runOut), "PASS") {
		return fmt.Errorf("tool-neutral development scenario did not pass: %s", runOut)
	}
	rc := Receipt{Schema: "tei.reference.order-rule-design.handover-receipt/v0", CandidateDigest: s.s3.Carried.CandidateDigest,
		Fingerprint: s.s3.Fingerprint, MainCommit: mainCommit, TaskHead: taskHead, TaskTree: taskTree, IntegrationHead: integHead,
		IntegrationTree: integTree, RebuildValidate: verr == nil, RebuildAccept: prev.Acceptable, RevisionAfter: prev.ModelRevisionAfter,
		RunSh: runKV, Qualification: "not-issued: qualify.sh requires the full repository-health-check, which this sample does not run"}
	_, ancErr := g(integ, "merge-base", "--is-ancestor", mainCommit, integHead)
	rc.ContainsMain = ancErr == nil
	if !rc.RebuildValidate || !rc.RebuildAccept || !rc.ContainsMain || integHead != taskHead {
		return fmt.Errorf("hand-over failed: %+v", rc)
	}
	c := s.s3.Carried
	c.MainCommit, c.TaskHead, c.TaskTree, c.IntegrationHead, c.IntegrationTree = mainCommit, taskHead, taskTree, integHead, integTree
	c.CandidateDigest = digestBytes(candInteg)
	s.s4 = stage4Out{Origin: origin, Task: task, Integ: integ, Receipt: rc, Carried: c}
	if p := s.ReverifyReceipt(ctx, rc, origin, integ); len(p) > 0 {
		return fmt.Errorf("fresh receipt does not reverify: %v", p)
	}
	return s.Chain.Append(4, c, map[string]any{
		"receipt": rc, "changedPaths": strings.Split(changed, "\n"), "mergeKind": "fast-forward by ordinary git fetch and merge --ff-only",
		"directEditPathAgrees": agree, "directEditRevision": directRev, "sharedAcceptanceRevisionAfter": prev.ModelRevisionAfter,
		"toolNeutralRun": runKV,
	})
}

func dir0(d string) string { _ = os.MkdirAll(d, 0o755); return d }

// directEditAgrees applies the same change by editing the two source files
// directly in a scratch copy, and compares the model revision with the one
// the shared acceptance preview computed. It is an observation: the two
// paths may differ in whitespace without differing in meaning.
func (s *Sample) directEditAgrees(ctx context.Context, sa, integ, dir, want string) (bool, string) {
	scratch := filepath.Join(dir, "direct-scratch")
	if err := copyTree(filepath.Join(integ, "reference", "workspace"), scratch); err != nil {
		return false, ""
	}
	vp := filepath.Join(scratch, "semantic", "sales", "validation.tei")
	raw, err := os.ReadFile(vp)
	if err != nil {
		return false, ""
	}
	raw = append(raw, []byte("\n\nvalidation "+RuleValidationID+" positive {\n    attribute "+RuleAttributeID+"\n}\n")...)
	if err := os.WriteFile(vp, raw, 0o644); err != nil {
		return false, ""
	}
	ap := filepath.Join(scratch, filepath.FromSlash(strings.TrimPrefix(ApplicationPlace, "")))
	app, err := os.ReadFile(ap)
	if err != nil {
		return false, ""
	}
	lines := strings.Split(string(app), "\n")
	last := -1
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "uses ") {
			last = i
		}
	}
	if last < 0 {
		return false, ""
	}
	lines = append(lines[:last+1], append([]string{"    uses validation " + RuleValidationID}, lines[last+1:]...)...)
	if err := os.WriteFile(ap, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return false, ""
	}
	rev, err := s.revisionWith(ctx, sa, scratch)
	return err == nil && rev == want, rev
}

func (s *Sample) revisionWith(ctx context.Context, sa, workspace string) (string, error) {
	raw, err := s.Tools.mustRun(ctx, sa, s.Repo, "revision", "-workspace", workspace)
	if err != nil {
		return "", err
	}
	var v struct {
		ModelRevision string `json:"modelRevision"`
	}
	err = json.Unmarshal(raw, &v)
	return v.ModelRevision, err
}
