package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/knowledgecontext"
	"tei/internal/sourceindex"
)

const candidateSchema = "tei.reference.change-candidate/v0"

// Candidateは、このReference専用の最小semantic deltaの運搬形式である。
// platform共通のProposal schemaやPatch DSLではなく、入口（file / CLI / Studio / AI）に依らず
// 同じ受入境界へ渡すために、意味の変更だけを持つ。
type Candidate struct {
	Schema            string      `json:"schema"`
	ID                string      `json:"id"`
	BaseModelRevision string      `json:"baseModelRevision"`
	Origin            Origin      `json:"origin"`
	Intent            string      `json:"intent"`
	Evidence          []Evidence  `json:"evidence,omitempty"`
	Delta             []Operation `json:"delta"`
}

// Originはprovenanceだけを保持する。fingerprint・受入判定・authoritative sourceへは影響しない。
type Origin struct {
	Kind  string            `json:"kind"`
	Actor string            `json:"actor,omitempty"`
	Notes map[string]string `json:"notes,omitempty"`
}

// Evidenceは、candidateの根拠を既存のCanonical ID、またはsubjectKind付きのSource Index identityとworkspace相対sourceで示す。
type Evidence struct {
	SubjectKind                     string                                        `json:"subjectKind,omitempty"`
	Subject                         string                                        `json:"subject,omitempty"`
	Source                          string                                        `json:"source,omitempty"`
	SourceFingerprint               string                                        `json:"sourceFingerprint,omitempty"`
	QualificationRequirements       *knowledgecontext.QualificationRequirements   `json:"qualificationRequirements,omitempty"`
	QualificationDependencyEvidence []knowledgecontext.QualificationEvidence      `json:"qualificationDependencyEvidence,omitempty"`
	Artifact                        *EvidenceArtifact                             `json:"artifact,omitempty"`
	Note                            string                                        `json:"note,omitempty"`
}

var revisionPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func decodeCandidate(data []byte) (Candidate, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var c Candidate
	if err := decoder.Decode(&c); err != nil {
		return Candidate{}, fmt.Errorf("candidateをJSONとして読めません: %w", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return Candidate{}, fmt.Errorf("candidateのJSONの後ろに余分な内容があります")
	}
	return c, nil
}

// normalizeCandidateは入力を正規形へ揃え、意味を持つ部分（base revisionとdelta）のfingerprintを返す。
// Origin / Intent / Evidenceはfingerprintへ含めないため、同じdeltaは入口に依らず同じfingerprintになる。
func normalizeCandidate(c Candidate) (Candidate, string, error) {
	if c.Schema != candidateSchema {
		return c, "", fmt.Errorf("schema %q は未対応です（%q を指定してください）", c.Schema, candidateSchema)
	}
	c.ID = strings.TrimSpace(c.ID)
	if c.ID == "" {
		return c, "", fmt.Errorf("candidate idが空です")
	}
	if !revisionPattern.MatchString(c.BaseModelRevision) {
		return c, "", fmt.Errorf("baseModelRevision %q は sha256:<64桁の16進> 形式ではありません", c.BaseModelRevision)
	}
	switch c.Origin.Kind {
	case "file", "cli", "studio", "ai":
	default:
		return c, "", fmt.Errorf("origin.kind %q は file / cli / studio / ai のいずれかである必要があります", c.Origin.Kind)
	}
	c.Intent = strings.TrimSpace(c.Intent)
	if c.Intent == "" {
		return c, "", fmt.Errorf("intent（何を直したいか）が空です")
	}
	if len(c.Delta) == 0 {
		return c, "", fmt.Errorf("deltaが空です")
	}

	evidence := make([]Evidence, 0, len(c.Evidence))
	evidenceSeen := map[string]struct{}{}
	for i, e := range c.Evidence {
		e.SubjectKind = strings.TrimSpace(e.SubjectKind)
		e.Subject = strings.TrimSpace(e.Subject)
		e.Source = strings.TrimSpace(e.Source)
		e.SourceFingerprint = strings.TrimSpace(e.SourceFingerprint)
		e.Note = strings.TrimSpace(e.Note)
		if e.Subject == "" && e.Source == "" && e.Artifact == nil {
			return c, "", fmt.Errorf("evidence[%d]にsubject / source / artifactのいずれもありません", i)
		}
		if e.Artifact != nil {
			if e.SubjectKind != "" || e.Subject != "" || e.Source != "" || e.SourceFingerprint != "" ||
				e.QualificationRequirements != nil || len(e.QualificationDependencyEvidence) != 0 {
				return c, "", fmt.Errorf("evidence[%d].artifactはSource Index Evidenceと同じitemへ混在できません", i)
			}
			normalizedArtifact, err := normalizeEvidenceArtifact(*e.Artifact)
			if err != nil {
				return c, "", fmt.Errorf("evidence[%d].artifact: %w", i, err)
			}
			e.Artifact = &normalizedArtifact
		}
		if e.SubjectKind != "" && e.Subject == "" {
			return c, "", fmt.Errorf("evidence[%d].subjectKindにはsubjectが必要です", i)
		}
		if e.SourceFingerprint != "" && e.Source == "" {
			return c, "", fmt.Errorf("evidence[%d].sourceFingerprintにはsourceが必要です", i)
		}
		if e.QualificationRequirements != nil &&
			(e.SubjectKind == "" || e.Subject == "" || e.Source == "" || e.SourceFingerprint == "") {
			return c, "", fmt.Errorf("evidence[%d].qualificationRequirementsにはsubjectKind / subject / source / sourceFingerprintが必要です", i)
		}
		requiredDependencySubjects := map[string]struct{}{}
		if e.QualificationRequirements != nil {
			normalizedRequirements, err := knowledgecontext.NormalizeQualificationRequirements(*e.QualificationRequirements)
			if err != nil {
				return c, "", fmt.Errorf("evidence[%d].qualificationRequirements: %w", i, err)
			}
			if normalizedRequirements.ExpectedKind != e.SubjectKind {
				return c, "", fmt.Errorf(
					"evidence[%d].qualificationRequirements.expectedKind %q はsubjectKind %q と一致しません",
					i,
					normalizedRequirements.ExpectedKind,
					e.SubjectKind,
				)
			}
			for _, requirement := range normalizedRequirements.RequiredDependencies {
				requiredDependencySubjects[candidateDependencySubjectKey(requirement.Kind, requirement.ID)] = struct{}{}
			}
			e.QualificationRequirements = &normalizedRequirements
		}
		if len(e.QualificationDependencyEvidence) > 0 && e.QualificationRequirements == nil {
			return c, "", fmt.Errorf("evidence[%d].qualificationDependencyEvidenceにはqualificationRequirementsが必要です", i)
		}
		seenDependencySubjects := map[string]struct{}{}
		for j := range e.QualificationDependencyEvidence {
			dependency := &e.QualificationDependencyEvidence[j]
			dependency.SubjectKind = strings.TrimSpace(dependency.SubjectKind)
			dependency.Subject = strings.TrimSpace(dependency.Subject)
			dependency.Source = strings.TrimSpace(dependency.Source)
			dependency.SourceFingerprint = strings.TrimSpace(dependency.SourceFingerprint)
			if dependency.SubjectKind == "" ||
				dependency.Subject == "" ||
				dependency.Source == "" ||
				!revisionPattern.MatchString(dependency.SourceFingerprint) {
				return c, "", fmt.Errorf("evidence[%d].qualificationDependencyEvidence[%d]はexact subject / source / fingerprintが必要です", i, j)
			}
			p, err := sourceindex.NormalizeWorkspacePath(dependency.Source)
			if err != nil {
				return c, "", fmt.Errorf("evidence[%d].qualificationDependencyEvidence[%d].source: %w", i, j, err)
			}
			dependency.Source = p
			subjectKey := candidateDependencySubjectKey(dependency.SubjectKind, dependency.Subject)
			if _, required := requiredDependencySubjects[subjectKey]; !required {
				return c, "", fmt.Errorf(
					"evidence[%d].qualificationDependencyEvidence[%d]のsubject %s:%s はrequiredDependenciesにありません",
					i,
					j,
					dependency.SubjectKind,
					dependency.Subject,
				)
			}
			seenDependencySubjects[subjectKey] = struct{}{}
		}
		for subjectKey := range requiredDependencySubjects {
			if _, found := seenDependencySubjects[subjectKey]; !found {
				return c, "", fmt.Errorf(
					"evidence[%d]のrequired dependency %q にauthoritative Evidenceがありません",
					i,
					subjectKey,
				)
			}
		}
		if len(e.QualificationDependencyEvidence) > 0 {
			dependencies := make([]knowledgecontext.QualificationEvidence, 0, len(e.QualificationDependencyEvidence))
			seenDependencies := map[string]struct{}{}
			for _, dependency := range e.QualificationDependencyEvidence {
				key := candidateEvidenceJSONKey(dependency)
				if _, duplicate := seenDependencies[key]; duplicate {
					continue
				}
				seenDependencies[key] = struct{}{}
				dependencies = append(dependencies, dependency)
			}
			sort.Slice(dependencies, func(i, j int) bool {
				return candidateEvidenceJSONKey(dependencies[i]) < candidateEvidenceJSONKey(dependencies[j])
			})
			e.QualificationDependencyEvidence = dependencies
		}
		if e.Subject != "" && e.SubjectKind == "" {
			if _, err := core.ParseCanonicalID(e.Subject); err != nil {
				return c, "", fmt.Errorf("evidence[%d].subject: %w", i, err)
			}
		}
		if e.Source != "" {
			p, err := sourceindex.NormalizeWorkspacePath(e.Source)
			if err != nil {
				return c, "", fmt.Errorf("evidence[%d].source: %w", i, err)
			}
			e.Source = p
		}
		if e.SourceFingerprint != "" && !revisionPattern.MatchString(e.SourceFingerprint) {
			return c, "", fmt.Errorf("evidence[%d].sourceFingerprint %q は sha256:<64桁の16進> 形式ではありません", i, e.SourceFingerprint)
		}
		key := candidateEvidenceJSONKey(e)
		if _, duplicate := evidenceSeen[key]; duplicate {
			continue
		}
		evidenceSeen[key] = struct{}{}
		evidence = append(evidence, e)
	}
	sort.Slice(evidence, func(i, j int) bool {
		return candidateEvidenceJSONKey(evidence[i]) < candidateEvidenceJSONKey(evidence[j])
	})
	c.Evidence = evidence

	ops := make([]Operation, 0, len(c.Delta))
	seen := map[string]bool{}
	for i, op := range c.Delta {
		normalized, err := normalizeOperation(op)
		if err != nil {
			return c, "", fmt.Errorf("delta[%d]: %w", i, err)
		}
		key := operationKey(normalized)
		if seen[key] {
			return c, "", fmt.Errorf("delta[%d]: %s がdelta内で重複しています", i, key)
		}
		seen[key] = true
		ops = append(ops, normalized)
	}
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Op != ops[j].Op {
			return ops[i].Op < ops[j].Op
		}
		return operationKey(ops[i]) < operationKey(ops[j])
	})
	c.Delta = ops
	return c, fingerprintOf(c.BaseModelRevision, ops), nil
}

func candidateDependencySubjectKey(kind, id string) string {
	return kind + "\x00" + id
}

func candidateEvidenceJSONKey(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func evidenceFingerprintOf(evidence []Evidence) string {
	if len(evidence) == 0 {
		return ""
	}
	raw, _ := json.Marshal(evidence)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func fingerprintOf(baseModelRevision string, ops []Operation) string {
	raw, _ := json.Marshal(struct {
		BaseModelRevision string      `json:"baseModelRevision"`
		Delta             []Operation `json:"delta"`
	}{baseModelRevision, ops})
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
