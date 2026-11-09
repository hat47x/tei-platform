package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/tools/internal/opensshwitness"
)

func repeatHex64(digit string) string {
	return strings.Repeat(digit, 64)
}

func witnessFixtureDecision(t *testing.T) Decision {
	t.Helper()
	baseRevision := "sha256:" + repeatHex64("b")
	return Decision{
		Schema:               decisionSchema,
		Decision:             "accepted",
		Reason:               "witness test",
		CandidateID:          "cand-1",
		CandidateFingerprint: "sha256:" + repeatHex64("c"),
		ReviewBinding: &ReviewBinding{
			Fingerprint: "sha256:" + repeatHex64("d"),
		},
		BaseModelRevision:   baseRevision,
		ModelRevisionBefore: baseRevision,
		SourceEdits: []SourceEdit{{
			Path:             "semantic/test.tei",
			Layer:            layerSemantic,
			AfterFingerprint: "sha256:" + repeatHex64("a"),
			InsertedText:     "+",
		}},
	}
}

func witnessFixtureRecord() DecisionWitnessRecord {
	return DecisionWitnessRecord{
		Schema:             decisionWitnessSchema,
		WorkspaceIdentity:  "ws-ref-001",
		DecisionRecordPath: decisionDir + "/" + repeatHex64("9") + ".json",
		SignerIdentity:     "reviewer@example.test",
		Signature:          "-----BEGIN SSH SIGNATURE-----\nfixture\n-----END SSH SIGNATURE-----\n",
		WitnessIssuedAt:    "20261028T120000Z",
	}
}

func TestDerivedWitnessPayloadBindsExactDecisionFields(t *testing.T) {
	d := witnessFixtureDecision(t)
	rec := witnessFixtureRecord()
	payload, err := deriveWitnessPayload(d, rec)
	if err != nil {
		t.Fatalf("derive witness payload: %v", err)
	}
	if payload.CandidateFingerprint != d.CandidateFingerprint {
		t.Fatalf("candidate fingerprintはexactにbindingされていません: %s", payload.CandidateFingerprint)
	}
	if payload.ReviewFingerprint != d.ReviewBinding.Fingerprint {
		t.Fatalf("review fingerprintはexactにbindingされていません: %s", payload.ReviewFingerprint)
	}
	if payload.BaseModelRevision != d.ModelRevisionBefore {
		t.Fatalf("base model revisionはexactにbindingされていません: %s", payload.BaseModelRevision)
	}
	if payload.Decision != opensshwitness.DecisionAccept {
		t.Fatalf("accepted recordのwitness semanticsはacceptでなければなりません: %s", payload.Decision)
	}
	if payload.DecisionReasonDigest != digestOf([]byte(d.Reason)) {
		t.Fatalf("reason digest mismatch: %s", payload.DecisionReasonDigest)
	}
	if payload.ScopeDigest != decisionScopeDigest(d) {
		t.Fatalf("scope digest mismatch: %s", payload.ScopeDigest)
	}
	if payload.WorkspaceIdentity != rec.WorkspaceIdentity {
		t.Fatalf("workspace identityはwitness記録から来ます: %s", payload.WorkspaceIdentity)
	}
	if payload.WitnessIssuedAt != rec.WitnessIssuedAt {
		t.Fatalf("issuedAt mismatch: %s", payload.WitnessIssuedAt)
	}
}

func TestDecisionScopeDigestIsContentAddressed(t *testing.T) {
	d := witnessFixtureDecision(t)
	first := decisionScopeDigest(d)
	more := d
	more.SourceEdits = []SourceEdit{
		d.SourceEdits[0],
		{
			Path:             "applications/app.tei",
			Layer:            layerApplication,
			AfterFingerprint: "sha256:" + repeatHex64("f"),
			InsertedText:     "+",
		},
	}
	withMore := decisionScopeDigest(more)
	if withMore == first {
		t.Fatal("source editの追加はscope digestを変える必要があります")
	}
	reordered := more
	reordered.SourceEdits = []SourceEdit{more.SourceEdits[1], more.SourceEdits[0]}
	if decisionScopeDigest(reordered) != withMore {
		t.Fatal("source edit順はscope digestに影響してはいけません")
	}
	tampered := d
	tampered.SourceEdits = []SourceEdit{d.SourceEdits[0]}
	tampered.SourceEdits[0].AfterFingerprint = "sha256:" + repeatHex64("e")
	if decisionScopeDigest(tampered) == first {
		t.Fatal("after fingerprintの改変はscope digestを変える必要があります")
	}
}

func TestRefusedDecisionIsNotWitnessable(t *testing.T) {
	d := witnessFixtureDecision(t)
	d.Decision = "refused"
	if _, err := decisionSemanticsOf(d); err == nil {
		t.Fatal("refused recordにはwitnessをbindingできません")
	}
}

func TestWitnessPayloadDerivationRefusesIncompleteBinding(t *testing.T) {
	d := witnessFixtureDecision(t)
	rec := witnessFixtureRecord()
	noReview := d
	noReview.ReviewBinding = nil
	if _, err := deriveWitnessPayload(noReview, rec); err == nil {
		t.Fatal("reviewBinding無しのrecordへwitnessをbindingできません")
	}
	noIdentity := rec
	noIdentity.WorkspaceIdentity = ""
	if _, err := deriveWitnessPayload(d, noIdentity); err == nil {
		t.Fatal("workspaceIdentity無しのwitnessはbindingできません")
	}
	noSignature := rec
	noSignature.Signature = ""
	if _, err := deriveWitnessPayload(d, noSignature); err == nil {
		t.Fatal("signature無しのwitnessはbindingできません")
	}
}

// witnessProviderBinaryは、testが使う実物providerとverify-time capabilityの有無を
// 返す。capability欠如はpassではなくskipにする（opensshwitness packageと同じ規約）。
func witnessProviderBinary(t *testing.T) string {
	t.Helper()
	if os.Getenv("TEI_SHARED_ACCEPTANCE_WITNESS_SKIP") == "1" {
		t.Skip("witness provider exercise disabled by TEI_SHARED_ACCEPTANCE_WITNESS_SKIP")
	}
	path, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("ssh-keygen unavailable; real OpenSSH witness not exercised")
	}
	if !probeVerifyTimeCapability(path) {
		t.Skip("provider build lacks -O verify-time; witness freshness cannot be pinned")
	}
	return path
}

// probeVerifyTimeCapabilityは、providerが-O verify-timeを知るかを、存在しない
// file群に対するverifyのrefusal文言で確かめる。
func probeVerifyTimeCapability(sshKeygen string) bool {
	cmd := exec.Command(sshKeygen, "-Y", "verify",
		"-O", "verify-time=20261028120000Z",
		"-f", "tei-probe-allowed-signers",
		"-I", "tei-probe-identity",
		"-n", "tei-probe-namespace",
		"-s", "tei-probe-signature")
	cmd.Stdin = strings.NewReader("")
	output, _ := cmd.CombinedOutput()
	return !strings.Contains(string(output), "verify-time")
}

// witnessTestKeyは、test用のed25519鍵をtemp directoryへ作る。
func witnessTestKey(t *testing.T, sshKeygen string) (string, string) {
	t.Helper()
	work := t.TempDir()
	key := filepath.Join(work, "witness-key")
	if out, err := exec.Command(sshKeygen, "-q", "-t", "ed25519", "-N", "", "-f", key).
		CombinedOutput(); err != nil {
		t.Skipf("ed25519 key generation failed: %v: %s", err, out)
	}
	public, err := os.ReadFile(key + ".pub")
	if err != nil {
		t.Fatalf("read public key: %v", err)
	}
	return key, strings.TrimSpace(string(public))
}

// writeWitnessPolicyは、identity・namespace・validityを固定したallowed-signers
// policyを書く。policyはoperator管理境界であり、candidateやworkspaceから導出しない。
func writeWitnessPolicy(t *testing.T, identity, namespace, publicKey string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "allowed_signers")
	content := identity + " namespaces=\"" + namespace + "\",valid-after=\"20200101Z\",valid-before=\"20990101Z\" " + publicKey + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

const witnessVerifyTime = "20261028120000Z"

// recordAcceptedAndRejectedForConflictは、同一candidateのacceptedと、同一review
// projection identityをbindingしたrejected recordを同じworkspaceへ置く。
//
// current Referenceのreject CLIはreviewをbindingしないため、CLIのrejectでは
// witness対象にならない（deriveWitnessPayloadがfail closedする）。conflict
// downgrade（Evidence Gate §10）の検証対象はwitness分類側であるため、ここでは
// 同一ReviewBinding・同一candidate fingerprintを持つrejected recordをledgerへ
// 直接置いて、両方qualifiedになった競合だけを再現する。
func recordAcceptedAndRejectedForConflict(t *testing.T, dir, candidatePath string) (Decision, Decision) {
	t.Helper()
	accepted := acceptViaCLI(t, dir, candidatePath)
	rejected := accepted
	rejected.Decision = "rejected"
	rejected.Reason = "witness conflict pressure"
	rejected.Stale = nil
	rejected.SourceChanged = false
	rejected.SourceEdits = nil
	rejected.SemanticDiff = nil
	rejected.ApplicationDiff = nil
	rejected.EnvironmentDiff = nil
	rejected.ModelRevisionBefore = accepted.ModelRevisionAfter
	rejected.ModelRevisionAfter = accepted.ModelRevisionAfter
	rejected.RecordPath = writeDecisionRecordFile(t, dir, rejected)
	return accepted, rejected
}

// writeDecisionRecordFileは、testが組み立てたDecisionをcontent-addressedの配置
// 規則に従ってDecision ledgerへ置く。
func writeDecisionRecordFile(t *testing.T, dir string, d Decision) string {
	t.Helper()
	encoded, err := encodeJSON(d)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(encoded)
	rel := decisionDir + "/" + hex.EncodeToString(sum[:]) + ".json"
	if err := os.MkdirAll(filepath.Join(dir, ".tei", "decisions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(rel)), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	return rel
}

// recordWitnessViaCLIは、witness-decisionをCLI経由で書き、そのverify設定もまとめて
// 渡す。
type witnessCLIOptions struct {
	key            string
	sshKeygen      string
	allowedSigners string
	verifyTime     string
	revocation     string
}

func recordWitnessViaCLI(t *testing.T, dir string, decision Decision, opts witnessCLIOptions) (int, string, string) {
	t.Helper()
	args := []string{
		"witness-decision",
		"-workspace", dir,
		"-decision", decision.RecordPath,
		"-workspace-identity", "ws-ref-001",
		"-key", opts.key,
		"-witness-identity", "reviewer@example.test",
		"-witness-issued-at", "20261028T120000Z",
	}
	if opts.sshKeygen != "" {
		args = append(args, "-witness-ssh-keygen", opts.sshKeygen)
	}
	if opts.allowedSigners != "" {
		args = append(args, "-witness-allowed-signers", opts.allowedSigners)
	}
	if opts.verifyTime != "" {
		args = append(args, "-witness-verify-time", opts.verifyTime)
	}
	if opts.revocation != "" {
		args = append(args, "-witness-revocation", opts.revocation)
	}
	return cliRun(t, args...)
}

func witnessFixtureOptions(sshKeygen, key, policy string) witnessCLIOptions {
	return witnessCLIOptions{
		key:            key,
		sshKeygen:      sshKeygen,
		allowedSigners: policy,
		verifyTime:     witnessVerifyTime,
	}
}

func decisionsWithWitnessVerification(t *testing.T, dir, policy string) DecisionIndex {
	t.Helper()
	code, out, errOut := cliRun(t, "decisions", "-workspace", dir,
		"-witness-allowed-signers", policy, "-witness-verify-time", witnessVerifyTime)
	if code != 0 {
		t.Fatalf("decisions code=%d err=%s", code, errOut)
	}
	var index DecisionIndex
	if err := json.Unmarshal([]byte(out), &index); err != nil {
		t.Fatalf("decisions出力を読めません: %v: %s", err, out)
	}
	return index
}

func TestQualifiedWitnessBindsAcceptedDecisionExactly(t *testing.T) {
	sshKeygen := witnessProviderBinary(t)
	dir := scratchWorkspaceBeforeSecondApplication(t)
	candidate := unitPriceCandidate(revisionOfDir(t, dir))
	candidatePath := candidateFile(t, candidate)
	decision := acceptViaCLI(t, dir, candidatePath)

	// namespaceはDecision record本文から再導出したものだけをpolicyへ許可する。
	rec := witnessFixtureRecord()
	rec.DecisionRecordPath = decision.RecordPath
	payload, err := deriveWitnessPayload(decision, rec)
	if err != nil {
		t.Fatalf("derive payload: %v", err)
	}
	namespace, err := payload.Namespace()
	if err != nil {
		t.Fatalf("namespace: %v", err)
	}
	key, publicKey := witnessTestKey(t, sshKeygen)
	policy := writeWitnessPolicy(t, "reviewer@example.test", namespace, publicKey)

	code, out, errOut := recordWitnessViaCLI(t, dir, decision, witnessFixtureOptions(sshKeygen, key, policy))
	if code != 0 {
		t.Fatalf("witness-decision code=%d err=%s out=%s", code, errOut, out)
	}
	var emitted struct {
		Record       string                     `json:"record"`
		Verification *witnessVerificationReport `json:"verification"`
	}
	if err := json.Unmarshal([]byte(out), &emitted); err != nil {
		t.Fatal(err)
	}
	if emitted.Verification == nil {
		t.Fatal("verify configuration有りではwitness検証が実行されるべきです")
	}
	if emitted.Verification.ActorStatus != actorStatusIdentityResolved ||
		emitted.Verification.AuthorityStatus != authorityStatusQualified ||
		emitted.Verification.WitnessStatus != witnessStatusVerified {
		t.Fatalf("exact bindingはqualifiedになっていません: %+v", emitted.Verification)
	}

	// decisions経路でも同じ分類に到達する。
	index := decisionsWithWitnessVerification(t, dir, policy)
	var target *DecisionRecord
	for i := range index.Records {
		if index.Records[i].Path == decision.RecordPath {
			target = &index.Records[i]
		}
	}
	if target == nil {
		t.Fatalf("record %s がindexにありません", decision.RecordPath)
	}
	if target.DecisionWitnessStatus != witnessStatusVerified ||
		target.DecisionActorStatus != actorStatusIdentityResolved ||
		target.DecisionAuthorityStatus != authorityStatusQualified {
		t.Fatalf("decisions経路の分類が不整合です: %+v", target)
	}
	if !strings.Contains(target.WitnessDiagnostics[0].Message, "evidenceFingerprint") {
		t.Fatalf("qualified witnessはevidence fingerprintをprovenanceとして残します: %+v", target.WitnessDiagnostics)
	}
}

func TestDeletedWitnessRevertsToUnresolved(t *testing.T) {
	sshKeygen := witnessProviderBinary(t)
	dir := scratchWorkspaceBeforeSecondApplication(t)
	candidate := unitPriceCandidate(revisionOfDir(t, dir))
	candidatePath := candidateFile(t, candidate)
	decision := acceptViaCLI(t, dir, candidatePath)

	rec := witnessFixtureRecord()
	rec.DecisionRecordPath = decision.RecordPath
	payload, err := deriveWitnessPayload(decision, rec)
	if err != nil {
		t.Fatal(err)
	}
	namespace, err := payload.Namespace()
	if err != nil {
		t.Fatal(err)
	}
	key, publicKey := witnessTestKey(t, sshKeygen)
	policy := writeWitnessPolicy(t, "reviewer@example.test", namespace, publicKey)
	code, _, errOut := recordWitnessViaCLI(t, dir, decision, witnessFixtureOptions(sshKeygen, key, policy))
	if code != 0 {
		t.Fatalf("witness-decision code=%d err=%s", code, errOut)
	}
	if index := decisionsWithWitnessVerification(t, dir, policy); index.Records[0].DecisionWitnessStatus != witnessStatusVerified {
		t.Fatalf("witness書込み直後はverifiedです: %+v", index.Records[0])
	}
	// EvidenceはCanonical sourceではない: 削除すればclaimが消え、meaningは変わらない。
	files, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	witnesses, err := loadDecisionWitnesses(files)
	if err != nil {
		t.Fatal(err)
	}
	files.Close()
	for _, witness := range witnesses {
		target := filepath.Join(dir, filepath.FromSlash(witnessFileName(mustEncodeWitness(t, witness))))
		if err := os.Remove(target); err != nil {
			t.Fatal(err)
		}
	}
	index := decisionsWithWitnessVerification(t, dir, policy)
	if index.Records[0].DecisionWitnessStatus != witnessStatusNone ||
		index.Records[0].DecisionActorStatus != actorStatusUnresolved ||
		index.Records[0].DecisionAuthorityStatus != authorityStatusUnresolved {
		t.Fatalf("witness削除後はclaimが完全に消える必要があります: %+v", index.Records[0])
	}
}

func mustEncodeWitness(t *testing.T, witness DecisionWitnessRecord) []byte {
	t.Helper()
	encoded, err := encodeJSON(witness)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestWrongPrincipalCannotResolveWitness(t *testing.T) {
	sshKeygen := witnessProviderBinary(t)
	dir := scratchWorkspaceBeforeSecondApplication(t)
	candidate := unitPriceCandidate(revisionOfDir(t, dir))
	candidatePath := candidateFile(t, candidate)
	decision := acceptViaCLI(t, dir, candidatePath)

	rec := witnessFixtureRecord()
	rec.DecisionRecordPath = decision.RecordPath
	rec.SignerIdentity = "impostor@example.test"
	payload, err := deriveWitnessPayload(decision, rec)
	if err != nil {
		t.Fatal(err)
	}
	namespace, err := payload.Namespace()
	if err != nil {
		t.Fatal(err)
	}
	key, publicKey := witnessTestKey(t, sshKeygen)
	// policyは別principalにしかnamespaceを与えない。
	policy := writeWitnessPolicy(t, "reviewer@example.test", namespace, publicKey)
	// impostorを名乗るwitness記録をledgerへ直接置く。witness-decisionはpolicyと
	// 対になる検証が前提の手続きのため、申告principalの不一致はこの経路でしか
	// 作れない。
	signature, err := opensshwitness.Sign(opensshwitness.SSHKeygenOption(sshKeygen), key, payload)
	if err != nil {
		t.Fatal(err)
	}
	message, err := payload.Message()
	if err != nil {
		t.Fatal(err)
	}
	rec.Signature = string(signature)
	rec.MessageDigest = digestOf([]byte(message))
	rec.SignatureDigest = digestOf(signature)
	decisionContent, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(decision.RecordPath)))
	if err != nil {
		t.Fatal(err)
	}
	rec.DecisionFingerprint = digestOf(decisionContent)
	writeWitnessRecordFile(t, dir, rec)
	index := decisionsWithWitnessVerification(t, dir, policy)
	target := findRecordByPath(t, index, decision.RecordPath)
	if target.DecisionActorStatus != actorStatusUnresolved ||
		target.DecisionAuthorityStatus != authorityStatusUnresolved {
		t.Fatalf("principal不一致は解決しないはずです: %+v", target)
	}
	if target.DecisionWitnessStatus == witnessStatusVerified {
		t.Fatalf("principal不一致のwitnessはverifiedになり得ません: %+v", target)
	}
	if !hasDiagnosticCode(target.WitnessDiagnostics, diagWitnessProviderRejected) {
		t.Fatalf("principal不一致はprovider拒絶のdiagnosticを残す必要があります: %+v", target.WitnessDiagnostics)
	}
}

// writeWitnessRecordFileは、testが組み立てたwitness記録をcontent-addressedの
// 配置規則に従ってledgerへ置く。
func writeWitnessRecordFile(t *testing.T, dir string, rec DecisionWitnessRecord) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".tei", "decision-witnesses"), 0o755); err != nil {
		t.Fatal(err)
	}
	encoded, err := encodeJSON(rec)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, filepath.FromSlash(witnessFileName(encoded)))
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
}

func hasDiagnosticCode(diagnostics []Diagnostic, code string) bool {
	for _, d := range diagnostics {
		if d.Code == code {
			return true
		}
	}
	return false
}

func findRecordByPath(t *testing.T, index DecisionIndex, path string) *DecisionRecord {
	t.Helper()
	for i := range index.Records {
		if index.Records[i].Path == path {
			return &index.Records[i]
		}
	}
	t.Fatalf("record %s がindexにありません", path)
	return nil
}

func TestForgedRejectWitnessAgainstAcceptedDecisionIsStructurallyInvalid(t *testing.T) {
	sshKeygen := witnessProviderBinary(t)
	dir := scratchWorkspaceBeforeSecondApplication(t)
	candidate := unitPriceCandidate(revisionOfDir(t, dir))
	candidatePath := candidateFile(t, candidate)
	decision := acceptViaCLI(t, dir, candidatePath)

	key, publicKey := witnessTestKey(t, sshKeygen)
	// 本物のmessageを検証可能なpolicyとして先に作るため、まず構造bindingを通すrec。
	baseRec := witnessFixtureRecord()
	baseRec.DecisionRecordPath = decision.RecordPath
	payload, err := deriveWitnessPayload(decision, baseRec)
	if err != nil {
		t.Fatal(err)
	}
	// acceptを許可するpolicyで署名しておき、witness payloadのsemanticsをrejectに
	// 差し替えたようなrecordを組み立てる。loaderはmessageをDecision本文から
	// 再導出するため、差し替えmessageを据えるとdigest照合で不成立になる。
	forged := payload
	forged.Decision = opensshwitness.DecisionReject
	signature, err := opensshwitness.Sign(opensshwitness.SSHKeygenOption(sshKeygen), key, forged)
	if err != nil {
		t.Fatal(err)
	}
	message, err := forged.Message()
	if err != nil {
		t.Fatal(err)
	}
	rec := baseRec
	rec.Signature = string(signature)
	rec.MessageDigest = digestOf([]byte(message))
	rec.SignatureDigest = digestOf(signature)
	// forged recordをwitness ledgerへ直接置く（witness-decision経由では書けない）。
	writeWitnessRecordFile(t, dir, rec)
	index := decisionsWithWitnessVerification(t, dir, writeWitnessPolicy(t, "reviewer@example.test", caNamespace(t, payload), publicKey))
	target := findRecordByPath(t, index, decision.RecordPath)
	if target.DecisionWitnessStatus != witnessStatusInvalid ||
		target.DecisionActorStatus != actorStatusUnresolved ||
		target.DecisionAuthorityStatus != authorityStatusUnresolved {
		t.Fatalf("改変messageを持つwitnessはinvalidのまま未解決であるべきです: %+v", target)
	}
	hasMismatch := false
	for _, d := range target.WitnessDiagnostics {
		if d.Code == diagWitnessMessageMismatch {
			hasMismatch = true
		}
	}
	if !hasMismatch {
		t.Fatalf("message digest不一致のdiagnosticが必要です: %+v", target.WitnessDiagnostics)
	}
}

func caNamespace(t *testing.T, payload opensshwitness.Payload) string {
	t.Helper()
	namespace, err := payload.Namespace()
	if err != nil {
		t.Fatal(err)
	}
	return namespace
}

func TestAcceptAndRejectWitnessesOnSameCandidateBecomeConflicted(t *testing.T) {
	sshKeygen := witnessProviderBinary(t)
	dir := scratchWorkspaceBeforeSecondApplication(t)
	candidate := unitPriceCandidate(revisionOfDir(t, dir))
	candidatePath := candidateFile(t, candidate)
	accepted, rejected := recordAcceptedAndRejectedForConflict(t, dir, candidatePath)

	key, publicKey := witnessTestKey(t, sshKeygen)
	acceptRec := witnessFixtureRecord()
	acceptRec.DecisionRecordPath = accepted.RecordPath
	acceptPayload, err := deriveWitnessPayload(accepted, acceptRec)
	if err != nil {
		t.Fatal(err)
	}
	acceptNamespace, err := acceptPayload.Namespace()
	if err != nil {
		t.Fatal(err)
	}
	rejectRec := acceptRec
	rejectRec.DecisionRecordPath = rejected.RecordPath
	rejectPayload, err := deriveWitnessPayload(rejected, rejectRec)
	if err != nil {
		t.Fatal(err)
	}
	rejectNamespace, err := rejectPayload.Namespace()
	if err != nil {
		t.Fatal(err)
	}
	// 同一identityへ、accept namespaceとreject namespaceの両方をpolicyで許可する。
	policy := filepath.Join(t.TempDir(), "allowed_signers")
	content := "reviewer@example.test namespaces=\"" + acceptNamespace + "\",valid-after=\"20200101Z\",valid-before=\"20990101Z\" " + publicKey + "\n" +
		"reviewer@example.test namespaces=\"" + rejectNamespace + "\",valid-after=\"20200101Z\",valid-before=\"20990101Z\" " + publicKey + "\n"
	if err := os.WriteFile(policy, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	options := witnessFixtureOptions(sshKeygen, key, policy)
	for _, target := range []Decision{accepted, rejected} {
		code, _, errOut := recordWitnessViaCLI(t, dir, target, options)
		if code != 0 {
			t.Fatalf("witness-decision code=%d err=%s", code, errOut)
		}
	}
	index := decisionsWithWitnessVerification(t, dir, policy)
	acceptTarget := findRecordByPath(t, index, accepted.RecordPath)
	rejectTarget := findRecordByPath(t, index, rejected.RecordPath)
	for _, target := range []*DecisionRecord{acceptTarget, rejectTarget} {
		if target.DecisionWitnessStatus != witnessStatusConflicted ||
			target.DecisionAuthorityStatus != authorityStatusConflicted {
			t.Fatalf("同一candidateのaccept/reject witness競合はconflictedに落ちる必要があります: %+v", target)
		}
		if target.DecisionActorStatus != actorStatusUnresolved {
			t.Fatalf("conflicted状態でactorを解決済みにしてはいけません: %+v", target)
		}
	}
}

func TestWitnessVerificationFlagsRequirePolicyAndTime(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	code, _, _ := cliRun(t, "decisions", "-workspace", dir, "-witness-allowed-signers", "x")
	if code != 2 {
		t.Fatalf("allow-signersだけではusageエラーであるべきです: code=%d", code)
	}
	code, _, _ = cliRun(t, "decisions", "-workspace", dir, "-witness-verify-time", witnessVerifyTime)
	if code != 2 {
		t.Fatalf("verify-timeだけではusageエラーであるべきです: code=%d", code)
	}
	code, out, errOut := cliRun(t, "decisions", "-workspace", dir)
	if code != 0 {
		t.Fatalf("flags無しのdecisionsは普通に読めます: code=%d err=%s out=%s", code, errOut, out)
	}
}
