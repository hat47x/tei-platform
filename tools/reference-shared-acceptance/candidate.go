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
	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

const (
	candidateSchema = "tei.reference.change-candidate/v0"
	opAddValidation = "add-validation"
)

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

// Evidenceは、candidateの根拠をqualified logical key（Canonical ID）またはworkspace相対のsource pathで示す。
type Evidence struct {
	Subject string `json:"subject,omitempty"`
	Source  string `json:"source,omitempty"`
	Note    string `json:"note,omitempty"`
}

type Operation struct {
	Op        string `json:"op"`
	Namespace string `json:"namespace"`
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Attribute string `json:"attribute"`
	Relation  string `json:"relation,omitempty"`
	Placement string `json:"placement,omitempty"`
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
	for i, e := range c.Evidence {
		e.Subject = strings.TrimSpace(e.Subject)
		e.Source = strings.TrimSpace(e.Source)
		if e.Subject == "" && e.Source == "" {
			return c, "", fmt.Errorf("evidence[%d]にsubjectもsourceもありません", i)
		}
		if e.Subject != "" {
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
		evidence = append(evidence, e)
	}
	c.Evidence = evidence

	ops := make([]Operation, 0, len(c.Delta))
	seen := map[string]bool{}
	for i, op := range c.Delta {
		normalized, err := normalizeOperation(op)
		if err != nil {
			return c, "", fmt.Errorf("delta[%d]: %w", i, err)
		}
		if seen[normalized.ID] {
			return c, "", fmt.Errorf("delta[%d]: validation %s がdelta内で重複しています", i, normalized.ID)
		}
		seen[normalized.ID] = true
		ops = append(ops, normalized)
	}
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Op != ops[j].Op {
			return ops[i].Op < ops[j].Op
		}
		return ops[i].ID < ops[j].ID
	})
	c.Delta = ops
	return c, fingerprintOf(c.BaseModelRevision, ops), nil
}

func normalizeOperation(op Operation) (Operation, error) {
	op.Op = strings.TrimSpace(op.Op)
	if op.Op != opAddValidation {
		return op, fmt.Errorf("operation %q は未対応です（このReferenceは %q だけを扱います）", op.Op, opAddValidation)
	}
	namespace, err := core.ParseCanonicalID(op.Namespace)
	if err != nil {
		return op, fmt.Errorf("namespace: %w", err)
	}
	id, err := core.ParseCanonicalID(op.ID)
	if err != nil {
		return op, fmt.Errorf("id: %w", err)
	}
	if !strings.HasPrefix(string(id), string(namespace)+".") {
		return op, fmt.Errorf("validation id %s はnamespace %s に属していません", id, namespace)
	}
	kind := semantic.ValidationKind(strings.TrimSpace(op.Kind))
	if !kind.Valid() {
		return op, fmt.Errorf("validation kind %q は未対応です", op.Kind)
	}
	attribute, err := core.ParseCanonicalID(op.Attribute)
	if err != nil {
		return op, fmt.Errorf("attribute: %w", err)
	}
	relation := core.CanonicalID("")
	switch kind {
	case semantic.ValidationPositive:
		if strings.TrimSpace(op.Relation) != "" {
			return op, fmt.Errorf("positive validation %s はrelationを受け付けません", id)
		}
	case semantic.ValidationRelationUnique:
		relation, err = core.ParseCanonicalID(op.Relation)
		if err != nil {
			return op, fmt.Errorf("relation-unique validation %s にはrelationが必要です: %w", id, err)
		}
	}
	placement := ""
	if strings.TrimSpace(op.Placement) != "" {
		placement, err = sourceindex.NormalizeWorkspacePath(op.Placement)
		if err != nil {
			return op, fmt.Errorf("placement: %w", err)
		}
	}
	return Operation{
		Op:        op.Op,
		Namespace: string(namespace),
		ID:        string(id),
		Kind:      string(kind),
		Attribute: string(attribute),
		Relation:  string(relation),
		Placement: placement,
	}, nil
}

func fingerprintOf(baseModelRevision string, ops []Operation) string {
	raw, _ := json.Marshal(struct {
		BaseModelRevision string      `json:"baseModelRevision"`
		Delta             []Operation `json:"delta"`
	}{baseModelRevision, ops})
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
