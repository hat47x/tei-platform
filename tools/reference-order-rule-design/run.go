package orderruledesign

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tei/tools/internal/opensshwitness"
)

// Environment variables of the through test.
const (
	RepoEnv   = "TEI_ORD_REPO"
	GoEnv     = "TEI_ORD_GO"
	OutDirEnv = "TEI_ORD_OUT_DIR"
)

const recordSchema = "tei.reference.order-rule-design.record/v0"

// Check is one fault injection or completion condition.
type Check struct {
	ID          string         `json:"id"`
	Requirement string         `json:"requirement"`
	Stage       string         `json:"stage"`
	Passed      bool           `json:"passed"`
	Observed    map[string]any `json:"observed"`
}

// AssetCall says whether an existing asset could be called unchanged.
type AssetCall struct {
	Stage    string `json:"stage"`
	Asset    string `json:"asset"`
	Callable string `json:"callable"`
	How      string `json:"how"`
}

// Record is the whole run record written as chain-record.json.
type Record struct {
	Schema      string            `json:"schema"`
	Chain       Chain             `json:"chain"`
	Through     ThroughResult     `json:"through"`
	Faults      []Check           `json:"faultInjections"`
	Completion  []Check           `json:"completionConditions"`
	AssetCalls  []AssetCall       `json:"assetCalls"`
	Gaps        []Gap             `json:"platformGaps"`
	NotVerified []string          `json:"notVerified"`
	Limits      []string          `json:"limits"`
	Protected   map[string]string `json:"protectedAssets"`
	Stable      string            `json:"stableSummaryDigest"`
	Passed      bool              `json:"passed"`
}

// Sample is one run of stages 1 to 6 in its own work directory.
type Sample struct {
	Repo  string
	Work  string
	Tools Tools
	Chain *Chain
	Rec   Record

	s1        stage1Out
	s2        stage2Out
	s3        stage3Out
	s4        stage4Out
	s5        stage5Out
	assurance assuranceOut
	matrix    []CheckerRow
	packet    ReviewPacket

	liveBaseline  string
	sg            signer
	payload       opensshwitness.Payload
	signature     []byte
	bundle        SubmissionBundle
	submission    SubmissionReceipt
	orderWorkflow SubmissionReceipt

	protectedBefore map[string]string
	faultNotes      map[string]string
}

// NewSample prepares a run. work must be outside the repository.
func NewSample(repo, work string, tools Tools) (*Sample, error) {
	s := &Sample{Repo: repo, Work: work, Tools: tools, Chain: NewChain(), faultNotes: map[string]string{}}
	s.Rec.Schema = recordSchema
	return s, nil
}

// FindRepo walks up from dir to the directory holding go.mod.
func FindRepo(dir string) (string, error) {
	if env := os.Getenv(RepoEnv); env != "" {
		return filepath.Abs(env)
	}
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d, nil
		}
		if filepath.Dir(d) == d {
			return "", fmt.Errorf("go.mod not found above %s", dir)
		}
	}
}

// protectedAssets fingerprints the shared assets the run must not change.
func (s *Sample) protectedAssets() (map[string]string, error) {
	out := map[string]string{}
	ws := filepath.Join(s.Repo, "reference", "workspace")
	var err error
	if out["reference/workspace"], err = treeDigest(ws); err != nil {
		return nil, err
	}
	if out["reference/world.json"], err = fileDigest(filepath.Join(s.Repo, "reference", "world.json")); err != nil {
		return nil, err
	}
	if out["reference/workspace/.tei/plugin-lock.json"], err = fileDigest(filepath.Join(ws, ".tei", "plugin-lock.json")); err != nil {
		return nil, err
	}
	if out["reference/workspace/.tei/decisions"], err = treeDigest(filepath.Join(ws, ".tei", "decisions")); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Sample) addFault(id, req, stage string, passed bool, observed map[string]any) {
	s.Rec.Faults = append(s.Rec.Faults, Check{ID: id, Requirement: req, Stage: stage, Passed: passed, Observed: observed})
}

func (s *Sample) faultsPassed(prefix string) (bool, []string) {
	ok, n := true, 0
	var ids []string
	for _, f := range s.Rec.Faults {
		if strings.HasPrefix(f.ID, prefix) {
			n++
			ids = append(ids, f.ID)
			ok = ok && f.Passed
		}
	}
	return ok && n > 0, ids
}

// Run executes stages 1-6, the fault injections and the completion checks.
func (s *Sample) Run(ctx context.Context) error {
	var err error
	if s.protectedBefore, err = s.protectedAssets(); err != nil {
		return err
	}
	steps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"stage 1", s.stage1}, {"stage 2", s.stage2}, {"stage 3", s.stage3}, {"stage 4", s.stage4},
		{"stage 5", s.stage5}, {"stage 6", s.stage6},
		{"faults", s.runFaults},
	}
	for _, st := range steps {
		if err := st.fn(ctx); err != nil {
			s.Finish()
			return fmt.Errorf("%s: %w", st.name, err)
		}
	}
	s.Finish()
	return nil
}

// Finish computes the completion conditions and decides whether the run
// passed.
func (s *Sample) Finish() {
	s.Rec.Chain = *s.Chain
	s.Rec.Through = VerifyThrough(*s.Chain)
	s.Rec.AssetCalls = AssetCalls()
	s.Rec.Gaps = Gaps()
	s.Rec.Limits = Limits()
	s.Rec.NotVerified = NotVerified()
	after, err := s.protectedAssets()
	s.Rec.Protected = after
	same := err == nil
	for k, v := range s.protectedBefore {
		same = same && after[k] == v
	}
	s.completion(same)
	passed := s.Rec.Through.Complete
	for _, list := range [][]Check{s.Rec.Faults, s.Rec.Completion} {
		for _, c := range list {
			passed = passed && c.Passed
		}
	}
	s.Rec.Passed = passed && len(s.Rec.Faults) > 0 && len(s.Rec.Completion) == 6
	s.Rec.Stable = s.stableSummary()
}

func (s *Sample) completion(protectedSame bool) {
	add := func(id, req, stage string, ok bool, obs map[string]any) {
		s.Rec.Completion = append(s.Rec.Completion, Check{ID: id, Requirement: req, Stage: stage, Passed: ok, Observed: obs})
	}
	thr, thrIDs := s.faultsPassed("through-")
	add("one-candidate-identity-across-stages", "通し", "1-6", s.Rec.Through.Complete && thr,
		map[string]any{"throughComplete": s.Rec.Through.Complete, "head": s.Rec.Through.Head, "negativeControls": thrIDs})
	pv, pvIDs := s.faultsPassed("provenance-")
	add("provenance-state-and-unresolved-count-to-the-signed-reason", "固有", "1,3,6", pv && s.provenanceSame(),
		map[string]any{"state": s.s1.Carried.ProvenanceState, "unresolved": s.s1.Carried.Unresolved, "reason": s.bundle.Decision.ReasonText, "cases": pvIDs})
	ck, ckIDs := s.faultsPassed("checker-")
	rw, rwIDs := s.faultsPassed("reuse-")
	add("checkers-listed-for-one-candidate-version-and-no-finding-is-not-a-basis", "固有", "2,5,6", ck && rw && len(VerifyMatrix(s.matrix)) == 0,
		map[string]any{"rows": len(s.matrix), "soundnessBasis": SoundnessBasis(s.matrix), "cases": append(ckIDs, rwIDs...)})
	wf1, wf1IDs := s.faultsPassed("wf01-")
	add("baseline-or-main-advance-stops-signing-and-receipt-reverify", "P-WF-01", "4-6", wf1, map[string]any{"cases": wf1IDs})
	mx, mxIDs := s.faultsPassed("mix-")
	wf2, wf2IDs := s.faultsPassed("wf02-")
	add("two-of-three-targets-and-in-flight-cases-stay-in-the-range", "P-MIX-01/P-WF-02", "5,6", mx && wf2,
		map[string]any{"mix": mxIDs, "wf02": wf2IDs})
	_ = rw
	add("signed-values-match-the-acceptance-path-recomputation-and-assets-unchanged", "固有", "4-6",
		len(s.submission.Problems) == 0 && s.submission.DecisionsUnchanged && !s.submission.Accepted && protectedSame,
		map[string]any{"submission": s.submission, "protectedAssetsUnchanged": protectedSame, "orderWorkflow": s.orderWorkflow.Mode})
}

// provenanceSame says the state and count read the same in the candidate
// attachment, the review packet and the signed reason.
func (s *Sample) provenanceSame() bool {
	var att Attachment
	if err := jsonUnmarshal(s.s3.Attachment, &att); err != nil {
		return false
	}
	r := s.bundle.Decision.Reason
	p := s.packet.Provenance
	c := s.s1.Carried
	return att.ProvenanceState == c.ProvenanceState && p.ProvenanceState == c.ProvenanceState && r.ProvenanceState == c.ProvenanceState &&
		att.Unresolved == *c.Unresolved && p.Unresolved == *c.Unresolved && r.Unresolved == *c.Unresolved
}

// stableSummary digests the judgments and diagnostics, not the run's own
// identities or bytes that may vary between runs.
func (s *Sample) stableSummary() string {
	faults := map[string]bool{}
	for _, f := range s.Rec.Faults {
		faults[f.ID] = f.Passed
	}
	comp := map[string]bool{}
	for _, f := range s.Rec.Completion {
		comp[f.ID] = f.Passed
	}
	classes := map[string]string{}
	for _, r := range s.matrix {
		classes[r.Checker] = r.Class + "/" + r.Verdict
	}
	return mustDigest(map[string]any{"faults": faults, "completion": comp, "classes": classes, "state": s.s1.Carried.ProvenanceState,
		"unresolved": s.s1.Carried.Unresolved, "through": s.Rec.Through.Complete})
}
