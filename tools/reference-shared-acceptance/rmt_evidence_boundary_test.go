package main

import (
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/tools/internal/rmtevidence"
)

// 外側のartifactFingerprintだけを書き換えた添付は、内側のtyped Evidenceが
// 正しくても拒否する。内側の改ざんを見る既存試験では、この照合は固定されない。
func TestRMTTranscriptionArtifactRejectsOuterFingerprintMismatch(t *testing.T) {
	payload, err := rmtevidence.CanonicalBytes(rmtEvidenceFixture(t, true))
	if err != nil {
		t.Fatal(err)
	}
	valid := EvidenceArtifact{
		Kind:                evidenceArtifactKindRMTTranscription,
		Schema:              rmtevidence.Schema,
		ArtifactFingerprint: sourceindex.Fingerprint(payload),
		Payload:             payload,
	}
	if _, err := normalizeEvidenceArtifact(valid); err != nil {
		t.Fatalf("対照となる正しい添付が拒否された: %v", err)
	}

	mismatched := valid
	mismatched.ArtifactFingerprint = "sha256:" + strings.Repeat("9", 64)
	if mismatched.ArtifactFingerprint == valid.ArtifactFingerprint {
		t.Fatal("試験用の外側指紋が正しい指紋と一致している")
	}
	if _, err := normalizeEvidenceArtifact(mismatched); err == nil ||
		!strings.Contains(err.Error(), "canonical payload") {
		t.Fatalf("外側のartifactFingerprintが不一致の添付が受理された: %v", err)
	}
	diagnostics := validateEvidenceArtifact(&mismatched)
	if len(diagnostics) != 1 || diagnostics[0].Code != "EVIDENCE_ARTIFACT_INVALID" {
		t.Fatalf("Shared Acceptanceの診断に現れない: %+v", diagnostics)
	}
}

// 対応していないkindは、payloadやschemaが整って見えても拒否する。
func TestEvidenceArtifactRejectsUnsupportedKind(t *testing.T) {
	payload, err := rmtevidence.CanonicalBytes(rmtEvidenceFixture(t, true))
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"rmt-observation", "unknown-kind", "RMT-TRANSCRIPTION-X"} {
		t.Run(kind, func(t *testing.T) {
			artifact := EvidenceArtifact{
				Kind:                kind,
				Schema:              rmtevidence.Schema,
				ArtifactFingerprint: sourceindex.Fingerprint(payload),
				Payload:             payload,
			}
			if _, err := normalizeEvidenceArtifact(artifact); err == nil ||
				!strings.Contains(err.Error(), "未対応") {
				t.Fatalf("未対応kind %q の添付が受理された: %v", kind, err)
			}
			if got := validateEvidenceArtifact(&artifact); len(got) != 1 || got[0].Code != "EVIDENCE_ARTIFACT_INVALID" {
				t.Fatalf("未対応kindが診断に現れない: %+v", got)
			}
		})
	}
}

// Reviewは、観測者Conformanceと外部Sourceの鮮度をShared Acceptanceが
// 確認していないことを、mapped / unmappedの両方で明示する。
func TestRMTReviewStatesUnverifiedObserverAndFreshness(t *testing.T) {
	for _, mapped := range []bool{true, false} {
		name := "unmapped"
		if mapped {
			name = "mapped"
		}
		t.Run(name, func(t *testing.T) {
			dir := scratchWorkspace(t)
			grounded, err := attachRMTTranscriptionEvidence(
				mustJSON(t, unitPriceCandidate(revisionOfDir(t, dir))),
				rmtEvidenceJSON(t, mapped),
			)
			if err != nil {
				t.Fatal(err)
			}
			ev, err := evaluate(dir, mustJSON(t, grounded))
			if err != nil {
				t.Fatal(err)
			}
			rendered := renderReviewProjection(buildReviewProjection(ev))
			for _, want := range []string{
				"observerConformance=not-verified",
				"externalSourceFreshness=not-verified-by-shared-acceptance",
			} {
				if !strings.Contains(rendered, want) {
					t.Fatalf("Reviewに %q がない:\n%s", want, rendered)
				}
			}
		})
	}
}

// 非公開のconstructや、別形式（Xeaf）の下に置かれたXeadのconstructを持つ
// Evidenceは、添付経路で拒否する。
func TestAttachRMTTranscriptionRejectsPrivateOrCrossFormatConstruct(t *testing.T) {
	dir := scratchWorkspace(t)
	candidate := mustJSON(t, unitPriceCandidate(revisionOfDir(t, dir)))
	cases := []struct {
		name      string
		format    string
		construct string
	}{
		{"XeafにXeadのconstruct", "xeaf", "IOPanel"},
		{"公開されていない顧客固有construct", "xead", "AcmeCustomerTable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// 通常の作成経路は同じ検査で失敗するため、検査前の値を直接JSONにする。
			value := rmtEvidenceFixture(t, true)
			value.SourceFormat = tc.format
			value.SourceConstruct = tc.construct
			if _, err := rmtevidence.Prepare(value); err == nil {
				t.Fatal("試験前提が崩れている: Prepareが受理した")
			}
			encoded, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := attachRMTTranscriptionEvidence(candidate, encoded); err == nil ||
				!strings.Contains(err.Error(), "標準construct") {
				t.Fatalf("添付経路が受理した、または理由が異なる: %v", err)
			}
		})
	}
}
