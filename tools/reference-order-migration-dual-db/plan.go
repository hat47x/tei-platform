package ordermigrationdualdb

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
)

// PlanSchema identifies the plan format of this sample.
const PlanSchema = "tei.reference.order-migration-plan/v0"

// Plan is the output of step 1. Digest covers every other field; later steps
// carry Digest and refuse to continue when it changes.
type Plan struct {
	Schema           string         `json:"schema"`
	Sample           string         `json:"sample"`
	Source           PlanSource     `json:"source"`
	Chunking         Chunking       `json:"chunking"`
	Chunks           []Chunk        `json:"chunks"`
	Reconciliation   Reconciliation `json:"reconciliation"`
	KnownDifferences []string       `json:"knownDifferences"`
	// ItemCodeMap is the legacy-to-new item code conversion observed in
	// step 1. A legacy code with more than one new code is one-to-many and
	// is never converted by guessing.
	ItemCodeMap []ItemCodeMapping `json:"itemCodeMap,omitempty"`
	Digest      string            `json:"digest,omitempty"`
}

// PlanSource records what was observed in the legacy database in step 1.
type PlanSource struct {
	Database string   `json:"database"`
	Tables   []string `json:"tables"`
	Observed Summary  `json:"observed"`
}

// Chunking declares how keys are cut into chunks.
type Chunking struct {
	Key       string `json:"key"`
	Collation string `json:"collation"`
	ChunkSize int    `json:"chunkSize"`
}

// Chunk is one key range [FromKey, ToKeyExclusive). An empty FromKey means
// unbounded below and an empty ToKeyExclusive unbounded above, so the chunks
// together cover every possible key, including keys inserted after step 1.
type Chunk struct {
	ID             string `json:"id"`
	FromKey        string `json:"fromKey"`
	ToKeyExclusive string `json:"toKeyExclusive"`
	ObservedOrders int    `json:"observedOrders"`
}

// Reconciliation declares the rules used by every later comparison.
type Reconciliation struct {
	Rules   []string `json:"rules"`
	RowHash string   `json:"rowHash"`
}

// BuildPlan cuts the observed keys into chunks and seals the plan.
func BuildPlan(obs Observation, chunkSize int) (Plan, error) {
	if chunkSize <= 0 {
		return Plan{}, errors.New("chunk size must be positive")
	}
	keys := make([]string, 0, len(obs.RowHashes))
	for k := range obs.RowHashes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var chunks []Chunk
	for start := 0; start < len(keys); start += chunkSize {
		end := start + chunkSize
		if end > len(keys) {
			end = len(keys)
		}
		from := keys[start]
		if start == 0 {
			from = ""
		}
		to := ""
		if end < len(keys) {
			to = keys[end]
		}
		chunks = append(chunks, Chunk{ID: fmt.Sprintf("c%04d", len(chunks)+1), FromKey: from, ToKeyExclusive: to, ObservedOrders: end - start})
	}
	if len(chunks) == 0 {
		chunks = []Chunk{{ID: "c0001"}}
	}
	p := Plan{
		Schema: PlanSchema,
		Sample: "order-migration-dual-db",
		Source: PlanSource{
			Database: "legacy",
			Tables:   []string{"omdd.legacy_order", "omdd.legacy_order_line"},
			Observed: obs.Summary,
		},
		Chunking:       Chunking{Key: "order_no", Collation: "C", ChunkSize: chunkSize},
		Chunks:         chunks,
		Reconciliation: Reconciliation{Rules: DefaultRules(), RowHash: RowHashAlgorithm},
		// P-DB-03 (dialect differences) is outside slice 1; nothing is declared.
		KnownDifferences: []string{},
	}
	return Seal(p)
}

// ItemCodeMapping is one row of the item code conversion.
type ItemCodeMapping struct {
	LegacyCode string `json:"legacyCode"`
	NewCode    string `json:"newCode"`
}

// KnownDifferenceItemCode is the known difference a plan declares when it
// carries an item code map: primary item codes are compared with the legacy
// rows after converting them back to legacy codes with the plan's map.
const KnownDifferenceItemCode = "item-code:primary-codes-compared-as-legacy-codes-via-plan-itemCodeMap"

// WithItemCodeMap adds the item code map to a plan, declares the matching
// known difference and reseals the plan.
func WithItemCodeMap(p Plan, m []ItemCodeMapping) (Plan, error) {
	if len(m) == 0 {
		return p, nil
	}
	m = append([]ItemCodeMapping(nil), m...)
	sort.Slice(m, func(i, j int) bool {
		if m[i].LegacyCode != m[j].LegacyCode {
			return m[i].LegacyCode < m[j].LegacyCode
		}
		return m[i].NewCode < m[j].NewCode
	})
	back := map[string]string{}
	for _, e := range m {
		if e.LegacyCode == "" || e.NewCode == "" {
			return Plan{}, errors.New("item code map has an empty code")
		}
		if prev, ok := back[e.NewCode]; ok && prev != e.LegacyCode {
			// The reverse conversion must be a function.
			return Plan{}, fmt.Errorf("new code %s maps back to both %s and %s", e.NewCode, prev, e.LegacyCode)
		}
		back[e.NewCode] = e.LegacyCode
	}
	p.ItemCodeMap = m
	p.KnownDifferences = append(append([]string{}, p.KnownDifferences...), KnownDifferenceItemCode)
	return Seal(p)
}

// Declares reports whether the plan declares a known difference.
func (p Plan) Declares(known string) bool {
	for _, k := range p.KnownDifferences {
		if k == known {
			return true
		}
	}
	return false
}

// ForwardCodes returns the new codes for a legacy code: none when the code
// is not in the map (it is kept as is), one when it converts, and more than
// one when it is one-to-many and cannot be converted without a decision.
func (p Plan) ForwardCodes(legacy string) []string {
	var out []string
	for _, e := range p.ItemCodeMap {
		if e.LegacyCode == legacy {
			out = append(out, e.NewCode)
		}
	}
	return out
}

// LegacyCode converts a primary item code back to the legacy code; codes
// outside the map are the same on both sides.
func (p Plan) LegacyCode(newCode string) string {
	for _, e := range p.ItemCodeMap {
		if e.NewCode == newCode {
			return e.LegacyCode
		}
	}
	return newCode
}

// Seal computes the digest over the plan without its digest field.
func Seal(p Plan) (Plan, error) {
	d, err := computeDigest(p)
	if err != nil {
		return Plan{}, err
	}
	p.Digest = d
	return p, nil
}

func computeDigest(p Plan) (string, error) {
	p.Digest = ""
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// Verify recomputes the digest and checks the structural rules.
func (p Plan) Verify() error {
	if p.Schema != PlanSchema {
		return fmt.Errorf("plan schema %q, want %q", p.Schema, PlanSchema)
	}
	want, err := computeDigest(p)
	if err != nil {
		return err
	}
	if p.Digest != want {
		return fmt.Errorf("plan digest mismatch: recorded %s, recomputed %s", p.Digest, want)
	}
	if len(p.Chunks) == 0 {
		return errors.New("plan has no chunks")
	}
	if p.Chunks[0].FromKey != "" || p.Chunks[len(p.Chunks)-1].ToKeyExclusive != "" {
		return errors.New("plan chunks must be unbounded at both ends")
	}
	for i := 1; i < len(p.Chunks); i++ {
		if p.Chunks[i].FromKey != p.Chunks[i-1].ToKeyExclusive {
			return fmt.Errorf("chunk %s does not start where %s ends", p.Chunks[i].ID, p.Chunks[i-1].ID)
		}
		if p.Chunks[i].FromKey <= p.Chunks[i-1].FromKey && i > 1 {
			return fmt.Errorf("chunk %s is not in ascending key order", p.Chunks[i].ID)
		}
	}
	return nil
}

// ChunkFor returns the chunk that covers key.
func (p Plan) ChunkFor(key string) Chunk {
	for _, c := range p.Chunks {
		if (c.FromKey == "" || key >= c.FromKey) && (c.ToKeyExclusive == "" || key < c.ToKeyExclusive) {
			return c
		}
	}
	return Chunk{}
}

// WritePlan writes the plan as indented JSON.
func WritePlan(path string, p Plan) error {
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

// ReadPlan reads and verifies a plan file. Unknown fields are refused.
func ReadPlan(path string) (Plan, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Plan{}, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var p Plan
	if err := dec.Decode(&p); err != nil {
		return Plan{}, fmt.Errorf("decode plan: %w", err)
	}
	if err := p.Verify(); err != nil {
		return Plan{}, err
	}
	return p, nil
}
