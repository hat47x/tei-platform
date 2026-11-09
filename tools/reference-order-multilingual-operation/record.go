package ordermultilingualoperation

import (
	"encoding/json"
	"fmt"
)

// Check is one observed pass/fail item (fault injection, completion
// condition, identity check).
type Check struct {
	ID          string `json:"id"`
	Requirement string `json:"requirement,omitempty"`
	Stage       string `json:"stage,omitempty"`
	Passed      bool   `json:"passed"`
	Observed    any    `json:"observed"`
}

// AssetCall records that an existing asset was called unchanged.
type AssetCall struct {
	Asset string `json:"asset"`
	Entry string `json:"entry"`
}

// StageRecord is one stage of the through record. Prev is the digest of the
// previous stage record, so a missing or swapped stage breaks the chain.
type StageRecord struct {
	Stage        int            `json:"stage"`
	Name         string         `json:"name"`
	Conditional  bool           `json:"conditional,omitempty"`
	Environment  string         `json:"environment"`
	Application  string         `json:"application"`
	Owner        string         `json:"owner"`
	SourceLocale string         `json:"sourceLocale"`
	LockFP       string         `json:"lockFingerprint,omitempty"`
	Release      string         `json:"release,omitempty"`
	ReleaseHash  string         `json:"releaseDigest,omitempty"`
	Assets       []AssetCall    `json:"assets"`
	Outputs      map[string]any `json:"outputs"`
	Prev         string         `json:"prev"`
	Digest       string         `json:"digest"`
}

// Gap is a platform gap observed while running the sample.
type Gap struct {
	Requirement string `json:"requirement"`
	Area        string `json:"area"`
	Gap         string `json:"gap"`
	Handling    string `json:"handling"`
}

// Record is the through record written to chain-record.json.
type Record struct {
	Schema      string          `json:"schema"`
	Unified     Unified         `json:"unified"`
	Identities  []AssetIdentity `json:"identities"`
	Stages      []StageRecord   `json:"stages"`
	Faults      []Check         `json:"faultInjections"`
	Completion  []Check         `json:"completionConditions"`
	Conditional []Check         `json:"conditionalConditions"`
	Gaps        []Gap           `json:"platformGaps"`
	Limits      []string        `json:"limits"`
	Through     ThroughResult   `json:"through"`
	Passed      bool            `json:"passed"`
}

// ThroughResult is the read-only verification of the chain.
type ThroughResult struct {
	Complete      bool     `json:"complete"`
	MissingStages []int    `json:"missingStages,omitempty"`
	Problems      []string `json:"problems,omitempty"`
}

func stageDigest(s StageRecord) string {
	s.Digest = ""
	return digestJSON(s)
}

// AddStage appends a stage record linked to the previous one.
func (r *Record) AddStage(s StageRecord) {
	if n := len(r.Stages); n > 0 {
		s.Prev = r.Stages[n-1].Digest
	}
	s.Environment = string(r.Unified.Environment)
	s.Application = string(r.Unified.Application)
	s.Owner = string(r.Unified.Owner.ID) + "@" + r.Unified.Owner.Version
	s.SourceLocale = r.Unified.SourceLocale
	// Round-trip the outputs through JSON so the digest covers exactly what
	// the record file holds.
	raw, err := json.Marshal(s.Outputs)
	if err != nil {
		panic(err)
	}
	var outputs map[string]any
	_ = json.Unmarshal(raw, &outputs)
	s.Outputs = outputs
	s.Digest = stageDigest(s)
	r.Stages = append(r.Stages, s)
}

func (r *Record) fault(id, req, stage string, passed bool, observed any) {
	r.Faults = append(r.Faults, Check{ID: id, Requirement: req, Stage: stage, Passed: passed, Observed: observed})
}

// VerifyThrough reads the record only. Stages 1-4 are mandatory, 5 and 6
// are conditional and must be present for this run as well. Every stage must
// carry the unified identity, link to its predecessor and keep its digest;
// stages 2, 3 and the baseline of 4 must share one lock fingerprint; stage 4
// must start from stage 3's bundle digest.
func VerifyThrough(r Record) ThroughResult {
	var res ThroughResult
	seen := map[int]StageRecord{}
	prev := ""
	for i, s := range r.Stages {
		if stageDigest(s) != s.Digest {
			res.Problems = append(res.Problems, fmt.Sprintf("stage %d digest does not match its content", s.Stage))
		}
		if s.Prev != prev {
			res.Problems = append(res.Problems, fmt.Sprintf("stage %d does not link to the previous stage record", s.Stage))
		}
		prev = s.Digest
		if i > 0 && s.Stage != r.Stages[i-1].Stage+1 {
			res.Problems = append(res.Problems, fmt.Sprintf("stage %d follows stage %d", s.Stage, r.Stages[i-1].Stage))
		}
		owner := string(r.Unified.Owner.ID) + "@" + r.Unified.Owner.Version
		if s.Environment != string(r.Unified.Environment) || s.Application != string(r.Unified.Application) ||
			s.Owner != owner || s.SourceLocale != r.Unified.SourceLocale {
			res.Problems = append(res.Problems, fmt.Sprintf("stage %d identity differs from the unified identity", s.Stage))
		}
		seen[s.Stage] = s
	}
	for st := 1; st <= 6; st++ {
		if _, ok := seen[st]; !ok {
			res.MissingStages = append(res.MissingStages, st)
		}
	}
	get := func(st int, key string) string {
		v, _ := seen[st].Outputs[key].(string)
		return v
	}
	if s2, ok := seen[2]; ok {
		if s3, ok := seen[3]; ok && s2.LockFP != s3.LockFP {
			res.Problems = append(res.Problems, "stage 3 lock fingerprint differs from stage 2")
		}
		if s4, ok := seen[4]; ok && s4.LockFP != s2.LockFP {
			res.Problems = append(res.Problems, "stage 4 lock fingerprint differs from stage 2")
		}
	}
	if _, ok := seen[4]; ok {
		if get(4, "bundleBefore") == "" || get(4, "bundleBefore") != get(3, "bundleDigest") {
			res.Problems = append(res.Problems, "stage 4 does not start from the stage 3 bundle digest")
		}
		if get(4, "bundleAfter") == "" || get(4, "bundleAfter") == get(4, "bundleBefore") {
			res.Problems = append(res.Problems, "stage 4 regenerated bundle digest is missing or unchanged")
		}
		if seen[4].Release == seen[3].Release {
			res.Problems = append(res.Problems, "stage 4 did not revise the source release")
		}
	}
	if _, ok := seen[6]; ok && get(6, "bundleDigest") != get(4, "bundleAfter") {
		res.Problems = append(res.Problems, "stage 6 does not use the stage 4 regenerated bundle")
	}
	res.Complete = len(res.MissingStages) == 0 && len(res.Problems) == 0
	return res
}
