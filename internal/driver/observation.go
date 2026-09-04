package driver

import (
	"context"
	"fmt"
	"strings"
)

// ObservationEvidenceNatureは、Observation readがどの性質のEvidenceから得られたかを示す。
//
// これはCanonicalな業務意味や固定taxonomyではない。第一Referenceではsyntheticと実資産由来の
// Evidenceを混同しないために保持する。将来のEvidence種別をこの三つへ閉じないため、未知の値も
// Coreで列挙制限しない。
type ObservationEvidenceNature string

const (
	ObservationEvidenceSynthetic        ObservationEvidenceNature = "synthetic"
	ObservationEvidenceExternalObserved ObservationEvidenceNature = "external-observed"
	ObservationEvidenceManagedHere      ObservationEvidenceNature = "managed-here"
)

// ObservationReadFuncは、既存Query / Contract Query / read adapter等の明示readを
// Observation用途で一回実行する最小runtime関数である。
//
// parameterやcredentialをEvidenceへ保持しないため、必要な入力はcaller側でbindした関数として渡す。
// この型自体が新しいCanonical read契約やObservation専用Adapterを定義するものではない。
type ObservationReadFunc func(context.Context) (any, error)

// ObservationReadExecutionは、Commandとは別に実行した明示readの再確認可能なruntime結果である。
//
// ReadIdentityはQuery / Contract Query / Binding / Adapter等、callerが実際に選択したread境界の
// identityを保持する。CanonicalIDであることは要求しない。synthetic test fixtureやHost固有readを
// Canonicalへ誤って昇格させないためである。
//
// Resultはreadで得た値であり、「期待状態に一致したか」というScenario上の評価とは分離する。
// 実行時parameter値やcredentialは保持しない。
type ObservationReadExecution struct {
	ReadIdentity string
	Nature       ObservationEvidenceNature
	Result       any
}

// ExecuteObservationReadは、Command Business Resultから独立した明示readを実行する。
//
// Command payload、HTTP status、Capability名等からread境界を推測しない。callerが選んだread identityと
// 実行関数を明示的に渡す。readに失敗した場合もidentity / Evidence natureは返し、Execution Error自体は
// errorとして分離する。これによりObservation失敗を成功済みCommand Business Resultへ畳まない。
func ExecuteObservationRead(
	ctx context.Context,
	readIdentity string,
	nature ObservationEvidenceNature,
	read ObservationReadFunc,
) (ObservationReadExecution, error) {
	identity := strings.TrimSpace(readIdentity)
	if identity == "" {
		return ObservationReadExecution{}, fmt.Errorf("Observation read identityが指定されていません")
	}
	if strings.TrimSpace(string(nature)) == "" {
		return ObservationReadExecution{}, fmt.Errorf("Observation Evidence natureが指定されていません")
	}
	if read == nil {
		return ObservationReadExecution{}, fmt.Errorf("Observation read %sの実行関数が指定されていません", identity)
	}

	execution := ObservationReadExecution{
		ReadIdentity: identity,
		Nature:       nature,
	}
	result, err := read(ctx)
	if err != nil {
		return execution, fmt.Errorf("Observation read %sを完了できませんでした: %w", identity, err)
	}
	execution.Result = result
	return execution, nil
}
