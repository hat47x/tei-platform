package pluginbuildsupply

import (
	"encoding/json"
	"fmt"
	"sort"
)

const chainSchema = "tei.reference.plugin-build-supply.chain/v0"

// Artifact roles each stage reports. A stage's own check only compares the
// roles inside the stage; the through check compares them across stages.
var stageRoles = map[int][]string{
	1: nil,
	2: {"first-success", "second-success", "stored"},
	3: {"first-build", "second-build", "registered"},
	4: {"retrieved", "invoked"},
}

var stageEngines = map[int][]string{
	3: {"transform", "static-response"},
	4: {"static-response"},
}

var stageNames = map[int]string{
	1: "build-intent-and-builder-resolution",
	2: "build-attempts-and-artifact-store",
	3: "self-build-reproducibility-convergence-registration",
	4: "process-plugin-call-and-identity-check",
}

// StageRecord is one link of the run record.
type StageRecord struct {
	Stage              int               `json:"stage"`
	Name               string            `json:"name"`
	PlanFingerprint    string            `json:"planFingerprint"`
	ArtifactDigests    map[string]string `json:"artifactDigests,omitempty"`
	EngineFingerprints map[string]string `json:"engineFingerprints,omitempty"`
	Detail             json.RawMessage   `json:"detail,omitempty"`
	Prev               string            `json:"prev"`
	Digest             string            `json:"digest"`
}

// Chain is the run record of stages 1 to 4.
type Chain struct {
	Schema  string        `json:"schema"`
	Records []StageRecord `json:"records"`
}

func NewChain() *Chain { return &Chain{Schema: chainSchema} }

func recordDigest(r StageRecord) (string, error) {
	r.Digest = ""
	return fingerprintJSON(r)
}

// Append links a stage record to the previous one.
func (c *Chain) Append(stage int, plan string, artifacts, engines map[string]string, detail any) error {
	raw, err := json.Marshal(detail)
	if err != nil {
		return err
	}
	r := StageRecord{Stage: stage, Name: stageNames[stage], PlanFingerprint: plan, ArtifactDigests: artifacts, EngineFingerprints: engines, Detail: raw}
	if n := len(c.Records); n > 0 {
		r.Prev = c.Records[n-1].Digest
	}
	if r.Digest, err = recordDigest(r); err != nil {
		return err
	}
	c.Records = append(c.Records, r)
	return nil
}

// SelfCheck is the stage's own check: it passes whenever the stage is
// internally consistent, whatever the other stages say.
func (r StageRecord) SelfCheck() error {
	if r.PlanFingerprint == "" {
		return fmt.Errorf("stage %d has no plan fingerprint", r.Stage)
	}
	roles, ok := stageRoles[r.Stage]
	if !ok {
		return fmt.Errorf("unknown stage %d", r.Stage)
	}
	var first string
	for _, role := range roles {
		d := r.ArtifactDigests[role]
		if d == "" {
			return fmt.Errorf("stage %d has no %s digest", r.Stage, role)
		}
		if first == "" {
			first = d
		} else if d != first {
			return fmt.Errorf("stage %d: %s differs inside the stage", r.Stage, role)
		}
	}
	for _, engine := range stageEngines[r.Stage] {
		if r.EngineFingerprints[engine] == "" {
			return fmt.Errorf("stage %d has no %s engine identity", r.Stage, engine)
		}
	}
	return nil
}

// ThroughResult says whether the run is complete as one flow.
type ThroughResult struct {
	Complete         bool     `json:"complete"`
	MissingStages    []int    `json:"missingStages,omitempty"`
	Mismatches       []string `json:"mismatches,omitempty"`
	SelfChecksPassed []int    `json:"selfChecksPassed"`
	PlanFingerprint  string   `json:"planFingerprint,omitempty"`
	ArtifactDigest   string   `json:"artifactDigest,omitempty"`
	EngineIdentity   string   `json:"staticResponseEngineFingerprint,omitempty"`
}

// VerifyThrough is read-only. It completes only when every stage is present
// once, the links hold, and the plan fingerprint, the artifact digest and the
// engine identity are one value across all stages that carry them.
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
		byStage[r.Stage] = r
		if err := r.SelfCheck(); err != nil {
			mismatch("%v", err)
		} else {
			res.SelfChecksPassed = append(res.SelfChecksPassed, r.Stage)
		}
	}
	for stage := 1; stage <= 4; stage++ {
		if _, ok := byStage[stage]; !ok {
			res.MissingStages = append(res.MissingStages, stage)
		}
	}
	stages := make([]int, 0, len(byStage))
	for s := range byStage {
		stages = append(stages, s)
	}
	sort.Ints(stages)
	if s1, ok := byStage[1]; ok {
		res.PlanFingerprint = s1.PlanFingerprint
		for _, s := range stages {
			if byStage[s].PlanFingerprint != s1.PlanFingerprint {
				mismatch("stage %d plan fingerprint %s differs from stage 1 %s", s, byStage[s].PlanFingerprint, s1.PlanFingerprint)
			}
		}
	}
	for _, s := range stages {
		r := byStage[s]
		for _, role := range stageRoles[s] {
			d := r.ArtifactDigests[role]
			if res.ArtifactDigest == "" {
				res.ArtifactDigest = d
			} else if d != res.ArtifactDigest {
				mismatch("stage %d %s digest %s differs from %s", s, role, d, res.ArtifactDigest)
			}
		}
		if e := r.EngineFingerprints["static-response"]; e != "" {
			if res.EngineIdentity == "" {
				res.EngineIdentity = e
			} else if e != res.EngineIdentity {
				mismatch("stage %d engine identity %s differs from %s", s, e, res.EngineIdentity)
			}
		}
	}
	res.Complete = len(res.MissingStages) == 0 && len(res.Mismatches) == 0
	return res
}
