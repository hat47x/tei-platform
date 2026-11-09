package orderwebmember

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// Identity is what every record of one step-6 run must carry unchanged: the
// read plan the projections came from, both projections, and the two units
// that served the run.
type Identity struct {
	ReadPlan             ReadPlanSource `json:"readPlan"`
	MemberProjection     string         `json:"memberProjectionDigest"`
	StaffProjection      string         `json:"staffProjectionDigest"`
	MemberUnitConfig     string         `json:"memberUnitConfigDigest"`
	StaffUnitConfig      string         `json:"staffUnitConfigDigest"`
	MemberUnitProjection string         `json:"memberUnitReportedProjectionDigest"`
	StaffUnitProjection  string         `json:"staffUnitReportedProjectionDigest"`
}

func (id Identity) complete() bool {
	return id.ReadPlan.QueryID != "" && id.ReadPlan.DataID != "" && id.MemberProjection != "" && id.StaffProjection != "" &&
		id.MemberUnitConfig != "" && id.StaffUnitConfig != "" &&
		id.MemberUnitProjection == id.MemberProjection && id.StaffUnitProjection == id.StaffProjection
}

// Record is one link of the run record.
type Record struct {
	Seq      int            `json:"seq"`
	Step     string         `json:"step"`
	Identity Identity       `json:"identity"`
	Result   map[string]any `json:"result"`
	Passed   bool           `json:"passed"`
	Prev     string         `json:"prev"`
	Digest   string         `json:"digest"`
}

// Chain is an append-only run record. Each record carries the identity and
// the digest of the previous record.
type Chain struct {
	Records []Record `json:"records"`
}

func recordDigest(r Record) string {
	r.Digest = ""
	raw, _ := json.Marshal(r)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func (c *Chain) Append(step string, id Identity, passed bool, result map[string]any) Record {
	prev := ""
	if n := len(c.Records); n > 0 {
		prev = c.Records[n-1].Digest
	}
	// Round-trip the result so the stored record is what its digest covers.
	raw, _ := json.Marshal(result)
	var norm map[string]any
	_ = json.Unmarshal(raw, &norm)
	r := Record{Seq: len(c.Records) + 1, Step: step, Identity: id, Result: norm, Passed: passed, Prev: prev}
	r.Digest = recordDigest(r)
	c.Records = append(c.Records, r)
	return r
}

var ErrChain = errors.New("run record chain broken")

// Verify checks the links, the digests, that every record carries the same
// complete identity, that every required step is present and passed.
func (c Chain) Verify(required []string) error {
	if len(c.Records) == 0 {
		return fmt.Errorf("%w: empty", ErrChain)
	}
	first := c.Records[0].Identity
	if !first.complete() {
		return fmt.Errorf("%w: identity incomplete or unit-reported projection differs: %+v", ErrChain, first)
	}
	have := map[string]bool{}
	prev := ""
	for i, r := range c.Records {
		if r.Seq != i+1 || r.Prev != prev || recordDigest(r) != r.Digest {
			return fmt.Errorf("%w: record %d (%s) link or digest", ErrChain, i+1, r.Step)
		}
		if r.Identity != first {
			return fmt.Errorf("%w: record %d (%s) identity differs", ErrChain, i+1, r.Step)
		}
		if !r.Passed {
			return fmt.Errorf("%w: record %d (%s) did not pass", ErrChain, i+1, r.Step)
		}
		have[r.Step] = true
		prev = r.Digest
	}
	var missing []string
	for _, s := range required {
		if !have[s] {
			missing = append(missing, s)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return fmt.Errorf("%w: missing steps %v", ErrChain, missing)
	}
	return nil
}

// Head is the digest of the last record.
func (c Chain) Head() string {
	if len(c.Records) == 0 {
		return ""
	}
	return c.Records[len(c.Records)-1].Digest
}
