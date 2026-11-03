package orderruledesign

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

// Attachment sits beside the candidate. It carries the provenance state and
// the unresolved list without entering the candidate's meaning fingerprint.
type Attachment struct {
	Schema              string           `json:"schema"`
	ProvenanceState     string           `json:"provenanceState"`
	Unresolved          Unresolved       `json:"unresolved"`
	SourceDigest        string           `json:"sourceDigest"`
	ContractDigest      string           `json:"contractDigest"`
	EvidenceFingerprint string           `json:"evidenceFingerprint"`
	UnresolvedList      []UnresolvedItem `json:"unresolvedList"`
	TemplateQuestions   int              `json:"templateQuestions"`
}

type stage3Out struct {
	Dir         string
	Candidate   []byte
	Attachment  []byte
	Preview     map[string]any
	PreviewRaw  []byte
	Fingerprint string
	Carried     Carried
	Detail      map[string]any
}

const studioEdit = `{"schema":"tei.reference.studio-edit/v0","id":"unit-price-positive","origin":"studio",` +
	`"intent":"order lineの単価が正であることを業務ruleとして明示する","view":"table",` +
	`"subject":"attribute:` + RuleAttributeID + `","operation":"add-validation","validationID":"` + RuleValidationID + `",` +
	`"validationKind":"positive","applicationUse":{"application":"` + RuleApplication + `","placement":"` + ApplicationPlace + `"}}`

var passRe = regexp.MustCompile(`(?m)^# (pass|fail) (\d+)`)

// templateStep calls reference/business-spec-templates unchanged: its own
// tests, and proposeAdoption, whose unresolved questions stay unresolved.
func (s *Sample) templateStep(ctx context.Context) (map[string]any, error) {
	res, err := runCLI(ctx, s.Tools.Env, s.Tools.Node, s.Repo, "--test", "reference/business-spec-templates/template-reference.test.mjs")
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, m := range passRe.FindAllStringSubmatch(string(res.Stdout), -1) {
		counts[m[1]], _ = strconv.Atoi(m[2])
	}
	script := `import {readFileSync} from "node:fs";
import {proposeAdoption} from "./reference/business-spec-templates/template-reference.mjs";
const t=JSON.parse(readFileSync("reference/business-spec-templates/order-entry.template.json","utf8"));
const p=proposeAdoption(t,{projectId:"order-rule-design",decisions:{}});
console.log(JSON.stringify({accepted:p.accepted,canonical:p.canonical,keys:Object.keys(p),unresolved:(p.unresolvedQuestions||p.unresolved||p.questions||[]).length}));`
	raw, err := s.Tools.mustRun(ctx, s.Tools.Node, s.Repo, "--input-type=module", "-e", script)
	if err != nil {
		return nil, err
	}
	var p map[string]any
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return map[string]any{"testsPass": counts["pass"], "testsFail": counts["fail"], "testExit": res.Exit,
		"proposalAccepted": p["accepted"], "proposalCanonical": p["canonical"], "unresolvedQuestions": p["unresolved"]}, nil
}

// buildCandidate authors the candidate for a stage 1 result. With no table
// node there is no evidence to attach: the candidate is built without a
// provenance evidence and the state travels as no-target.
func (s *Sample) buildCandidate(ctx context.Context, s1 stage1Out, s2 stage2Out, dir string, questions int) (stage3Out, error) {
	out := stage3Out{Dir: dir}
	editPath := filepath.Join(dir, "edit.json")
	if err := writeFile(editPath, []byte(studioEdit)); err != nil {
		return out, err
	}
	cand, err := s.Tools.mustRun(ctx, s.Tools.Node, s.Repo, "reference/studio-authoring/cli.mjs", "propose", "-workspace", s2.R0, "-edit-file", editPath)
	if err != nil {
		return out, err
	}
	candPath := filepath.Join(dir, "candidate.json")
	if err := writeFile(candPath, cand); err != nil {
		return out, err
	}
	if s1.Evidence != nil {
		evPath := filepath.Join(dir, "evidence.json")
		if err := writeFile(evPath, s1.EvidenceBytes); err != nil {
			return out, err
		}
		cand2 := filepath.Join(dir, "candidate-with-evidence.json")
		if _, err := s.Tools.mustRun(ctx, s.Tools.SharedAcceptance, s.Repo, "attach-rmt-transcription", "-candidate", candPath, "-rmt-transcription", evPath, "-out", cand2); err != nil {
			return out, err
		}
		candPath = cand2
	}
	out.Candidate, err = os.ReadFile(candPath)
	if err != nil {
		return out, err
	}
	att := Attachment{Schema: "tei.reference.order-rule-design.attachment/v0", ProvenanceState: s1.Carried.ProvenanceState,
		Unresolved: *s1.Carried.Unresolved, SourceDigest: s1.Carried.SourceDigest, ContractDigest: s1.Carried.ContractDigest,
		EvidenceFingerprint: s1.Carried.EvidenceFingerprint, UnresolvedList: s2.Unresolved, TemplateQuestions: questions}
	attRaw, _ := json.MarshalIndent(att, "", "  ")
	attPath := filepath.Join(dir, "attachment.json")
	if err := writeFile(attPath, attRaw); err != nil {
		return out, err
	}
	out.Attachment = attRaw
	prev, err := s.Tools.mustRun(ctx, s.Tools.SharedAcceptance, s.Repo, "preview", "-workspace", s2.R0, "-candidate", candPath)
	if err != nil {
		return out, err
	}
	out.PreviewRaw = prev
	if err := json.Unmarshal(prev, &out.Preview); err != nil {
		return out, err
	}
	out.Fingerprint, _ = out.Preview["candidateFingerprint"].(string)
	// Read the values back from the files the stage wrote.
	var back Attachment
	rb, _ := os.ReadFile(attPath)
	if err := json.Unmarshal(rb, &back); err != nil {
		return out, err
	}
	var cj struct {
		BaseModelRevision string `json:"baseModelRevision"`
	}
	_ = json.Unmarshal(out.Candidate, &cj)
	c := s2.Carried
	c.ProvenanceState, c.Unresolved = back.ProvenanceState, &back.Unresolved
	c.SourceDigest, c.ContractDigest, c.EvidenceFingerprint = back.SourceDigest, back.ContractDigest, back.EvidenceFingerprint
	c.BaselineRevision = cj.BaseModelRevision
	c.CandidateDigest, c.CandidateFingerprint, c.AttachmentDigest = digestBytes(out.Candidate), out.Fingerprint, digestBytes(rb)
	out.Carried = c
	acceptable, _ := out.Preview["acceptable"].(bool)
	out.Detail = map[string]any{
		"candidateId": "unit-price-positive", "previewAcceptable": acceptable, "modelRevisionAfter": out.Preview["modelRevisionAfter"],
		"evidenceFingerprintOfCandidate": out.Preview["evidenceFingerprint"], "provenanceEvidenceAttached": s1.Evidence != nil,
		"verificationEvidence": map[bool]string{true: "transcription evidence attached", false: "not created: no table node, carried as no-target"}[s1.Evidence != nil],
		"attachment":           "provenance state and unresolved list sit beside the candidate; the meaning fingerprint excludes them",
		"templateQuestions":    questions,
	}
	return out, nil
}

func (s *Sample) stage3(ctx context.Context) error {
	tpl, err := s.templateStep(ctx)
	if err != nil {
		return err
	}
	if tpl["testsFail"] != 0 || tpl["testsPass"] == 0 {
		return fmt.Errorf("business-spec-templates tests: %v", tpl)
	}
	q, _ := tpl["unresolvedQuestions"].(float64)
	out, err := s.buildCandidate(ctx, s.s1, s.s2, filepath.Join(s.Work, "stage3"), int(q))
	if err != nil {
		return err
	}
	if ok, _ := out.Preview["acceptable"].(bool); !ok {
		return fmt.Errorf("the candidate is not acceptable on R0")
	}
	out.Detail["businessSpecTemplates"] = tpl
	s.s3 = out
	return s.Chain.Append(3, out.Carried, out.Detail)
}
