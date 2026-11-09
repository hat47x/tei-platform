package orderruledesign

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tei/tools/internal/opensshwitness"
)

const (
	witnessWorkspace = "ws-order-rule-design"
	signerPrincipal  = "designer@example.test"
	issuedAt         = "20261029T000000Z"
	verifyTime       = "20261029000000Z"
)

// requiredCheckers are the checker outputs every review packet must list,
// whatever their class.
var requiredCheckers = []string{
	"tools/reference-business-dfd@R0", "tools/reference-subsystem-cud@R0",
	"tools/reference-business-dfd@R1", "tools/reference-subsystem-cud@R1",
	"tools/reference-change-assurance", "tools/reference-shared-acceptance:preview",
	"tools/reference-spec-review", "tools/reference-purpose-preflight@R1",
	"tools/reference-purpose-preflight@R0", "tools/reference-purpose-preflight:order-approval-fixture",
	"tools/reference-tool-neutral-development:direct-edit",
}

var noFindingVerdicts = map[string]bool{"no-finding": true, "partial-no-finding": true, "no-finding-in-provided-evidence": true}

// VerifyMatrix returns the ways the checker list fails: a required checker
// missing, an unknown class, nothing evaluated, or a list that only
// collects "no finding".
func VerifyMatrix(rows []CheckerRow) []string {
	var out []string
	add := func(f string, a ...any) { out = append(out, fmt.Sprintf(f, a...)) }
	seen := map[string]CheckerRow{}
	for _, r := range rows {
		seen[r.Checker] = r
		if r.Class != ClassEvaluated && r.Class != ClassOutOfScope && r.Class != ClassInsufficient {
			add("%s has class %q", r.Checker, r.Class)
		}
		if r.Basis == "" {
			add("%s states no basis for its class", r.Checker)
		}
	}
	for _, name := range requiredCheckers {
		if _, ok := seen[name]; !ok {
			add("checker %s is not listed", name)
		}
	}
	evaluated, allNoFinding := 0, len(rows) > 0
	for _, r := range rows {
		if r.Class == ClassEvaluated {
			evaluated++
		}
		if !noFindingVerdicts[r.Verdict] {
			allNoFinding = false
		}
	}
	if evaluated == 0 {
		add("no checker evaluated the candidate's rule")
	}
	if allNoFinding {
		add("the list only collects no-finding verdicts")
	}
	return out
}

// SoundnessBasis counts what may support "the rule has no problem": only
// evaluated rows. An out-of-scope or insufficient-input row contributes
// nothing, whatever it said.
func SoundnessBasis(rows []CheckerRow) int {
	n := 0
	for _, r := range rows {
		if r.Class == ClassEvaluated {
			n++
		}
	}
	return n
}

// ReviewPacket is the review packet: the same provenance values as the
// candidate attachment, the checker list and the impact range digest.
type ReviewPacket struct {
	Schema            string          `json:"schema"`
	CandidateFP       string          `json:"candidateFingerprint"`
	BaselineRevision  string          `json:"baselineRevision"`
	ReviewFingerprint string          `json:"reviewFingerprint"`
	Provenance        Attachment      `json:"provenance"`
	Checkers          []CheckerRow    `json:"checkers"`
	SoundnessBasis    int             `json:"soundnessBasis"`
	ImpactRangeDigest string          `json:"impactRangeDigest"`
	AssuranceDigest   string          `json:"assuranceDigest"`
	SpecReview        json.RawMessage `json:"specReview"`
	InFlight          []InFlight      `json:"inFlightCases"`
}

func reasonFromPacket(p ReviewPacket) Reason {
	r := Reason{ProvenanceState: p.Provenance.ProvenanceState, Unresolved: p.Provenance.Unresolved, SourceDigest: p.Provenance.SourceDigest,
		ContractDigest: p.Provenance.ContractDigest, EvidenceFingerprint: p.Provenance.EvidenceFingerprint,
		AssuranceDigest: p.AssuranceDigest, ImpactRangeDigest: p.ImpactRangeDigest, MatrixDigest: mustDigest(p.Checkers), InFlight: len(p.InFlight)}
	for _, c := range p.InFlight {
		if c.Disposition == "" {
			r.Undecided++
		}
	}
	return r
}

func reasonText(r Reason) string {
	return fmt.Sprintf("provenance=%s; unresolved=%d (source-observation %d, mapping %d, semantic-candidate %d); in-flight=%d undecided=%d; assurance=%s",
		r.ProvenanceState, r.Unresolved.Total, r.Unresolved.SourceObservation, r.Unresolved.Mapping, r.Unresolved.SemanticCandidate,
		r.InFlight, r.Undecided, r.AssuranceDigest)
}

// checkerRowsStage6 runs the checkers that stage 6 owns and gathers the
// rows of the earlier stages, so the list covers one candidate version.
func (s *Sample) checkerRows(ctx context.Context) ([]CheckerRow, json.RawMessage, error) {
	dir := filepath.Join(s.Work, "stage6")
	rows := append([]CheckerRow{}, s.s2.Rows...)
	rows[0].Checker += "@R0"
	rows[1].Checker += "@R0"
	r1 := s.s5.R1
	world1, err := s.Tools.mustRun(ctx, s.Tools.Tei, s.Repo, "world", "-workspace", r1)
	if err != nil {
		return nil, nil, err
	}
	d1, err := s.runDFD(ctx, r1, 6)
	if err != nil {
		return nil, nil, err
	}
	d1.Checker += "@R1"
	c1, _, err := s.runCUD(ctx, r1, world1, 6)
	if err != nil {
		return nil, nil, err
	}
	c1.Checker += "@R1"
	rows = append(rows, d1, c1)
	a := s.assurance
	rows = append(rows, CheckerRow{Checker: "tools/reference-change-assurance", Stage: 5, Input: "R0 and R1 copies", Class: ClassEvaluated,
		Verdict: a.Action, Findings: len(a.Impact), OutputDigest: a.Digest,
		Basis:    "the impact lists " + RuleValidationID + " and the consumer application",
		Observed: map[string]any{"executionAllowed": a.Exec, "emitsExecutionTargets": a.HasTargetKeys}})
	ok, _ := s.s3.Preview["acceptable"].(bool)
	rows = append(rows, CheckerRow{Checker: "tools/reference-shared-acceptance:preview", Stage: 3, Input: "candidate on R0", Class: ClassEvaluated,
		Verdict: map[bool]string{true: "acceptable-in-preview", false: "not-acceptable"}[ok], OutputDigest: digestBytes(s.s3.PreviewRaw),
		Basis: "the semantic diff adds " + RuleValidationID, Observed: map[string]any{"previewAcceptable": ok}})
	// spec-review: the knowledge context of the new rule on R1 and the preview.
	kc, err := s.Tools.mustRun(ctx, s.Tools.Tei, s.Repo, "knowledge-context", "-workspace", r1, "-subject-kind", "semantic-validation", "-subject-id", RuleValidationID)
	if err != nil {
		return nil, nil, err
	}
	kcPath, prPath := filepath.Join(dir, "knowledge-context.json"), filepath.Join(dir, "preview.json")
	if err := writeFile(kcPath, kc); err != nil {
		return nil, nil, err
	}
	if err := writeFile(prPath, s.s3.PreviewRaw); err != nil {
		return nil, nil, err
	}
	sr, err := s.Tools.mustRun(ctx, s.Tools.SpecReview, s.Repo, "-case-id", "order-rule-design", "-knowledge-context", kcPath, "-shared-acceptance", prPath)
	if err != nil {
		return nil, nil, err
	}
	var srv struct {
		Status        string           `json:"status"`
		Findings      []map[string]any `json:"findings"`
		MissingInputs []string         `json:"missingInputs"`
	}
	_ = json.Unmarshal(sr, &srv)
	srRow := CheckerRow{Checker: "tools/reference-spec-review", Stage: 6, Input: "knowledge context of the rule on R1 + preview", Verdict: srv.Status,
		Findings: len(srv.Findings), OutputDigest: digestBytes(sr), Observed: map[string]any{"missingInputs": srv.MissingInputs}}
	if mentions(sr, RuleValidationID) {
		srRow.Class, srRow.Basis = ClassEvaluated, "its evidence names "+RuleValidationID+"; missing inputs stay listed"
	} else {
		srRow.Class, srRow.Basis = ClassInsufficient, "its evidence does not name the rule"
	}
	rows = append(rows, srRow)
	// purpose-preflight on R1, on R0 and on its own fixture rule.
	pp := func(ws string) ([]byte, map[string]any, error) {
		raw, err := s.Tools.mustRun(ctx, s.Tools.PurposePreflight, s.Repo, "-workspace", ws, "-application", RuleApplication,
			"-subject-kind", "semantic-validation", "-subject-id", RuleValidationID)
		var v map[string]any
		_ = json.Unmarshal(raw, &v)
		return raw, v, err
	}
	raw1, v1, err := pp(r1)
	if err != nil {
		return nil, nil, err
	}
	row := CheckerRow{Checker: "tools/reference-purpose-preflight@R1", Stage: 6, Input: "R1 copy, " + RuleValidationID, OutputDigest: digestBytes(raw1),
		Verdict: fmt.Sprint(v1["ruleMeaning"]), Observed: map[string]any{"executionAllowed": v1["executionAllowed"], "unresolved": v1["unresolved"]}}
	if v1["ruleMeaning"] == "canonical-validation-declared" {
		row.Class, row.Basis = ClassEvaluated, "the rule is declared and the application selects it; human acceptance and runtime stay unresolved"
	} else {
		row.Class, row.Basis = ClassInsufficient, "the rule is not established on R1"
	}
	rows = append(rows, row)
	raw0, v0, err := pp(s.s2.R0)
	if err != nil {
		return nil, nil, err
	}
	row0 := CheckerRow{Checker: "tools/reference-purpose-preflight@R0", Stage: 6, Input: "R0 copy, " + RuleValidationID, OutputDigest: digestBytes(raw0),
		Verdict: fmt.Sprint(v0["ruleMeaning"]), Observed: map[string]any{"unresolved": v0["unresolved"]}}
	row0.Class, row0.Basis = ClassInsufficient, "the rule does not exist on the baseline R0, so there is nothing to examine"
	rows = append(rows, row0)
	fx, err := s.Tools.mustRun(ctx, s.Tools.PurposePreflight, s.Repo, "-input", "tools/reference-purpose-preflight/order-approval.json")
	if err != nil {
		return nil, nil, err
	}
	rows = append(rows, CheckerRow{Checker: "tools/reference-purpose-preflight:order-approval-fixture", Stage: 6, Input: "synthetic approval rule cases",
		Class: ClassOutOfScope, Verdict: "no-finding", OutputDigest: digestBytes(fx), Observed: map[string]any{"mentionsUnitPrice": mentions(fx, "unit_price", "unitPrice")},
		Basis: "its cases judge an approval threshold on the order total, not a unit price rule"})
	de := s.s4.Receipt.RunSh
	rows = append(rows, CheckerRow{Checker: "tools/reference-tool-neutral-development:direct-edit", Stage: 4, Input: "repository checkout", Class: ClassEvaluated,
		Verdict: "no-finding", OutputDigest: mustDigest(de), Basis: "the scenario adds " + RuleValidationID + " by editing the source file directly",
		Observed: map[string]any{"changed": de["changed"], "resultCommit": de["result"]}})
	return rows, sr, nil
}

// preflightInput is what the signing preflight recomputes against.
type preflightInput struct {
	Chain       Chain
	BaselineDir string // the live baseline another party may advance
	Origin      string
	Integ       string
	Receipt     Receipt
	SourcePath  string
	Range       ImpactRange
	Matrix      []CheckerRow
}

// Preflight recomputes what stages 4 and 5 recorded and returns the reasons
// not to sign. Signing happens only when it returns none.
func (s *Sample) Preflight(ctx context.Context, in preflightInput) []string {
	var out []string
	add := func(f string, a ...any) { out = append(out, fmt.Sprintf(f, a...)) }
	th := VerifyThrough(in.Chain)
	if len(th.Mismatches) > 0 || len(th.MissingStages) != 1 || th.MissingStages[0] != 6 {
		add("stages 1-5 are not one chain: missing %v mismatches %v", th.MissingStages, th.Mismatches)
	}
	var c5 Carried
	for _, r := range in.Chain.Records {
		if r.Stage == 5 {
			c5 = r.Carried
		}
	}
	if rev, err := s.revisionOf(ctx, in.BaselineDir); err != nil {
		add("baseline revision: %v", err)
	} else if rev != c5.BaselineRevision {
		add("the baseline moved: recomputed %s, stage 5 recorded %s", rev, c5.BaselineRevision)
	}
	for _, p := range s.ReverifyReceipt(ctx, in.Receipt, in.Origin, in.Integ) {
		add("receipt: %s", p)
	}
	if d, err := fileDigest(in.SourcePath); err != nil || d != c5.SourceDigest {
		add("the legacy material changed or is unreadable: %s, stage 1 recorded %s", d, c5.SourceDigest)
	}
	for _, p := range VerifyImpactRange(in.Range, inFlightCases(c5.BaselineRevision, nil)) {
		add("impact range: %s", p)
	}
	for _, c := range in.Range.InFlight {
		if c.Disposition == "" {
			add("in-flight case %s started at %s has no decided handling (finish on R0 or migrate)", c.CaseID, c.StartedVersion)
		}
	}
	for _, p := range VerifyMatrix(in.Matrix) {
		add("checkers: %s", p)
	}
	return out
}

type signer struct {
	Key, Allowed string
}

func (s *Sample) newSigner(ctx context.Context, dir string, scopedTo opensshwitness.Payload) (signer, error) {
	key := filepath.Join(dir, "designer")
	if _, err := s.Tools.mustRun(ctx, "ssh-keygen", dir, "-q", "-t", "ed25519", "-N", "", "-f", key); err != nil {
		return signer{}, err
	}
	pub, err := os.ReadFile(key + ".pub")
	if err != nil {
		return signer{}, err
	}
	ns, err := scopedTo.Namespace()
	if err != nil {
		return signer{}, err
	}
	allowed := filepath.Join(dir, "allowed_signers")
	line := fmt.Sprintf("%s namespaces=\"%s\",valid-after=\"20200101Z\",valid-before=\"20990101Z\" %s", signerPrincipal, ns, strings.TrimSpace(string(pub)))
	return signer{Key: key, Allowed: allowed}, os.WriteFile(allowed, []byte(line+"\n"), 0o644)
}

func (s *Sample) payloadFor(candidateFP, reviewFP, base string, reason Reason, decision string) opensshwitness.Payload {
	return opensshwitness.Payload{WorkspaceIdentity: witnessWorkspace, CandidateFingerprint: candidateFP, ReviewFingerprint: reviewFP,
		BaseModelRevision: base, Decision: decision, DecisionReasonDigest: mustDigest(reason),
		ScopeDigest:     mustDigest(map[string]any{"candidate": candidateFP, "application": RuleApplication, "targets": len(executionTargets)}),
		WitnessIssuedAt: issuedAt, WitnessNonce: strings.TrimPrefix(candidateFP, "sha256:")[:16]}
}

func (s *Sample) verifySignature(sg signer, principal string, p opensshwitness.Payload, sig []byte) (opensshwitness.Evidence, error) {
	return opensshwitness.Verify(opensshwitness.ProviderOptions{AllowedSigners: sg.Allowed, VerifyTime: verifyTime}, principal, p, sig)
}

func (s *Sample) stage6(ctx context.Context) error {
	dir := filepath.Join(s.Work, "stage6")
	rows, sr, err := s.checkerRows(ctx)
	if err != nil {
		return err
	}
	s.matrix = rows
	// The review fingerprint comes from the review on the baseline copy, as
	// the acceptance path computes it.
	reviewFP := s.s5.Review
	s.liveBaseline = filepath.Join(s.Work, "baseline-live")
	if err := copyTree(filepath.Join(s.Repo, "reference", "workspace"), s.liveBaseline); err != nil {
		return err
	}
	packet := ReviewPacket{Schema: "tei.reference.order-rule-design.review-packet/v0", CandidateFP: s.s3.Fingerprint,
		BaselineRevision: s.s2.Revision, ReviewFingerprint: reviewFP, Checkers: rows, SoundnessBasis: SoundnessBasis(rows),
		ImpactRangeDigest: s.s5.Carried.ImpactRangeDigest, AssuranceDigest: s.s5.Carried.AssuranceDigest, SpecReview: sr, InFlight: s.s5.Range.InFlight}
	var att Attachment
	if err := json.Unmarshal(s.s3.Attachment, &att); err != nil {
		return err
	}
	packet.Provenance = att
	packetRaw, _ := json.MarshalIndent(packet, "", "  ")
	if err := writeFile(filepath.Join(dir, "review-packet.json"), packetRaw); err != nil {
		return err
	}
	s.packet = packet
	pre := s.Preflight(ctx, preflightInput{Chain: *s.Chain, BaselineDir: s.liveBaseline, Origin: s.s4.Origin, Integ: s.s4.Integ,
		Receipt: s.s4.Receipt, SourcePath: s.s1.SourcePath, Range: s.s5.Range, Matrix: rows})
	if len(pre) > 0 {
		return fmt.Errorf("the signing preflight refused: %v", pre)
	}
	reason := reasonFromPacket(packet)
	payload := s.payloadFor(s.s3.Fingerprint, reviewFP, s.s2.Revision, reason, opensshwitness.DecisionAccept)
	sg, err := s.newSigner(ctx, dir, payload)
	if err != nil {
		return err
	}
	sig, err := opensshwitness.Sign("", sg.Key, payload)
	if err != nil {
		return err
	}
	ev, err := s.verifySignature(sg, signerPrincipal, payload, sig)
	if err != nil {
		return err
	}
	if ev.GateStatus != opensshwitness.GateAuthorityQualified {
		return fmt.Errorf("the signature did not qualify: %s", ev.Diagnosis)
	}
	s.sg, s.payload, s.signature = sg, payload, sig
	ns, _ := payload.Namespace()
	msg, _ := payload.Message()
	bundle := SubmissionBundle{Schema: bundleSchema, Target: "tools/reference-shared-acceptance (replaceable: order-workflow common acceptance)",
		CandidateJSON: s.s3.Candidate, CandidateDigest: s.s3.Carried.CandidateDigest, CandidateFingerprint: s.s3.Fingerprint,
		BaselineRevision: s.s2.Revision, Provenance: att, ReviewFingerprint: reviewFP, PacketDigest: digestBytes(packetRaw),
		Decision: SignedDecision{Payload: payload, Principal: signerPrincipal, Namespace: ns, Signature: sig, Evidence: ev, Reason: reason, ReasonText: reasonText(reason)},
		Acceptance: AcceptanceState{Performed: false, Status: "submitted-not-accepted",
			Reason: "this sample submits a candidate and stops; acceptance belongs to the common acceptance path"}}
	if p := VerifyBundle(bundle); len(p) > 0 {
		return fmt.Errorf("bundle: %v", p)
	}
	s.bundle = bundle
	bundleRaw, _ := json.MarshalIndent(bundle, "", "  ")
	if err := writeFile(filepath.Join(dir, "submission-bundle.json"), bundleRaw); err != nil {
		return err
	}
	rc, err := sharedAcceptanceSubmitter{s: s, Baseline: s.liveBaseline, Work: dir}.Submit(ctx, bundle)
	if err != nil {
		return err
	}
	if len(rc.Problems) > 0 || rc.Accepted {
		return fmt.Errorf("submission: %+v", rc)
	}
	s.submission = rc
	nc, _ := notConnectedSubmitter{}.Submit(ctx, bundle)
	s.orderWorkflow = nc
	c := s.s5.Carried
	c.ReviewFingerprint, c.PacketDigest = reviewFP, bundle.PacketDigest
	c.ReasonDigest = payload.DecisionReasonDigest
	c.MessageDigest, c.SignatureDigest = ev.MessageDigest, ev.SignatureDigest
	// Stage 6 reads the provenance back from the bundle it signed.
	c.ProvenanceState, c.Unresolved = bundle.Decision.Reason.ProvenanceState, &bundle.Decision.Reason.Unresolved
	c.EvidenceFingerprint, c.SourceDigest = bundle.Decision.Reason.EvidenceFingerprint, bundle.Decision.Reason.SourceDigest
	c.ContractDigest = bundle.Decision.Reason.ContractDigest
	c.AssuranceDigest, c.ImpactRangeDigest = bundle.Decision.Reason.AssuranceDigest, bundle.Decision.Reason.ImpactRangeDigest
	c.CandidateFingerprint, c.CandidateDigest, c.BaselineRevision = bundle.CandidateFingerprint, bundle.CandidateDigest, bundle.BaselineRevision
	return s.Chain.Append(6, c, map[string]any{
		"checkers": rows, "soundnessBasis": packet.SoundnessBasis, "reason": bundle.Decision.ReasonText, "namespace": ns, "message": msg,
		"signatureEvidence": ev, "submission": rc, "orderWorkflowSubmission": nc, "acceptance": bundle.Acceptance,
		"decisionMeaning": "the decider's sign-off on submitting the candidate; no source was written and nothing was accepted",
	})
}
