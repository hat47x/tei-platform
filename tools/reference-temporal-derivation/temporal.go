// Package temporalderivation is a read-only, synthetic reference for Data
// Derivation D14 (#1727). It separates the business valid time, the recorded
// time, the rule version, the evaluation time, and a business-confirmed saved
// fact, and labels a recalculation as historical replay, what-if, or current
// computation.
//
// It is not a bitemporal database, a history store, a rule registry, an
// authorization engine, or an accepted API. It does not write a Source. It never
// invents a historical price, never calls a current-rule result a historical
// replay, and never updates a confirmed saved fact with a recalculation.
package temporalderivation

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// Result kinds.
const (
	KindHistoricalReplay   = "historical-replay"
	KindWhatIf             = "what-if"
	KindCurrentComputation = "current-computation"
	KindConfirmedFactRead  = "confirmed-fact-read"
)

// Report codes.
const (
	CodeHistoricalReplay       = "HISTORICAL_REPLAY"
	CodeWhatIf                 = "WHAT_IF"
	CodeCurrentComputation     = "CURRENT_COMPUTATION"
	CodeConfirmedFactRead      = "CONFIRMED_FACT_READ"
	CodeUnknownHistoricalPrice = "UNKNOWN_HISTORICAL_PRICE"
	CodeUnknownRuleVersion     = "UNKNOWN_RULE_VERSION"
	CodeAuthorizationDenied    = "AUTHORIZATION_DENIED"
	CodeInvalidRequest         = "INVALID_REQUEST"
)

// PricePoint is one valid-time interval's price. ValidFrom is YYYY-MM-DD.
type PricePoint struct {
	ValidFrom string `json:"validFrom"`
	Price     string `json:"price"`
}

// RuleVersion is one accepted rule version. Multiplier is applied to
// quantity × price; version 1 uses 1.
type RuleVersion struct {
	Version    string `json:"version"`
	Multiplier string `json:"multiplier"`
}

// ConfirmedSaved is a business-confirmed saved fact. It is read, never
// recalculated.
type ConfirmedSaved struct {
	Value       string `json:"value"`
	ValidTime   string `json:"validTime,omitempty"`
	RuleVersion string `json:"ruleVersion,omitempty"`
}

// Input is the whole synthetic request.
type Input struct {
	Kind string `json:"kind"`
	// Quantity is the order quantity.
	Quantity string `json:"quantity"`
	// AsOf is the business valid time the caller asks about (YYYY-MM-DD).
	AsOf string `json:"asOf,omitempty"`
	// RuleVersion is the rule version the caller asks for.
	RuleVersion string `json:"ruleVersion,omitempty"`
	// PriceHistory is the provider's valid-time price history.
	PriceHistory []PricePoint `json:"priceHistory,omitempty"`
	// HistoricalPricesAvailable is true only when the provider can serve the
	// historical price. A false value with a current price must not be turned
	// into a historical estimate.
	HistoricalPricesAvailable bool `json:"historicalPricesAvailable"`
	// Rules are the accepted rule versions the caller can use.
	Rules []RuleVersion `json:"rules"`
	// ConfirmedSaved is the saved fact, when one exists.
	ConfirmedSaved *ConfirmedSaved `json:"confirmedSaved,omitempty"`
	// Authorization is "allowed" or "denied".
	Authorization string `json:"authorization"`
	// EvaluationAt is when the recalculation runs. It is never the valid time.
	EvaluationAt string `json:"evaluationAt,omitempty"`
}

// Report is the read-only result.
type Report struct {
	Kind               string `json:"kind"`
	Code               string `json:"code"`
	Value              string `json:"value,omitempty"`
	PriceUsed          string `json:"priceUsed,omitempty"`
	PriceValidFrom     string `json:"priceValidFrom,omitempty"`
	RuleVersion        string `json:"ruleVersion,omitempty"`
	Reproductible      bool   `json:"reproductible"`
	IsHistoricalReplay bool   `json:"isHistoricalReplay"`
	UpdatesSavedFact   bool   `json:"updatesSavedFact"`
	ActorTime          string `json:"evaluationAt,omitempty"`
	Message            string `json:"message,omitempty"`
}

func reject(kind, code, message string) Report {
	return Report{Kind: kind, Code: code, Message: message}
}

// Evaluate labels and, when the evidence allows it, computes one requested
// result. It never writes anything.
func Evaluate(in Input) Report {
	kind := strings.TrimSpace(in.Kind)
	if strings.TrimSpace(in.Authorization) == "denied" {
		return reject(kind, CodeAuthorizationDenied, "認可されていないため、値も履歴の存在も開示しない。")
	}
	switch kind {
	case KindConfirmedFactRead:
		return evaluateConfirmed(kind, in)
	case KindHistoricalReplay:
		return evaluateHistorical(kind, in)
	case KindWhatIf:
		return evaluateWhatIf(kind, in)
	case KindCurrentComputation:
		return evaluateCurrent(kind, in)
	default:
		return reject(kind, CodeInvalidRequest, fmt.Sprintf("未知の結果種別 %q。", in.Kind))
	}
}

func evaluateConfirmed(kind string, in Input) Report {
	if in.ConfirmedSaved == nil {
		return reject(kind, CodeInvalidRequest, "確定保存値の参照には保存済み事実が必要。")
	}
	return Report{
		Kind: kind, Code: CodeConfirmedFactRead,
		Value:       in.ConfirmedSaved.Value,
		RuleVersion: in.ConfirmedSaved.RuleVersion,
		ActorTime:   in.EvaluationAt,
		Message:     "保存済みの業務確定事実をそのまま返す。導出の再実行ではない。",
	}
}

func evaluateHistorical(kind string, in Input) Report {
	if strings.TrimSpace(in.AsOf) == "" {
		return reject(kind, CodeInvalidRequest, "過去再計算には業務有効時点が必要。")
	}
	price, point, ok := historicalPrice(in.PriceHistory, in.AsOf)
	if !in.HistoricalPricesAvailable || !ok {
		return reject(kind, CodeUnknownHistoricalPrice,
			fmt.Sprintf("%s の当時価格をProviderが提供していない。現行価格から推定しない。", in.AsOf))
	}
	rule, ok := findRule(in.Rules, in.RuleVersion)
	if !ok {
		return reject(kind, CodeUnknownRuleVersion,
			fmt.Sprintf("当時の規則版 %q が取得できない。別版の値で再現を装わない。", in.RuleVersion))
	}
	value, err := compute(in.Quantity, price.Price, rule.Multiplier)
	if err != nil {
		return reject(kind, CodeInvalidRequest, err.Error())
	}
	return Report{
		Kind: kind, Code: CodeHistoricalReplay,
		Value: value, PriceUsed: price.Price, PriceValidFrom: point.ValidFrom,
		RuleVersion: rule.Version, Reproductible: true, IsHistoricalReplay: true,
		ActorTime: in.EvaluationAt,
		Message:   "当時の価格と当時の規則版で再評価した。保存済みの確定事実は変更しない。",
	}
}

func evaluateWhatIf(kind string, in Input) Report {
	rule, ok := findRule(in.Rules, in.RuleVersion)
	if !ok {
		return reject(kind, CodeUnknownRuleVersion, fmt.Sprintf("規則版 %q が取得できない。", in.RuleVersion))
	}
	price, point, ok := historicalPrice(in.PriceHistory, in.AsOf)
	if !ok {
		price, point, ok = currentPrice(in.PriceHistory)
		if !ok {
			return reject(kind, CodeUnknownHistoricalPrice, "利用できる価格がない。")
		}
	}
	value, err := compute(in.Quantity, price.Price, rule.Multiplier)
	if err != nil {
		return reject(kind, CodeInvalidRequest, err.Error())
	}
	return Report{
		Kind: kind, Code: CodeWhatIf,
		Value: value, PriceUsed: price.Price, PriceValidFrom: point.ValidFrom,
		RuleVersion: rule.Version, Reproductible: false, IsHistoricalReplay: false,
		ActorTime: in.EvaluationAt,
		Message:   "意図的に異なる規則版を当てた試算。当時の確定結果でも再現でもない。",
	}
}

func evaluateCurrent(kind string, in Input) Report {
	rule, ok := findRule(in.Rules, in.RuleVersion)
	if !ok {
		return reject(kind, CodeUnknownRuleVersion, fmt.Sprintf("規則版 %q が取得できない。", in.RuleVersion))
	}
	price, point, ok := currentPrice(in.PriceHistory)
	if !ok {
		return reject(kind, CodeUnknownHistoricalPrice, "利用できる価格がない。")
	}
	value, err := compute(in.Quantity, price.Price, rule.Multiplier)
	if err != nil {
		return reject(kind, CodeInvalidRequest, err.Error())
	}
	return Report{
		Kind: kind, Code: CodeCurrentComputation,
		Value: value, PriceUsed: price.Price, PriceValidFrom: point.ValidFrom,
		RuleVersion: rule.Version, Reproductible: false, IsHistoricalReplay: false,
		ActorTime: in.EvaluationAt,
		Message:   "現在取得できる入力と指定した規則版による計算。過去の確定金額とは無関係。",
	}
}

// historicalPrice returns the latest price whose valid-from is on or before
// asOf. It never extrapolates before the first known valid-from.
func historicalPrice(history []PricePoint, asOf string) (PricePoint, PricePoint, bool) {
	sorted := sortedHistory(history)
	var chosen PricePoint
	found := false
	for _, point := range sorted {
		if point.ValidFrom <= asOf {
			chosen = point
			found = true
		}
	}
	return chosen, chosen, found
}

// currentPrice returns the latest known price point.
func currentPrice(history []PricePoint) (PricePoint, PricePoint, bool) {
	sorted := sortedHistory(history)
	if len(sorted) == 0 {
		return PricePoint{}, PricePoint{}, false
	}
	last := sorted[len(sorted)-1]
	return last, last, true
}

func sortedHistory(history []PricePoint) []PricePoint {
	sorted := make([]PricePoint, len(history))
	copy(sorted, history)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ValidFrom < sorted[j].ValidFrom })
	return sorted
}

func findRule(rules []RuleVersion, version string) (RuleVersion, bool) {
	for _, rule := range rules {
		if rule.Version == version {
			return rule, true
		}
	}
	return RuleVersion{}, false
}

func compute(quantity, price, multiplier string) (string, error) {
	q, ok := new(big.Rat).SetString(strings.TrimSpace(quantity))
	if !ok {
		return "", fmt.Errorf("invalid quantity %q", quantity)
	}
	p, ok := new(big.Rat).SetString(strings.TrimSpace(price))
	if !ok {
		return "", fmt.Errorf("invalid price %q", price)
	}
	m := new(big.Rat).SetInt64(1)
	if strings.TrimSpace(multiplier) != "" {
		parsed, ok := new(big.Rat).SetString(strings.TrimSpace(multiplier))
		if !ok {
			return "", fmt.Errorf("invalid multiplier %q", multiplier)
		}
		m = parsed
	}
	value := new(big.Rat).Mul(q, p)
	value.Mul(value, m)
	return formatRat(value), nil
}

// formatRat writes an exact integer when possible, otherwise a decimal with at
// most six fractional digits. It does not introduce a rounding policy.
func formatRat(r *big.Rat) string {
	if r.IsInt() {
		return r.Num().String()
	}
	return strings.TrimRight(strings.TrimRight(r.FloatString(6), "0"), ".")
}
