package externaladoptionauthority

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Environment variables of the through test.
const (
	RepoEnv   = "TEI_EAA_REPO"
	GoEnv     = "TEI_EAA_GO"
	OutDirEnv = "TEI_EAA_OUT_DIR"
)

const recordSchema = "tei.reference.external-adoption-authority.record/v0"

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
	Schema     string            `json:"schema"`
	Chain      Chain             `json:"chain"`
	Through    ThroughResult     `json:"through"`
	Trust      TrustHistory      `json:"trustPoints"`
	Report     *CompletionReport `json:"completionReport,omitempty"`
	Faults     []Check           `json:"faultInjections"`
	Completion []Check           `json:"completionConditions"`
	AssetCalls []AssetCall       `json:"assetCalls"`
	Gaps       []Gap             `json:"platformGaps"`
	Limits     []string          `json:"limits"`
	Passed     bool              `json:"passed"`
}

// Sample is one run of stages 1 to 4 in its own work directory.
type Sample struct {
	Repo  string
	Work  string
	Tools Tools
	Trust TrustHistory
	Chain *Chain
	Rec   Record

	p1     TrustPoint
	s1     stage1Out
	s2     stage2Out
	s3     stage3Out
	host   *Host
	proj   project
	pinned PinnedSource
}

// NewSample prepares a run. work must be outside the repository.
func NewSample(repo, work string, tools Tools) (*Sample, error) {
	s := &Sample{Repo: repo, Work: work, Tools: tools, Chain: NewChain()}
	s.Rec.Schema = recordSchema
	owner, err := NewGrant(OwnerAuthorityClass, OwnerScope(ExtensionNamespace), OwnerIssuer, "owner policy 1", true)
	if err != nil {
		return nil, err
	}
	k1 := NewSignerKey(SignerRef, "1.0.0", 1)
	signer, err := NewGrant(SignerAuthorityClass, OwnerScope("sui"), issuerOf(k1), "signer policy 1", true)
	if err != nil {
		return nil, err
	}
	s.p1 = s.Trust.Add(TrustPoint{Label: "initial", OwnerGrants: ownerList(owner), SignerGrants: ownerList(signer)}.WithKeys(k1))
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

func (s *Sample) stage1(ctx context.Context) error {
	p, err := ReadPinnedTable(filepath.Join(s.Repo, "reference", "sui-adoption", "README.md"))
	if err != nil {
		return err
	}
	s.pinned = p
	ext, err := WriteExternalCopy(p, filepath.Join(s.Work, "external", "current"))
	if err != nil {
		return err
	}
	s.s1, err = runStage1(ctx, s.Tools, s.Repo, filepath.Join(s.Work, "stage1"), ext)
	if err != nil {
		return err
	}
	allFalse := true
	for _, r := range s.s1.Reports {
		allFalse = allFalse && !*r.Mapping.Lossless && r.Mapping.ParentPresenceSemantics == "unresolved"
	}
	return s.Chain.Append(1, s.s1.Carried, map[string]any{
		"fixtureValid":           s.s1.FixtureValid,
		"reportDigests":          s.s1.ReportDigests,
		"pinCheck":               s.s1.Pin,
		"externalComparedWith":   "copy of the pinned table of reference/sui-adoption/README.md",
		"unresolved":             s.s1.Loss.Unresolved,
		"shadowOnly":             s.s1.Loss.ShadowOnly,
		"shapeOnly":              s.s1.Loss.ShapeOnly,
		"assetReportsLossless":   !allFalse,
		"reportSchema":           s.s1.Reports[stage1Fixtures[0].File].SchemaVersion,
		"writeAttempted":         s.s1.Reports[stage1Fixtures[0].File].WriteAttempted,
		"canonicalizedInstances": s.s1.Reports[stage1Fixtures[0].File].Canonicalized,
	})
}

func (s *Sample) stage2(ctx context.Context) error {
	out, detail, err := s.runStage2(ctx, s.s1, s.s1.Carried, s.p1, filepath.Join(s.Work, "stage2"))
	if err != nil {
		return err
	}
	s.s2 = out
	return s.Chain.Append(2, out.Carried, detail)
}

func (s *Sample) stage3(ctx context.Context) error {
	out, detail, err := s.runStage3(ctx, s.s1, s.s2, s.s2.Carried, s.p1, filepath.Join(s.Work, "stage3"))
	if err != nil {
		return err
	}
	s.s3 = out
	return s.Chain.Append(3, out.Carried, detail)
}

func (s *Sample) stage4(ctx context.Context) error {
	h, proj, detail, err := s.runStage4(ctx, s.s1, s.s2, s.s3, s.s3.Carried, s.p1, filepath.Join(s.Work, "stage4"))
	if err != nil {
		return err
	}
	s.host, s.proj = h, proj
	return s.Chain.Append(4, s.s3.Carried, detail)
}

// Run executes stages 1-4, the fault injections and the completion checks,
// then verifies the chain and the completion report by reading them only.
func (s *Sample) Run(ctx context.Context) error {
	steps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"stage 1", s.stage1}, {"stage 2", s.stage2}, {"stage 3", s.stage3}, {"stage 4", s.stage4},
		{"P-GOV-01", s.faultPGOV01}, {"P-INT-03", s.faultPINT03}, {"P-OPS-02", s.faultPOPS02},
		{"lossless", s.faultLossless}, {"meaning change", s.faultMeaningChange}, {"substitution", s.faultSubstitution},
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

// Finish computes the completion conditions, writes the report and the
// gaps, and decides whether the run passed.
func (s *Sample) Finish() {
	s.Rec.Chain = *s.Chain
	s.Rec.Through = VerifyThrough(*s.Chain)
	s.Rec.Trust = s.Trust
	s.Rec.AssetCalls = AssetCalls()
	s.Rec.Gaps = Gaps()
	s.Rec.Limits = Limits()
	rep, repErr := BuildCompletionReport(*s.Chain)
	var repMismatch []string
	if repErr == nil {
		s.Rec.Report = &rep
		repMismatch = VerifyCompletionReport(*s.Chain, rep)
	}
	sub, subIDs := s.faultsPassed("substitution-")
	thr, thrIDs := s.faultsPassed("through-")
	s.Rec.Completion = append(s.Rec.Completion, Check{ID: "one-pinned-version-carried-to-acceptance", Requirement: "通し", Stage: "1-4",
		Passed: s.Rec.Through.Complete && sub && thr,
		Observed: map[string]any{"throughComplete": s.Rec.Through.Complete, "head": s.Rec.Through.Head,
			"substitutionStops": subIDs, "throughNegativeControls": thrIDs}})
	loss, lossIDs := s.faultsPassed("lossless-")
	s.Rec.Completion = append(s.Rec.Completion, Check{ID: "lossless-false-carried-to-completion-report", Requirement: "固有", Stage: "1-4",
		Passed: loss && s.Rec.Through.Lossless != nil && !*s.Rec.Through.Lossless && repErr == nil && !rep.Loss.Lossless,
		Observed: map[string]any{"throughLossless": s.Rec.Through.Lossless, "reportLossless": rep.Loss.Lossless, "cases": lossIDs}})
	gov, _ := s.faultsPassed("pgov01-")
	intr, intIDs := s.faultsPassed("pint03-")
	ops, opsIDs := s.faultsPassed("pops02-")
	s.Rec.Completion = append(s.Rec.Completion, Check{ID: "trust-judged-at-the-judgment-point", Requirement: "P-OPS-02/P-INT-03", Stage: "2-4",
		Passed: intr && ops, Observed: map[string]any{"pint03": intIDs, "pops02": opsIDs}})
	mean, meanIDs := s.faultsPassed("meaning-")
	var selected any
	if d := stageDetail(*s.Chain, 4); d != nil {
		selected = string(d["selectedEntries"])
	}
	s.Rec.Completion = append(s.Rec.Completion, Check{ID: "acceptance-one-to-one-and-meaning-change-refused", Requirement: "固有", Stage: "4",
		Passed: mean && selected == "1", Observed: map[string]any{"selectedEntries": selected, "cases": meanIDs}})
	rneg, rnegIDs := s.faultsPassed("report-")
	s.Rec.Completion = append(s.Rec.Completion, Check{ID: "completion-report-distinguishes-and-does-not-overclaim", Requirement: "固有", Stage: "報告",
		Passed: repErr == nil && len(repMismatch) == 0 && rneg,
		Observed: map[string]any{"reportError": errString(repErr), "mismatches": repMismatch, "negativeControls": rnegIDs}})
	passed := s.Rec.Through.Complete && gov
	for _, list := range [][]Check{s.Rec.Faults, s.Rec.Completion} {
		for _, c := range list {
			passed = passed && c.Passed
		}
	}
	s.Rec.Passed = passed && len(s.Rec.Faults) > 0 && len(s.Rec.Completion) == 5
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// AssetCalls records, per stage, whether the existing asset was called
// unchanged and how.
func AssetCalls() []AssetCall {
	return []AssetCall{
		{"1", "reference/sui-adoption", "固定入力として複製して使う", "workspaceとM1〜M4を作業ディレクトリへ複製する。READMEの固定表を外部側の固定コピーとして読む"},
		{"1", "tools/reference-sui-adoption", "CLIで呼べる", "package mainのため関数は呼べない。go buildしたCLIを複製の中で四つの事例に実行し、報告(schema v2)を読む"},
		{"2", "tools/reference-semantic-extension-catalog", "CLIで呼べる(評価は呼べない)", "-require、-interface-conformance、-namespace-authority、-publisher-trust、-evaluator-binding、-semantic-lock-out、-validation-plan-out、-verify-semantic-lock(-evidence)、-verify-validation-planを使う。評価関数はCLIに出ていない"},
		{"2", "tools/reference-sui-adoption/hierarchy、tools/reference-semantic-extension-catalog/acyclicgraph", "ライブラリとして呼べる", "工程1と同じ判定を共有パッケージで評価し、二重に実装しない"},
		{"2〜4", "tools/reference-authority-evidence-trust/trust", "ライブラリとして呼べる", "名前空間所有者(カタログ経由)と署名者(サンプルのLookup)を同じVerifyで判定する"},
		{"3", "reference/semantic-distribution", "ライブラリとして呼べる", "MaterializeReference、InspectSelectedReference"},
		{"3", "reference/semantic-build-bridge", "ライブラリとして呼べる", "InspectLockedExportで構築計画と固定記録を照合して読む"},
		{"3", "reference/semantic-composition-preflight", "ライブラリとして呼べる", "SemanticPublicationMessageとInspectPublished(論理時刻を渡す)"},
		{"4", "reference/semantic-composition-preflight", "ライブラリとして呼べる(パス取込みだけ)", "InspectCurrentProjectMeaningを合成ホストで呼ぶ。データのみの配布物は受理照合の対象にならないため、パス取込みの意味の照合値を配布物の値と突き合わせる"},
	}
}
