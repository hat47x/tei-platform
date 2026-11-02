package orderruledesign

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The subject of the candidate: the rule that limits the unit price to
// positive values.
const (
	RuleValidationID = "sales.order_line.unit_price_positive"
	RuleAttributeID  = "sales.order_line.unit_price"
	RuleApplication  = "order_entry"
	ApplicationPlace = "applications/order-entry/application.tei"
	ReviewScope      = "order-entry-reference"
)

// Classes of one checker output.
const (
	ClassEvaluated    = "evaluated"          // 対象にできた: the output is about the candidate's rule
	ClassOutOfScope   = "out-of-scope"       // 対象外: the checker examines another rule
	ClassInsufficient = "insufficient-input" // 入力不足: the checker could not examine anything
)

// CheckerRow is one checker's output for the same candidate version, with
// what it covered. "No finding" is a verdict, not a class: an out-of-scope
// checker can say no finding about a rule it never looked at.
type CheckerRow struct {
	Checker      string         `json:"checker"`
	Stage        int            `json:"stage"`
	Input        string         `json:"input"`
	Class        string         `json:"class"`
	Verdict      string         `json:"verdict"`
	Findings     int            `json:"findings"`
	Basis        string         `json:"basis"`
	OutputDigest string         `json:"outputDigest"`
	Observed     map[string]any `json:"observed,omitempty"`
}

// EvalTarget is something the unit price rule should be evaluated against
// before it is confirmed.
type EvalTarget struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Basis string `json:"basis"`
}

// UnresolvedItem is a diagnostic that carries no authority: it is listed so
// it stays visible, and never read as accepted meaning.
type UnresolvedItem struct {
	Source    string `json:"source"`
	Code      string `json:"code"`
	Count     int    `json:"count"`
	Authority string `json:"authority"`
}

type stage2Out struct {
	R0         string
	Revision   string
	World      []byte
	Targets    []EvalTarget
	Rows       []CheckerRow
	Unresolved []UnresolvedItem
	Carried    Carried
}

func (s *Sample) revisionOf(ctx context.Context, workspace string) (string, error) {
	raw, err := s.Tools.mustRun(ctx, s.Tools.SharedAcceptance, s.Repo, "revision", "-workspace", workspace)
	if err != nil {
		return "", err
	}
	var v struct {
		ModelRevision string `json:"modelRevision"`
	}
	if err := json.Unmarshal(raw, &v); err != nil || v.ModelRevision == "" {
		return "", fmt.Errorf("revision output: %v %s", err, raw)
	}
	return v.ModelRevision, nil
}

func (s *Sample) stage2(ctx context.Context) error {
	r0 := filepath.Join(s.Work, "stage2", "r0")
	if err := copyTree(filepath.Join(s.Repo, "reference", "workspace"), r0); err != nil {
		return err
	}
	rev, err := s.revisionOf(ctx, r0)
	if err != nil {
		return err
	}
	world, err := s.Tools.mustRun(ctx, s.Tools.Tei, s.Repo, "world", "-workspace", r0)
	if err != nil {
		return err
	}
	out := stage2Out{R0: r0, Revision: rev, World: world}
	targets, err := evaluationTargets(world)
	if err != nil {
		return err
	}
	out.Targets = targets
	// The checkers read the baseline copy R0 and the candidate's subject.
	if r, err := s.runDFD(ctx, r0, 2); err != nil {
		return err
	} else {
		out.Rows = append(out.Rows, r)
	}
	if r, items, err := s.runCUD(ctx, r0, world, 2); err != nil {
		return err
	} else {
		out.Rows = append(out.Rows, r)
		out.Unresolved = append(out.Unresolved, items...)
	}
	// The unresolved list of stage 1 enters stage 2 as an input; stage 2
	// lists it without authority and records the count it received.
	u := s.s1.Carried.Unresolved
	out.Unresolved = append(out.Unresolved,
		UnresolvedItem{"stage1:source-observation", "contract-cannot-handle", u.SourceObservation, "none"},
		UnresolvedItem{"stage1:mapping", "mapping-unresolved", u.Mapping, "none"},
		UnresolvedItem{"stage1:semantic-candidate", "semantic-classification-unresolved", u.SemanticCandidate, "none"})
	sort.SliceStable(out.Unresolved, func(i, j int) bool { return out.Unresolved[i].Source < out.Unresolved[j].Source })
	c := s.s1.Carried
	c.BaselineRevision = rev
	c.TargetsDigest = mustDigest(out.Targets)
	out.Carried = c
	s.s2 = out
	return s.Chain.Append(2, c, map[string]any{
		"receivedFromStage1": map[string]any{"provenanceState": s.s1.Carried.ProvenanceState, "unresolved": u},
		"workspaceCopy":      "copy of reference/workspace (R0); the shared directory is never written",
		"baselineRevision":   rev,
		"evaluationTargets":  out.Targets,
		"checkers":           out.Rows,
		"unresolvedList":     out.Unresolved,
		"unresolvedNote":     "listed without authority; not accepted meaning",
	})
}

// evaluationTargets finds the applications that use the semantic namespace
// of the unit price attribute. It reads the World JSON that `tei world`
// derives; it does not read source text.
func evaluationTargets(world []byte) ([]EvalTarget, error) {
	var w struct {
		Applications []struct {
			ID           string   `json:"id"`
			SemanticUses []string `json:"semanticUses"`
			Validations  []string `json:"validations"`
		} `json:"applications"`
	}
	if err := json.Unmarshal(world, &w); err != nil {
		return nil, err
	}
	if !strings.Contains(string(world), `"`+RuleAttributeID+`"`) {
		return nil, nil
	}
	ns := strings.SplitN(RuleAttributeID, ".", 2)[0]
	var out []EvalTarget
	for _, a := range w.Applications {
		if contains(a.SemanticUses, ns) {
			out = append(out, EvalTarget{"application", a.ID,
				"the application uses semantic namespace " + ns + ", which declares " + RuleAttributeID + "; it does not yet select " + RuleValidationID})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func mentions(raw []byte, subjects ...string) bool {
	text := string(raw)
	for _, s := range subjects {
		if strings.Contains(text, s) {
			return true
		}
	}
	return false
}

// runDFD runs the business DFD projector on a workspace. It draws the order
// approval procedure, so a rule on the unit price is outside its subject.
func (s *Sample) runDFD(ctx context.Context, workspace string, stage int) (CheckerRow, error) {
	raw, err := s.Tools.mustRun(ctx, s.Tools.BusinessDFD, s.Repo, "-workspace", workspace, "-format", "json")
	if err != nil {
		return CheckerRow{}, err
	}
	row := CheckerRow{Checker: "tools/reference-business-dfd", Stage: stage, Input: "workspace copy + order-entry procedure fixture",
		Verdict: "no-finding", OutputDigest: digestBytes(raw)}
	var v struct {
		ProcedureID string `json:"procedureId"`
	}
	_ = json.Unmarshal(raw, &v)
	row.Observed = map[string]any{"procedure": v.ProcedureID, "mentionsUnitPrice": mentions(raw, "unit_price", "unitPrice")}
	if mentions(raw, "unit_price", "unitPrice") {
		row.Class, row.Basis = ClassEvaluated, "the output names the unit price"
	} else {
		row.Class, row.Basis = ClassOutOfScope, "the output draws the order-handling procedure and never names the unit price rule"
	}
	return row, nil
}

// runCUD runs the subsystem CUD diagnostic on the World of a workspace with
// an empty fact set. The sample has no accepted ownership or function I/O
// facts for the order line, so the diagnostic has nothing to examine.
func (s *Sample) runCUD(ctx context.Context, workspace string, world []byte, stage int) (CheckerRow, []UnresolvedItem, error) {
	dir := filepath.Join(s.Work, fmt.Sprintf("cud-stage%d", stage))
	worldPath, factsPath := filepath.Join(dir, "world.json"), filepath.Join(dir, "facts.json")
	if err := writeFile(worldPath, world); err != nil {
		return CheckerRow{}, nil, err
	}
	facts, _ := json.Marshal(map[string]any{"schemaVersion": "subsystem-cud-candidate-v1",
		"worldJSONFingerprint": digestBytes(world), "owners": []any{}, "functionIO": []any{}, "procedureSteps": []any{}})
	if err := writeFile(factsPath, facts); err != nil {
		return CheckerRow{}, nil, err
	}
	// The diagnostic exits nonzero for an unresolved report; both are read.
	res, err := runCLI(ctx, s.Tools.Env, s.Tools.SubsystemCUD, s.Repo, "-world", worldPath, "-facts", factsPath)
	if err != nil {
		return CheckerRow{}, nil, err
	}
	var v struct {
		Status      string `json:"status"`
		Diagnostics []struct {
			Code string `json:"code"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(res.Stdout, &v); err != nil {
		return CheckerRow{}, nil, fmt.Errorf("subsystem-cud output: %v: %s", err, res.Stdout)
	}
	row := CheckerRow{Checker: "tools/reference-subsystem-cud", Stage: stage, Input: "World of the workspace + empty candidate facts",
		Verdict: v.Status, Findings: len(v.Diagnostics), OutputDigest: digestBytes(res.Stdout),
		Observed: map[string]any{"exit": res.Exit, "workspace": filepath.Base(workspace)}}
	var items []UnresolvedItem
	empty := false
	for _, d := range v.Diagnostics {
		items = append(items, UnresolvedItem{"subsystem-cud", d.Code, 1, "none"})
		if d.Code == "EMPTY_CANDIDATE_SCOPE" {
			empty = true
		}
	}
	if empty {
		row.Class, row.Basis = ClassInsufficient, "EMPTY_CANDIDATE_SCOPE: no declared ownership, function I/O or procedure step to examine"
	} else {
		row.Class, row.Basis = ClassOutOfScope, "it examines subsystem ownership and CRUD intent, not a value rule"
	}
	return row, items, nil
}

// copyAndWrite is used by the stages to put a fresh file next to a copy.
func copyAndWrite(path string, raw []byte) error {
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, raw, 0o644)
}
