package orderruledesign

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"tei/tools/internal/opensshwitness"
)

// forge rewrites one record's carried values and relinks, as a forger would.
// Every record's own link digest then holds, so each stage alone cannot tell.
func forge(c Chain, stage int, mut func(*Carried)) Chain {
	f := c.Clone()
	for i := range f.Records {
		if f.Records[i].Stage == stage {
			mut(&f.Records[i].Carried)
		}
	}
	f.Relink()
	return f
}

func (s *Sample) prefixChain() Chain {
	c := s.Chain.Clone()
	if len(c.Records) > 5 {
		c.Records = c.Records[:5]
	}
	return c
}

func (s *Sample) preInput() preflightInput {
	return preflightInput{Chain: s.prefixChain(), BaselineDir: s.liveBaseline, Origin: s.s4.Origin, Integ: s.s4.Integ,
		Receipt: s.s4.Receipt, SourcePath: s.s1.SourcePath, Range: s.s5.Range, Matrix: s.matrix}
}

func anyHas(list []string, sub string) bool {
	for _, v := range list {
		if strings.Contains(v, sub) {
			return true
		}
	}
	return false
}

func (s *Sample) runFaults(ctx context.Context) error {
	for _, f := range []func(context.Context) error{
		s.faultsThrough, s.faultsProvenance, s.faultsChecker, s.faultsReuse, s.faultsMix, s.faultsWF02, s.faultsWF01,
	} {
		if err := f(ctx); err != nil {
			return err
		}
	}
	return nil
}

// through- : replace one identity, keep every link digest valid. Each
// stage's own check passes; only the through check fails.
func (s *Sample) faultsThrough(ctx context.Context) error {
	other := "sha256:" + strings.Repeat("ab", 32)
	cases := []struct {
		name  string
		stage int
		mut   func(*Carried)
	}{
		{"candidate-fingerprint-at-signature", 6, func(c *Carried) { c.CandidateFingerprint = other }},
		{"candidate-digest-at-handover", 4, func(c *Carried) { c.CandidateDigest = other }},
		{"baseline-revision-at-assurance", 5, func(c *Carried) { c.BaselineRevision = other }},
		{"task-head-at-handover", 4, func(c *Carried) { c.TaskHead = strings.Repeat("0", 40) }},
		{"integration-tree-at-impact", 5, func(c *Carried) { c.IntegrationTree = strings.Repeat("1", 40) }},
		{"assurance-digest-at-signature", 6, func(c *Carried) { c.AssuranceDigest = other }},
		{"source-digest-at-candidate", 3, func(c *Carried) { c.SourceDigest = other }},
		{"trial-revision-is-baseline", 5, func(c *Carried) { c.TrialRevision = c.BaselineRevision }},
		{"impact-range-at-signature", 6, func(c *Carried) { c.ImpactRangeDigest = other }},
		{"main-commit-at-impact", 5, func(c *Carried) { c.MainCommit = strings.Repeat("2", 40) }},
	}
	for _, tc := range cases {
		f := forge(*s.Chain, tc.stage, tc.mut)
		th := VerifyThrough(f)
		s.addFault("through-"+tc.name, "通し", "1-6", standaloneOK(f) && !th.Complete && len(th.Mismatches) > 0,
			map[string]any{"eachStageLinkHolds": standaloneOK(f), "throughComplete": th.Complete, "mismatches": th.Mismatches})
	}
	// A stage that is missing or out of order is also not a through.
	drop := s.Chain.Clone()
	drop.Records = append(drop.Records[:2], drop.Records[3:]...)
	drop.Relink()
	th := VerifyThrough(drop)
	s.addFault("through-stage-dropped", "通し", "3", !th.Complete && len(th.MissingStages) > 0, map[string]any{"missing": th.MissingStages})
	return nil
}

// provenance- : legacy material faults.
func (s *Sample) faultsProvenance(ctx context.Context) error {
	dir := filepath.Join(s.Work, "faults", "provenance")
	// The material gains descriptions the contract cannot handle.
	more, err := s.runStage1(ctx, "more-unsupported", sourceReplaced, filepath.Join(dir, "more"))
	if err != nil {
		return err
	}
	main := s.s1.Carried
	grew := more.Carried.Unresolved.SourceObservation > main.Unresolved.SourceObservation
	f := forge(*s.Chain, 1, func(c *Carried) { *c = more.Carried })
	th := VerifyThrough(f)
	s.addFault("provenance-unsupported-description-grows-count", "固有", "1,3,6", grew && more.Carried.ProvenanceState == StateBound && !th.Complete,
		map[string]any{"main": main.Unresolved, "more": more.Carried.Unresolved, "stateStaysBound": more.Carried.ProvenanceState, "throughComplete": th.Complete})
	// The material is replaced after stage 1.
	rep := filepath.Join(dir, "replaced.xead")
	if err := writeFile(rep, []byte(sourceReplaced)); err != nil {
		return err
	}
	res, err := runCLI(ctx, s.Tools.Env, s.Tools.CrossView, dir, "-input", rep, "-contract", s.s1.ContractPath, "-expect-accounting", s.s1.AccountingPath)
	if err != nil {
		return err
	}
	in := s.preInput()
	in.SourcePath = rep
	pre := s.Preflight(ctx, in)
	s.addFault("provenance-source-replaced-after-stage1", "固有", "1,6", res.Exit != 0 && anyHas(pre, "legacy material changed"),
		map[string]any{"crossViewExpectAccountingExit": res.Exit, "preflight": pre})
	// No table node, and a table node no path reaches.
	for _, v := range []struct{ id, src, want string }{{"no-target", sourceNoTarget, StateNoTarget}, {"table-only", sourceTableOnly, StateTableOnly}} {
		o, err := s.runStage1(ctx, v.id, v.src, filepath.Join(dir, v.id))
		if err != nil {
			return err
		}
		cand, err := s.buildCandidate(ctx, o, s.s2, filepath.Join(dir, v.id, "s3"), 4)
		if err != nil {
			return err
		}
		var att Attachment
		_ = json.Unmarshal(cand.Attachment, &att)
		hasEvidence := strings.Contains(string(cand.Candidate), "rmt-transcription")
		pk := s.packet
		pk.Provenance = att
		reason := reasonFromPacket(pk)
		obs := map[string]any{"state": o.Carried.ProvenanceState, "evidenceInCandidate": hasEvidence, "reasonState": reason.ProvenanceState,
			"unresolved": reason.Unresolved, "sameReasonDigestAsMain": mustDigest(reason) == s.payload.DecisionReasonDigest}
		ok := o.Carried.ProvenanceState == v.want && reason.ProvenanceState == v.want && reason.Unresolved == *o.Carried.Unresolved &&
			mustDigest(reason) != s.payload.DecisionReasonDigest && (v.want == StateNoTarget) == !hasEvidence
		if v.want == StateNoTarget {
			res, err := runCLI(ctx, s.Tools.Env, s.Tools.SourceAdapter, dir, "-format", "xead", "-input", o.SourcePath, "-contract", o.ContractPath, "-candidate-evidence-node", "99")
			if err != nil {
				return err
			}
			ok = ok && res.Exit != 0 && o.Carried.EvidenceFingerprint == "none"
			obs["adapterRefusesAnInventedNode"] = res.Exit != 0
		}
		s.addFault("provenance-"+v.id+"-carried-to-the-reason", "固有", "1,3,6", ok, obs)
	}
	// Dropping the count or the state at a later stage is caught.
	zero := forge(*s.Chain, 3, func(c *Carried) { c.Unresolved = &Unresolved{} })
	zero = forge(zero, 6, func(c *Carried) { c.ProvenanceState = StateTableOnly })
	th = VerifyThrough(zero)
	s.addFault("provenance-state-or-count-changed-at-a-later-stage", "固有", "3,6", standaloneOK(zero) && !th.Complete,
		map[string]any{"mismatches": th.Mismatches})
	// The signature binds the reason: a reason with a smaller count does not verify.
	pl := s.payload
	smaller := s.bundle.Decision.Reason
	smaller.Unresolved.Total = 0
	pl.DecisionReasonDigest = mustDigest(smaller)
	ev, verr := s.verifySignature(s.sg, signerPrincipal, pl, s.signature)
	b := s.bundle
	b.Decision.Reason = smaller
	b.Decision.Payload = pl
	bp := VerifyBundle(b)
	s.addFault("provenance-signature-binds-the-reason", "固有", "6", (verr != nil || ev.GateStatus != opensshwitness.GateAuthorityQualified) && len(bp) > 0,
		map[string]any{"verifyError": errString(verr), "gate": ev.GateStatus, "bundleProblems": bp})
	return nil
}

// checker- : the list of checker outputs.
func (s *Sample) faultsChecker(ctx context.Context) error {
	want := map[string]string{
		"tools/reference-business-dfd@R0": ClassOutOfScope, "tools/reference-business-dfd@R1": ClassOutOfScope,
		"tools/reference-subsystem-cud@R0": ClassInsufficient, "tools/reference-subsystem-cud@R1": ClassInsufficient,
		"tools/reference-purpose-preflight:order-approval-fixture": ClassOutOfScope,
		"tools/reference-purpose-preflight@R0":                     ClassInsufficient,
		"tools/reference-purpose-preflight@R1":                     ClassEvaluated,
		"tools/reference-spec-review":                              ClassEvaluated,
		"tools/reference-change-assurance":                         ClassEvaluated,
		"tools/reference-shared-acceptance:preview":                ClassEvaluated,
	}
	got := map[string]string{}
	for _, r := range s.matrix {
		got[r.Checker] = r.Class
	}
	same := true
	for k, v := range want {
		same = same && got[k] == v
	}
	s.addFault("checker-each-class-follows-the-output", "固有", "2,5,6", same && len(VerifyMatrix(s.matrix)) == 0,
		map[string]any{"classes": got})
	var noFinding, dropped, relabeled []CheckerRow
	for _, r := range s.matrix {
		if noFindingVerdicts[r.Verdict] {
			noFinding = append(noFinding, r)
		}
		if r.Checker != "tools/reference-business-dfd@R0" {
			dropped = append(dropped, r)
		}
		r.Class = ClassOutOfScope
		relabeled = append(relabeled, r)
	}
	p1, p2, p3 := VerifyMatrix(noFinding), VerifyMatrix(dropped), VerifyMatrix(relabeled)
	s.addFault("checker-no-finding-only-list-refused", "固有", "6", anyHas(p1, "only collects") && len(noFinding) > 0,
		map[string]any{"rows": len(noFinding), "problems": p1})
	s.addFault("checker-list-without-an-out-of-scope-checker-refused", "固有", "6", anyHas(p2, "not listed"), map[string]any{"problems": p2})
	s.addFault("checker-list-with-nothing-evaluated-refused", "固有", "6", anyHas(p3, "no checker evaluated"), map[string]any{"problems": p3})
	var rest []CheckerRow
	nfOutside := 0
	for _, r := range s.matrix {
		if r.Class != ClassEvaluated {
			rest = append(rest, r)
			if noFindingVerdicts[r.Verdict] {
				nfOutside++
			}
		}
	}
	s.addFault("checker-out-of-scope-no-finding-is-not-a-basis", "固有", "2,5,6", SoundnessBasis(rest) == 0 && nfOutside > 0 && SoundnessBasis(s.matrix) > 0,
		map[string]any{"noFindingOutsideScope": nfOutside, "basisOfThoseRows": SoundnessBasis(rest), "basisOfEvaluatedRows": SoundnessBasis(s.matrix)})
	return nil
}

// reuse- : past evidence and the trial version.
func (s *Sample) faultsReuse(ctx context.Context) error {
	main := s.s5.Range
	reuse := 0
	for _, c := range main.Reuse {
		if c.Action == "reuse-candidate" {
			reuse++
		}
	}
	bad := func(mut func(*ImpactRange)) []string {
		r := main
		r.Reuse = append([]ReuseCandidate{}, main.Reuse...)
		mut(&r)
		return VerifyImpactRange(r, inFlightCases(main.Baseline, nil))
	}
	p1 := bad(func(r *ImpactRange) { r.Reuse[0].Pass = true })
	p2 := bad(func(r *ImpactRange) { r.ExecAllowed = true })
	p3 := bad(func(r *ImpactRange) { r.TrialState, r.ProjectAccepted = "accepted", true })
	s.addFault("reuse-candidate-is-not-a-pass-or-a-permission", "固有", "5",
		reuse > 0 && len(VerifyImpactRange(main, inFlightCases(main.Baseline, nil))) == 0 && len(p1) > 0 && len(p2) > 0 && !s.assurance.Exec,
		map[string]any{"reuseCandidates": reuse, "passRead": p1, "executionAllowedRead": p2, "assuranceExecutionAllowed": s.assurance.Exec})
	trialDecisions, err := treeDigest(filepath.Join(s.s5.R1, ".tei", "decisions"))
	if err != nil {
		return err
	}
	now, err := s.protectedAssets()
	if err != nil {
		return err
	}
	s.addFault("reuse-trial-version-is-kept-apart-from-acceptance", "固有", "5",
		len(p3) > 0 && main.TrialState == TrialState && !main.ProjectAccepted && trialDecisions != s.protectedBefore["reference/workspace/.tei/decisions"] &&
			now["reference/workspace/.tei/decisions"] == s.protectedBefore["reference/workspace/.tei/decisions"],
		map[string]any{"trialState": main.TrialState, "trialWroteItsOwnDecisionInTheCopy": trialDecisions != s.protectedBefore["reference/workspace/.tei/decisions"],
			"sharedDecisionsUnchanged": now["reference/workspace/.tei/decisions"] == s.protectedBefore["reference/workspace/.tei/decisions"], "problemsWhenRelabeled": p3})
	return nil
}

// mix- : P-MIX-01.
func (s *Sample) faultsMix(ctx context.Context) error {
	a := s.assurance
	derivs := []Derivation{{executionTargets[0].Target, executionTargets[0].Ref, s.s3.Fingerprint},
		{executionTargets[1].Target, executionTargets[1].Ref, s.s3.Fingerprint},
		{executionTargets[2].Target, executionTargets[2].Ref, s.s2.Revision}}
	r := BuildImpactRange(s.s2.Revision, s.s3.Fingerprint, s.s5.Carried.TrialRevision, a, derivs, s.s5.Range.InFlight, s.s5.Range.Reuse)
	var open []string
	for _, t := range r.Targets {
		if !t.Derived && t.Recheck {
			open = append(open, t.Target)
		}
	}
	_, affected := expansionStep()
	short := r
	short.Targets = r.Targets[:2]
	ps := VerifyImpactRange(short, inFlightCases(r.Baseline, nil))
	hidden := r
	hidden.Targets = append([]TargetEntry{}, r.Targets...)
	hidden.Targets[2].Recheck = false
	ph := VerifyImpactRange(hidden, inFlightCases(r.Baseline, nil))
	s.addFault("mix-two-of-three-derived-leaves-the-third-in-recheck", "P-MIX-01", "5",
		len(r.Targets) == 3 && len(open) == 1 && open[0] == "batch" && len(VerifyImpactRange(r, inFlightCases(r.Baseline, nil))) == 0 &&
			contains(affected, executionTargets[2].Ref) && anyHas(ps, "execution targets") && anyHas(ph, "not derived"),
		map[string]any{"targets": r.Targets, "stillInRecheck": open, "expansionAffectedRefs": affected, "twoTargetRangeProblems": ps,
			"hiddenRecheckProblems": ph, "assuranceEmitsExecutionTargets": a.HasTargetKeys})
	return nil
}

// wf02- : in-flight cases at the old version.
func (s *Sample) faultsWF02(ctx context.Context) error {
	main := s.s5.Range
	listed := len(main.InFlight) == 3
	for _, c := range main.InFlight {
		listed = listed && c.StartedVersion == s.s2.Revision
	}
	missing := main
	missing.InFlight = main.InFlight[:2]
	pm := VerifyImpactRange(missing, inFlightCases(main.Baseline, nil))
	s.addFault("wf02-in-flight-cases-and-versions-are-in-the-range", "P-WF-02", "5", listed && anyHas(pm, "missing from the range"),
		map[string]any{"cases": main.InFlight, "rangeWithoutACase": pm})
	// Undecided handling stops signing; fully decided does not.
	none := s.preInput()
	none.Range.InFlight = inFlightCases(s.s2.Revision, nil)
	one := s.preInput()
	one.Range.InFlight = inFlightCases(s.s2.Revision, map[string]string{"SO-1001": "finish-on-r0", "SO-1002": "migrate"})
	pn, p1 := s.Preflight(ctx, none), s.Preflight(ctx, one)
	pok := s.Preflight(ctx, s.preInput())
	s.addFault("wf02-undecided-case-stops-signing", "P-WF-02", "6", anyHas(pn, "no decided handling") && anyHas(p1, "SO-1003") && len(pok) == 0,
		map[string]any{"noneDecided": pn, "oneUndecided": p1, "allDecided": pok})
	return nil
}

func (s *Sample) otherCandidate(rev string) []byte {
	b, _ := json.Marshal(map[string]any{"schema": "tei.reference.change-candidate/v0", "id": "other-party-candidate", "baseModelRevision": rev,
		"origin": map[string]any{"kind": "cli"}, "intent": "another party's change that moves the baseline",
		"delta": []any{map[string]any{"op": "add-validation", "namespace": "sales", "id": "sales.order.total_positive", "kind": "positive", "attribute": "sales.order.total"}}})
	return b
}

// wf01- : the baseline or main moves, or a signature is reused.
func (s *Sample) wf01Advance(ctx context.Context) error {
	dir := filepath.Join(s.Work, "faults", "wf01")
	rev, err := s.revisionOf(ctx, s.liveBaseline)
	if err != nil {
		return err
	}
	cand := filepath.Join(dir, "other.json")
	if err := writeFile(cand, s.otherCandidate(rev)); err != nil {
		return err
	}
	rv, err := s.Tools.mustRun(ctx, s.Tools.SharedAcceptance, s.Repo, "review", "-workspace", s.liveBaseline, "-candidate", cand, "-format", "json")
	if err != nil {
		return err
	}
	var r struct {
		ReviewFingerprint string `json:"reviewFingerprint"`
	}
	_ = json.Unmarshal(rv, &r)
	_, err = s.Tools.mustRun(ctx, s.Tools.SharedAcceptance, s.Repo, "accept", "-workspace", s.liveBaseline, "-candidate", cand, "-expected-review-fingerprint", r.ReviewFingerprint)
	return err
}

func (s *Sample) faultsWF01(ctx context.Context) error {
	// 1: another party's acceptance moves the baseline after stage 5.
	if err := s.wf01Advance(ctx); err != nil {
		return err
	}
	pre := s.Preflight(ctx, s.preInput())
	signed := len(pre) == 0
	s.addFault("wf01-baseline-advanced-after-stage5-stops-signing", "P-WF-01", "5,6", anyHas(pre, "the baseline moved") && !signed,
		map[string]any{"preflight": pre, "signed": signed})
	if err := copyBack(s.liveBaseline, filepath.Join(s.Repo, "reference", "workspace")); err != nil {
		return err
	}
	// 2: main moves after stage 4.
	oldMain := s.s4.Receipt.MainCommit
	if _, err := s.Tools.git(ctx, s.s4.Origin, "commit", "-q", "--allow-empty", "-m", "another party's integration"); err != nil {
		return err
	}
	probs := s.ReverifyReceipt(ctx, s.s4.Receipt, s.s4.Origin, s.s4.Integ)
	pre2 := s.Preflight(ctx, s.preInput())
	if _, err := s.Tools.git(ctx, s.s4.Origin, "reset", "-q", "--hard", oldMain); err != nil {
		return err
	}
	s.addFault("wf01-main-advanced-after-stage4-fails-receipt-reverify", "P-WF-01", "4,6", anyHas(probs, "main advanced") && anyHas(pre2, "main advanced"),
		map[string]any{"reverify": probs, "preflight": pre2})
	// 3: the working history is replaced.
	if _, err := s.Tools.git(ctx, s.s4.Integ, "replace", "-f", s.s4.Receipt.IntegrationHead, oldMain); err != nil {
		return err
	}
	probs = s.ReverifyReceipt(ctx, s.s4.Receipt, s.s4.Origin, s.s4.Integ)
	if _, err := s.Tools.git(ctx, s.s4.Integ, "replace", "-d", s.s4.Receipt.IntegrationHead); err != nil {
		return err
	}
	s.addFault("wf01-history-replaced-fails-receipt-reverify", "P-WF-01", "4", anyHas(probs, "refs/replace"), map[string]any{"reverify": probs})
	if p := s.ReverifyReceipt(ctx, s.s4.Receipt, s.s4.Origin, s.s4.Integ); len(p) > 0 {
		return fmt.Errorf("the repositories were not restored: %v", p)
	}
	// 4: the signature is reused.
	reuse := func(id string, principal string, mut func(*opensshwitness.Payload)) {
		pl := s.payload
		mut(&pl)
		ev, err := s.verifySignature(s.sg, principal, pl, s.signature)
		refused := err != nil || ev.GateStatus != opensshwitness.GateAuthorityQualified
		s.addFault("wf01-signature-reuse-"+id, "P-WF-01", "6", refused,
			map[string]any{"error": errString(err), "gate": ev.GateStatus, "diagnosis": ev.Diagnosis})
	}
	reuse("other-candidate", signerPrincipal, func(p *opensshwitness.Payload) { p.CandidateFingerprint = "sha256:" + strings.Repeat("cd", 32) })
	reuse("other-signer", "someone-else@example.test", func(*opensshwitness.Payload) {})
	reuse("as-reject", signerPrincipal, func(p *opensshwitness.Payload) { p.Decision = opensshwitness.DecisionReject })
	reuse("other-baseline", signerPrincipal, func(p *opensshwitness.Payload) { p.BaseModelRevision = "sha256:" + strings.Repeat("ef", 32) })
	// 5: the acceptance path recomputes a different value than the signed one.
	b := s.bundle
	changed := []byte(strings.ReplaceAll(string(b.CandidateJSON), "unit_price_positive", "unit_price_strict"))
	b.CandidateJSON, b.CandidateDigest = changed, digestBytes(changed)
	rc, err := sharedAcceptanceSubmitter{s: s, Baseline: s.liveBaseline, Work: filepath.Join(s.Work, "faults", "wf01")}.Submit(ctx, b)
	vb := VerifyBundle(s.bundle)
	altered := s.bundle
	altered.CandidateJSON = changed
	s.addFault("wf01-recomputed-value-mismatch-stops-the-submission", "P-WF-01", "6",
		err == nil && anyHas(rc.Problems, "candidate fingerprint") && len(vb) == 0 && len(VerifyBundle(altered)) > 0 && !rc.Accepted,
		map[string]any{"submissionProblems": rc.Problems, "alteredBundleProblems": VerifyBundle(altered), "unalteredBundleProblems": vb})
	return nil
}

// copyBack restores a copy from the shared baseline.
func copyBack(dst, src string) error {
	if err := removeAll(dst); err != nil {
		return err
	}
	return copyTree(src, dst)
}
