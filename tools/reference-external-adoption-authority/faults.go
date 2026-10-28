package externaladoptionauthority

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tei/internal/plugin"
	semanticbuildbridge "tei/reference/semantic-build-bridge"
	semanticcompositionpreflight "tei/reference/semantic-composition-preflight"
	semanticdistribution "tei/reference/semantic-distribution"
	authoritytrust "tei/tools/reference-authority-evidence-trust/trust"
)

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *Sample) faultDir(name string) string { return filepath.Join(s.Work, "faults", name) }

// faultPGOV01 changes one of the four dependency blobs in a copy of the
// external side, one at a time, and moves the revision alone.
func (s *Sample) faultPGOV01(ctx context.Context) error {
	for i, name := range pinnedFiles {
		ext := s.pinned
		ext.Blobs = map[string]string{}
		for k, v := range s.pinned.Blobs {
			ext.Blobs[k] = v
		}
		ext.Blobs[name] = strings.Repeat("0", 39) + string(rune('1'+i))
		dir := s.faultDir("pgov01-blob-" + name)
		path, err := WriteExternalCopy(ext, filepath.Join(dir, "external"))
		if err != nil {
			return err
		}
		out, err := runStage1(ctx, s.Tools, s.Repo, filepath.Join(dir, "stage1"), path)
		stopped := errors.Is(err, ErrInputMismatch)
		s.addFault("pgov01-blob-"+name, "P-GOV-01", "1",
			stopped && !exists(filepath.Join(dir, "stage1", "loss-record.json")) && !exists(out.ReportsDir) && out.Carried.LossRecordDigest == "",
			map[string]any{"changedBlob": name, "stopped": stopped, "error": errString(err),
				"lossRecordWritten": exists(filepath.Join(dir, "stage1", "loss-record.json")), "blobMismatches": out.Pin.BlobMismatches})
	}
	ext := s.pinned
	ext.Revision = strings.Repeat("f", 40)
	dir := s.faultDir("pgov01-revision-only")
	path, err := WriteExternalCopy(ext, filepath.Join(dir, "external"))
	if err != nil {
		return err
	}
	out, err := runStage1(ctx, s.Tools, s.Repo, filepath.Join(dir, "stage1"), path)
	same := err == nil && out.Carried.MappingReportDigest == s.s1.Carried.MappingReportDigest &&
		out.Carried.LossRecordDigest == s.s1.Carried.LossRecordDigest
	s.addFault("pgov01-revision-only", "P-GOV-01", "1", same && out.Pin.RevisionAdvanced,
		map[string]any{"error": errString(err), "revisionAdvanced": out.Pin.RevisionAdvanced,
			"sameReportAndLossDigests": same, "note": "freshness is bound to the four blobs; the moved revision is recorded, not a stop"})
	return nil
}

func writeRival(dir string, src string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	var d map[string]any
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	d["id"] = ExtensionID + "-rival"
	out, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(filepath.Join(dir), append(out, '\n'))
}

type claimRun struct {
	Variant     string   `json:"variant"`
	Exit        int      `json:"exit"`
	Unresolved  []string `json:"unresolved,omitempty"`
	Error       string   `json:"error,omitempty"`
	LockWritten bool     `json:"lockWritten"`
}

func (s *Sample) claimRun(ctx context.Context, variant string, ci catalogInputs, lock string) (claimRun, error) {
	// First without a lock request, to read the catalog the CLI prints;
	// then with one, to see whether a lock is written.
	res, err := runCLI(ctx, s.Tools.Catalog, s.Repo, ci.args()...)
	if err != nil {
		return claimRun{}, err
	}
	withLock, err := runCLI(ctx, s.Tools.Catalog, s.Repo, append(ci.args(), "-semantic-lock-out", lock)...)
	if err != nil {
		return claimRun{}, err
	}
	r := claimRun{Variant: variant, Exit: res.Exit, Error: res.Stderr, LockWritten: exists(lock) || withLock.Exit == 0}
	var cat catalogOut
	if json.Unmarshal(res.Stdout, &cat) == nil {
		for _, u := range cat.Unresolved {
			r.Unresolved = append(r.Unresolved, u.Code+": "+u.Detail)
		}
		sort.Strings(r.Unresolved)
	}
	return r, nil
}

// faultPINT03 lets two providers claim the extension namespace.
func (s *Sample) faultPINT03(ctx context.Context) error {
	base := s.s2.Inputs
	rivalReq := ExtensionID + "-rival@1"

	// Two descriptors claim the namespace and no authority decides: the
	// order of -require and the order of the descriptor files are swapped.
	var runs []claimRun
	for _, v := range []struct {
		name, file string
		reqs       []string
	}{
		{"rival-file-first/require-original-first", "aaa-rival.json", append(append([]string{}, base.Requires...), rivalReq)},
		{"rival-file-last/require-rival-first", "zzz-rival.json", append([]string{rivalReq}, base.Requires...)},
	} {
		dir := s.faultDir("pint03-no-authority-" + strings.SplitN(v.name, "/", 2)[0])
		ci := base
		ci.Extensions = filepath.Join(dir, "extensions")
		if err := copyTree(base.Extensions, ci.Extensions); err != nil {
			return err
		}
		if err := writeRival(filepath.Join(ci.Extensions, v.file), filepath.Join(base.Extensions, "sui-acyclic-1.0.0.json")); err != nil {
			return err
		}
		ci.Requires, ci.Authorities, ci.Grants, ci.Bindings = v.reqs, nil, nil, nil
		r, err := s.claimRun(ctx, v.name, ci, filepath.Join(dir, "semantic-lock.json"))
		if err != nil {
			return err
		}
		runs = append(runs, r)
	}
	sameOutcome := strings.Join(runs[0].Unresolved, "|") == strings.Join(runs[1].Unresolved, "|")
	conflict := false
	for _, u := range runs[0].Unresolved {
		conflict = conflict || strings.HasPrefix(u, "namespace-ownership-conflict") && strings.Contains(u, ExtensionNamespace)
	}
	refused := true
	for _, r := range runs {
		refused = refused && r.Exit != 0 && !r.LockWritten
	}
	s.addFault("pint03-two-claimants-without-authority", "P-INT-03", "2", refused && conflict && sameOutcome,
		map[string]any{"runs": runs, "sameUnresolvedInBothOrders": sameOutcome})

	// Two trusted authority statements name different owners.
	dir := s.faultDir("pint03-two-authorities")
	ext := filepath.Join(dir, "extensions")
	if err := copyTree(base.Extensions, ext); err != nil {
		return err
	}
	if err := writeRival(filepath.Join(ext, "rival.json"), filepath.Join(base.Extensions, "sui-acyclic-1.0.0.json")); err != nil {
		return err
	}
	rivalIssuer := authoritytrust.Issuer{ID: "rival.namespace_publisher", Version: "1.0.0"}
	rivalAuth, err := s.writeJSON(filepath.Join(dir, "rival-authority.json"), map[string]any{
		"namespace": ExtensionNamespace, "extension": ExtensionID + "-rival", "issuer": rivalIssuer,
		"evidenceFingerprint": digestBytes([]byte("synthetic rival authority")),
	})
	if err != nil {
		return err
	}
	rivalGrant, err := NewGrant(OwnerAuthorityClass, OwnerScope(ExtensionNamespace), rivalIssuer, "rival policy", true)
	if err != nil {
		return err
	}
	grants, err := writeOwnerGrants(filepath.Join(dir, "grants"), append(ownerList(s.p1.OwnerGrants...), rivalGrant))
	if err != nil {
		return err
	}
	runs = nil
	for i, auths := range [][]string{{base.Authorities[0], rivalAuth}, {rivalAuth, base.Authorities[0]}} {
		ci := base
		ci.Extensions, ci.Authorities, ci.Grants, ci.Bindings = ext, auths, grants, nil
		ci.Requires = append(append([]string{}, base.Requires...), rivalReq)
		r, err := s.claimRun(ctx, []string{"original-first", "rival-first"}[i], ci, filepath.Join(dir, "lock-"+[]string{"a", "b"}[i]+".json"))
		if err != nil {
			return err
		}
		runs = append(runs, r)
	}
	refused = true
	names := func(msg string) string {
		var got []string
		for _, n := range []string{ExtensionID + "-rival", ExtensionID} {
			if strings.Contains(msg, n+" and") || strings.HasSuffix(strings.TrimSpace(msg), n) {
				got = append(got, n)
			}
		}
		sort.Strings(got)
		return strings.Join(got, ",")
	}
	for _, r := range runs {
		refused = refused && r.Exit != 0 && !r.LockWritten && strings.Contains(r.Error, "conflicting authority grants")
	}
	sameNames := names(runs[0].Error) == names(runs[1].Error) && names(runs[0].Error) != ""
	s.addFault("pint03-two-trusted-authorities", "P-INT-03", "2", refused && sameNames,
		map[string]any{"runs": runs, "sameClaimantsNamedInBothOrders": sameNames,
			"messageOrderFollowsInput": runs[0].Error != runs[1].Error})

	// Two grants for the same issuer and scope: the trust judgment itself is
	// ambiguous.
	dir = s.faultDir("pint03-ambiguous-grant")
	second, err := NewGrant(OwnerAuthorityClass, OwnerScope(ExtensionNamespace), OwnerIssuer, "owner policy duplicate", true)
	if err != nil {
		return err
	}
	runs = nil
	for i, order := range [][]authoritytrust.Grant{{s.p1.OwnerGrants[0], second}, {second, s.p1.OwnerGrants[0]}} {
		g, err := writeOwnerGrants(filepath.Join(dir, []string{"a", "b"}[i]), order)
		if err != nil {
			return err
		}
		ci := base
		ci.Grants, ci.Bindings = g, nil
		r, err := s.claimRun(ctx, []string{"first-grant-first", "duplicate-first"}[i], ci, filepath.Join(dir, "lock-"+[]string{"a", "b"}[i]+".json"))
		if err != nil {
			return err
		}
		runs = append(runs, r)
	}
	refused = true
	for _, r := range runs {
		refused = refused && r.Exit != 0 && !r.LockWritten && strings.Contains(r.Error, "trust grant is ambiguous")
	}
	s.addFault("pint03-ambiguous-trust-grant", "P-INT-03", "2", refused && runs[0].Error == runs[1].Error,
		map[string]any{"runs": runs, "sameErrorInBothOrders": runs[0].Error == runs[1].Error})
	return nil
}

func mustGrant(class, scope string, issuer authoritytrust.Issuer, policy string, active bool) authoritytrust.Grant {
	g, err := NewGrant(class, scope, issuer, policy, active)
	if err != nil {
		panic(err)
	}
	return g
}

// faultPOPS02 revokes and rotates trust at later logical points and judges
// stages 2, 3 and 4 there.
func (s *Sample) faultPOPS02(ctx context.Context) error {
	k1, _ := s.p1.Key(SignerRef)
	k2 := NewSignerKey(SignerRef, "2.0.0", 2)
	ownerOn := s.p1.OwnerGrants
	ownerOff := ownerList(mustGrant(OwnerAuthorityClass, OwnerScope(ExtensionNamespace), OwnerIssuer, "owner policy 1", false))
	signerOn := s.p1.SignerGrants
	signerOff := ownerList(mustGrant(SignerAuthorityClass, OwnerScope("sui"), issuerOf(k1), "signer policy 1", false))
	pOwnerRevoked := s.Trust.Add(TrustPoint{Label: "owner-grant-revoked", OwnerGrants: ownerOff, SignerGrants: signerOn}.WithKeys(k1))
	pOwnerAbsent := s.Trust.Add(TrustPoint{Label: "owner-grant-absent", SignerGrants: signerOn}.WithKeys(k1))
	pSignerRevoked := s.Trust.Add(TrustPoint{Label: "signer-grant-revoked", OwnerGrants: ownerOn, SignerGrants: signerOff}.WithKeys(k1))
	pRotated := s.Trust.Add(TrustPoint{Label: "signer-key-rotated", OwnerGrants: ownerOn,
		SignerGrants: append(ownerList(signerOff...), mustGrant(SignerAuthorityClass, OwnerScope("sui"), issuerOf(k2), "signer policy 2", true))}.WithKeys(k2))
	pRestored := s.Trust.Add(TrustPoint{Label: "owner-grant-reissued", SignerGrants: signerOn,
		OwnerGrants: ownerList(mustGrant(OwnerAuthorityClass, OwnerScope(ExtensionNamespace), OwnerIssuer, "owner policy 2", true))}.WithKeys(k1))
	pExpired := s.Trust.Add(TrustPoint{Label: "after-publication-expiry", Now: PublicationValidUntil.Add(1),
		OwnerGrants: ownerOn, SignerGrants: signerOn}.WithKeys(k1))

	for _, c := range []struct {
		id   string
		p    TrustPoint
		want string
	}{{"pops02-stage2-owner-revoked", pOwnerRevoked, "revoked"}, {"pops02-stage2-owner-absent", pOwnerAbsent, "no current trust grant"}} {
		dir := s.faultDir(c.id)
		_, _, err := s.runStage2(ctx, s.s1, s.s1.Carried, c.p, dir)
		s.addFault(c.id, "P-OPS-02", "2", err != nil && strings.Contains(err.Error(), c.want) && !exists(filepath.Join(dir, "semantic-lock.json")),
			map[string]any{"trustPoint": c.p.Seq, "error": errString(err), "lockWritten": exists(filepath.Join(dir, "semantic-lock.json"))})
	}

	_, _, err := s.runStage3(ctx, s.s1, s.s2, s.s2.Carried, pOwnerRevoked, s.faultDir("pops02-stage3-owner-revoked"))
	s.addFault("pops02-stage3-owner-revoked", "P-OPS-02", "3", err != nil && strings.Contains(err.Error(), "not qualified"),
		map[string]any{"trustPoint": pOwnerRevoked.Seq, "error": errString(err),
			"stage2QualifiedAtPoint": s.p1.Seq})
	for _, c := range []struct {
		id   string
		p    TrustPoint
		want string
	}{
		{"pops02-stage3-signer-revoked", pSignerRevoked, "revoked"},
		{"pops02-stage3-signer-key-rotated", pRotated, "rejected"},
		{"pops02-stage3-publication-expired", pExpired, "expired"},
	} {
		j := judgeSigner(s.s3.Publication, c.p, nil)
		s.addFault(c.id, "P-OPS-02", "3", !j.Qualified && strings.Contains(j.Error, c.want), map[string]any{"judgment": j})
	}
	calls := 0
	during := func(sub semanticcompositionpreflight.PublisherSubject) (semanticcompositionpreflight.PublisherTrust, error) {
		calls++
		if calls == 1 {
			return s.p1.SignerLookup()(sub)
		}
		return pRotated.SignerLookup()(sub)
	}
	j := judgeSigner(s.s3.Publication, s.p1, during)
	s.addFault("pops02-stage3-key-rotated-during-read", "P-OPS-02", "3",
		!j.Qualified && j.Error == semanticcompositionpreflight.ErrPublisherTrustChanged.Error(),
		map[string]any{"judgment": j, "lookupCalls": calls})

	// Stage 4: the decision made at the first point is judged again at
	// later points. The decision itself does not change.
	before, _ := digestJSON(s.host.decisions[ApplicationID])
	var rows []Rejudgment
	pattern := map[int]bool{s.p1.Seq: true, pOwnerRevoked.Seq: false, pOwnerAbsent.Seq: false,
		pSignerRevoked.Seq: false, pRotated.Seq: false, pRestored.Seq: true, pExpired.Seq: false}
	ok := true
	for _, p := range []TrustPoint{s.p1, pOwnerRevoked, pOwnerAbsent, pSignerRevoked, pRotated, pRestored, pExpired} {
		r, err := s.rejudge(ctx, s.host, s.proj, s.s2, s.s3, s.s1.LossPath, p, s.faultDir("pops02-stage4"))
		if err != nil {
			return err
		}
		rows = append(rows, r)
		ok = ok && r.Established == pattern[p.Seq] && r.Meaning == "ok" && r.Loss == "ok"
	}
	after, _ := digestJSON(s.host.decisions[ApplicationID])
	s.addFault("pops02-stage4-rejudged-at-each-point", "P-OPS-02", "4", ok && before == after,
		map[string]any{"decisionDecidedAtTrustPoint": s.host.decisions[ApplicationID].External.DecidedAtTrustPoint,
			"decisionUnchanged": before == after, "rejudgments": rows})
	return nil
}

// faultLossless rewrites lossless along the way and checks it is detected.
func (s *Sample) faultLossless(ctx context.Context) error {
	allFalse := true
	for _, r := range s.s1.Reports {
		allFalse = allFalse && !*r.Mapping.Lossless && r.Mapping.ParentPresenceSemantics == "unresolved"
	}
	s.addFault("lossless-asset-reports-false", "固有", "1", allFalse && !s.s1.Loss.Lossless,
		map[string]any{"reports": len(s.s1.Reports), "allLosslessFalse": allFalse, "lossRecordUnresolved": s.s1.Loss.Unresolved})

	t := true
	for _, stage := range []int{2, 3, 4} {
		forged := s.Chain.Clone()
		forged.Records[stage-1].Carried.Lossless = &t
		forged.Relink()
		res := VerifyThrough(forged)
		detected := !res.Complete && strings.Contains(strings.Join(res.Mismatches, "|"), "reads lossless as true")
		s.addFault("lossless-chain-rewritten-true-at-stage-"+string(rune('0'+stage)), "固有", "1-4", detected,
			map[string]any{"relinked": true, "mismatches": res.Mismatches})
	}
	forged := s.Chain.Clone()
	forged.Records[3].Carried.Lossless = nil
	forged.Relink()
	res := VerifyThrough(forged)
	s.addFault("lossless-chain-dropped-at-stage-4", "固有", "4", !res.Complete, map[string]any{"mismatches": res.Mismatches})

	raw, err := os.ReadFile(s.s1.LossPath)
	if err != nil {
		return err
	}
	rewritten := bytes.Replace(raw, []byte(`"lossless": false`), []byte(`"lossless": true`), 1)
	path := filepath.Join(s.faultDir("lossless-loss-record"), "loss-record.json")
	if err := writeFile(path, rewritten); err != nil {
		return err
	}
	_, _, errCarried := readLossRecord(path, s.s1.Carried.LossRecordDigest)
	_, _, errOwn := readLossRecord(path, digestBytes(rewritten))
	forgedC := s.s3.Carried
	forgedC.LossRecordDigest = digestBytes(rewritten)
	plan, err := distributionPlan(forgedC)
	if err != nil {
		return err
	}
	locked, err := plugin.LockBuildOutput(plan, s.s3.Carried.ArtifactDigest)
	if err != nil {
		return err
	}
	payload, err := os.ReadFile(s.s3.PayloadPath)
	if err != nil {
		return err
	}
	_, bridgeErr := semanticbuildbridge.InspectLockedExport(plan, s.s3.Lock, s.s3.Locked, payload, s.s3.Expected)
	pub := s.s3.Publication
	pub.Asset.Build = locked
	pub.Asset.Lock = plugin.BuildArtifactLock{FormatVersion: plugin.BuildArtifactLockFormatVersion, Builds: []plugin.LockedBuild{locked}}
	sig := judgeSigner(pub, s.p1, nil)
	s.addFault("lossless-loss-record-rewritten", "固有", "1-3",
		errors.Is(errCarried, ErrInputMismatch) && errors.Is(errOwn, ErrInputMismatch) && bridgeErr != nil && !sig.Qualified,
		map[string]any{"againstCarriedDigest": errString(errCarried), "againstItsOwnDigest": errString(errOwn),
			"planFingerprintChanged":      locked.PlanFingerprint != s.s3.Locked.PlanFingerprint,
			"buildBridgeWithOriginalLock": errString(bridgeErr), "signedPreflight": sig.Error})

	rep, err := BuildCompletionReport(*s.Chain)
	if err != nil {
		return err
	}
	for id, edit := range map[string]func(*CompletionReport){
		"report-lossless-rewritten-true":   func(r *CompletionReport) { r.Loss.Lossless = true },
		"report-claims-current-conformity": func(r *CompletionReport) { r.CurrentExternalApplication.Conformance = "conforms" },
		"report-claims-real-environment": func(r *CompletionReport) {
			r.RealEnvironment.Confirmed = true
			r.Evidence = append(r.Evidence, EvidenceItem{"stage 4", NatureReal, "forged"})
		},
		"report-claims-adoption": func(r *CompletionReport) { r.AdoptionEstablished = true },
	} {
		raw, _ := json.Marshal(rep)
		var forged CompletionReport
		_ = json.Unmarshal(raw, &forged)
		edit(&forged)
		mm := VerifyCompletionReport(*s.Chain, forged)
		s.addFault(id, "固有", "報告", len(mm) > 0, map[string]any{"mismatches": mm})
	}
	return nil
}

// editProviderMeaning changes the meaning of the copied provider namespace.
func editProviderMeaning(providerRoot string) error {
	path := filepath.Join(providerRoot, "semantic", "sui", "island.tei")
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	changed := bytes.Replace(raw, []byte("from std.identifier"), []byte("from std.text"), 1)
	if bytes.Equal(raw, changed) {
		return errors.New("provider meaning edit did not apply")
	}
	return os.WriteFile(path, changed, 0o644)
}

// faultMeaningChange accepts once, then changes only the provider's
// meaning, or only moves its file.
func (s *Sample) faultMeaningChange(ctx context.Context) error {
	src := filepath.Join(s.s1.InputDir, "workspace")
	for _, c := range []struct {
		id     string
		edit   func(project) error
		expect bool
	}{
		{"meaning-provider-meaning-changed", func(p project) error { return editProviderMeaning(p.Provider) }, false},
		{"meaning-provider-file-moved", func(p project) error {
			from := filepath.Join(p.Provider, "semantic", "sui", "island.tei")
			to := filepath.Join(p.Provider, "semantic", "sui", "hierarchy", "island.tei")
			if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
				return err
			}
			return os.Rename(from, to)
		}, true},
	} {
		dir := s.faultDir(c.id)
		proj, err := s.setupProject(src, filepath.Join(dir, "project"))
		if err != nil {
			return err
		}
		h := NewHost()
		if _, err := h.Accept(proj.Consumer, externalFrom(s.s3.Carried, s.p1), "synthetic://external-adoption-authority/decision/"+c.id); err != nil {
			return err
		}
		before, err := s.rejudge(ctx, h, proj, s.s2, s.s3, s.s1.LossPath, s.p1, dir)
		if err != nil {
			return err
		}
		if err := c.edit(proj); err != nil {
			return err
		}
		after, err := s.rejudge(ctx, h, proj, s.s2, s.s3, s.s1.LossPath, s.p1, dir)
		if err != nil {
			return err
		}
		digest, derr := semanticDigestOf(proj.Provider, "sui")
		passed := before.Established && after.Established == c.expect && after.SourceOnly == "ok"
		if !c.expect {
			passed = passed && strings.Contains(after.Meaning, "PROJECT_MEANING_SEMANTIC_DIGEST_MISMATCH") && digest != s.s3.Carried.SemanticDigest
		} else {
			passed = passed && digest == s.s3.Carried.SemanticDigest
		}
		s.addFault(c.id, "固有", "4", passed && derr == nil, map[string]any{
			"before": before, "after": after, "providerSemanticDigestAfter": digest,
			"acceptedSemanticDigest": s.s3.Carried.SemanticDigest})
	}
	return nil
}

// faultSubstitution swaps what one stage hands to the next and checks the
// next stage stops as an input mismatch; it also edits the chain.
func (s *Sample) faultSubstitution(ctx context.Context) error {
	// A stage 1 report from another revision.
	dir := s.faultDir("substitution-stage1-report")
	s1 := s.s1
	s1.InputDir, s1.ReportsDir, s1.LossPath = filepath.Join(dir, "input"), filepath.Join(dir, "reports"), filepath.Join(dir, "loss-record.json")
	for _, p := range [][2]string{{s.s1.InputDir, s1.InputDir}, {s.s1.ReportsDir, s1.ReportsDir}} {
		if err := copyTree(p[0], p[1]); err != nil {
			return err
		}
	}
	if err := copyFile(s.s1.LossPath, s1.LossPath); err != nil {
		return err
	}
	rep := filepath.Join(s1.ReportsDir, stage1Fixtures[1].File)
	raw, err := os.ReadFile(rep)
	if err != nil {
		return err
	}
	if err := os.WriteFile(rep, bytes.Replace(raw, []byte(s.pinned.Revision), []byte(strings.Repeat("e", 40)), 1), 0o644); err != nil {
		return err
	}
	_, _, err = s.runStage2(ctx, s1, s.s1.Carried, s.p1, filepath.Join(dir, "stage2"))
	s.addFault("substitution-stage1-report-from-another-revision", "通し", "2", errors.Is(err, ErrInputMismatch),
		map[string]any{"error": errString(err)})

	// Another provider meaning handed to stage 3.
	dir = s.faultDir("substitution-stage3-provider")
	s1 = s.s1
	s1.InputDir = filepath.Join(dir, "input")
	if err := copyTree(s.s1.InputDir, s1.InputDir); err != nil {
		return err
	}
	if err := editProviderMeaning(filepath.Join(s1.InputDir, "workspace")); err != nil {
		return err
	}
	_, _, err = s.runStage3(ctx, s1, s.s2, s.s2.Carried, s.p1, filepath.Join(dir, "stage3"))
	s.addFault("substitution-stage3-provider-meaning-swapped", "通し", "3", errors.Is(err, ErrInputMismatch),
		map[string]any{"error": errString(err)})

	// Another distribution handed to stage 4.
	dir = s.faultDir("substitution-stage4-distribution")
	ns, err := loadNamespace(filepath.Join(s.s1.InputDir, "workspace"), "sui")
	if err != nil {
		return err
	}
	other, err := semanticdistribution.MaterializeReference(semanticdistribution.Identity{PackageID: PackageID, Version: "1.0.1"}, ns)
	if err != nil {
		return err
	}
	s3 := s.s3
	s3.PayloadPath = filepath.Join(dir, "distribution.json")
	if err := writeFile(s3.PayloadPath, other); err != nil {
		return err
	}
	_, _, _, err = s.runStage4(ctx, s.s1, s.s2, s3, s.s3.Carried, s.p1, filepath.Join(dir, "stage4"))
	s.addFault("substitution-stage4-distribution-swapped", "通し", "4", errors.Is(err, ErrInputMismatch),
		map[string]any{"error": errString(err), "sameSemanticDigest": true})

	// Chain edits.
	for _, c := range []struct {
		id   string
		edit func(*Chain)
		want string
	}{
		{"through-stage2-missing", func(c *Chain) { c.Records = append(c.Records[:1:1], c.Records[2:]...); c.Relink() }, "missing"},
		{"through-stage4-artifact-swapped", func(c *Chain) {
			c.Records[3].Carried.ArtifactDigest = digestBytes([]byte("other"))
			c.Relink()
		}, "artifactDigest"},
		{"through-stage3-semantic-digest-swapped", func(c *Chain) {
			c.Records[2].Carried.SemanticDigest = digestBytes([]byte("other"))
			c.Relink()
		}, "semanticDigest"},
		{"through-stage2-source-revision-swapped", func(c *Chain) {
			c.Records[1].Carried.Source.Revision = strings.Repeat("e", 40)
			c.Relink()
		}, "pinned SUI source"},
		{"through-stage3-edited-without-relink", func(c *Chain) { c.Records[2].Carried.Package = "other@1.0.0" }, "does not match its digest"},
	} {
		forged := s.Chain.Clone()
		c.edit(&forged)
		res := VerifyThrough(forged)
		text := strings.Join(res.Mismatches, "|")
		if len(res.MissingStages) > 0 {
			text += "|missing"
		}
		s.addFault(c.id, "通し", "1-4", !res.Complete && strings.Contains(text, c.want),
			map[string]any{"missingStages": res.MissingStages, "mismatches": res.Mismatches})
	}
	return nil
}
