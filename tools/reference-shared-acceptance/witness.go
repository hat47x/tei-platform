package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"tei/tools/internal/opensshwitness"
)

// Decision witnessは、Decision recordの本文全体へOpenSSH署名でexactにbindingする
// 付加Evidenceである。providerが`authority-qualified`と出したwitnessも、accepted
// meaning・Source書込みを表さず、その決定が人会得されたという証明でもない
// （Evidence Gate 2026-10 §5 / §6）。Decision ledgerと同様にCanonical sourceでは
// なく、消去してもmodel revisionもaccepted meaningも変わらない。
//
// policy・verify time・revocationは、loaderへ渡すWitnessVerifyOptions（CLIの
// flags）というmanaged configuration boundaryだけから得る。workspace・candidate・
// decision recordの内容からpolicyを補完せず、provider不在でstatusを解決にせず、
// 自己申告actorへfallbackしない。

const (
	witnessDir            = ".tei/decision-witnesses"
	decisionWitnessSchema = "tei.reference.decision-witness/v0"
)

// DecisionWitnessRecordは、Decision record本文への署名と、provider検証に必要な
// provenanceを記録する。namespaceと署名対象messageの本文はDecision record本文から
// 再導出するため、recordには含めない。
type DecisionWitnessRecord struct {
	Schema string `json:"schema"`
	// WorkspaceIdentityはnamespaceへ埋め込むoperator管理のtokenである。
	// workspaceの実在directory名やrevisionから導出せず、policy側のnamespace許可と
	// 対になる。
	WorkspaceIdentity string `json:"workspaceIdentity"`
	// DecisionRecordPathはbinding先のDecision recordのworkspace相対pathである。
	DecisionRecordPath string `json:"decisionRecordPath"`
	// DecisionFingerprintはbinding先Decision record本文のsha256である。record本文の
	// 一部にではなくproposal全体にbindingする。
	DecisionFingerprint string `json:"decisionFingerprint"`
	// SignerIdentityは署名者のprincipal provenanceである。authorityはproviderが
	// policyで確定するまで未解決のまま扱う。
	SignerIdentity  string `json:"signerIdentity"`
	Signature       string `json:"signature"`
	WitnessIssuedAt string `json:"witnessIssuedAt"`
	WitnessNonce    string `json:"witnessNonce,omitempty"`
	// MessageDigest / SignatureDigestはprovenance用の内容参照である。loaderは
	// 再導出したmessageのdigestと照合し、provider有無に関わらずrecord側の改変を
	// 検知する。
	MessageDigest   string `json:"messageDigest,omitempty"`
	SignatureDigest string `json:"signatureDigest,omitempty"`
}

// WitnessVerifyOptionsは、Decision ledger読み込み時のwitness検証のmanaged
// configuration boundaryである。nilならprovider検証を行わず、statusを解決しない。
type WitnessVerifyOptions struct {
	// AllowedSignersはoperator管理のallowed-signers policy fileである。
	AllowedSigners string
	// VerifyTimeは検証時刻（YYYYMMDDHHMMSSZ）で必須である。新鮮性を固定しない
	// 検証は実行しない。
	VerifyTime string
	// Revocationは任意のKRL / revoked-keys fileである。
	Revocation string
}

// witnessVerifyInternalは、provider binaryの明示指定（Reference・test用）を含む
// 内部渡しである。通常のflagsではprovider binaryはPATHから選ばれる。
type witnessVerifyInternal struct {
	WitnessVerifyOptions
	SSHKeygen string
}

// DecisionRecordに現れるwitness状態。Evidence Gate 2026-10 §5のgate stateに対応する。
const (
	witnessStatusNone       = "none"
	witnessStatusRecorded   = "recorded"
	witnessStatusVerified   = "verified"
	witnessStatusInvalid    = "invalid"
	witnessStatusConflicted = "conflicted"

	actorStatusUnresolved       = "unresolved"
	actorStatusIdentityResolved = "identity-resolved"

	authorityStatusUnresolved = "unresolved"
	authorityStatusQualified  = "authority-qualified"
	authorityStatusConflicted = "authority-conflicted"
)

// witness検証のdiagnostic codes。DecisionRecordのwitnessDiagnosticsへ入る。
const (
	diagWitnessTargetNotBindable       = "WITNESS_TARGET_NOT_BINDABLE"
	diagWitnessMessageMismatch         = "WITNESS_MESSAGE_MISMATCH"
	diagWitnessSignatureDigestMismatch = "WITNESS_SIGNATURE_DIGEST_MISMATCH"
	diagWitnessProviderRejected        = "WITNESS_PROVIDER_REJECTED"
	diagWitnessProviderUnavailable     = "WITNESS_PROVIDER_UNAVAILABLE"
	diagWitnessProviderUnsupported     = "WITNESS_PROVIDER_UNSUPPORTED"
	diagWitnessDecisionConflict        = "WITNESS_DECISION_CONFLICT"
	diagWitnessVerified                = "WITNESS_VERIFIED"
)

var witnessVerifyTimePattern = regexp.MustCompile(`^\d{8}\d{6}Z$`)

// decisionSemanticsOfは、witness対象になり得るDecision本体のsemanticsを返す。
// 受入拒否（refused）は人・AIの決定の記録ではないため、witness対象から外す。
func decisionSemanticsOf(d Decision) (string, error) {
	switch d.Decision {
	case "accepted":
		return opensshwitness.DecisionAccept, nil
	case "rejected":
		return opensshwitness.DecisionReject, nil
	default:
		return "", fmt.Errorf(
			"decision %qにはwitnessをbindingできません（accepted / rejectedの本文だけがwitness対象です）", d.Decision)
	}
}

// decisionScopeDigestは、Decision recordのsource編集全体を一つのsha256へ凝縮する。
// path・前後fingerprint・createdの全てを含み、順序を固定する。編集が無い場合
// （reject等）も空リストのdigestとして定義し、「未指定」と「空scope」を区別しない。
func decisionScopeDigest(d Decision) string {
	lines := make([]string, 0, len(d.SourceEdits))
	for _, e := range d.SourceEdits {
		lines = append(lines, fmt.Sprintf(
			"%s\t%s\t%s\tcreated=%t",
			e.Path, e.BeforeFingerprint, e.AfterFingerprint, e.Created,
		))
	}
	sort.Strings(lines)
	return digestOf([]byte(strings.Join(lines, "\n")))
}

// digestOfは、sha256:<hex>形式の内容参照を生成する。
func digestOf(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// deriveWitnessPayloadは、Decision record本文とwitness記録側のoperator管理値だけ
// からwitness messageを再導出する。binding先がrecord本文へ凝縮されているため、
// recordの改変・飲み替えが検証で検知できる。
func deriveWitnessPayload(d Decision, rec DecisionWitnessRecord) (opensshwitness.Payload, error) {
	switch {
	case rec.Signature == "":
		return opensshwitness.Payload{}, errors.New("witness recordにsignatureがありません")
	}
	return deriveWitnessPayloadCore(d, rec)
}

// deriveWitnessPayloadCoreは、signature有無を除いたbinding整合の検証とpayload
// 導出を行う。署名前（buildDecisionWitness）でも使える。
func deriveWitnessPayloadCore(d Decision, rec DecisionWitnessRecord) (opensshwitness.Payload, error) {
	payload := opensshwitness.Payload{
		WorkspaceIdentity: rec.WorkspaceIdentity,
		WitnessIssuedAt:   rec.WitnessIssuedAt,
		WitnessNonce:      rec.WitnessNonce,
	}
	switch {
	case d.CandidateFingerprint == "":
		return payload, errors.New("decision recordにcandidateFingerprintが無く、witnessをbindingできません")
	case d.ModelRevisionBefore == "":
		return payload, errors.New("decision recordにmodelRevisionBeforeが無く、基準revisionへbindingできません")
	case rec.WorkspaceIdentity == "":
		return payload, errors.New("witness recordにworkspaceIdentityがありません")
	case rec.SignerIdentity == "":
		return payload, errors.New("witness recordにsignerIdentityがありません")
	}
	semantics, err := decisionSemanticsOf(d)
	if err != nil {
		return payload, err
	}
	if d.ReviewBinding == nil || d.ReviewBinding.Fingerprint == "" {
		return payload, errors.New("decision recordにreviewBinding fingerprintが無く、review Projectionへのbindingが成立しません")
	}
	payload.CandidateFingerprint = d.CandidateFingerprint
	payload.ReviewFingerprint = d.ReviewBinding.Fingerprint
	payload.BaseModelRevision = d.ModelRevisionBefore
	payload.Decision = semantics
	payload.DecisionReasonDigest = digestOf([]byte(d.Reason))
	payload.ScopeDigest = decisionScopeDigest(d)
	return payload, nil
}

// witnessFileNameは、witness記録をcontent-addressedに配置するworkspace相対path。
func witnessFileName(encoded []byte) string {
	sum := sha256.Sum256(encoded)
	return witnessDir + "/" + hex.EncodeToString(sum[:]) + ".json"
}

// buildDecisionWitnessは、binding先Decision recordを読み、本文から導出したmessageを
// Reference harness keyで署名し、witness記録をcontent-addressed ledgerへ書き込む。
// 決定の内容・policyには触れず、authoritative sourceを書き換えない。
func buildDecisionWitness(root, privateKeyFile string, rec DecisionWitnessRecord, sshKeygen opensshwitness.SSHKeygenOption) (DecisionWitnessRecord, string, error) {
	if privateKeyFile == "" {
		return rec, "", errors.New("witness-decisionにはReference署名用の秘密鍵file（-key）が必須です")
	}
	relative := strings.ReplaceAll(rec.DecisionRecordPath, "\\", "/")
	if !strings.HasPrefix(relative, decisionDir+"/") {
		return rec, "", fmt.Errorf("witnessのbinding先 %q はDecision recordではありません", relative)
	}
	files, err := os.OpenRoot(root)
	if err != nil {
		return rec, "", fmt.Errorf("open Workspace root for Decision witness ledger: %w", err)
	}
	defer files.Close()
	content, err := ledgerReadFile(files, relative)
	if err != nil {
		return rec, "", fmt.Errorf("binding先Decision recordを読めません: %w", err)
	}
	sum := sha256.Sum256(content)
	if relative != decisionDir+"/"+hex.EncodeToString(sum[:])+".json" {
		return rec, "", fmt.Errorf(
			"binding先Decision record %s: 内容hashがfile名と一致しないため、witnessをbindingせず停止します", relative)
	}
	var d Decision
	if err := decodeStrict(content, &d); err != nil {
		return rec, "", fmt.Errorf("binding先Decision record %s: %w", relative, err)
	}
	payload, err := deriveWitnessPayloadCore(d, rec)
	if err != nil {
		return rec, "", fmt.Errorf("binding先Decision record %s: %w", relative, err)
	}
	message, err := payload.Message()
	if err != nil {
		return rec, "", fmt.Errorf("binding先Decision record %s: %w", relative, err)
	}
	signature, err := opensshwitness.Sign(sshKeygen, privateKeyFile, payload)
	if err != nil {
		return rec, "", fmt.Errorf("witness signatureを生成できません: %w", err)
	}
	rec.DecisionFingerprint = "sha256:" + hex.EncodeToString(sum[:])
	rec.Signature = string(signature)
	rec.MessageDigest = digestOf([]byte(message))
	rec.SignatureDigest = digestOf(signature)
	if rec.Schema == "" {
		rec.Schema = decisionWitnessSchema
	}
	encoded, err := encodeJSON(rec)
	if err != nil {
		return rec, "", err
	}
	rel := witnessFileName(encoded)
	if err := ledgerWriteFile(files, rel, encoded); err != nil {
		return rec, "", err
	}
	return rec, rel, nil
}

// loadDecisionWitnessesは、witness ledgerを読み、content-addressの一致を検証する。
// ledgerが無い場合はnilを返す。
func loadDecisionWitnesses(files *os.Root) ([]DecisionWitnessRecord, error) {
	entries, err := ledgerReadDir(files, witnessDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	var witnesses []DecisionWitnessRecord
	for _, name := range names {
		relative := witnessDir + "/" + name
		content, err := ledgerReadFile(files, relative)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(content)
		if name != hex.EncodeToString(sum[:])+".json" {
			return nil, fmt.Errorf("decision-witness record %s: content hashがfile名と一致しません", relative)
		}
		var rec DecisionWitnessRecord
		if err := decodeStrict(content, &rec); err != nil {
			return nil, fmt.Errorf("decision-witness record %s: %w", relative, err)
		}
		witnesses = append(witnesses, rec)
	}
	return witnesses, nil
}

// bindDecisionWitnessesは、Decision recordsへwitness ledgerをexactにbindingし、
// witnessStatus・decisionActorStatus・decisionAuthorityStatusを分類する。
//
// 更新規則（fail closed）:
//   - binding物が無い → witnessStatus=none、actor/authority=unresolvedのまま
//   - binding物が存在し、本文・message再導出・signature digestの照合が通る →
//     （provider検証に届かない場合も）recorded。actor/authorityは解決しない
//   - binding先digest不一致・semantics対象外・message/signature digest不一致 →
//     invalid（他のbinding物がverifiedならverifiedは失わない）
//   - provider検証configurationが揃った場合は全binding物を検証し、providerが
//     authority-qualifiedと出したものだけが actor=identity-resolved /
//     authority=authority-qualified をもたらす。provider不在・verify-time非対応・
//     拒絶は全て解決に至らず、diagnosticsに理由を残す
//   - 同一candidate fingerprintのaccepted / rejectedが両方qualifiedになった場合は
//     どちらもauthority-conflictedへdowngradeする（Evidence Gate §10）
func bindDecisionWitnesses(records []*DecisionRecord, digestOfRecord map[string]string, witnesses []DecisionWitnessRecord, verify *witnessVerifyInternal) error {
	targets := make(map[string]*DecisionRecord, len(records))
	for _, record := range records {
		targets[record.Path] = record
	}
	for i := range witnesses {
		rec := witnesses[i]
		digest, ok := digestOfRecord[rec.DecisionRecordPath]
		if !ok {
			// 存在しない・消去済みrecordへのbindingは無力であり、statusを上げない。
			continue
		}
		record := targets[rec.DecisionRecordPath]
		if digest != rec.DecisionFingerprint {
			record.WitnessDiagnostics = append(record.WitnessDiagnostics, Diagnostic{
				Code: diagWitnessMessageMismatch,
				Message: fmt.Sprintf(
					"witnessのdecisionFingerprintがbinding先Decision record本文と一致しません: witness=%s record=%s",
					rec.DecisionFingerprint, digest,
				),
			})
			record.witnessFailure()
			continue
		}
		payload, err := deriveWitnessPayload(record.Decision, rec)
		if err != nil {
			record.WitnessDiagnostics = append(record.WitnessDiagnostics, Diagnostic{
				Code:    diagWitnessTargetNotBindable,
				Message: err.Error(),
			})
			record.witnessFailure()
			continue
		}
		message, err := payload.Message()
		if err != nil {
			// deriveWitnessPayloadが妥当なpayloadを返しているため、ここには到達しない。
			record.WitnessDiagnostics = append(record.WitnessDiagnostics, Diagnostic{
				Code:    diagWitnessTargetNotBindable,
				Message: err.Error(),
			})
			record.witnessFailure()
			continue
		}
		recomputedMessageDigest := digestOf([]byte(message))
		if rec.MessageDigest != "" && rec.MessageDigest != recomputedMessageDigest {
			record.WitnessDiagnostics = append(record.WitnessDiagnostics, Diagnostic{
				Code: diagWitnessMessageMismatch,
				Message: fmt.Sprintf(
					"witness記録のmessageDigestがDecision recordから再導出したdigestと一致しません: recorded=%s recomputed=%s",
					rec.MessageDigest, recomputedMessageDigest,
				),
			})
			record.witnessFailure()
			continue
		}
		recomputedSignatureDigest := digestOf([]byte(rec.Signature))
		if rec.SignatureDigest != "" && rec.SignatureDigest != recomputedSignatureDigest {
			record.WitnessDiagnostics = append(record.WitnessDiagnostics, Diagnostic{
				Code: diagWitnessSignatureDigestMismatch,
				Message: fmt.Sprintf(
					"witness記録のsignatureDigestがsignature本文と一致しません: recorded=%s recomputed=%s",
					rec.SignatureDigest, recomputedSignatureDigest,
				),
			})
			record.witnessFailure()
			continue
		}
		// 構造的なbindingが成立した。provider検証に届かなくてもstatusは
		// 少なくともrecordedである（content-addressで改変検知可能な Evidence）。
		record.markWitnessBound()
		if verify == nil {
			continue
		}
		options := opensshwitness.ProviderOptions{
			SSHKeygen:      verify.SSHKeygen,
			AllowedSigners: verify.AllowedSigners,
			Revocation:     verify.Revocation,
			VerifyTime:     verify.VerifyTime,
		}
		evidence, verifyErr := opensshwitness.Verify(options, rec.SignerIdentity, payload, []byte(rec.Signature))
		switch {
		case errors.Is(verifyErr, opensshwitness.ErrProviderUnavailable):
			record.WitnessDiagnostics = append(record.WitnessDiagnostics, Diagnostic{
				Code:    diagWitnessProviderUnavailable,
				Message: verifyErr.Error(),
			})
		case errors.Is(verifyErr, opensshwitness.ErrVerifyTimeUnsupported):
			record.WitnessDiagnostics = append(record.WitnessDiagnostics, Diagnostic{
				Code:    diagWitnessProviderUnsupported,
				Message: verifyErr.Error(),
			})
		case verifyErr != nil:
			// options・payloadの不整合は、読み込み側のmisconfigurationとして
			// 黙って無視せずfailureとして返す。
			return fmt.Errorf("decision witness for %s: %w", rec.DecisionRecordPath, verifyErr)
		case evidence.GateStatus == opensshwitness.GateAuthorityQualified:
			// providerがpolicy・namespace・pinned verify-time・任意KRLを含めて
			// qualifiedと出した。namespaceはrecord本文から再導出済みなので、
			// qualifiedはこのDecision recordへのexact bindingを意味する。
			record.markWitnessVerified(rec.SignerIdentity, evidence)
		default:
			reason := evidence.ProviderReason
			if reason == "" {
				reason = evidence.Diagnosis
			}
			record.WitnessDiagnostics = append(record.WitnessDiagnostics, Diagnostic{
				Code: diagWitnessProviderRejected,
				Message: fmt.Sprintf("providerがwitnessをauthority-qualifiedにしませんでした: diagnosis=%s reason=%s",
					evidence.Diagnosis, reason),
			})
		}
	}
	// 同一candidateでのaccepted / rejected双方がqualifiedになった場合、winnerを
	// 決められないので紛れたrecordsをconflictedへdowngradeする。
	byCandidate := map[string][]*DecisionRecord{}
	for _, record := range records {
		if record.DecisionAuthorityStatus == authorityStatusQualified {
			byCandidate[record.Decision.CandidateFingerprint] = append(
				byCandidate[record.Decision.CandidateFingerprint], record)
		}
	}
	for _, group := range byCandidate {
		if len(group) < 2 {
			continue
		}
		semantics := map[string]bool{}
		for _, record := range group {
			semantics[record.Decision.Decision] = true
		}
		if len(semantics) < 2 {
			continue
		}
		for _, record := range group {
			record.DecisionAuthorityStatus = authorityStatusConflicted
			record.DecisionActorStatus = actorStatusUnresolved
			record.DecisionWitnessStatus = witnessStatusConflicted
			record.WitnessDiagnostics = append(record.WitnessDiagnostics, Diagnostic{
				Code: diagWitnessDecisionConflict,
				Message: fmt.Sprintf(
					"同一candidate %s に対しaccepted / rejected双方のwitnessがauthority-qualifiedになったため、winnerを決められません",
					record.Decision.CandidateFingerprint),
			})
		}
	}
	return nil
}

// witnessFailureは、一つのbinding物が構造的に壊れていることを記録する。他の
// binding物がverifiedであればverifiedな実績は失われない。
func (r *DecisionRecord) witnessFailure() {
	if r.DecisionWitnessStatus == witnessStatusVerified || r.DecisionWitnessStatus == witnessStatusConflicted {
		return
	}
	r.DecisionWitnessStatus = witnessStatusInvalid
}

// markWitnessBoundは、構造的に通ったbinding物を記録する。
func (r *DecisionRecord) markWitnessBound() {
	if r.DecisionWitnessStatus == witnessStatusNone {
		r.DecisionWitnessStatus = witnessStatusRecorded
	}
}

// markWitnessVerifiedは、providerがqualifiedと出したwitnessをrecordへ反映する。
// 異なるprincipalが同一namespace（workspace・semantics・scopeが同一）へauthorityを
// 持つことはpolicyの範囲なので、conflictにはしない。
func (r *DecisionRecord) markWitnessVerified(signer string, evidence opensshwitness.Evidence) {
	r.DecisionWitnessStatus = witnessStatusVerified
	r.DecisionActorStatus = actorStatusIdentityResolved
	r.DecisionAuthorityStatus = authorityStatusQualified
	message := fmt.Sprintf(
		"signer=%s namespace=%s verifyTime=%s evidenceFingerprint=%s",
		signer, evidence.Namespace, evidence.VerifyTime, evidence.EvidenceFingerprint,
	)
	for _, d := range r.WitnessDiagnostics {
		if d.Code == diagWitnessVerified && d.Message == message {
			return
		}
	}
	r.WitnessDiagnostics = append(r.WitnessDiagnostics, Diagnostic{
		Code:    diagWitnessVerified,
		Message: message,
	})
}

// witnessVerifyOptionsForは、CLI flagsのwitness検証configurationを検証し、verifier
// を組み立てる。allowed-signersとverify-timeは対で必須であり、片方だけは受理しない。
// 何も指定しなければnilを返し、loaderはprovider検証をしない。
func witnessVerifyOptionsFor(allowedSigners, verifyTime, revocation, sshKeygen string) (*witnessVerifyInternal, error) {
	if allowedSigners == "" && verifyTime == "" && revocation == "" {
		return nil, nil
	}
	if allowedSigners == "" || verifyTime == "" {
		return nil, errors.New("witness検証には -witness-allowed-signers と -witness-verify-time の両方が必須です")
	}
	if !witnessVerifyTimePattern.MatchString(verifyTime) {
		return nil, errors.New("-witness-verify-timeはYYYYMMDDHHMMSSZ形式で指定してください")
	}
	return &witnessVerifyInternal{
		WitnessVerifyOptions: WitnessVerifyOptions{
			AllowedSigners: allowedSigners,
			VerifyTime:     verifyTime,
			Revocation:     revocation,
		},
		SSHKeygen: sshKeygen,
	}, nil
}

// witnessOptionsFromFlagsは、flagsで与えたwitness検証configurationをloader用の
// optionsへ変換する。loader層ではprovider binaryをPATHから選ぶため、SSHKeygenの
// 明示指定はlessonの範囲外である。
func witnessOptionsFromFlags(allowedSigners, verifyTime, revocation string) (*WitnessVerifyOptions, error) {
	internal, err := witnessVerifyOptionsFor(allowedSigners, verifyTime, revocation, "")
	if err != nil {
		return nil, err
	}
	if internal == nil {
		return nil, nil
	}
	return &internal.WitnessVerifyOptions, nil
}

// witnessVerificationReportは、witness-decisionの書込み直後検証（任意）の結果を
// CLI出力へ出す形である。qualifiedでも受理状態は変わらない。
type witnessVerificationReport struct {
	ActorStatus     string       `json:"actorStatus"`
	AuthorityStatus string       `json:"authorityStatus"`
	WitnessStatus   string       `json:"witnessStatus"`
	Diagnostics     []Diagnostic `json:"diagnostics,omitempty"`
}

// verifiedDecisionRecordは、witness検証configuration付きでDecision ledgerを
// 再読み込みし、binding先recordの分類結果を返す。
func verifiedDecisionRecord(root, decisionPath string, witness *WitnessVerifyOptions) (*DecisionRecord, error) {
	index, err := loadDecisionRecords(root, witness)
	if err != nil {
		return nil, err
	}
	for i := range index.Records {
		if index.Records[i].Path == decisionPath {
			return &index.Records[i], nil
		}
	}
	return nil, fmt.Errorf("decision record %s がledgerに見つかりません", decisionPath)
}
