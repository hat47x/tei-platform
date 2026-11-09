package orderrollouthost

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"tei/internal/deployment/durable"
)

// Record is one link of the run record of one deployment run.
type Record struct {
	Seq      int            `json:"seq"`
	Step     string         `json:"step"`
	Identity Identity       `json:"identity"`
	Result   map[string]any `json:"result"`
	Passed   bool           `json:"passed"`
	Prev     string         `json:"prev"`
	Digest   string         `json:"digest"`
}

// Chain is an append-only run record. Each record carries the identity the
// step received (read from that step's own input, not copied from step 1)
// and the digest of the previous record.
type Chain struct {
	Records []Record `json:"records"`
}

func recordDigest(r Record) string {
	r.Digest = ""
	raw, _ := json.Marshal(r)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Append adds a record.
func (c *Chain) Append(step string, id Identity, passed bool, result map[string]any) Record {
	prev := ""
	if n := len(c.Records); n > 0 {
		prev = c.Records[n-1].Digest
	}
	raw, _ := json.Marshal(result)
	var norm map[string]any
	_ = json.Unmarshal(raw, &norm)
	r := Record{Seq: len(c.Records) + 1, Step: step, Identity: id, Result: norm, Passed: passed, Prev: prev}
	r.Digest = recordDigest(r)
	c.Records = append(c.Records, r)
	return r
}

// ErrChain is returned when the run record does not verify.
var ErrChain = errors.New("run record chain broken")

// Verify checks the links, the digests, that every record carries the same
// complete identity, and that every required step is present and passed.
func (c Chain) Verify(required []string) error {
	if len(c.Records) == 0 {
		return fmt.Errorf("%w: empty", ErrChain)
	}
	first := c.Records[0].Identity
	if !first.complete() {
		return fmt.Errorf("%w: identity incomplete: %+v", ErrChain, first)
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

// Handover is the chain of supervisors across steps 2 to 4.
type Handover struct {
	CallLease, TakeoverLease, HandleLease string
	CallOwner, TakeoverPrevOwner          string
	CallToken, TakeoverPrevToken          uint64
	TakeoverOwner, HandleOwner            string
	TakeoverToken, HandleToken            uint64
}

// Verify checks that the lease, the owner and the fencing token are handed
// over unchanged and that the new token is larger.
func (h Handover) Verify() error {
	var errs []string
	if h.CallLease == "" || h.CallLease != h.TakeoverLease || h.CallLease != h.HandleLease {
		errs = append(errs, fmt.Sprintf("lease %q/%q/%q", h.CallLease, h.TakeoverLease, h.HandleLease))
	}
	if h.CallOwner != h.TakeoverPrevOwner || h.CallToken != h.TakeoverPrevToken {
		errs = append(errs, fmt.Sprintf("previous supervisor %s/%d vs %s/%d", h.CallOwner, h.CallToken, h.TakeoverPrevOwner, h.TakeoverPrevToken))
	}
	if h.TakeoverToken <= h.CallToken {
		errs = append(errs, fmt.Sprintf("token did not increase: %d -> %d", h.CallToken, h.TakeoverToken))
	}
	if h.HandleOwner != h.TakeoverOwner || h.HandleToken != h.TakeoverToken {
		errs = append(errs, fmt.Sprintf("handle supervisor %s/%d vs takeover %s/%d", h.HandleOwner, h.HandleToken, h.TakeoverOwner, h.TakeoverToken))
	}
	if len(errs) > 0 {
		return fmt.Errorf("%w: handover: %s", ErrChain, strings.Join(errs, "; "))
	}
	return nil
}

// SaveRun appends the run record to orhc.run_record.
func SaveRun(ctx context.Context, pool *pgxpool.Pool, run durable.Run) error {
	raw, err := json.Marshal(run)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `INSERT INTO orhc.run_record(run_id,revision,status,run) VALUES ($1,$2,$3,$4)`, run.RunID, run.Revision, string(run.Status), raw)
	return err
}

// LeakScan reports which of the sensitive values appear in the text. Empty
// values are ignored. The report names the index of the value, never the
// value.
func LeakScan(text string, sensitive map[string]string) []string {
	var found []string
	for name, v := range sensitive {
		if v != "" && strings.Contains(text, v) {
			found = append(found, name)
		}
	}
	sort.Strings(found)
	return found
}

// DumpTables returns every row of the tables as JSON lines.
func DumpTables(ctx context.Context, pool *pgxpool.Pool, tables []string) (string, error) {
	var b strings.Builder
	for _, t := range tables {
		rows, err := pool.Query(ctx, fmt.Sprintf(`SELECT to_jsonb(x)::text FROM %s x`, t))
		if err != nil {
			return "", err
		}
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err != nil {
				rows.Close()
				return "", err
			}
			b.WriteString(t + " " + s + "\n")
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return "", err
		}
	}
	return b.String(), nil
}

// ControlTables and SalesRecordTables are the persistent records of steps 1-5.
var (
	ControlTables = []string{"orhc.ownership_seq", "orhc.supervisor_ownership", "orhc.ownership_event", "orhc.resource_lease", "orhc.lease_event",
		"orhc.invocation_handle", "orhc.handle_event", "orhc.call_event", "orhc.barrier", "orhc.app_instance", "orhc.run_record"}
	SalesRecordTables = []string{"tei_atlas.job", "tei_atlas.revision"}
)
