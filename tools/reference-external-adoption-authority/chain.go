package externaladoptionauthority

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

const chainSchema = "tei.reference.external-adoption-authority.chain/v0"

var stageNames = map[int]string{
	1: "external-contract-mapping-shadow-probe",
	2: "extension-namespace-owner-semantic-lock",
	3: "semantic-distribution-signature-preflight",
	4: "project-acceptance-and-current-rejudgment",
}

// PinnedSource is the SUI version that stage 1 shadowed: one revision and
// the four dependency blobs. Freshness is bound to the blobs, not to the
// revision (reference/sui-adoption/README.md).
type PinnedSource struct {
	Repository string            `json:"repository"`
	Revision   string            `json:"revision"`
	Blobs      map[string]string `json:"blobs"`
}

func (p PinnedSource) equal(o PinnedSource) bool {
	if p.Repository != o.Repository || p.Revision != o.Revision || len(p.Blobs) != len(o.Blobs) {
		return false
	}
	for k, v := range p.Blobs {
		if o.Blobs[k] != v {
			return false
		}
	}
	return true
}

// Carried is what one stage hands to the next. It holds identities and
// digests only. Point-in-time trust qualifications are deliberately not
// carried: every stage judges trust again at its own logical point.
type Carried struct {
	Source                    PinnedSource `json:"source"`
	InputDigest               string       `json:"inputDigest"`
	MappingReportDigest       string       `json:"mappingReportDigest"`
	Namespace                 string       `json:"namespace"`
	Subject                   string       `json:"subject"`
	Lossless                  *bool        `json:"lossless"`
	ParentPresence            string       `json:"parentPresenceSemantics"`
	LossRecordDigest          string       `json:"lossRecordDigest"`
	SemanticDigest            string       `json:"semanticDigest,omitempty"`
	SemanticSetFingerprint    string       `json:"semanticSetFingerprint,omitempty"`
	SemanticLockDigest        string       `json:"semanticLockDigest,omitempty"`
	ValidationPlanFingerprint string       `json:"validationPlanFingerprint,omitempty"`
	ExtensionNamespace        string       `json:"extensionNamespace,omitempty"`
	NamespaceOwner            string       `json:"namespaceOwner,omitempty"`
	OwnerIssuer               string       `json:"ownerIssuer,omitempty"`
	ArtifactDigest            string       `json:"artifactDigest,omitempty"`
	PlanFingerprint           string       `json:"planFingerprint,omitempty"`
	Package                   string       `json:"package,omitempty"`
	SignerRef                 string       `json:"signerRef,omitempty"`
	SignerKeyFingerprint      string       `json:"signerKeyFingerprint,omitempty"`
}

// StageRecord is one link of the run record.
type StageRecord struct {
	Stage   int             `json:"stage"`
	Name    string          `json:"name"`
	Carried Carried         `json:"carried"`
	Detail  json.RawMessage `json:"detail,omitempty"`
	Prev    string          `json:"prev"`
	Digest  string          `json:"digest"`
}

// Chain is the run record of stages 1 to 4.
type Chain struct {
	Schema  string        `json:"schema"`
	Records []StageRecord `json:"records"`
}

func NewChain() *Chain { return &Chain{Schema: chainSchema} }

func digestJSON(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return digestBytes(raw), nil
}

func digestBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func recordDigest(r StageRecord) (string, error) {
	r.Digest = ""
	return digestJSON(r)
}

// Append links a stage record to the previous one.
func (c *Chain) Append(stage int, carried Carried, detail any) error {
	raw, err := json.Marshal(detail)
	if err != nil {
		return err
	}
	r := StageRecord{Stage: stage, Name: stageNames[stage], Carried: carried, Detail: raw}
	if n := len(c.Records); n > 0 {
		r.Prev = c.Records[n-1].Digest
	}
	if r.Digest, err = recordDigest(r); err != nil {
		return err
	}
	c.Records = append(c.Records, r)
	return nil
}

// Relink recomputes the links and digests, as a forger who rewrites one
// record would. It exists for the negative controls only.
func (c *Chain) Relink() {
	prev := ""
	for i := range c.Records {
		c.Records[i].Prev = prev
		c.Records[i].Digest, _ = recordDigest(c.Records[i])
		prev = c.Records[i].Digest
	}
}

// Clone returns a deep copy through JSON.
func (c Chain) Clone() Chain {
	raw, _ := json.Marshal(c)
	var out Chain
	_ = json.Unmarshal(raw, &out)
	return out
}

// ThroughResult says whether the run is complete as one flow.
type ThroughResult struct {
	Complete      bool     `json:"complete"`
	MissingStages []int    `json:"missingStages,omitempty"`
	Mismatches    []string `json:"mismatches,omitempty"`
	Head          string   `json:"head,omitempty"`
	Lossless      *bool    `json:"lossless"`
}

type carriedField struct {
	name  string
	intro int
	get   func(Carried) string
}

var carriedFields = []carriedField{
	{"inputDigest", 1, func(c Carried) string { return c.InputDigest }},
	{"mappingReportDigest", 1, func(c Carried) string { return c.MappingReportDigest }},
	{"namespace", 1, func(c Carried) string { return c.Namespace }},
	{"subject", 1, func(c Carried) string { return c.Subject }},
	{"parentPresenceSemantics", 1, func(c Carried) string { return c.ParentPresence }},
	{"lossRecordDigest", 1, func(c Carried) string { return c.LossRecordDigest }},
	{"semanticDigest", 2, func(c Carried) string { return c.SemanticDigest }},
	{"semanticSetFingerprint", 2, func(c Carried) string { return c.SemanticSetFingerprint }},
	{"semanticLockDigest", 2, func(c Carried) string { return c.SemanticLockDigest }},
	{"validationPlanFingerprint", 2, func(c Carried) string { return c.ValidationPlanFingerprint }},
	{"extensionNamespace", 2, func(c Carried) string { return c.ExtensionNamespace }},
	{"namespaceOwner", 2, func(c Carried) string { return c.NamespaceOwner }},
	{"ownerIssuer", 2, func(c Carried) string { return c.OwnerIssuer }},
	{"artifactDigest", 3, func(c Carried) string { return c.ArtifactDigest }},
	{"planFingerprint", 3, func(c Carried) string { return c.PlanFingerprint }},
	{"package", 3, func(c Carried) string { return c.Package }},
	{"signerRef", 3, func(c Carried) string { return c.SignerRef }},
	{"signerKeyFingerprint", 3, func(c Carried) string { return c.SignerKeyFingerprint }},
}

// VerifyThrough is read-only. It completes only when every stage is present
// once and in order, the links hold, the pinned SUI version and every
// carried identity and digest is one value from the stage that introduces
// it to stage 4, and lossless is false (never absent, never true) at every
// stage.
func VerifyThrough(c Chain) ThroughResult {
	res := ThroughResult{}
	mismatch := func(format string, args ...any) {
		res.Mismatches = append(res.Mismatches, fmt.Sprintf(format, args...))
	}
	if c.Schema != chainSchema {
		mismatch("schema %q", c.Schema)
	}
	byStage := map[int]StageRecord{}
	prev := ""
	for i, r := range c.Records {
		if r.Prev != prev {
			mismatch("record %d (stage %d) is not linked to the previous record", i, r.Stage)
		}
		if d, err := recordDigest(r); err != nil || d != r.Digest {
			mismatch("record %d (stage %d) does not match its digest", i, r.Stage)
		}
		prev = r.Digest
		if _, dup := byStage[r.Stage]; dup {
			mismatch("stage %d appears twice", r.Stage)
		}
		if r.Stage != i+1 {
			mismatch("record %d is stage %d, out of order", i, r.Stage)
		}
		byStage[r.Stage] = r
	}
	res.Head = prev
	for stage := 1; stage <= 4; stage++ {
		if _, ok := byStage[stage]; !ok {
			res.MissingStages = append(res.MissingStages, stage)
		}
	}
	s1, ok := byStage[1]
	if ok {
		if s1.Carried.Source.Repository == "" || s1.Carried.Source.Revision == "" || len(s1.Carried.Source.Blobs) != 4 {
			mismatch("stage 1 does not pin one revision and four dependency blobs")
		}
		for s := 2; s <= 4; s++ {
			if r, ok := byStage[s]; ok && !r.Carried.Source.equal(s1.Carried.Source) {
				mismatch("stage %d pinned SUI source differs from stage 1", s)
			}
		}
	}
	for _, f := range carriedFields {
		intro, ok := byStage[f.intro]
		if !ok {
			continue
		}
		want := f.get(intro.Carried)
		if want == "" {
			mismatch("stage %d does not record %s", f.intro, f.name)
			continue
		}
		for s := f.intro + 1; s <= 4; s++ {
			if r, ok := byStage[s]; ok && f.get(r.Carried) != want {
				mismatch("stage %d %s %q differs from stage %d %q", s, f.name, f.get(r.Carried), f.intro, want)
			}
		}
	}
	for s := 1; s <= 4; s++ {
		r, ok := byStage[s]
		if !ok {
			continue
		}
		switch {
		case r.Carried.Lossless == nil:
			mismatch("stage %d does not carry lossless", s)
		case *r.Carried.Lossless:
			mismatch("stage %d reads lossless as true", s)
		}
		if r.Carried.ParentPresence != "unresolved" {
			mismatch("stage %d parent presence semantics is %q, not unresolved", s, r.Carried.ParentPresence)
		}
	}
	if len(res.Mismatches) == 0 {
		f := false
		res.Lossless = &f
	}
	res.Complete = len(res.MissingStages) == 0 && len(res.Mismatches) == 0
	return res
}
