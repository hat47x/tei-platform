package derivationcache

// CycleInput is the cache invalidation input for A→B→A history changes.
type CycleInput struct {
	CacheRequested bool
	// CachedSource is the source fingerprint captured at cache time.
	CachedSource string
	// CurrentSource is the current source fingerprint.
	CurrentSource string
	// SourceHistory is the ordered list of source fingerprints observed
	// between cache time and now. An A→B→A sequence appears as
	// [A, B, A] where A == CurrentSource.
	SourceHistory []string
}

// CycleReport is the read-only result for A→B→A cycle detection.
type CycleReport struct {
	Code      string
	Accepted  bool
	ValueNull bool
	Value     string
	Message   string
	ABA       bool
}

func rejectCycle(code, message string) CycleReport {
	return CycleReport{Code: code, ValueNull: true, Message: message}
}

// ReuseCycle checks whether a cached value is still valid when the source
// has undergone an A→B→A cycle. Fingerprint equality alone is not enough:
// an intermediate change means the cache provenance is not equivalent to a
// fresh evaluation.
func ReuseCycle(in CycleInput) CycleReport {
	if !in.CacheRequested {
		return CycleReport{Code: CodeDirectEvaluation, Accepted: true, Value: "",
			Message: "単純な純粋計算に履歴の鮮度を必須にしない。"}
	}
	if in.CachedSource != in.CurrentSource {
		return rejectCycle(CodeStaleSourceTime,
			"現在の情報源時点がキャッシュと異なる。評価時刻が同じでも旧値を現在値として返さない。")
	}
	if len(in.SourceHistory) == 0 {
		// No history provided; treat as cache reuse without cycle evidence.
		return CycleReport{Code: CodeCacheReused, Accepted: true, Value: "",
			Message: "履歴が未提供のため、現在のfingerprint一致を再利用として受理。"}
	}
	// Detect A→B→A or longer cycles. If the current fingerprint matches the
	// cached fingerprint but the history contains any different intermediate
	// fingerprint, an intermediate change occurred.
	for _, fp := range in.SourceHistory {
		if fp != in.CurrentSource {
			return CycleReport{
				Code:      CodeSnapshotIntegrityUnproven,
				Accepted:  false,
				ValueNull: true,
				ABA:       true,
				Message:   "A→B→Aのような中間変化が観測された。fingerprint一致では古い適用資格を復活させない。",
			}
		}
	}
	return CycleReport{Code: CodeCacheReused, Accepted: true, Value: "",
		Message: "履歴に中間変化がなく、現在のfingerprintが一致した再利用。"}
}
