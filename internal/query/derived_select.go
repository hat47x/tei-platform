package query

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic/derivationref"
)

// DerivedSelect is a deliberately narrow, target-neutral D2 execution contract.
// It does not replace native SQL, the accepted semantic model, or authorization.
type DerivedSelect struct {
	Rule       derivationref.Definition
	Binding    derivationref.Binding
	Inputs     map[string]core.CanonicalID // rule input name -> explicitly mapped semantic attribute
	Where      *DerivedWhere
	Order      []DerivedOrder
	Limit      int
	HasLimit   bool
	Offset     int
	MaxRows    int // mandatory upper bound on a complete, unpaginated source
}

type DerivedWhere struct {
	Op      string // =, !=, <, <=, >, >=
	Decimal string
}

type DerivedOrder struct {
	Attribute core.CanonicalID
	Descending bool
	Kind string // decimal or text; explicit, never inferred from an identifier
}

type DerivedRow struct {
	Values map[core.CanonicalID]derivationref.Input
}

type DerivedSelectResult struct {
	Rows  []DerivedRow
	Count int // after WHERE but before OFFSET / LIMIT
}

// CompleteSourceEvidence must originate from a trusted analyzer of the exact
// native SQL. It is NOT proof of Provider authority or complete retrieval.
type CompleteSourceEvidence struct {
	SQLSHA256 string
	IR        DerivedIR
}

var ErrDerivedUnsupported = errors.New("derived SELECT is unsupported")
var ErrDerivedUnauthorized = errors.New("derived SELECT attribute authorization denied")
var ErrDerivedIncomplete = errors.New("derived SELECT requires complete bounded source")

func DigestNativeSQL(source string) string {
	sum := sha256.Sum256([]byte(source))
	return hex.EncodeToString(sum[:])
}

// ValidateDerivedSelect refuses uncertain SQL source shapes and unsafe plans
// BEFORE dispatch. An analyzed single-table projection is only a prerequisite;
// the trusted Executor must independently attest to complete retrieval.
func ValidateDerivedSelect(q EmbeddedSQL, p DerivedSelect, evidence CompleteSourceEvidence, canRead func(core.CanonicalID) bool) error {
	if evidence.SQLSHA256 == "" || evidence.SQLSHA256 != DigestNativeSQL(q.Source) ||
		evidence.IR.Query != q.ID || evidence.IR.Coverage.Parse != CoverageFull ||
		evidence.IR.Coverage.Lowering != CoverageFull {
		return fmt.Errorf("%w: missing or stale full SQL analysis", ErrDerivedUnsupported)
	}
	if len(evidence.IR.Relational.Nodes) == 0 || evidence.IR.Relational.Root == "" {
		return fmt.Errorf("%w: missing relational source graph", ErrDerivedUnsupported)
	}
	seen := map[string]bool{}
	sources, projects := 0, 0
	for _, node := range evidence.IR.Relational.Nodes {
		if node.ID == "" || seen[node.ID] {
			return fmt.Errorf("%w: invalid relational node identity", ErrDerivedUnsupported)
		}
		seen[node.ID] = true
		switch node.Kind {
		case RelSource:
			sources++
			if node.Source == "" || (node.SourceKind != "" && node.SourceKind != "table") {
				return fmt.Errorf("%w: ambiguous or non-table source", ErrDerivedUnsupported)
			}
		case RelProject:
			projects++
			// A SQL computed projection, including aggregates or subqueries, cannot
			// serve as a proven raw row source for Go-side derivation.
			for _, expr := range node.Expressions {
				if !simpleProjection(expr) {
					return fmt.Errorf("%w: computed native SQL projection", ErrDerivedUnsupported)
				}
			}
		default:
			// In particular reject pre-derive WHERE, JOIN, DISTINCT, GROUP,
			// SORT and LIMIT; never silently use the wrong result universe.
			return fmt.Errorf("%w: physical operation %s before derivation", ErrDerivedUnsupported, node.Kind)
		}
	}
	if sources != 1 || projects != 1 || !seen[evidence.IR.Relational.Root] {
		return fmt.Errorf("%w: unsupported native SQL source shape", ErrDerivedUnsupported)
	}
	if p.MaxRows < 1 || p.Offset < 0 || (p.HasLimit && p.Limit < 0) || (!p.HasLimit && p.Limit != 0) {
		return fmt.Errorf("%w: missing bounds or invalid window", ErrDerivedUnsupported)
	}
	if p.Rule.ID == "" || !p.Rule.Pure || p.Binding.RuleID != p.Rule.ID ||
		p.Binding.Scope != "query" || p.Binding.TargetID == "" ||
		(p.Rule.Operator != "subtract" && p.Rule.Operator != "multiply") ||
		len(p.Rule.Inputs) != 2 || len(p.Inputs) != 2 {
		return fmt.Errorf("%w: unresolved pure derivation binding/operator", ErrDerivedUnsupported)
	}
	columns := map[core.CanonicalID]bool{}
	if canRead == nil {
		return ErrDerivedUnauthorized
	}
	for _, mapping := range q.ResultColumns {
		if mapping.Attribute == "" || columns[mapping.Attribute] {
			return fmt.Errorf("%w: ambiguous explicit result mapping", ErrDerivedUnsupported)
		}
		columns[mapping.Attribute] = true
		// All returned raw attributes need their own disclosure authority.
		if !canRead(mapping.Attribute) {
			return ErrDerivedUnauthorized
		}
	}
	target := core.CanonicalID(p.Binding.TargetID)
	if columns[target] || !canRead(target) {
		if columns[target] {
			return fmt.Errorf("%w: derived target shadows mapped raw attribute", ErrDerivedUnsupported)
		}
		return ErrDerivedUnauthorized
	}
	used := map[core.CanonicalID]bool{}
	for _, name := range p.Rule.Inputs {
		attr := p.Inputs[name]
		if name == "" || attr == "" || !columns[attr] || used[attr] {
			return fmt.Errorf("%w: unresolved, duplicate or unprojected input", ErrDerivedUnsupported)
		}
		used[attr] = true
	}
	if p.Where != nil {
		if _, err := parseDecimal(p.Where.Decimal); err != nil {
			return fmt.Errorf("%w: invalid filter value: %v", ErrDerivedUnsupported, err)
		}
		switch p.Where.Op {
		case "=", "!=", "<", "<=", ">", ">=":
		default:
			return fmt.Errorf("%w: unsupported filter operator", ErrDerivedUnsupported)
		}
	}
	if (p.HasLimit || p.Offset > 0) && len(p.Order) == 0 {
		return fmt.Errorf("%w: paginated query needs explicit deterministic sort", ErrDerivedUnsupported)
	}
	sortAttrs := map[core.CanonicalID]bool{}
	for _, ordering := range p.Order {
		if ordering.Attribute == "" || sortAttrs[ordering.Attribute] ||
			(ordering.Attribute != target && !columns[ordering.Attribute]) ||
			(ordering.Kind != "decimal" && ordering.Kind != "text") {
			return fmt.Errorf("%w: invalid ordering attribute or type", ErrDerivedUnsupported)
		}
		if ordering.Attribute == target && ordering.Kind != "decimal" {
			return fmt.Errorf("%w: derived decimal ordering must be numeric", ErrDerivedUnsupported)
		}
		sortAttrs[ordering.Attribute] = true
	}
	return nil
}

func simpleProjection(expr string) bool {
	// Deliberately conservative: no aliases, wildcard, functions, quoting,
	// expressions or hidden row-shape transformations in this first slice.
	if expr == "" {
		return false
	}
	for _, part := range strings.Split(expr, ".") {
		if part == "" {
			return false
		}
		for i, r := range part {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				r == '_' || (i > 0 && r >= '0' && r <= '9')) {
				return false
			}
		}
	}
	return true
}

func parseDecimal(s string) (*big.Rat, error) {
	if s == "" || strings.ContainsAny(s, "/eE \t\n") {
		return nil, fmt.Errorf("not a finite decimal literal")
	}
	for i, r := range s {
		if (r < '0' || r > '9') && r != '.' && !(i == 0 && (r == '+' || r == '-')) {
			return nil, fmt.Errorf("not a decimal literal")
		}
	}
	v, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, fmt.Errorf("invalid decimal literal")
	}
	return v, nil
}

func exactDecimal(r *big.Rat) (string, error) {
	d := new(big.Int).Set(r.Denom())
	scale := 0
	for _, factor := range []int64{2, 5} {
		n := 0
		f := big.NewInt(factor)
		for {
			q, rem := new(big.Int), new(big.Int)
			q.QuoRem(d, f, rem)
			if rem.Sign() != 0 {
				break
			}
			d = q
			n++
		}
		if n > scale {
			scale = n
		}
	}
	if d.Cmp(big.NewInt(1)) != 0 {
		return "", fmt.Errorf("%w: nonterminating decimal needs rounding contract", ErrDerivedUnsupported)
	}
	return r.FloatString(scale), nil
}

func deriveValue(rule derivationref.Definition, a, b derivationref.Input) (derivationref.Input, error) {
	switch rule.Operator {
	case "multiply":
		return derivationref.Multiply(a, b)
	case "subtract":
		for _, v := range []derivationref.Input{a, b} {
			if v.State != derivationref.Present && v.State != derivationref.Null {
				return derivationref.Input{}, fmt.Errorf("%w: input state %s", derivationref.ErrInput, v.State)
			}
		}
		if a.State == derivationref.Null || b.State == derivationref.Null {
			return derivationref.Input{State: derivationref.Null}, nil
		}
		left, err := parseDecimal(a.Decimal)
		if err != nil {
			return derivationref.Input{}, fmt.Errorf("%w: left input: %v", derivationref.ErrOperator, err)
		}
		right, err := parseDecimal(b.Decimal)
		if err != nil {
			return derivationref.Input{}, fmt.Errorf("%w: right input: %v", derivationref.ErrOperator, err)
		}
		value, err := exactDecimal(new(big.Rat).Sub(left, right))
		if err != nil {
			return derivationref.Input{}, err
		}
		return derivationref.Input{State: derivationref.Present, Decimal: value}, nil
	default:
		return derivationref.Input{}, derivationref.ErrOperator
	}
}

type selectedRow struct {
	row  DerivedRow
	keys []derivedKey
}
type derivedKey struct {
	text string
	number *big.Rat
}

func orderCompare(a, b selectedRow, orders []DerivedOrder) int {
	for i, o := range orders {
		cmp := strings.Compare(a.keys[i].text, b.keys[i].text)
		if o.Kind == "decimal" {
			cmp = a.keys[i].number.Cmp(b.keys[i].number)
		}
		if cmp != 0 {
			if o.Descending {
				return -cmp
			}
			return cmp
		}
	}
	return 0
}

// EvaluateDerivedSelect performs derivation before filtering, then sorts the
// whole filtered set, counts it, and applies offset and limit LAST.
func EvaluateDerivedSelect(p DerivedSelect, rows []DerivedRow, complete bool) (DerivedSelectResult, error) {
	if !complete || p.MaxRows <= 0 || len(rows) > p.MaxRows {
		return DerivedSelectResult{}, ErrDerivedIncomplete
	}
	if len(p.Rule.Inputs) != 2 || p.Binding.TargetID == "" || (p.Rule.Operator != "subtract" && p.Rule.Operator != "multiply") {
		return DerivedSelectResult{}, ErrDerivedUnsupported
	}
	target := core.CanonicalID(p.Binding.TargetID)
	var threshold *big.Rat
	if p.Where != nil {
		var err error
		threshold, err = parseDecimal(p.Where.Decimal)
		if err != nil {
			return DerivedSelectResult{}, err
		}
	}
	chosen := make([]selectedRow, 0, len(rows))
	for _, source := range rows {
		a, ok := source.Values[p.Inputs[p.Rule.Inputs[0]]]
		if !ok {
			a = derivationref.Input{State: derivationref.Missing}
		}
		b, ok := source.Values[p.Inputs[p.Rule.Inputs[1]]]
		if !ok {
			b = derivationref.Input{State: derivationref.Missing}
		}
		value, err := deriveValue(p.Rule, a, b)
		if err != nil {
			return DerivedSelectResult{}, fmt.Errorf("%w: cannot derive row: %v", ErrDerivedUnsupported, err)
		}
		fields := make(map[core.CanonicalID]derivationref.Input, len(source.Values)+1)
		for k, v := range source.Values {
			fields[k] = v
		}
		fields[target] = value
		if p.Where != nil {
			if value.State == derivationref.Null {
				continue // SQL UNKNOWN, not zero and not an execution error
			}
			left, err := parseDecimal(value.Decimal)
			if err != nil {
				return DerivedSelectResult{}, err
			}
			cmp := left.Cmp(threshold)
			keep := (p.Where.Op == "=" && cmp == 0) || (p.Where.Op == "!=" && cmp != 0) ||
				(p.Where.Op == "<" && cmp < 0) || (p.Where.Op == "<=" && cmp <= 0) ||
				(p.Where.Op == ">" && cmp > 0) || (p.Where.Op == ">=" && cmp >= 0)
			if !keep {
				continue
			}
		}
		item := selectedRow{row: DerivedRow{Values: fields}}
		for _, o := range p.Order {
			in, exists := fields[o.Attribute]
			if !exists || in.State != derivationref.Present {
				return DerivedSelectResult{}, fmt.Errorf("%w: absent or NULL sort field", ErrDerivedUnsupported)
			}
			key := derivedKey{text: in.Decimal}
			if o.Kind == "decimal" {
				key.number, err = parseDecimal(in.Decimal)
				if err != nil {
					return DerivedSelectResult{}, fmt.Errorf("%w: imprecise/invalid numeric sort field", ErrDerivedUnsupported)
				}
			}
			item.keys = append(item.keys, key)
		}
		chosen = append(chosen, item)
	}
	sort.SliceStable(chosen, func(i, j int) bool { return orderCompare(chosen[i], chosen[j], p.Order) < 0 })
	if p.HasLimit || p.Offset > 0 {
		if len(p.Order) == 0 {
			return DerivedSelectResult{}, ErrDerivedUnsupported
		}
		for i := 1; i < len(chosen); i++ {
			if orderCompare(chosen[i-1], chosen[i], p.Order) == 0 {
				return DerivedSelectResult{}, fmt.Errorf("%w: sort does not uniquely order rows for stable pagination", ErrDerivedUnsupported)
			}
		}
	}
	count := len(chosen)
	start := p.Offset
	if start > count {
		start = count
	}
	end := count
	if p.HasLimit && p.Limit < end-start {
		end = start + p.Limit
	}
	result := DerivedSelectResult{Count: count, Rows: make([]DerivedRow, 0, end-start)}
	for _, row := range chosen[start:end] {
		result.Rows = append(result.Rows, row.row)
	}
	return result, nil
}
