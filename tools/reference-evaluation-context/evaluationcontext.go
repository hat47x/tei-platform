// Package evaluationcontext is a read-only, synthetic reference for Data
// Derivation D12 (#1720). It separates a rule's typed business inputs from
// declared evaluation arguments and host-supplied trusted context, and checks
// that a time-dependent rule is evaluated from an explicit, fixed evaluation
// value rather than a hidden clock.
//
// It is not a trusted Invocation Context provider, an authority engine, a
// calendar product, or an accepted API. It does not write a Source. It never
// falls back to the host OS clock, never picks a timezone silently, never
// reads a clock per row, and never certifies an implementation that hides I/O
// inside a pure rule.
package evaluationcontext

import (
	"fmt"
	"math/big"
	"strings"
	"time"
) // Result codes.
const (
	CodeDays                  = "OK_DAYS"
	CodeAmount                = "OK_AMOUNT"
	CodeEvaluationDateMissing = "EVALUATION_DATE_MISSING"
	CodeTimezoneUndeclared    = "TIMEZONE_UNDECLARED"
	CodeRowClockReference     = "NONDETERMINISTIC_REEVALUATION"
	CodeHiddenIO              = "HIDDEN_IO_IN_RULE"
	CodeContextUnproven       = "CONTEXT_UNPROVEN"
	CodeUnsupportedRule       = "UNSUPPORTED_RULE"
	CodeInvalidRequest        = "INVALID_REQUEST"
)

// Rule kinds.
const (
	KindElapsedDays    = "elapsed-days"
	KindQuantityAmount = "quantity-amount"
)

// Input is the whole synthetic request.
type Input struct {
	// Kind selects the rule under evaluation.
	Kind string `json:"kind"`
	// RuleVersion is the pinned rule version the result is reproducible
	// against. Changing it is a different explanation, not a silent rerun.
	RuleVersion string `json:"ruleVersion,omitempty"`
	// StartDate is the typed business input of elapsed-days.
	StartDate string `json:"startDate,omitempty"`
	// EvaluationDate is the declared evaluation argument (as-of date). It is a
	// declared value, not a host clock read.
	EvaluationDate         string `json:"evaluationDate,omitempty"`
	EvaluationDateProvided bool   `json:"evaluationDateProvided"`
	// TimezoneNeeded is true when the rule's meaning depends on a business
	// calendar or a counterparty timezone.
	TimezoneNeeded bool `json:"timezoneNeeded"`
	// Timezone is the declared timezone for the evaluation argument.
	Timezone string `json:"timezone,omitempty"`
	// Quantity and UnitPrice are the typed business inputs of quantity-amount.
	Quantity  string `json:"quantity,omitempty"`
	UnitPrice string `json:"unitPrice,omitempty"`
	// RowClockReference is true when the realization reads a clock per row.
	RowClockReference bool `json:"rowClockReference"`
	// ExternalFetch is true when the realization performs hidden network I/O.
	ExternalFetch bool `json:"externalFetch"`
	// ProviderImpliedClock and ProviderImpliedLocale are true when the target or
	// provider implies a clock or locale that the contract did not declare.
	ProviderImpliedClock  bool `json:"providerImpliedClock,omitempty"`
	ProviderImpliedLocale bool `json:"providerImpliedLocale,omitempty"`
}

// Report is the read-only result.
type Report struct {
	Code        string   `json:"code"`
	Accepted    bool     `json:"accepted"`
	Value       string   `json:"value,omitempty"`
	RuleVersion string   `json:"ruleVersion,omitempty"`
	Unknown     []string `json:"unknown,omitempty"`
	Message     string   `json:"message,omitempty"`
}

func reject(code, message string) Report {
	return Report{Code: code, Message: message}
}

func parseDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(value))
}

// Evaluate checks one evaluation-context contract. It never reads a clock and
// never writes anything.
func Evaluate(in Input) Report {
	if in.RowClockReference {
		return reject(CodeRowClockReference,
			"一行ごとの隠れた時計参照を禁止する。評価開始時に確定した基準時刻から計算する。")
	}
	if in.ExternalFetch {
		return reject(CodeHiddenIO,
			"純粋なRule本文へ隠れたI/Oを混ぜない。事前取得した型付き入力か既存Query/Capabilityを明示する。AI生成実装を純粋と無条件認定しない。")
	}
	if in.ProviderImpliedClock || in.ProviderImpliedLocale {
		unknown := []string{}
		if in.ProviderImpliedClock {
			unknown = append(unknown, "providerImpliedClock")
		}
		if in.ProviderImpliedLocale {
			unknown = append(unknown, "providerImpliedLocale")
		}
		return Report{
			Code: CodeContextUnproven, RuleVersion: in.RuleVersion, Unknown: unknown,
			Message: "Provider/Target固有の暗黙Clock/Localeは適合未証明。宣言された評価文脈なしに同等としない（#1694へ接続）。",
		}
	}

	switch strings.TrimSpace(in.Kind) {
	case KindQuantityAmount:
		// A simple rule needs no clock or locale declaration.
		quantity, ok := new(big.Rat).SetString(strings.TrimSpace(in.Quantity))
		if !ok {
			return reject(CodeInvalidRequest, fmt.Sprintf("invalid quantity %q", in.Quantity))
		}
		unitPrice, ok := new(big.Rat).SetString(strings.TrimSpace(in.UnitPrice))
		if !ok {
			return reject(CodeInvalidRequest, fmt.Sprintf("invalid unit price %q", in.UnitPrice))
		}
		return Report{Code: CodeAmount, Accepted: true, RuleVersion: in.RuleVersion,
			Value:   formatExact(quantity.Mul(quantity, unitPrice)),
			Message: "時計・Locale宣言なしで評価する。"}

	case KindElapsedDays:
		if !in.EvaluationDateProvided {
			return reject(CodeEvaluationDateMissing,
				"評価日が未提供。fail-closed とし、検証側のOS時刻へfallbackしない。")
		}
		if in.TimezoneNeeded && strings.TrimSpace(in.Timezone) == "" {
			return reject(CodeTimezoneUndeclared,
				"業務暦・タイムゾーンが結果に影響する。timezone未指定で一方を勝手に採用しない。")
		}
		start, err := parseDate(in.StartDate)
		if err != nil {
			return reject(CodeInvalidRequest, fmt.Sprintf("invalid startDate %q", in.StartDate))
		}
		evaluation, err := parseDate(in.EvaluationDate)
		if err != nil {
			return reject(CodeInvalidRequest, fmt.Sprintf("invalid evaluationDate %q", in.EvaluationDate))
		}
		days := int(evaluation.Sub(start).Hours() / 24)
		return Report{Code: CodeDays, Accepted: true, RuleVersion: in.RuleVersion, Value: fmt.Sprintf("%d", days),
			Message: "宣言された評価引数から計算する。同じ固定評価日は再評価しても変わらない。"}

	default:
		return reject(CodeUnsupportedRule, fmt.Sprintf("unsupported rule kind %q", in.Kind))
	}
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
