package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const reviewSchema = "tei.reference.approval-review/v0"

const (
	reviewCoverageAvailable    = "available"
	reviewCoveragePartial      = "partial"
	reviewCoverageNotEvaluated = "not-evaluated"
)

type ReviewSection struct {
	Code     string   `json:"code"`
	Title    string   `json:"title"`
	Coverage string   `json:"coverage"`
	Summary  string   `json:"summary,omitempty"`
	Sources  []string `json:"sources,omitempty"`
	Items    []string `json:"items,omitempty"`
	Note     string   `json:"note,omitempty"`
}

type ReviewCoverageGap struct {
	Code     string `json:"code"`
	Title    string `json:"title"`
	Coverage string `json:"coverage"`
}

type ReviewCoverageSummary struct {
	Available    int                 `json:"available"`
	Partial      int                 `json:"partial"`
	NotEvaluated int                 `json:"notEvaluated"`
	Incomplete   []ReviewCoverageGap `json:"incomplete,omitempty"`
}

type ReviewProjection struct {
	Schema               string                `json:"schema"`
	CandidateID          string                `json:"candidateId,omitempty"`
	Intent               string                `json:"intent,omitempty"`
	CandidateFingerprint string                `json:"candidateFingerprint,omitempty"`
	EvidenceFingerprint  string                `json:"evidenceFingerprint,omitempty"`
	ReviewFingerprint    string                `json:"reviewFingerprint,omitempty"`
	PreviewAcceptable    bool                  `json:"previewAcceptable"`
	Coverage             ReviewCoverageSummary `json:"coverage"`
	Sections             []ReviewSection       `json:"sections"`
}

func buildReviewProjection(ev evaluation) ReviewProjection {
	p := ev.preview
	review := ReviewProjection{
		Schema:               reviewSchema,
		CandidateID:          p.CandidateID,
		Intent:               ev.candidate.Intent,
		CandidateFingerprint: p.CandidateFingerprint,
		EvidenceFingerprint:  p.EvidenceFingerprint,
		PreviewAcceptable:    p.Acceptable,
	}

	_, semanticEvaluated := reviewStage(p, "semantic-diff")
	_, impactEvaluated := reviewStage(p, "impact")
	normalizeStage, normalizeReached := reviewStage(p, "normalize")
	normalizeEvaluated := normalizeReached && normalizeStage.Status == "pass"
	applyStage, applyReached := reviewStage(p, "apply")
	applyEvaluated := applyReached && applyStage.Status == "pass"

	a1 := ReviewSection{
		Code:     "A1",
		Title:    "意味差分",
		Coverage: reviewCoverageNotEvaluated,
		Sources:  []string{"preview.semanticDiff", "preview.applicationDiff", "preview.environmentDiff"},
	}
	if semanticEvaluated {
		a1.Coverage = reviewCoverageAvailable
		a1.Summary = fmt.Sprintf(
			"semantic %d件 / application %d件 / environment %d件",
			len(p.SemanticDiff),
			len(p.ApplicationDiff),
			len(p.EnvironmentDiff),
		)
		for _, change := range allChanges(p) {
			a1.Items = append(a1.Items, fmt.Sprintf("%s %s:%s", change.Change, change.Kind, change.ID))
		}
		if len(a1.Items) == 0 {
			a1.Items = []string{"変更subjectなし"}
		}
	} else {
		a1.Note = "semantic-diff stageが完了していないため、意味差分を確定できません"
	}
	review.Sections = append(review.Sections, a1)

	a2 := ReviewSection{
		Code:     "A2",
		Title:    "影響",
		Coverage: reviewCoverageNotEvaluated,
		Sources:  []string{"preview.impact", "preview.unresolved"},
	}
	if impactEvaluated {
		a2.Coverage = reviewCoverageAvailable
		a2.Summary = fmt.Sprintf("impact %d本 / unresolved %d件", len(p.Impact), len(p.Unresolved))
		for _, edge := range p.Impact {
			a2.Items = append(a2.Items, fmt.Sprintf("%s -> %s via %s: %s", edge.From, edge.To, edge.Via, edge.Reason))
		}
		if len(a2.Items) == 0 {
			a2.Items = []string{"exact identityから導出されたimpact edgeなし"}
		}
		if len(p.Unresolved) > 0 {
			a2.Coverage = reviewCoveragePartial
			a2.Note = "解析できなかった範囲があります。A4 未解決点を確認してください"
		}
	} else {
		a2.Note = "impact stageが完了していないため、影響範囲を確定できません"
	}
	review.Sections = append(review.Sections, a2)

	a3 := ReviewSection{
		Code:     "A3",
		Title:    "例外・診断",
		Coverage: reviewCoveragePartial,
		Sources:  []string{"preview.stages[*].diagnostics", "candidate.evidence[*].artifact"},
		Note:     "受入stageと添付Evaluation artifactのdiagnosticを表示します。Evaluationのpass / fail / unknownは根拠であり、それ自体からAccept / Rejectを自動導出しません",
	}
	diagnosticCount := 0
	evaluationCount := 0
	for _, stage := range p.Stages {
		for _, diagnostic := range stage.Diagnostics {
			diagnosticCount++
			item := stage.Name + ": " + diagnostic.Code + " - " + diagnostic.Message
			if diagnostic.Subject != "" {
				item += " (" + diagnostic.Subject + ")"
			}
			a3.Items = append(a3.Items, item)
		}
	}
	for _, evidence := range ev.candidate.Evidence {
		evaluation, ok := semanticEvaluationFromEvidence(evidence)
		if !ok {
			continue
		}
		evaluationCount++
		a3.Items = append(a3.Items, fmt.Sprintf(
			"semantic evaluation: status=%s subject=%s:%s constraint=%s/%s evaluator=%s artifact=%s",
			evaluation.Status,
			evaluation.Subject.Kind,
			evaluation.Subject.ID,
			evaluation.ConstraintExtension,
			evaluation.ConstraintID,
			evaluation.EvaluatorImplementationID,
			evidence.Artifact.ArtifactFingerprint,
		))
		for _, diagnostic := range evaluation.Diagnostics {
			diagnosticCount++
			a3.Items = append(a3.Items, "semantic evaluation: "+diagnostic.Code+" - "+diagnostic.Message)
		}
	}
	a3.Summary = fmt.Sprintf("diagnostic %d件 / semantic evaluation %d件", diagnosticCount, evaluationCount)
	if len(a3.Items) == 0 {
		a3.Items = []string{"受入stageのerror diagnosticなし / semantic evaluation artifactなし"}
	}
	review.Sections = append(review.Sections, a3)

	a4 := ReviewSection{
		Code:     "A4",
		Title:    "未解決点",
		Coverage: reviewCoverageNotEvaluated,
		Sources:  []string{"preview.unresolved"},
	}
	if impactEvaluated {
		a4.Coverage = reviewCoverageAvailable
		a4.Summary = fmt.Sprintf("unresolved %d件", len(p.Unresolved))
		for _, unresolved := range p.Unresolved {
			a4.Items = append(a4.Items, unresolved.Code+" - "+unresolved.Message)
		}
		if len(a4.Items) == 0 {
			a4.Items = []string{"Previewが明示した未解決点なし"}
		}
	} else {
		a4.Note = "impact stageが完了していないため、未解決点の一覧を確定できません"
	}
	review.Sections = append(review.Sections, a4)

	a5 := ReviewSection{
		Code:     "A5",
		Title:    "権限の変更",
		Coverage: reviewCoverageNotEvaluated,
		Sources:  []string{"candidate.delta", "preview.unresolved", "preview.changedCollections"},
	}
	if normalizeEvaluated {
		a5.Coverage = reviewCoveragePartial
		a5.Summary = "runtime authority grant 0件 / effective authority impact 未評価"
		a5.Items = append(a5.Items,
			"candidateがauthorするruntime authority grant: 0件（現行operationはPermission grant / Resource binding / lease / execution principalをauthorしません）",
		)
		var applicationCount, bindingCount int
		for _, op := range ev.candidate.Delta {
			switch op.Op {
			case opAddApplication:
				applicationCount++
			case opAddEnvironmentBinding:
				bindingCount++
			}
		}
		if applicationCount > 0 {
			a5.Items = append(a5.Items, fmt.Sprintf(
				"Application追加: %d件（Environment binding / Provider / Permission grantはadd-applicationに含まれません）",
				applicationCount,
			))
		}
		if bindingCount > 0 {
			a5.Items = append(a5.Items, fmt.Sprintf(
				"Environment binding追加: %d件（datasource aliasとinstanceの対応であり、runtime authority grantではありません）",
				bindingCount,
			))
		}
		for _, unresolved := range p.Unresolved {
			if strings.Contains(unresolved.Code, "AUTHORITY") ||
				strings.Contains(unresolved.Code, "PERMISSION") ||
				unresolved.Code == "ENVIRONMENT_BINDING_NOT_AUTHORED" {
				a5.Items = append(a5.Items, unresolved.Code+" - "+unresolved.Message)
			}
		}
		if len(p.ChangedCollections) > 0 {
			a5.Items = append(a5.Items, "changed collections: "+strings.Join(p.ChangedCollections, ", "))
		}
		a5.Note = "既存のPermission grant / leaseが変更後のApplicationや新しいbindingへどう及ぶかは評価していません。candidate自身がgrantをauthorしないことと、実効権限が変わらないことは別です"
	} else {
		a5.Note = "normalize stageが完了していないため、candidateがauthorする権限変更の範囲も確定できません"
	}
	review.Sections = append(review.Sections, a5)

	a6 := ReviewSection{
		Code:     "A6",
		Title:    "native codeへ残る箇所",
		Coverage: reviewCoverageNotEvaluated,
		Sources:  []string{"candidate.delta", "preview.changedCollections", "preview.unchangedCollections"},
	}
	if normalizeEvaluated {
		a6.Coverage = reviewCoveragePartial
		a6.Summary = "native code authoring 0件 / residual native dependency impact 未評価"
		a6.Items = append(a6.Items,
			"candidateがauthorするnative code: 0件（現行operationはcustomGoBindingsをauthorしません）",
		)
		switch {
		case containsString(p.ChangedCollections, "customGoBindings"):
			a6.Items = append(a6.Items, "customGoBindings collection: changed")
		case containsString(p.UnchangedCollections, "customGoBindings"):
			a6.Items = append(a6.Items, "customGoBindings collection: unchanged")
		default:
			a6.Items = append(a6.Items, "customGoBindings collectionの状態は、このPreviewの到達段階では未確認")
		}
		a6.Note = "既存custom implementationのどこが変更subjectに依存し、残存・再実装・確認を要するかはまだ追跡していません"
	} else {
		a6.Note = "normalize stageが完了していないため、native code authoringの有無も確定できません"
	}
	review.Sections = append(review.Sections, a6)

	a7 := ReviewSection{
		Code:     "A7",
		Title:    "逆追跡",
		Coverage: reviewCoverageNotEvaluated,
		Sources: []string{
			"candidate.evidence",
			"preview.candidateFingerprint",
			"preview.evidenceFingerprint",
			"preview.sourceEdits",
		},
	}
	if normalizeEvaluated {
		a7.Coverage = reviewCoveragePartial
		a7.Summary = fmt.Sprintf("Evidence %d件 / source edit %d件", len(ev.candidate.Evidence), len(p.SourceEdits))
		a7.Items = append(a7.Items,
			"candidateFingerprint: "+p.CandidateFingerprint,
			"baseModelRevision: "+p.BaseModelRevision,
			"currentModelRevision: "+p.CurrentModelRevision,
		)
		if p.EvidenceFingerprint != "" {
			a7.Items = append(a7.Items, "evidenceFingerprint: "+p.EvidenceFingerprint)
		}
		for _, evidence := range ev.candidate.Evidence {
			a7.Items = append(a7.Items, describeReviewEvidence(evidence))
			for _, dependency := range evidence.QualificationDependencyEvidence {
				a7.Items = append(a7.Items, fmt.Sprintf(
					"dependency Evidence: %s:%s source=%s fingerprint=%s",
					dependency.SubjectKind,
					dependency.Subject,
					dependency.Source,
					dependency.SourceFingerprint,
				))
			}
		}
		if applyEvaluated {
			for _, edit := range p.SourceEdits {
				a7.Items = append(a7.Items, fmt.Sprintf(
					"source edit: %s layer=%s after=%s",
					edit.Path,
					edit.Layer,
					edit.AfterFingerprint,
				))
			}
		}
		a7.Note = "Source Index Evidenceとartifact Evidenceを分離して表示します。artifact Evidenceは受理時に見たexact artifactを固定しますが、現在のEvaluator / Conformance状態を再実行・再証明したことにはなりません"
	} else {
		a7.Note = "normalize stageが完了していないため、candidate / Evidence identityを確定できません"
	}
	review.Sections = append(review.Sections, a7)

	a8 := ReviewSection{
		Code:     "A8",
		Title:    "承認単位の大きさ",
		Coverage: reviewCoverageNotEvaluated,
		Sources: []string{
			"preview.semanticDiff",
			"preview.applicationDiff",
			"preview.environmentDiff",
			"acceptance-policy.maxSubjectChanges",
		},
	}
	if semanticEvaluated {
		n := len(allChanges(p))
		a8.Coverage = reviewCoverageAvailable
		a8.Summary = fmt.Sprintf("%d / %d", n, maxSubjectChanges)
		a8.Items = []string{fmt.Sprintf("変更subject数 %d / 上限 %d", n, maxSubjectChanges)}
		if n > maxSubjectChanges {
			a8.Note = "上限を超えているため、acceptance-policyが拒否します。分割できない理由を示して例外承認する経路はありません"
		}
	} else {
		a8.Note = "semantic-diff stageが完了していないため、変更subject数を評価できません"
	}
	review.Sections = append(review.Sections, a8)

	review.Coverage = summarizeReviewCoverage(review.Sections)
	review.ReviewFingerprint = fingerprintReviewProjection(review)
	return review
}

func fingerprintReviewProjection(review ReviewProjection) string {
	review.ReviewFingerprint = ""
	raw, _ := json.Marshal(review)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func summarizeReviewCoverage(sections []ReviewSection) ReviewCoverageSummary {
	var summary ReviewCoverageSummary
	for _, section := range sections {
		switch section.Coverage {
		case reviewCoverageAvailable:
			summary.Available++
		case reviewCoveragePartial:
			summary.Partial++
		default:
			summary.NotEvaluated++
		}
		if section.Coverage != reviewCoverageAvailable {
			summary.Incomplete = append(summary.Incomplete, ReviewCoverageGap{
				Code:     section.Code,
				Title:    section.Title,
				Coverage: section.Coverage,
			})
		}
	}
	return summary
}

func reviewStage(p Preview, name string) (Stage, bool) {
	for _, stage := range p.Stages {
		if stage.Name != name {
			continue
		}
		return stage, stage.Status == "pass" || stage.Status == "fail"
	}
	return Stage{}, false
}

func describeReviewEvidence(e Evidence) string {
	var parts []string
	if e.Artifact != nil {
		parts = append(parts,
			"artifactKind="+e.Artifact.Kind,
			"artifactSchema="+e.Artifact.Schema,
			"artifactFingerprint="+e.Artifact.ArtifactFingerprint,
		)
		if evaluation, ok := semanticEvaluationFromEvidence(e); ok {
			parts = append(parts,
				"status="+string(evaluation.Status),
				"subject="+evaluation.Subject.Kind+":"+evaluation.Subject.ID,
				"constraint="+evaluation.ConstraintExtension+"/"+evaluation.ConstraintID,
				"semanticSet="+evaluation.SemanticSetFingerprint,
				"evaluationFingerprint="+evaluation.EvidenceFingerprint,
			)
		}
		if rmt, ok := rmtTranscriptionFromEvidence(e); ok {
			parts = append(parts,
				"sourceFormat="+rmt.SourceFormat,
				"sourceFormatVersion="+rmt.SourceFormatVersion,
				"sourceArtifact="+rmt.SourceArtifactDigest,
				"sourceToken="+rmt.SourceToken,
				"mappingContract="+rmt.MappingContractFingerprint,
				"mappingStatus="+string(rmt.MappingStatus),
				"rmtEvidenceFingerprint="+rmt.EvidenceFingerprint,
			)
			if rmt.SourceConstructKnown {
				parts = append(parts, "sourceConstruct="+rmt.SourceConstruct)
			}
			if len(rmt.Classification) > 0 {
				parts = append(parts, "classification="+strings.Join(rmt.Classification, ","))
			}
			if rmt.Extension != "" {
				parts = append(parts, "extension="+rmt.Extension)
			}
		}
	}
	switch {
	case e.SubjectKind != "" && e.Subject != "":
		parts = append(parts, "subject="+e.SubjectKind+":"+e.Subject)
	case e.Subject != "":
		parts = append(parts, "subject="+e.Subject)
	}
	if e.Source != "" {
		parts = append(parts, "source="+e.Source)
	}
	if e.SourceFingerprint != "" {
		parts = append(parts, "fingerprint="+e.SourceFingerprint)
	}
	if e.Note != "" {
		parts = append(parts, "note="+e.Note)
	}
	if len(parts) == 0 {
		return "Evidence: （表示可能な参照なし）"
	}
	return "Evidence: " + strings.Join(parts, " ")
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func renderReviewProjection(review ReviewProjection) string {
	var b strings.Builder
	b.WriteString("# Shared Acceptance Review\n")
	if review.CandidateID != "" {
		fmt.Fprintf(&b, "candidate: %s\n", review.CandidateID)
	}
	if review.Intent != "" {
		fmt.Fprintf(&b, "intent: %s\n", review.Intent)
	}
	if review.CandidateFingerprint != "" {
		fmt.Fprintf(&b, "candidate fingerprint: %s\n", review.CandidateFingerprint)
	}
	if review.EvidenceFingerprint != "" {
		fmt.Fprintf(&b, "evidence fingerprint: %s\n", review.EvidenceFingerprint)
	}
	if review.ReviewFingerprint != "" {
		fmt.Fprintf(&b, "review fingerprint: %s\n", review.ReviewFingerprint)
	}
	if review.PreviewAcceptable {
		b.WriteString("machine preview: 受入可能\n")
	} else {
		b.WriteString("machine preview: 受入不可\n")
	}
	fmt.Fprintf(
		&b,
		"coverage: 表示可能 %d / 一部 %d / 未評価 %d\n",
		review.Coverage.Available,
		review.Coverage.Partial,
		review.Coverage.NotEvaluated,
	)
	if len(review.Coverage.Incomplete) > 0 {
		var gaps []string
		for _, gap := range review.Coverage.Incomplete {
			gaps = append(gaps, fmt.Sprintf(
				"%s %s [%s]",
				gap.Code,
				gap.Title,
				reviewCoverageLabel(gap.Coverage),
			))
		}
		fmt.Fprintf(&b, "確認が残るsection: %s\n", strings.Join(gaps, ", "))
	}

	for _, section := range review.Sections {
		fmt.Fprintf(&b, "\n## %s %s [%s]\n", section.Code, section.Title, reviewCoverageLabel(section.Coverage))
		if section.Summary != "" {
			fmt.Fprintf(&b, "要約: %s\n", section.Summary)
		}
		for _, item := range section.Items {
			fmt.Fprintf(&b, "- %s\n", item)
		}
		if len(section.Sources) > 0 {
			fmt.Fprintf(&b, "根拠: %s\n", strings.Join(section.Sources, ", "))
		}
		if section.Note != "" {
			fmt.Fprintf(&b, "注記: %s\n", section.Note)
		}
	}
	return b.String()
}

func reviewCoverageLabel(coverage string) string {
	switch coverage {
	case reviewCoverageAvailable:
		return "表示可能"
	case reviewCoveragePartial:
		return "一部"
	default:
		return "未評価"
	}
}
