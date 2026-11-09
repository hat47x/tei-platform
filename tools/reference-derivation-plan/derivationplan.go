// Package derivationplan is a read-only, synthetic reference for Data
// Derivation D2 (#1640). It checks a target-neutral query plan over derived
// values: the dependency order of Derive, Filter, Sort, Limit/Offset and Count
// before any physical SQL runs, and refuses the plans that would return a
// wrong row set.
//
// It is not a SQL optimizer, a dialect rewriter, a distributed query engine, an
// authorization decision, or an accepted API. It does not write a Source. It
// never decides a row set from a physical row order or a generated identifier,
// never guarantees a stable page split without a declared complete order, and
// never collapses a multiplied reference into one row silently.
package derivationplan

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// Result codes.
const (
	CodeRows               = "RESULT_ROWS"
	CodeCount              = "RESULT_COUNT"
	CodeDeriveUnresolved   = "DERIVED_ATTR_UNRESOLVED"
	CodeIllegalPlanOrder   = "ILLEGAL_PLAN_ORDER"
	CodeNoStableOrder      = "NO_STABLE_ORDER"
	CodeUnknownField       = "UNKNOWN_FIELD"
	CodeIncompleteInput    = "INCOMPLETE_INPUT"
	CodeNullInput          = "NULL_INPUT"
	CodeComputeUnavailable = "COMPUTE_UNAVAILABLE"
	CodeInvalidRequest     = "INVALID_REQUEST"
)

// Step operations.
const (
	OpDerive = "derive"
	OpFilter = "filter"
	OpSort   = "sort"
	OpLimit  = "limit"
	OpCount  = "count"
)

// Sort directions.
const (
	SortAsc  = "asc"
	SortDesc = "desc"
)

// Element states of one row's inputs.
const (
	StatePresent = "present"
	StateNull    = "null"
	StateMissing = "missing"
)

// Row is one entity of the synthetic dataset. A missing or null input element
// keeps its declared state; the value is read only when the state is present.
type Row struct {
	ID            string `json:"id"`
	StockState    string `json:"stockState,omitempty"`
	Stock         string `json:"stock,omitempty"`
	ReservedState string `json:"reservedState,omitempty"`
	Reserved      string `json:"reserved,omitempty"`
}

// DerivedRule is the resolved derivation the plan refers to. It is the rule's
// meaning contract, not the physical SQL text.
type DerivedRule struct {
	Field      string `json:"field"`
	Expression string `json:"expression"`
}

// PlanStep is one step of a query plan.
type PlanStep struct {
	Op         string `json:"op"`
	Field      string `json:"field,omitempty"`
	Direction  string `json:"direction,omitempty"`
	TieBreaker string `json:"tieBreaker,omitempty"`
	Comparator string `json:"comparator,omitempty"`
	Operand    string `json:"operand,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
}

// Input is the whole synthetic request.
type Input struct {
	Rows         []Row         `json:"rows"`
	DerivedRules []DerivedRule `json:"derivedRules"`
	Steps        []PlanStep    `json:"steps"`
}

// Report is the read-only result.
type Report struct {
	Code     string   `json:"code"`
	Accepted bool     `json:"accepted"`
	IDs      []string `json:"ids,omitempty"`
	Count    *int     `json:"count,omitempty"`
	// NullInputCount is the number of rows whose element was a declared null
	// while the plan used the element.
	NullInputCount *int   `json:"nullInputCount,omitempty"`
	Message        string `json:"message,omitempty"`
}

func reject(code, message string) Report {
	return Report{Code: code, Message: message}
}

type rowState struct {
	id string
	// field values keyed by field name. A nil *big.Rat means a declared null; an
	// absent field is not derived (or resolved) yet.
	values map[string]*big.Rat
}

func rawRow(raw Row) (rowState, string) {
	row := rowState{id: raw.ID, values: map[string]*big.Rat{}}
	for _, element := range []struct {
		field string
		state string
		value string
	}{
		{"stock", raw.StockState, raw.Stock},
		{"reserved", raw.ReservedState, raw.Reserved},
	} {
		state := element.state
		if state == "" {
			state = StatePresent
		}
		switch state {
		case StatePresent:
			value, ok := new(big.Rat).SetString(strings.TrimSpace(element.value))
			if !ok {
				return rowState{}, fmt.Sprintf("行 %q の入力 %q が数値でない。", raw.ID, element.field)
			}
			row.values[element.field] = value
		case StateNull:
			row.values[element.field] = nil
		case StateMissing:
			return rowState{}, fmt.Sprintf("行 %q の入力 %q が未取得。", raw.ID, element.field)
		default:
			return rowState{}, fmt.Sprintf("unknown state %q", state)
		}
	}
	return row, ""
}

// splitExpression accepts exactly `<field> - <field>`.
func splitExpression(expression string) (string, string, error) {
	parts := strings.Fields(expression)
	if len(parts) != 3 || parts[1] != "-" {
		return "", "", fmt.Errorf("未対応の式 %q。対応するのは <field> - <field> のみ。", expression)
	}
	return parts[0], parts[2], nil
}

func isIdentifier(field string) bool {
	for _, ch := range field {
		if !(ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')) {
			return false
		}
	}
	return field != ""
}

func fieldKnown(field string, rules map[string]DerivedRule) bool {
	if isRawField(field) {
		return true
	}
	_, ok := rules[field]
	return ok
}

func isRawField(field string) bool {
	switch field {
	case "stock", "reserved", "id":
		return true
	default:
		return false
	}
}

func appliedDerive(field string, applied map[string]DerivedRule) bool {
	_, ok := applied[field]
	return ok
}

func limitSeenBefore(steps []PlanStep, index int) bool {
	for i := 0; i < index; i++ {
		if steps[i].Op == OpLimit {
			return true
		}
	}
	return false
}

func limitSeenAfter(steps []PlanStep, index int) bool {
	for i := index + 1; i < len(steps); i++ {
		if steps[i].Op == OpLimit {
			return true
		}
	}
	return false
}

func comparatorLegal(op string) bool {
	switch op {
	case ">", ">=", "<", "<=", "=":
		return true
	default:
		return false
	}
}

func applies(value, operand *big.Rat, op string) bool {
	if value == nil || operand == nil {
		return false
	}
	c := value.Cmp(operand)
	switch op {
	case ">":
		return c > 0
	case ">=":
		return c >= 0
	case "<":
		return c < 0
	case "<=":
		return c <= 0
	case "=":
		return c == 0
	}
	return false
}

// lessOrders a declared null last, then the numeric order.
func less(a, b *big.Rat) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	return a.Cmp(b) < 0
}

// lessRows is the ordering for one sort step.
func lessRows(a, b rowState, field, direction, tieBreaker string) bool {
	ava, bva := a.values[field], b.values[field]
	switch {
	case ava == nil && bva == nil:
	case ava == nil:
		// a declared null goes last, regardless of direction
		return false
	case bva == nil:
		return true
	default:
		cmp := ava.Cmp(bva)
		if direction == SortDesc {
			cmp = -cmp
		}
		if cmp != 0 {
			return cmp < 0
		}
	}
	if strings.TrimSpace(tieBreaker) == "id" {
		return a.id < b.id
	}
	if tieBreaker != "" {
		return less(a.values[tieBreaker], b.values[tieBreaker])
	}
	return false
}

// Run checks one plan and returns its result or a distinct refusal. It never
// writes anything.
func Run(in Input) Report {
	if len(in.Steps) == 0 {
		return reject(CodeInvalidRequest, "計画ステップが未指定。")
	}
	rules := map[string]DerivedRule{}
	for _, rule := range in.DerivedRules {
		if !isIdentifier(rule.Field) {
			return reject(CodeInvalidRequest, fmt.Sprintf("導出規則の項目名 %q が語として不適切。", rule.Field))
		}
		if _, duplicate := rules[rule.Field]; duplicate {
			return reject(CodeInvalidRequest, fmt.Sprintf("導出規則 %q が重複。", rule.Field))
		}
		rules[rule.Field] = rule
	}

	rows := make([]rowState, 0, len(in.Rows))
	for _, raw := range in.Rows {
		row, message := rawRow(raw)
		if message != "" {
			return reject(CodeIncompleteInput, message)
		}
		rows = append(rows, row)
	}

	// First pass: plan legality, before any value is computed.
	applied := map[string]DerivedRule{}
	for i, step := range in.Steps {
		switch step.Op {
		case OpDerive:
			if _, ok := rules[step.Field]; !ok {
				return reject(CodeUnknownField, fmt.Sprintf("解決されていない導出規則 %q。", step.Field))
			}
		case OpFilter, OpSort:
			if limitSeenBefore(in.Steps, i) {
				return reject(CodeIllegalPlanOrder,
					fmt.Sprintf("ステップ%d の %q が Limit 以降にある。取得後にその場で付け足した値で絞り替えない。", i+1, step.Op))
			}
			if !fieldKnown(step.Field, rules) {
				return reject(CodeUnknownField, fmt.Sprintf("未解決の項目 %q。", step.Field))
			}
			if !isRawField(step.Field) && !appliedDerive(step.Field, applied) {
				return reject(CodeDeriveUnresolved,
					fmt.Sprintf("導出項目 %q が導出ステップの前に使用されている。", step.Field))
			}
			if step.Op == OpSort {
				if step.Direction == "" {
					return reject(CodeNoStableOrder, "整列方向が未指定。物理行順や生成IDを業務順位にしない。")
				}
				if step.Direction != SortAsc && step.Direction != SortDesc {
					return reject(CodeInvalidRequest, fmt.Sprintf("未知の方向 %q。", step.Direction))
				}
				if limitSeenAfter(in.Steps, i) && strings.TrimSpace(step.TieBreaker) == "" {
					return reject(CodeNoStableOrder,
						fmt.Sprintf("安定ページ分割には完全な順序が必要。同値の %q で扱える二次順序を宣言しない。", step.Field))
				}
				if strings.TrimSpace(step.TieBreaker) != "" && !fieldKnown(step.TieBreaker, rules) {
					return reject(CodeUnknownField, fmt.Sprintf("未解決の二次順序の項目 %q。", step.TieBreaker))
				}
			} else {
				if !comparatorLegal(step.Comparator) {
					return reject(CodeInvalidRequest, fmt.Sprintf("未知の比較演算子 %q。", step.Comparator))
				}
				if _, ok := new(big.Rat).SetString(strings.TrimSpace(step.Operand)); !ok {
					return reject(CodeInvalidRequest, fmt.Sprintf("比較の値 %q が数値でない。", step.Operand))
				}
			}
		case OpLimit:
			if step.Limit < 1 {
				return reject(CodeInvalidRequest, "limit は1以上。")
			}
			if step.Offset < 0 {
				return reject(CodeInvalidRequest, "offset が負。")
			}
		case OpCount:
		default:
			return reject(CodeInvalidRequest, fmt.Sprintf("unknown operation %q", step.Op))
		}
		if step.Op == OpDerive {
			applied[step.Field] = rules[step.Field]
		}
	}

	// Second pass: execute the plan over the current row set.
	nullHits := 0
	current := rows
	for _, step := range in.Steps {
		switch step.Op {
		case OpDerive:
			left, right, err := splitExpression(rules[step.Field].Expression)
			if err != nil {
				return reject(CodeComputeUnavailable, err.Error())
			}
			for _, field := range []string{left, right} {
				if !fieldKnown(field, rules) {
					return reject(CodeComputeUnavailable, fmt.Sprintf("式 %q が未解決の語 %q。", rules[step.Field].Expression, field))
				}
			}
			for i := range current {
				leftValue, leftKnown := current[i].values[left]
				rightValue, rightKnown := current[i].values[right]
				if !leftKnown || !rightKnown {
					return reject(CodeComputeUnavailable, fmt.Sprintf("式 %q が未解決の語を参照する。", rules[step.Field].Expression))
				}
				if leftValue == nil || rightValue == nil {
					current[i].values[step.Field] = nil
					continue
				}
				current[i].values[step.Field] = new(big.Rat).Sub(leftValue, rightValue)
			}
		case OpFilter:
			operand, _ := new(big.Rat).SetString(strings.TrimSpace(step.Operand))
			kept := make([]rowState, 0, len(current))
			for _, row := range current {
				value, known := row.values[step.Field]
				if !known {
					// Unresolved: never happens after the legality pass.
					continue
				}
				if value == nil {
					// SQL-like: a null comparison stays unmatched.
					nullHits++
					continue
				}
				if applies(value, operand, step.Comparator) {
					kept = append(kept, row)
				}
			}
			current = kept
		case OpSort:
			filter := func(i, j int) bool {
				return lessRows(current[i], current[j], step.Field, step.Direction, step.TieBreaker)
			}
			sort.SliceStable(current, filter)
		case OpLimit:
			offset := step.Offset
			if offset >= len(current) {
				current = nil
				break
			}
			end := offset + step.Limit
			if end > len(current) {
				end = len(current)
			}
			current = current[offset:end]
		case OpCount:
			count := len(current)
			report := Report{Code: CodeCount, Accepted: true, Count: &count, Message: "導出後に Filter と Count の依存順を守った件数。"}
			if nullHits > 0 {
				n := nullHits
				report.NullInputCount = &n
			}
			return report
		default:
			return reject(CodeInvalidRequest, "unknown operation.")
		}
	}

	ids := make([]string, 0, len(current))
	for _, row := range current {
		ids = append(ids, row.id)
	}
	report := Report{Code: CodeRows, Accepted: true, IDs: ids}
	if nullHits > 0 {
		n := nullHits
		report.NullInputCount = &n
		report.Message = "NULL の比較値は一致しない扱い。NULL を0や未取得と混同しない。"
	}
	return report
}
