package legacyauthorization

import (
	"context"
	"strings"
)

// AuthorityEpochs は既存認可規則が参照した正本の変更版を表す。
// 権限根拠の全変更経路が版を単調増加させることはProvider側の責務である。
// 版が一致していても正本と書込み先が別トランザクションなら原子性を保証しない。
type AuthorityEpochs struct {
	Source      string
	SubjectLink uint64
	Membership  uint64
	Policy      uint64
}

func (e AuthorityEpochs) valid() bool {
	return strings.TrimSpace(e.Source) != "" &&
		!strings.ContainsAny(e.Source, "\x00\r\n") &&
		e.SubjectLink != 0 && e.Membership != 0 && e.Policy != 0
}

// EpochReader は既存の主体・所属・認可規則の各正本から版条件を取得する。
// ReadはInputに対応する同一の認可スナップショットを返さなければならない。
// すべての認可根拠の変更が版更新と同じ境界で行われることが導入条件。
// 単にリモートから複製した版や呼出側で組み立てた数値は使わない。
type EpochReader interface {
	ReadAuthorizationEpochs(context.Context, Input) (AuthorityEpochs, error)
}
