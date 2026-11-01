// Package derivationcache is a read-only, synthetic reference for Data
// Derivation D3 (#1641). It checks the smallest freshness contract for a
// derived value: when no cache is requested a simple pure computation needs no
// freshness declaration, and when a cache is requested its reuse is decided by
// the rule version, the input version, the evaluation context class and the
// information source time point.
//
// It is not a distributed cache product, a history store, an authorization
// engine, a persistence executor, or an accepted API. It does not write a
// Source. It never returns an old value as a current one, never overwrites a
// business-confirmed saved fact through a cache path or a re-render, and never
// claims a cross-source snapshot alignment that is not provided.
package derivationcache

import "fmt"

// Result codes.
const (
	CodeDirectEvaluation          = "DIRECT_EVALUATION"
	CodeCacheReused               = "CACHE_REUSED"
	CodeConfirmedSavedImmutable   = "CONFIRMED_SAVED_IMMUTABLE"
	CodeStaleInput                = "STALE_INPUT"
	CodeStaleRule                 = "STALE_RULE"
	CodeStaleSourceTime           = "STALE_SOURCE_TIME"
	CodeSourceTimeUnspecified     = "SOURCE_TIME_UNSPECIFIED"
	CodeContextMismatch           = "CONTEXT_MISMATCH"
	CodeSnapshotIntegrityUnproven = "SNAPSHOT_INTEGRITY_UNPROVEN"
	CodeInvalidRequest            = "INVALID_REQUEST"
)

// Entry is the provenance of one cached precomputed result.
type Entry struct {
	RuleVersion     string `json:"ruleVersion"`
	InputVersion    string `json:"inputVersion"`
	ContextClass    string `json:"contextClass"`
	SourceTimePoint string `json:"sourceTimePoint"`
	// EvaluatedAt is when the value was computed. It is informational only: a
	// different evaluation wall-clock time does not make a cache stale, and the
	// same wall-clock time does not prove the source did not move.
	EvaluatedAt string `json:"evaluatedAt,omitempty"`
	// Value is the cached value, never overwriting a confirmed business fact.
	Value string `json:"value,omitempty"`
}

// Context is the current evaluation context.
type Context struct {
	RuleVersion     string `json:"ruleVersion"`
	InputVersion    string `json:"inputVersion"`
	ContextClass    string `json:"contextClass"`
	SourceTimePoint string `json:"sourceTimePoint"`
	// EvaluatedAt is the current evaluation time. It is kept apart from the
	// source time point and is never used as a reuse criterion.
	EvaluatedAt string `json:"evaluatedAt,omitempty"`
}

// Input is the whole synthetic request.
type Input struct {
	// CacheRequested is false for a simple pure computation; no cache contract is
	// then required.
	CacheRequested bool `json:"cacheRequested"`
	// Entry is the cached result under test.
	Entry *Entry `json:"entry,omitempty"`
	// Current is the evaluation context under test.
	Current *Context `json:"current,omitempty"`
	// ConfirmedSaved is true when the value under consideration is a
	// business-confirmed saved fact, not a cache.
	ConfirmedSaved bool `json:"confirmedSaved"`
	// ConfirmedValue is the saved business fact.
	ConfirmedValue string `json:"confirmedValue,omitempty"`
	// NewComputation is the value a fresh evaluation would produce. It is
	// reported only as an unconfirmed computation.
	NewComputation string `json:"newComputation,omitempty"`
	// MultiSource is true when several information sources feed the evaluation.
	MultiSource bool `json:"multiSource"`
	// SnapshotsAligned is true only when the sources share one snapshot.
	SnapshotsAligned bool `json:"snapshotsAligned"`
}

// Report is the read-only result.
type Report struct {
	Code      string `json:"code"`
	Accepted  bool   `json:"accepted"`
	Value     string `json:"value,omitempty"`
	ValueNull bool   `json:"valueNull,omitempty"`
	Message   string `json:"message,omitempty"`
}

func reject(code, message string) Report {
	return Report{Code: code, Message: message}
}

// Reuse checks one cache reuse decision. It never writes anything.
func Reuse(in Input) Report {
	if !in.CacheRequested {
		return Report{Code: CodeDirectEvaluation, Accepted: true, Value: in.NewComputation,
			Message: "単純な純粋計算に鮮度・履歴・保存宣言を必須にしない。"}
	}
	if in.Entry == nil || in.Current == nil {
		return reject(CodeInvalidRequest, "キャッシュ判定には Entry と Current の双方が必要。")
	}

	if in.ConfirmedSaved {
		return Report{Code: CodeConfirmedSavedImmutable, Accepted: true, Value: in.ConfirmedValue,
			Message: fmt.Sprintf("確定保存値 %s はマスターや規則の変更で上書きしない。新規の未確定計算は %s。", in.ConfirmedValue, in.NewComputation)}
	}

	if in.MultiSource && !in.SnapshotsAligned {
		return reject(CodeSnapshotIntegrityUnproven, "複数情報源の時点を一致させられない。強い整合性を偽装せず、#1708 へ委譲する。")
	}

	if in.Entry.RuleVersion != in.Current.RuleVersion {
		return Report{Code: CodeStaleRule, ValueNull: true,
			Message: fmt.Sprintf("規則版 %s に対するキャッシュが版 %s。根拠なく旧値を流用しない。", in.Current.RuleVersion, in.Entry.RuleVersion)}
	}
	if in.Entry.InputVersion != in.Current.InputVersion {
		return Report{Code: CodeStaleInput, ValueNull: true,
			Message: fmt.Sprintf("入力版 %s に対するキャッシュが版 %s。旧値を現在値として返さない。", in.Current.InputVersion, in.Entry.InputVersion)}
	}
	if in.Entry.ContextClass != in.Current.ContextClass {
		return Report{Code: CodeContextMismatch, ValueNull: true,
			Message: "権限・tenant・情報源時点が異なる結果を同一キャッシュとして共有しない。"}
	}
	// The source time point is part of the cache identity. An empty current
	// source time point cannot prove that the source did not move, so it is
	// unproven rather than reused.
	if in.Current.SourceTimePoint == "" {
		return Report{Code: CodeSourceTimeUnspecified, ValueNull: true,
			Message: "現在の情報源時点が未指定。時点を確かめずに旧値を再利用しない。"}
	}
	if in.Entry.SourceTimePoint != in.Current.SourceTimePoint {
		return Report{Code: CodeStaleSourceTime, ValueNull: true,
			Message: fmt.Sprintf("情報源時点が %s から %s へ変わった。評価時刻が同じでも旧値を現在値として返さない。", in.Entry.SourceTimePoint, in.Current.SourceTimePoint)}
	}
	return Report{Code: CodeCacheReused, Accepted: true, Value: in.Entry.Value,
		Message: "規則版・入力版・評価文脈・情報源時点が一致した再利用。"}
}
