// Package readconsistency is a read-only, synthetic reference for Data
// Derivation D11 (#1708). It checks the read-consistency guarantee of the input
// set of one derivation evaluation: whether the required level is guaranteed by
// the provider, whether several inputs belong to one snapshot, and how a
// non-atomic read is reported.
//
// It is not a distributed transaction manager, a global snapshot API, a generic
// cache, an authorization decision, or an accepted API. It does not write a
// Source. It never mixes two revisions into a value and presents it as
// confirmed, never treats a lost snapshot token as NULL or zero, and never
// claims a strong cross-provider guarantee that no provider provides.
package readconsistency

import (
	"fmt"
	"math/big"
	"strings"
)

// Result codes.
const (
	CodeOK                     = "RESULT_OK"
	CodeNonAtomicRead          = "NON_ATOMIC_READ"
	CodeMixedSnapshot          = "MIXED_SNAPSHOT"
	CodeUnsupported            = "UNSUPPORTED_CONSISTENCY"
	CodeBestEffortNotAccepted  = "BEST_EFFORT_NOT_ACCEPTED"
	CodeInputUnavailable       = "INPUT_UNAVAILABLE"
	CodeInputMissing           = "INPUT_MISSING"
	CodeAbsentRelation         = "ABSENT_RELATION"
	CodeNullInput              = "NULL_INPUT"
	CodeRetryRequiresReauth    = "RETRY_REQUIRES_REAUTHORIZATION"
	CodeAuthorizationDelegated = "AUTHORIZATION_DELEGATED"
	CodeInvalidRequest         = "INVALID_REQUEST"
)

// Consistency levels.
const (
	LevelBestEffort              = "best-effort"
	LevelSameRecord              = "same-record"
	LevelVersionAligned          = "version-aligned"
	LevelSameTransactionSnapshot = "same-transaction-snapshot"
	GuaranteeNone                = "none"
)

// Operations.
const (
	OpSubtract = "subtract"
	OpSum      = "sum"
)

// Read states.
const (
	ReadPresent        = "present"
	ReadMissing        = "missing"
	ReadAbsentRelation = "absent-relation"
	ReadNull           = "null"
)

// Read is one input read of the evaluation.
type Read struct {
	InputID  string `json:"inputId"`
	Revision string `json:"revision"`
	State    string `json:"state"`
	Value    string `json:"value,omitempty"`
}

// Input is the whole synthetic request.
type Input struct {
	// RequiredLevel is the consistency the contract requires.
	RequiredLevel string `json:"requiredLevel"`
	// ProviderGuarantee is what the provider actually guarantees.
	ProviderGuarantee string `json:"providerGuarantee"`
	// MultiProvider is true when the inputs come from several providers.
	MultiProvider bool `json:"multiProvider"`
	// AcceptBestEffort is true only when the case explicitly accepts a non-atomic
	// read.
	AcceptBestEffort bool `json:"acceptBestEffort"`
	// SnapshotTokenLost is true when the provider lost the snapshot mid-read.
	SnapshotTokenLost bool `json:"snapshotTokenLost"`
	// InputUnavailable is true when an input could not be read.
	InputUnavailable bool `json:"inputUnavailable"`
	// AuthorizationDenied is true when authorization was refused.
	AuthorizationDenied bool `json:"authorizationDenied"`
	// Retry, AuthorizationVersionChanged and RetryReauthorized describe a retry.
	Retry                       bool `json:"retry"`
	AuthorizationVersionChanged bool `json:"authorizationVersionChanged"`
	RetryReauthorized           bool `json:"retryReauthorized"`
	// Operation combines the reads.
	Operation string `json:"operation"`
	Reads     []Read `json:"reads"`
}

// Report is the read-only result.
type Report struct {
	Code     string `json:"code"`
	Accepted bool   `json:"accepted"`
	Result   string `json:"result,omitempty"`
	Mixed    string `json:"mixedValue,omitempty"`
	Message  string `json:"message,omitempty"`
}

func reject(code, message string) Report {
	return Report{Code: code, Message: message}
}

func levelRank(level string) (int, bool) {
	switch level {
	case LevelBestEffort:
		return 1, true
	case LevelSameRecord:
		return 2, true
	case LevelVersionAligned:
		return 3, true
	case LevelSameTransactionSnapshot:
		return 4, true
	default:
		return 0, false
	}
}

func guaranteeRank(guarantee string) (int, bool) {
	switch guarantee {
	case GuaranteeNone:
		return 0, true
	case LevelSameRecord:
		return 2, true
	case LevelVersionAligned:
		return 3, true
	case LevelSameTransactionSnapshot:
		return 4, true
	default:
		return 0, false
	}
}

// Evaluate checks one evaluation's input set. It never writes anything.
func Evaluate(in Input) Report {
	required, ok := levelRank(in.RequiredLevel)
	if !ok {
		return reject(CodeInvalidRequest, fmt.Sprintf("unknown required level %q", in.RequiredLevel))
	}
	if in.ProviderGuarantee == "" {
		in.ProviderGuarantee = GuaranteeNone
	}
	if _, ok := guaranteeRank(in.ProviderGuarantee); !ok {
		return reject(CodeInvalidRequest, fmt.Sprintf("unknown provider guarantee %q", in.ProviderGuarantee))
	}
	if in.Operation != OpSubtract && in.Operation != OpSum {
		return reject(CodeInvalidRequest, fmt.Sprintf("unsupported operation %q", in.Operation))
	}

	if in.AuthorizationDenied {
		return reject(CodeAuthorizationDelegated, "行・テナント・項目の認可は #1693 へ委譲する。")
	}
	if in.Retry && in.AuthorizationVersionChanged && !in.RetryReauthorized {
		return reject(CodeRetryRequiresReauth, "再試行で認可版が変化した。前の読取り文脈を転用せず再認可する。")
	}
	if in.InputUnavailable || in.SnapshotTokenLost {
		return reject(CodeInputUnavailable, "入力が未取得、またはsnapshotが失効した。NULLや0として扱わない。")
	}
	for _, read := range in.Reads {
		if read.State == ReadMissing {
			return reject(CodeInputMissing, fmt.Sprintf("入力 %q が不在。", read.InputID))
		}
		if read.State == ReadAbsentRelation {
			return reject(CodeAbsentRelation, fmt.Sprintf("入力 %q の参照先に一致する行が無い。", read.InputID))
		}
		if read.State == ReadNull {
			return reject(CodeNullInput, fmt.Sprintf("入力 %q がNULL。0として扱わない。", read.InputID))
		}
	}
	if len(in.Reads) < 2 {
		return reject(CodeInputMissing, "評価に必要な入力集合が不足している。")
	}

	// Guarantee check. best-effort needs no provider guarantee but does need an
	// explicit case acceptance.
	guarantee, _ := guaranteeRank(in.ProviderGuarantee)
	if in.RequiredLevel == LevelBestEffort {
		if !in.AcceptBestEffort {
			return reject(CodeBestEffortNotAccepted, "best-effort は案件の明示受容がある場合に限定する。")
		}
	} else {
		if in.MultiProvider && in.RequiredLevel == LevelSameTransactionSnapshot {
			return reject(CodeUnsupported, "異なるProvider間の原子的snapshotは保証できない。根拠なしに成立扱いにしない。")
		}
		if required > guarantee {
			return reject(CodeUnsupported,
				fmt.Sprintf("要求 %q はProvider保証 %q に無い。", in.RequiredLevel, in.ProviderGuarantee))
		}
	}

	revision := in.Reads[0].Revision
	consistent := true
	for _, read := range in.Reads {
		if read.Revision != revision {
			consistent = false
			break
		}
	}

	value, err := combine(in.Reads, in.Operation)
	if err != nil {
		return reject(CodeInvalidRequest, err.Error())
	}

	if !consistent {
		// A strong level must not return a value that belongs to no revision.
		if in.RequiredLevel == LevelVersionAligned || in.RequiredLevel == LevelSameTransactionSnapshot {
			return Report{Code: CodeMixedSnapshot, Mixed: value,
				Message: "複数Revisionを混ぜた値。確定値として返さず再読取りする。"}
		}
		if !in.AcceptBestEffort {
			return Report{Code: CodeMixedSnapshot, Mixed: value,
				Message: "非原子の読取り。案件の明示受容がないため確定値として返さない。"}
		}
		return Report{Code: CodeNonAtomicRead, Accepted: true, Result: value,
			Message: "非原子の読取り。取得範囲と非原子的状態を示し、強い整合を偽装しない。"}
	}

	return Report{Code: CodeOK, Accepted: true, Result: value}
}

// combine folds the reads with the operation.
func combine(reads []Read, operation string) (string, error) {
	total := new(big.Rat)
	for i, read := range reads {
		value, ok := new(big.Rat).SetString(strings.TrimSpace(read.Value))
		if !ok {
			return "", fmt.Errorf("invalid value %q for input %q", read.Value, read.InputID)
		}
		if i == 0 {
			total.Set(value)
			continue
		}
		switch operation {
		case OpSum:
			total.Add(total, value)
		case OpSubtract:
			total.Sub(total, value)
		}
	}
	return formatExact(total), nil
}

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
