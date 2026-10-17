// Package relationaggregate is a read-only, synthetic reference for Data
// Derivation D9 (#1706). It checks the meaning of a relation collection that is
// bound to a derivation input: the collection shape, whether the collection was
// fully retrieved, JOIN fan-out, duplicate counting, the empty set, and NULL
// elements.
//
// It is not a Query planner, a SQL executor, an authorization decision, a
// generic ORM, or an accepted API. It does not write a Source. A JOIN fan-out is
// never silently turned into a larger business total, an incomplete collection is
// never reported as a complete total, and an undeclared empty-set or NULL policy
// is never assumed to be zero.
package relationaggregate

import (
	"fmt"
	"math/big"
	"strings"
)

// Result codes.
const (
	CodeOK                       = "OK"
	CodeEmptySetZero             = "EMPTY_SET_ZERO"
	CodeEmptySetNull             = "EMPTY_SET_NULL"
	CodeEmptySetPolicyUndeclared = "EMPTY_SET_POLICY_UNDECLARED"
	CodeNullIgnored              = "NULL_IGNORED"
	CodeNullPropagated           = "NULL_PROPAGATED"
	CodeNullPolicyUndeclared     = "NULL_ELEMENT_POLICY_UNDECLARED"
	CodeIncompleteSet            = "INCOMPLETE_SET"
	CodeInputUnavailable         = "INPUT_UNAVAILABLE"
	CodeJoinFanOutDetected       = "JOIN_FANOUT_DETECTED"
	CodeMultiplicityViolation    = "MULTIPLICITY_VIOLATION"
	CodeScalarReduceRequired     = "SCALAR_REDUCE_REQUIRED"
	CodeAuthorizationDelegated   = "AUTHORIZATION_DELEGATED"
	CodeUnsupported              = "UNSUPPORTED_OPERATION"
	CodeInvalidRequest           = "INVALID_REQUEST"
)

// Collection shapes.
const (
	ShapeOne  = "one"
	ShapeMany = "many"
)

// Target shapes.
const (
	TargetAggregate = "aggregate"
	TargetScalar    = "scalar"
)

// Element states.
const (
	ElementPresent          = "present"
	ElementNull             = "null"
	ElementInputUnavailable = "input-unavailable"
)

// Empty-set policies.
const (
	EmptySetZero = "zero"
	EmptySetNull = "null"
)

// NULL policies.
const (
	NullIgnore    = "ignore"
	NullPropagate = "propagate"
)

// Element is one member of the relation collection.
type Element struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Amount string `json:"amount,omitempty"`
}

// Input is the whole synthetic request.
type Input struct {
	// Shape is the declared multiplicity of the relation binding.
	Shape string `json:"shape"`
	// Complete is true only when the whole business collection was retrieved. A
	// display page or a truncated provider response is not complete.
	Complete bool `json:"complete"`
	// Elements are the distinct business elements of the collection.
	Elements []Element `json:"elements"`
	// JoinFanOut is the number of copies of each element that a JOIN with another
	// relation would introduce. 1 means no fan-out.
	JoinFanOut int `json:"joinFanOut,omitempty"`
	// TargetShape is where the collection is bound.
	TargetShape string `json:"targetShape"`
	// Reduce is the explicit reduction required to bind many to scalar.
	Reduce string `json:"reduce,omitempty"`
	// EmptySetPolicy declares the SUM(empty) meaning. An empty string means it is
	// not declared here.
	EmptySetPolicy string `json:"emptySetPolicy,omitempty"`
	// NullPolicy declares how a NULL element is treated.
	NullPolicy string `json:"nullPolicy,omitempty"`
	// Authorization is "allowed" or "denied".
	Authorization string `json:"authorization"`
	// Op is the operation; only "sum" is owned here.
	Op string `json:"op"`
}

// Report is the read-only result.
type Report struct {
	Code         string `json:"code"`
	Accepted     bool   `json:"accepted"`
	Result       string `json:"result,omitempty"`
	ResultNull   bool   `json:"resultNull,omitempty"`
	NaiveJoinSum string `json:"naiveJoinSum,omitempty"`
	Message      string `json:"message,omitempty"`
}

func reject(code, message string) Report {
	return Report{Code: code, Message: message}
}

// Evaluate checks the collection binding and returns the distinct-element total
// or a distinct refusal. It never writes anything.
func Evaluate(in Input) Report {
	if strings.TrimSpace(in.Op) != "sum" {
		return reject(CodeUnsupported, fmt.Sprintf("unsupported operation %q", in.Op))
	}
	if strings.TrimSpace(in.Authorization) == "denied" {
		return reject(CodeAuthorizationDelegated, "行レベル権限の判断は #1693 へ委譲する。")
	}
	if strings.TrimSpace(in.Shape) == ShapeOne && len(in.Elements) != 1 {
		return reject(CodeMultiplicityViolation,
			fmt.Sprintf("多重度 one の Relation が %d 件一致した。先頭要素を選択しない。", len(in.Elements)))
	}
	if strings.TrimSpace(in.Shape) == ShapeMany && strings.TrimSpace(in.TargetShape) == TargetScalar && strings.TrimSpace(in.Reduce) == "" {
		return reject(CodeScalarReduceRequired, "many を scalar 引数へ結ぶには明示の reduce 規則が必要。")
	}

	for _, element := range in.Elements {
		if strings.TrimSpace(element.State) == ElementInputUnavailable {
			return reject(CodeInputUnavailable, "集合の一部要素の入力が未取得。")
		}
	}
	if !in.Complete {
		return reject(CodeIncompleteSet, "集合が完全取得されていない。部分和を全受注額として返さない。")
	}
	if strings.TrimSpace(in.Shape) == ShapeOne {
		// one is exactly one element at this point; no fan-out concept applies.
		in.JoinFanOut = 0
	}

	sum, hasNull, err := sumPresent(in.Elements)
	if err != nil {
		return reject(CodeInvalidRequest, err.Error())
	}

	if len(in.Elements) == 0 {
		switch strings.TrimSpace(in.EmptySetPolicy) {
		case EmptySetZero:
			return Report{Code: CodeEmptySetZero, Accepted: true, Result: "0"}
		case EmptySetNull:
			return Report{Code: CodeEmptySetNull, Accepted: true, ResultNull: true}
		default:
			return reject(CodeEmptySetPolicyUndeclared, "空集合の SUM 方針が宣言されていない。勝手に0へ確定しない。")
		}
	}

	if hasNull && strings.TrimSpace(in.NullPolicy) == "" {
		return reject(CodeNullPolicyUndeclared, "NULL要素の処理方針が宣言されていない。空集合や未取得と混同しない。")
	}
	if hasNull && strings.TrimSpace(in.NullPolicy) == NullPropagate {
		return Report{Code: CodeNullPropagated, Accepted: true, ResultNull: true}
	}

	fanOut := in.JoinFanOut
	if fanOut < 1 {
		fanOut = 1
	}
	if fanOut > 1 {
		naive := new(big.Rat).Mul(sum, new(big.Rat).SetInt64(int64(fanOut)))
		return Report{
			Code:         CodeJoinFanOutDetected,
			Accepted:     true,
			Result:       formatExact(sum),
			NaiveJoinSum: formatExact(naive),
			Message:      "JOIN による fan-out を検出した。業務上の合計は distinct 要素の和であり、naive な JOIN 和は二重計上である。",
		}
	}

	code := CodeOK
	if hasNull {
		code = CodeNullIgnored
	}
	return Report{Code: code, Accepted: true, Result: formatExact(sum)}
}

// sumPresent adds the present amounts and reports whether a NULL element exists.
func sumPresent(elements []Element) (*big.Rat, bool, error) {
	total := new(big.Rat)
	hasNull := false
	for _, element := range elements {
		switch strings.TrimSpace(element.State) {
		case ElementPresent:
			value, ok := new(big.Rat).SetString(strings.TrimSpace(element.Amount))
			if !ok {
				return nil, false, fmt.Errorf("invalid amount %q for element %q", element.Amount, element.ID)
			}
			total.Add(total, value)
		case ElementNull:
			hasNull = true
		case ElementInputUnavailable:
			// handled before this loop
		default:
			return nil, false, fmt.Errorf("invalid element state %q", element.State)
		}
	}
	return total, hasNull, nil
}

// formatExact writes r exactly when its denominator is a product of 2s and 5s.
func formatExact(r *big.Rat) string {
	scale, ok := exactScale(r)
	if !ok {
		return ""
	}
	return r.FloatString(scale)
}

func exactScale(r *big.Rat) (int, bool) {
	den := new(big.Int).Set(r.Denom())
	count := func(factor *big.Int) int {
		n := 0
		for {
			q, rem := new(big.Int).QuoRem(den, factor, new(big.Int))
			if rem.Sign() != 0 {
				break
			}
			den = q
			n++
		}
		return n
	}
	twos := count(big.NewInt(2))
	fives := count(big.NewInt(5))
	if den.Cmp(big.NewInt(1)) != 0 {
		return 0, false
	}
	if twos > fives {
		return twos, true
	}
	return fives, true
}
