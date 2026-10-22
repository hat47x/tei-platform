package orderassistoffline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// Identity is what every record of one C5 run must carry unchanged: the
// realizations each site ran and the policy the case was pinned to.
type Identity struct {
	QuoteID              string `json:"quoteId"`
	RuleVersion          string `json:"ruleVersion"`
	TerminalAGeneration2 string `json:"terminalARealizationDigest"`
	TerminalBGeneration1 string `json:"terminalBRealizationDigest"`
	ServerRealization    string `json:"serverRealizationDigest"`
	BatchRealization     string `json:"batchRealizationDigest"`
	WASIRunner           string `json:"wasiRunnerDigest"`
	PolicyDigest         string `json:"policyDigest"`
}

func (id Identity) complete() bool {
	return id.QuoteID != "" && id.RuleVersion != "" && id.TerminalAGeneration2 != "" && id.TerminalBGeneration1 != "" &&
		id.ServerRealization != "" && id.BatchRealization != "" && id.WASIRunner != "" && id.PolicyDigest != ""
}

// Record is one link of the run record. QuoteDigest is the quote identity
// the step acted on ("" for steps before the quote exists on the server).
type Record struct {
	Seq         int            `json:"seq"`
	Step        string         `json:"step"`
	Identity    Identity       `json:"identity"`
	QuoteDigest string         `json:"quoteDigest"`
	Result      map[string]any `json:"result"`
	Passed      bool           `json:"passed"`
	Prev        string         `json:"prev"`
	Digest      string         `json:"digest"`
}

// Chain is an append-only run record.
type Chain struct {
	Records []Record `json:"records"`
}

func recordDigest(r Record) string {
	r.Digest = ""
	raw, _ := json.Marshal(r)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Append adds a record linked to the previous one.
func (c *Chain) Append(step string, id Identity, quoteDigest string, passed bool, result map[string]any) Record {
	prev := ""
	if n := len(c.Records); n > 0 {
		prev = c.Records[n-1].Digest
	}
	raw, _ := json.Marshal(result)
	var norm map[string]any
	_ = json.Unmarshal(raw, &norm)
	r := Record{Seq: len(c.Records) + 1, Step: step, Identity: id, QuoteDigest: quoteDigest, Result: norm, Passed: passed, Prev: prev}
	r.Digest = recordDigest(r)
	c.Records = append(c.Records, r)
	return r
}

// ErrChain is a broken run record.
var ErrChain = errors.New("run record chain broken")

// Verify checks links, digests, one complete identity, that each required
// step is present and passed, and that every step from carryFrom onwards
// acted on one and the same quote digest.
func (c Chain) Verify(required []string, carryFrom string) error {
	if len(c.Records) == 0 {
		return fmt.Errorf("%w: empty", ErrChain)
	}
	first := c.Records[0].Identity
	if !first.complete() {
		return fmt.Errorf("%w: identity incomplete: %+v", ErrChain, first)
	}
	have := map[string]bool{}
	prev := ""
	carried := ""
	carrying := false
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
		if r.Step == carryFrom {
			carrying, carried = true, r.QuoteDigest
			if carried == "" {
				return fmt.Errorf("%w: %s carries no quote digest", ErrChain, r.Step)
			}
		} else if carrying && r.QuoteDigest != carried {
			return fmt.Errorf("%w: record %d (%s) acted on %s, not the carried %s", ErrChain, i+1, r.Step, r.QuoteDigest, carried)
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
	if !carrying {
		return fmt.Errorf("%w: carry step %s missing", ErrChain, carryFrom)
	}
	return nil
}
