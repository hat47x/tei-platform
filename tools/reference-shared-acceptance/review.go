package main

import (
	"fmt"
	"sort"
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
	Sources  []string `json:"sources,omitempty"`
	Items    []string `json:"items,omitempty"`
	Note     string   `json:"note,omitempty"`
}

type ReviewProjection struct {
	Schema               string          `json:"schema"`
	CandidateID          string          `json:"candidateId,omitempty"`
	Intent               string          `json:"intent,omitempty"`
	CandidateFingerprint string          `json:"candidateFingerprint,omitempty"`
	EvidenceFingerprint  string          `json:"evidenceFingerprint,omitempty"`
	Acceptable           bool            `json:"acceptable"`
	Sections             []ReviewSection `json:"sections"`
}

func buildReviewProjection(ev evaluation) ReviewProjection {
	p := ev.preview
	review := ReviewProjection{
		Schema:               reviewSchema,
		CandidateID:          p.CandidateID,
		Intent:               ev.candidate.Intent,
		CandidateFingerprint: p.CandidateFingerprint,
		EvidenceFingerprint:  p.EvidenceFingerprint,
		Acceptable:           p.Acceptable,
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
		Sources:  []string{"preview.stages[*].diagnostics"},
		Note:     "受入stageのdiagnosticは表示しますが、業務上の例外を独立に列挙する専用Projectionはまだありません",
	}
	for _, stage := range p.Stages {
		for _, diagnostic := range stage.Diagnostics {
			item := stage.Name + ": " + diagnostic.Code + " - " + diagnostic.Message
			if diagnostic.Subject != "" {
				item += " (" + diagnostic.Subject + ")"
			}
			a3.Items = append(a3.Items, item)
		}
	}
	if len(a3.Items) == 0 {
		a3.Items = []string{"受入stageのerror diagnosticなし"}
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
		Sources:  []string{"preview.unresolved", "preview.changedCollections"},
		Note:     "権限差分を判定する専用項目はまだありません。下記は既存Previewから読める関連signalであり、権限変更の完全な判定ではありません",
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
	if len(a5.Items) == 0 {
		a5.Items = []string{"権限変更について安全に断定できる専用情報なし"}
	}
	review.Sections = append(review.Sections, a5)

	a6 := ReviewSection{
		Code:     "A6",
		Title:    "native codeへ残る箇所",
		Coverage: reviewCoverageNotEvaluated,
		Sources:  []string{"preview.changedCollections", "preview.unchangedCollections"},
		Note:     "native codeの残存箇所を列挙する専用項目はまだありません。collection差分は間接signalに留めます",
	}
	switch {
	case containsString(p.ChangedCollections, "customGoBindings"):
		a6.Items = []string{"customGoBindings collection: changed"}
	case containsString(p.UnchangedCollections, "customGoBindings"):
		a6.Items = []string{"customGoBindings collection: unchanged（native code全体が不変であることまでは示しません）"}
	default:
		a6.Items = []string{"customGoBindings collectionの状態をこのPreviewから確認できません"}
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
		a7.Note = "fingerprint・Evidence・source pathへ戻れますが、各Review項目からCanonical subjectとsourceへの直接linkはまだありません"
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
		a8.Items = []string{fmt.Sprintf("変更subject数 %d / 上限 %d", n, maxSubjectChanges)}
		if n > maxSubjectChanges {
			a8.Note = "上限を超えているため、acceptance-policyが拒否します。分割できない理由を示して例外承認する経路はありません"
		}
	} else {
		a8.Note = "semantic-diff stageが完了していないため、変更subject数を評価できません"
	}
	review.Sections = append(review.Sections, a8)

	return review
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
	if review.Acceptable {
		b.WriteString("preview: 受入可能\n")
	} else {
		b.WriteString("preview: 受入不可\n")
	}

	for _, section := range review.Sections {
		fmt.Fprintf(&b, "\n## %s %s [%s]\n", section.Code, section.Title, reviewCoverageLabel(section.Coverage))
		items := append([]string(nil), section.Items...)
		sort.Strings(items)
		for _, item := range items {
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
