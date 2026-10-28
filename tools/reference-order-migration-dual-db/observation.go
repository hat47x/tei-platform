// Package ordermigrationdualdb is the slice-1 (steps 1-3) implementation of
// the order-migration-dual-db technical validation sample. It observes a
// legacy order database, builds a digest-bound migration plan, copies the
// legacy rows at one recorded snapshot, migrates chunk by chunk into a
// primary database with a sample-local effect outbox, and delivers stock
// reservations to a separate inventory database.
//
// Everything here is sample-specific. It deliberately does not live under
// internal/: the chunk resume points, the cross-database commit boundary and
// the compensation rule are not core declarations yet (see README).
package ordermigrationdualdb

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
)

// RowHashAlgorithm names the canonical row encoding used for row hashes.
const RowHashAlgorithm = "sha256-canonical-order-v1"

// Order is one order backlog entry with its lines, as read from any of the
// three table sets (legacy, staged copy, primary). Numeric values are kept as
// decimal strings read with ::text so no floating point is involved.
type Order struct {
	OrderNo      string
	CustomerCode string
	OrderedOn    string // YYYY-MM-DD
	Status       string
	Lines        []Line
}

// Line is one order line.
type Line struct {
	LineNo    int
	ItemCode  string
	Qty       int64
	UnitPrice string
	Amount    string
}

// RowHash returns the canonical hash of one order including its lines.
func RowHash(o Order) string {
	lines := append([]Line(nil), o.Lines...)
	sort.Slice(lines, func(i, j int) bool { return lines[i].LineNo < lines[j].LineNo })
	var b strings.Builder
	field := func(name, value string) {
		b.WriteString(name)
		b.WriteByte('=')
		b.WriteString(strconv.Itoa(len(value)))
		b.WriteByte(':')
		b.WriteString(value)
		b.WriteByte(0x1f)
	}
	field("order_no", o.OrderNo)
	field("customer_code", o.CustomerCode)
	field("ordered_on", o.OrderedOn)
	field("status", o.Status)
	b.WriteByte(0x1e)
	for _, l := range lines {
		field("line_no", strconv.Itoa(l.LineNo))
		field("item_code", l.ItemCode)
		field("qty", strconv.FormatInt(l.Qty, 10))
		field("unit_price", l.UnitPrice)
		field("amount", l.Amount)
		b.WriteByte(0x1e)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Summary is the aggregate part of an observation. It is what the plan and
// the snapshot record carry; per-row hashes stay with the full Observation.
type Summary struct {
	OrderCount       int               `json:"orderCount"`
	LineCount        int               `json:"lineCount"`
	AmountSum        string            `json:"amountSum"`
	MonthlyAmountSum map[string]string `json:"monthlyAmountSum"`
	RowsetDigest     string            `json:"rowsetDigest"`
}

// Observation is a summary plus the per-key row hashes.
type Observation struct {
	Summary
	RowHashes map[string]string `json:"-"`
}

// Observe computes the reconciliation values for a set of orders.
func Observe(orders []Order) (Observation, error) {
	obs := Observation{
		Summary:   Summary{MonthlyAmountSum: map[string]string{}},
		RowHashes: map[string]string{},
	}
	total := new(big.Int)
	monthly := map[string]*big.Int{}
	for _, o := range orders {
		if _, dup := obs.RowHashes[o.OrderNo]; dup {
			return Observation{}, fmt.Errorf("duplicate order key %q", o.OrderNo)
		}
		if len(o.OrderedOn) != len("2006-01-02") {
			return Observation{}, fmt.Errorf("order %s: ordered_on %q is not YYYY-MM-DD", o.OrderNo, o.OrderedOn)
		}
		month := o.OrderedOn[:7]
		if monthly[month] == nil {
			monthly[month] = new(big.Int)
		}
		for _, l := range o.Lines {
			amount, ok := new(big.Int).SetString(l.Amount, 10)
			if !ok {
				return Observation{}, fmt.Errorf("order %s line %d: amount %q is not an integer yen value", o.OrderNo, l.LineNo, l.Amount)
			}
			total.Add(total, amount)
			monthly[month].Add(monthly[month], amount)
			obs.LineCount++
		}
		obs.OrderCount++
		obs.RowHashes[o.OrderNo] = RowHash(o)
	}
	obs.AmountSum = total.String()
	for m, v := range monthly {
		obs.MonthlyAmountSum[m] = v.String()
	}
	obs.RowsetDigest = RowsetDigest(obs.RowHashes)
	return obs, nil
}

// RowsetDigest hashes the sorted key/row-hash pairs. Keys are compared as
// bytes, matching COLLATE "C" in PostgreSQL.
func RowsetDigest(rowHashes map[string]string) string {
	keys := make([]string, 0, len(rowHashes))
	for k := range rowHashes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		fmt.Fprintf(h, "%d:%s=%s\n", len(k), k, rowHashes[k])
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// Rule names used in the plan's reconciliation rules.
const (
	RuleOrderCount     = "order-count"
	RuleLineCount      = "line-count"
	RuleAmountSum      = "amount-sum"
	RuleMonthlyAmount  = "monthly-amount-sum"
	RuleKeySet         = "key-set"
	RuleRowHash        = "row-hash"
	RuleRowsetDigest   = "rowset-digest"
	maxReportedKeyDiff = 20
)

// DefaultRules is the rule set every plan built by this sample declares.
func DefaultRules() []string {
	return []string{RuleOrderCount, RuleLineCount, RuleAmountSum, RuleMonthlyAmount, RuleKeySet, RuleRowHash, RuleRowsetDigest}
}

// Difference is one failed reconciliation rule.
type Difference struct {
	Rule   string   `json:"rule"`
	Source string   `json:"source,omitempty"`
	Target string   `json:"target,omitempty"`
	Keys   []string `json:"keys,omitempty"`
}

// Compare applies the declared rules. An unknown rule is a difference, not a
// silent pass.
func Compare(rules []string, source, target Observation) []Difference {
	var diffs []Difference
	for _, rule := range rules {
		switch rule {
		case RuleOrderCount:
			if source.OrderCount != target.OrderCount {
				diffs = append(diffs, Difference{Rule: rule, Source: strconv.Itoa(source.OrderCount), Target: strconv.Itoa(target.OrderCount)})
			}
		case RuleLineCount:
			if source.LineCount != target.LineCount {
				diffs = append(diffs, Difference{Rule: rule, Source: strconv.Itoa(source.LineCount), Target: strconv.Itoa(target.LineCount)})
			}
		case RuleAmountSum:
			if source.AmountSum != target.AmountSum {
				diffs = append(diffs, Difference{Rule: rule, Source: source.AmountSum, Target: target.AmountSum})
			}
		case RuleMonthlyAmount:
			months := map[string]bool{}
			for m := range source.MonthlyAmountSum {
				months[m] = true
			}
			for m := range target.MonthlyAmountSum {
				months[m] = true
			}
			var bad []string
			for m := range months {
				if source.MonthlyAmountSum[m] != target.MonthlyAmountSum[m] {
					bad = append(bad, m)
				}
			}
			if len(bad) > 0 {
				sort.Strings(bad)
				diffs = append(diffs, Difference{Rule: rule, Keys: bad})
			}
		case RuleKeySet:
			var missing, extra []string
			for k := range source.RowHashes {
				if _, ok := target.RowHashes[k]; !ok {
					missing = append(missing, "missing:"+k)
				}
			}
			for k := range target.RowHashes {
				if _, ok := source.RowHashes[k]; !ok {
					extra = append(extra, "extra:"+k)
				}
			}
			if len(missing)+len(extra) > 0 {
				keys := append(missing, extra...)
				sort.Strings(keys)
				diffs = append(diffs, Difference{Rule: rule, Keys: capKeys(keys)})
			}
		case RuleRowHash:
			var bad []string
			for k, h := range source.RowHashes {
				if th, ok := target.RowHashes[k]; ok && th != h {
					bad = append(bad, k)
				}
			}
			if len(bad) > 0 {
				sort.Strings(bad)
				diffs = append(diffs, Difference{Rule: rule, Keys: capKeys(bad)})
			}
		case RuleRowsetDigest:
			if source.RowsetDigest != target.RowsetDigest {
				diffs = append(diffs, Difference{Rule: rule, Source: source.RowsetDigest, Target: target.RowsetDigest})
			}
		default:
			diffs = append(diffs, Difference{Rule: rule, Source: "unknown rule"})
		}
	}
	return diffs
}

func capKeys(keys []string) []string {
	if len(keys) <= maxReportedKeyDiff {
		return keys
	}
	out := append([]string(nil), keys[:maxReportedKeyDiff]...)
	return append(out, fmt.Sprintf("... %d more", len(keys)-maxReportedKeyDiff))
}
