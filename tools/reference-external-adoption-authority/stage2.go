package externaladoptionauthority

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
	authoritytrust "tei/tools/reference-authority-evidence-trust/trust"
	"tei/tools/reference-semantic-extension-catalog/acyclicgraph"
	suihierarchy "tei/tools/reference-sui-adoption/hierarchy"
)

// The extension stage 2 attaches. The descriptors are the unchanged
// testdata of tools/reference-semantic-extension-catalog.
const (
	ExtensionID         = "tei.reference.sui-acyclic"
	ExtensionNamespace  = "reference.sui.acyclic"
	ExtensionConstraint = "acyclic"
	ContractExtension   = "tei.reference.acyclic-contract"
	EvaluatorExtension  = "tei.reference.acyclic-evaluator"
	EvaluatorInterface  = "reference.graph.acyclic.evaluate"
	EvaluatorImpl       = "reference.graph.acyclic-evaluator/v1"
	// CoverageExtended is the stage 1 coverage item the extension covers.
	CoverageExtended = "acyclic-hierarchy"
)

var extensionDescriptorFiles = []string{
	"sui-acyclic-1.0.0.json",
	"acyclic-contract-1.0.0.json",
	"acyclic-evaluator-1.0.0.json",
}

// OwnerIssuer is the synthetic issuer of the namespace authority statement.
var OwnerIssuer = authoritytrust.Issuer{ID: "sui_sensemaking.namespace_publisher", Version: "1.0.0"}

type catalogInputs struct {
	Workspace   string
	Extensions  string
	Requires    []string
	Conformance []string
	Authorities []string
	Grants      []string
	Bindings    []string
}

func (ci catalogInputs) args() []string {
	args := []string{"-workspace", ci.Workspace, "-extensions", ci.Extensions}
	for _, r := range ci.Requires {
		args = append(args, "-require", r)
	}
	for _, p := range ci.Conformance {
		args = append(args, "-interface-conformance", p)
	}
	for _, p := range ci.Authorities {
		args = append(args, "-namespace-authority", p)
	}
	for _, p := range ci.Grants {
		args = append(args, "-publisher-trust", p)
	}
	for _, p := range ci.Bindings {
		args = append(args, "-evaluator-binding", p)
	}
	return args
}

type catalogUnresolved struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

type catalogOut struct {
	ResolutionFingerprint  string `json:"resolutionFingerprint"`
	EvidenceFingerprint    string `json:"evidenceFingerprint"`
	CoreSubjectFingerprint string `json:"coreSubjectFingerprint"`
	NamespaceAuthorities   []struct {
		Namespace                     string `json:"namespace"`
		Extension                     string `json:"extension"`
		EvidenceFingerprint           string `json:"evidenceFingerprint"`
		TrustQualificationFingerprint string `json:"trustQualificationFingerprint"`
	} `json:"namespaceAuthorities"`
	Constraints []struct {
		Extension string `json:"extension"`
		ID        string `json:"id"`
		Subject   struct {
			Kind string `json:"kind"`
			ID   string `json:"id"`
		} `json:"subject"`
		EvaluatorProviderExtension string `json:"evaluatorProviderExtension"`
	} `json:"constraints"`
	Unresolved      []catalogUnresolved `json:"unresolved"`
	AuthorityLimits []string            `json:"authorityLimits"`
}

type stage2Out struct {
	Dir      string
	Inputs   catalogInputs
	LockPath string
	PlanPath string
	Carried  Carried
}

// OwnerJudgment is one judgment of the namespace owner at one trust point.
type OwnerJudgment struct {
	Point     int    `json:"trustPoint"`
	Label     string `json:"trustPointLabel"`
	Qualified bool   `json:"qualified"`
	Exit      int    `json:"exit"`
	Error     string `json:"error,omitempty"`
}

func (s *Sample) writeJSON(path string, v any) (string, error) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return path, writeFile(path, append(raw, '\n'))
}

// contractFingerprintFromDescriptor reads the contract fingerprint the
// evaluator descriptor declares, so no fingerprint is written by hand.
func contractFingerprintFromDescriptor(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var d struct {
		ProvidesInterfaces []struct {
			Interface           string `json:"interface"`
			ContractFingerprint string `json:"contractFingerprint"`
		} `json:"providesInterfaces"`
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		return "", err
	}
	for _, p := range d.ProvidesInterfaces {
		if p.Interface == EvaluatorInterface {
			return p.ContractFingerprint, nil
		}
	}
	return "", fmt.Errorf("%s does not provide %s", path, EvaluatorInterface)
}

func descriptorConstraintSubject(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var d struct {
		Constraints []struct {
			ID      string `json:"id"`
			Subject struct {
				Kind string `json:"kind"`
				ID   string `json:"id"`
			} `json:"subject"`
		} `json:"constraints"`
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		return "", err
	}
	for _, c := range d.Constraints {
		if c.ID == ExtensionConstraint {
			return c.Subject.Kind + ":" + c.Subject.ID, nil
		}
	}
	return "", fmt.Errorf("%s has no constraint %s", path, ExtensionConstraint)
}

// prepareCatalogInputs copies the three descriptors and writes the
// conformance, namespace authority and evaluator binding inputs. Trust
// grants are written per judgment, from the trust point used.
func (s *Sample) prepareCatalogInputs(dir, workspaceDir string) (catalogInputs, error) {
	ci := catalogInputs{
		Workspace:  workspaceDir,
		Extensions: filepath.Join(dir, "extensions"),
		Requires:   []string{ExtensionID + "@1", EvaluatorExtension + "@1", ContractExtension + "@1"},
	}
	src := filepath.Join(s.Repo, "tools", "reference-semantic-extension-catalog", "testdata", "extensions")
	for _, f := range extensionDescriptorFiles {
		if err := copyFile(filepath.Join(src, f), filepath.Join(ci.Extensions, f)); err != nil {
			return ci, err
		}
	}
	contractFP, err := contractFingerprintFromDescriptor(filepath.Join(ci.Extensions, "acyclic-evaluator-1.0.0.json"))
	if err != nil {
		return ci, err
	}
	conf, err := s.writeJSON(filepath.Join(dir, "conformance.json"), map[string]string{
		"providerExtension":   EvaluatorExtension,
		"interface":           EvaluatorInterface,
		"contractFingerprint": contractFP,
		"profile":             "reference.conformance.acyclic-graph-v1",
		"evidenceFingerprint": digestBytes([]byte("synthetic conformance evidence for " + EvaluatorExtension)),
	})
	if err != nil {
		return ci, err
	}
	auth, err := s.writeJSON(filepath.Join(dir, "namespace-authority.json"), map[string]any{
		"namespace":           ExtensionNamespace,
		"extension":           ExtensionID,
		"issuer":              OwnerIssuer,
		"evidenceFingerprint": digestBytes([]byte("synthetic namespace authority " + ExtensionNamespace + " -> " + ExtensionID)),
	})
	if err != nil {
		return ci, err
	}
	// The artifact digest is the digest of the shared acyclic evaluator's
	// source file: the evaluator is a Go package, not a built artifact.
	graphSrc, err := os.ReadFile(filepath.Join(s.Repo, "tools", "reference-semantic-extension-catalog", "acyclicgraph", "graph.go"))
	if err != nil {
		return ci, err
	}
	bind, err := s.writeJSON(filepath.Join(dir, "evaluator-binding.json"), map[string]string{
		"constraintExtension": ExtensionID,
		"constraintId":        ExtensionConstraint,
		"implementationId":    EvaluatorImpl,
		"artifactDigest":      digestBytes(graphSrc),
	})
	if err != nil {
		return ci, err
	}
	ci.Conformance = []string{conf}
	ci.Authorities = []string{auth}
	ci.Bindings = []string{bind}
	return ci, nil
}

// judgeOwner re-verifies the semantic lock's namespace owner against the
// owner grants of one trust point, with the unchanged catalog CLI.
func (s *Sample) judgeOwner(ctx context.Context, ci catalogInputs, lockPath string, p TrustPoint, dir string) (OwnerJudgment, error) {
	grants, err := writeOwnerGrants(filepath.Join(dir, fmt.Sprintf("trust-point-%d", p.Seq)), p.OwnerGrants)
	if err != nil {
		return OwnerJudgment{}, err
	}
	args := []string{"-workspace", ci.Workspace, "-extensions", ci.Extensions,
		"-verify-semantic-lock", lockPath, "-verify-semantic-lock-evidence"}
	for _, c := range ci.Conformance {
		args = append(args, "-interface-conformance", c)
	}
	for _, a := range ci.Authorities {
		args = append(args, "-namespace-authority", a)
	}
	for _, g := range grants {
		args = append(args, "-publisher-trust", g)
	}
	res, err := runCLI(ctx, s.Tools.Catalog, s.Repo, args...)
	if err != nil {
		return OwnerJudgment{}, err
	}
	j := OwnerJudgment{Point: p.Seq, Label: p.Label, Exit: res.Exit, Error: res.Stderr}
	if res.Exit == 0 {
		var out struct {
			Replay   bool `json:"semanticReplayValid"`
			Evidence bool `json:"acceptedEvidenceBindingsValid"`
		}
		if err := json.Unmarshal(res.Stdout, &out); err != nil {
			return j, err
		}
		j.Qualified = out.Replay && out.Evidence
	}
	return j, nil
}

// semanticDigestOf loads a workspace and returns the digest of one
// namespace, computed as workspace imports compute it.
func semanticDigestOf(root, namespace string) (string, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return "", err
	}
	for _, ns := range loaded.World.Semantics {
		if string(ns.ID) == namespace {
			raw, err := json.Marshal(ns)
			if err != nil {
				return "", err
			}
			return sourceindex.Fingerprint(raw), nil
		}
	}
	return "", fmt.Errorf("namespace %s is not in %s", namespace, root)
}

// SharedEvaluation is the verdict of the shared hierarchy package and of the
// shared acyclic graph package on one fixture.
type SharedEvaluation struct {
	Fixture          string `json:"fixture"`
	Stage1Valid      bool   `json:"stage1FixtureValid"`
	HierarchyValid   bool   `json:"sharedHierarchyValid"`
	AcyclicOutcome   string `json:"acyclicOutcome"`
	AcyclicDiagnosis string `json:"acyclicDiagnosis,omitempty"`
}

func evaluateShared(inputDir string, stage1 map[string]bool) ([]SharedEvaluation, error) {
	var out []SharedEvaluation
	for _, fx := range stage1Fixtures {
		doc, err := suihierarchy.ReadFile(filepath.Join(inputDir, fx.File))
		if err != nil {
			return nil, err
		}
		ev := SharedEvaluation{Fixture: fx.File, Stage1Valid: stage1[fx.File],
			HierarchyValid: len(suihierarchy.Validate(doc)) == 0}
		snap := acyclicgraph.Snapshot{}
		for _, is := range doc.Islands {
			snap.Nodes = append(snap.Nodes, is.ID)
			if is.ParentIslandID != nil {
				snap.Edges = append(snap.Edges, acyclicgraph.Edge{From: is.ID, To: *is.ParentIslandID})
			}
		}
		diags, err := acyclicgraph.ValidateAcyclic(snap)
		switch {
		case err != nil:
			ev.AcyclicOutcome, ev.AcyclicDiagnosis = "evaluator-error", err.Error()
		case len(diags) > 0:
			ev.AcyclicOutcome, ev.AcyclicDiagnosis = "fail", diags[0].Code
		default:
			ev.AcyclicOutcome = "pass"
		}
		out = append(out, ev)
	}
	return out, nil
}

// checkStage1Handoff is stage 2's entry check: the stage 1 input copy, the
// four reports and the loss record must still be what stage 1 fixed.
func checkStage1Handoff(s1 stage1Out, c Carried) (LossRecord, error) {
	input, err := treeDigest(s1.InputDir)
	if err != nil {
		return LossRecord{}, err
	}
	if input != c.InputDigest {
		return LossRecord{}, fmt.Errorf("%w: stage 1 input digest %s, carried %s", ErrInputMismatch, input, c.InputDigest)
	}
	mapping, _, err := mappingDigestFromDir(s1.ReportsDir)
	if err != nil {
		return LossRecord{}, err
	}
	if mapping != c.MappingReportDigest {
		return LossRecord{}, fmt.Errorf("%w: mapping report digest %s, carried %s", ErrInputMismatch, mapping, c.MappingReportDigest)
	}
	loss, _, err := readLossRecord(s1.LossPath, c.LossRecordDigest)
	if err != nil {
		return LossRecord{}, err
	}
	if !loss.Source.equal(c.Source) || loss.MappingReportDigest != c.MappingReportDigest {
		return LossRecord{}, fmt.Errorf("%w: loss record names another version or report", ErrInputMismatch)
	}
	return loss, nil
}

// runStage2 attaches the acyclic extension to the stage 1 subject, judges
// the namespace owner at the given trust point and writes the semantic lock
// and the validation plan with the unchanged catalog CLI.
func (s *Sample) runStage2(ctx context.Context, s1 stage1Out, c Carried, p TrustPoint, dir string) (stage2Out, map[string]any, error) {
	out := stage2Out{Dir: dir}
	loss, err := checkStage1Handoff(s1, c)
	if err != nil {
		return out, nil, err
	}
	covered := false
	for _, id := range loss.ShadowOnly {
		covered = covered || id == CoverageExtended
	}
	if !covered {
		return out, nil, fmt.Errorf("%w: coverage item %s is not shadow-only in the loss record", ErrInputMismatch, CoverageExtended)
	}
	ci, err := s.prepareCatalogInputs(dir, filepath.Join(s1.InputDir, "workspace"))
	if err != nil {
		return out, nil, err
	}
	subj, err := descriptorConstraintSubject(filepath.Join(ci.Extensions, "sui-acyclic-1.0.0.json"))
	if err != nil {
		return out, nil, err
	}
	if subj != c.Subject {
		return out, nil, fmt.Errorf("%w: extension constraint subject %s, stage 1 subject %s", ErrInputMismatch, subj, c.Subject)
	}
	grants, err := writeOwnerGrants(filepath.Join(dir, fmt.Sprintf("trust-point-%d", p.Seq)), p.OwnerGrants)
	if err != nil {
		return out, nil, err
	}
	resolveInputs := ci
	resolveInputs.Grants = grants
	out.LockPath = filepath.Join(dir, "semantic-lock.json")
	out.PlanPath = filepath.Join(dir, "validation-plan.json")
	args := append(resolveInputs.args(), "-semantic-lock-out", out.LockPath, "-validation-plan-out", out.PlanPath)
	res, err := runCLI(ctx, s.Tools.Catalog, s.Repo, args...)
	if err != nil {
		return out, nil, err
	}
	if res.Exit != 0 {
		return out, map[string]any{"exit": res.Exit, "stderr": res.Stderr},
			fmt.Errorf("catalog did not resolve at trust point %d: exit %d: %s", p.Seq, res.Exit, firstLine(res.Stderr, res.Stdout))
	}
	var cat catalogOut
	if err := json.Unmarshal(res.Stdout, &cat); err != nil {
		return out, nil, err
	}
	if len(cat.Unresolved) > 0 || len(cat.NamespaceAuthorities) != 1 ||
		cat.NamespaceAuthorities[0].Namespace != ExtensionNamespace || cat.NamespaceAuthorities[0].Extension != ExtensionID {
		return out, nil, fmt.Errorf("catalog output is not the expected single owner: %+v", cat)
	}
	if err := writeFile(filepath.Join(dir, "catalog.json"), res.Stdout); err != nil {
		return out, nil, err
	}
	lockRaw, err := os.ReadFile(out.LockPath)
	if err != nil {
		return out, nil, err
	}
	var lock struct {
		SemanticSetFingerprint string `json:"semanticSetFingerprint"`
		NamespaceAuthorities   []struct {
			Namespace string `json:"namespace"`
			Extension string `json:"extension"`
		} `json:"namespaceAuthorities"`
	}
	if err := json.Unmarshal(lockRaw, &lock); err != nil {
		return out, nil, err
	}
	planRaw, err := os.ReadFile(out.PlanPath)
	if err != nil {
		return out, nil, err
	}
	var plan struct {
		SemanticSetFingerprint string `json:"semanticSetFingerprint"`
		PlanFingerprint        string `json:"planFingerprint"`
	}
	if err := json.Unmarshal(planRaw, &plan); err != nil {
		return out, nil, err
	}
	if lock.SemanticSetFingerprint != cat.ResolutionFingerprint || plan.SemanticSetFingerprint != cat.ResolutionFingerprint {
		return out, nil, fmt.Errorf("lock, plan and catalog disagree on the semantic set")
	}
	verifyPlan, err := runCLI(ctx, s.Tools.Catalog, s.Repo, append(ci.args(), "-verify-validation-plan", out.PlanPath)...)
	if err != nil {
		return out, nil, err
	}
	if verifyPlan.Exit != 0 {
		return out, nil, fmt.Errorf("validation plan did not verify: %s", verifyPlan.Stderr)
	}
	owner, err := s.judgeOwner(ctx, ci, out.LockPath, p, dir)
	if err != nil {
		return out, nil, err
	}
	if !owner.Qualified {
		return out, nil, fmt.Errorf("namespace owner is not qualified at trust point %d: %s", p.Seq, owner.Error)
	}
	shared, err := evaluateShared(s1.InputDir, s1.FixtureValid)
	if err != nil {
		return out, nil, err
	}
	for _, ev := range shared {
		if ev.HierarchyValid != ev.Stage1Valid {
			return out, nil, fmt.Errorf("shared hierarchy package disagrees with stage 1 on %s", ev.Fixture)
		}
	}
	semDigest, err := semanticDigestOf(ci.Workspace, c.Namespace)
	if err != nil {
		return out, nil, err
	}
	out.Inputs = ci
	out.Carried = c
	out.Carried.SemanticDigest = semDigest
	out.Carried.SemanticSetFingerprint = cat.ResolutionFingerprint
	out.Carried.SemanticLockDigest = digestBytes(lockRaw)
	out.Carried.ValidationPlanFingerprint = plan.PlanFingerprint
	out.Carried.ExtensionNamespace = ExtensionNamespace
	out.Carried.NamespaceOwner = ExtensionID
	out.Carried.OwnerIssuer = OwnerIssuer.ID + "@" + OwnerIssuer.Version
	var remaining []string
	for _, id := range append(append([]string{}, loss.Unresolved...), loss.ShadowOnly...) {
		if id != CoverageExtended {
			remaining = append(remaining, id)
		}
	}
	sort.Strings(remaining)
	detail := map[string]any{
		"trustPoint":                    p.Seq,
		"ownerJudgment":                 owner,
		"trustQualificationFingerprint": cat.NamespaceAuthorities[0].TrustQualificationFingerprint,
		"coreSubjectFingerprint":        cat.CoreSubjectFingerprint,
		"evidenceFingerprint":           cat.EvidenceFingerprint,
		"coverageExtended":              CoverageExtended,
		"coverageNotExtended":           remaining,
		"parentPresenceStillUnresolved": contains(loss.Unresolved, "parent-presence"),
		"sharedEvaluation":              shared,
		"authorityLimits":               cat.AuthorityLimits,
		"validationPlanVerified":        true,
	}
	return out, detail, nil
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func firstLine(stderr string, stdout []byte) string {
	if stderr != "" {
		return stderr
	}
	if len(stdout) > 400 {
		return string(stdout[:400])
	}
	return string(stdout)
}
