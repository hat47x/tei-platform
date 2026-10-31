package orderruledesign

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

const chainSchema = "tei.reference.order-rule-design.chain/v0"

var stageNames = map[int]string{
	1: "legacy-source-reading-and-provenance-state",
	2: "baseline-workspace-matching-and-evaluation-targets",
	3: "change-candidate-authoring",
	4: "candidate-handover-and-rebuild",
	5: "impact-range-and-evidence-reuse",
	6: "review-preflight-and-signed-decision",
}

// Provenance states of the table that corresponds to the order line in the
// legacy material. The state says how far the candidate's origin is bound.
const (
	StateBound     = "bound"      // 結び付いた: the table node maps and a function and task path reaches it
	StateTableOnly = "table-only" // 表だけ: the table node maps, no path reaches it
	StateNoTarget  = "no-target"  // 対象なし: the material has no such table node
)

// Unresolved is the count of legacy descriptions the contract cannot handle.
// It is carried unchanged to the signed reason and never folded into
// accepted meaning.
type Unresolved struct {
	Total             int `json:"total"`
	SourceObservation int `json:"sourceObservation"`
	Mapping           int `json:"mapping"`
	SemanticCandidate int `json:"semanticCandidate"`
}

// Carried is what one stage hands to the next. It holds identities and
// digests only. Later stages record the values they read back from their own
// artifacts, so a stage that drops or changes a value is visible.
type Carried struct {
	SourceDigest          string      `json:"sourceDigest,omitempty"`
	ContractDigest        string      `json:"contractDigest,omitempty"`
	ReceiptFingerprint    string      `json:"receiptFingerprint,omitempty"`
	AccountingFingerprint string      `json:"accountingFingerprint,omitempty"`
	ProvenanceState       string      `json:"provenanceState,omitempty"`
	Unresolved            *Unresolved `json:"unresolved,omitempty"`
	EvidenceFingerprint   string      `json:"evidenceFingerprint,omitempty"`

	BaselineRevision string `json:"baselineRevision,omitempty"`
	TargetsDigest    string `json:"targetsDigest,omitempty"`

	CandidateDigest      string `json:"candidateDigest,omitempty"`
	CandidateFingerprint string `json:"candidateFingerprint,omitempty"`
	AttachmentDigest     string `json:"attachmentDigest,omitempty"`

	MainCommit      string `json:"mainCommit,omitempty"`
	TaskHead        string `json:"taskHead,omitempty"`
	TaskTree        string `json:"taskTree,omitempty"`
	IntegrationHead string `json:"integrationHead,omitempty"`
	IntegrationTree string `json:"integrationTree,omitempty"`

	AssuranceDigest   string `json:"assuranceDigest,omitempty"`
	TrialRevision     string `json:"trialRevision,omitempty"`
	ImpactRangeDigest string `json:"impactRangeDigest,omitempty"`

	ReviewFingerprint string `json:"reviewFingerprint,omitempty"`
	PacketDigest      string `json:"packetDigest,omitempty"`
	ReasonDigest      string `json:"reasonDigest,omitempty"`
	MessageDigest     string `json:"messageDigest,omitempty"`
	SignatureDigest   string `json:"signatureDigest,omitempty"`
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

// Chain is the run record of stages 1 to 6.
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

func mustDigest(v any) string {
	d, err := digestJSON(v)
	if err != nil {
		return "sha256:unmarshalable"
	}
	return d
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
}

type carriedField struct {
	name  string
	intro int
	get   func(Carried) string
}

func unresolvedString(c Carried) string {
	if c.Unresolved == nil {
		return ""
	}
	return fmt.Sprintf("total=%d,sourceObservation=%d,mapping=%d,semanticCandidate=%d",
		c.Unresolved.Total, c.Unresolved.SourceObservation, c.Unresolved.Mapping, c.Unresolved.SemanticCandidate)
}

// carriedFields lists every identity that must be one value from the stage
// that introduces it to stage 6.
var carriedFields = []carriedField{
	{"sourceDigest", 1, func(c Carried) string { return c.SourceDigest }},
	{"contractDigest", 1, func(c Carried) string { return c.ContractDigest }},
	{"receiptFingerprint", 1, func(c Carried) string { return c.ReceiptFingerprint }},
	{"accountingFingerprint", 1, func(c Carried) string { return c.AccountingFingerprint }},
	{"provenanceState", 1, func(c Carried) string { return c.ProvenanceState }},
	{"unresolved", 1, unresolvedString},
	{"evidenceFingerprint", 1, func(c Carried) string { return c.EvidenceFingerprint }},
	{"baselineRevision", 2, func(c Carried) string { return c.BaselineRevision }},
	{"targetsDigest", 2, func(c Carried) string { return c.TargetsDigest }},
	{"candidateDigest", 3, func(c Carried) string { return c.CandidateDigest }},
	{"candidateFingerprint", 3, func(c Carried) string { return c.CandidateFingerprint }},
	{"attachmentDigest", 3, func(c Carried) string { return c.AttachmentDigest }},
	{"mainCommit", 4, func(c Carried) string { return c.MainCommit }},
	{"taskHead", 4, func(c Carried) string { return c.TaskHead }},
	{"taskTree", 4, func(c Carried) string { return c.TaskTree }},
	{"integrationHead", 4, func(c Carried) string { return c.IntegrationHead }},
	{"integrationTree", 4, func(c Carried) string { return c.IntegrationTree }},
	{"assuranceDigest", 5, func(c Carried) string { return c.AssuranceDigest }},
	{"trialRevision", 5, func(c Carried) string { return c.TrialRevision }},
	{"impactRangeDigest", 5, func(c Carried) string { return c.ImpactRangeDigest }},
	{"reviewFingerprint", 6, func(c Carried) string { return c.ReviewFingerprint }},
	{"packetDigest", 6, func(c Carried) string { return c.PacketDigest }},
	{"reasonDigest", 6, func(c Carried) string { return c.ReasonDigest }},
	{"messageDigest", 6, func(c Carried) string { return c.MessageDigest }},
	{"signatureDigest", 6, func(c Carried) string { return c.SignatureDigest }},
}

// VerifyThrough is read-only. It completes only when every stage is present
// once and in order, the links hold, and every identity is one value from
// the stage that introduces it to stage 6.
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
	for stage := 1; stage <= 6; stage++ {
		if _, ok := byStage[stage]; !ok {
			res.MissingStages = append(res.MissingStages, stage)
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
		for s := f.intro + 1; s <= 6; s++ {
			r, ok := byStage[s]
			if !ok {
				continue
			}
			if got := f.get(r.Carried); got != want {
				mismatch("stage %d %s %q differs from stage %d %q", s, f.name, got, f.intro, want)
			}
		}
	}
	// The integration head equals the task head: the candidate was handed
	// over by an ordinary fast-forward, not rewritten.
	if r, ok := byStage[4]; ok && r.Carried.TaskHead != r.Carried.IntegrationHead {
		mismatch("stage 4 integration head differs from the task head")
	}
	// The trial acceptance revision is never the baseline.
	if r, ok := byStage[5]; ok && r.Carried.TrialRevision == r.Carried.BaselineRevision {
		mismatch("stage 5 trial revision equals the baseline: nothing was applied")
	}
	res.Complete = len(res.MissingStages) == 0 && len(res.Mismatches) == 0
	return res
}

// standaloneOK is what one stage can check alone: its own link digest holds.
// A forger who relinks passes it for every stage.
func standaloneOK(c Chain) bool {
	for _, r := range c.Records {
		if d, err := recordDigest(r); err != nil || d != r.Digest {
			return false
		}
	}
	return true
}
