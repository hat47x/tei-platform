package realization

import "tei/internal/core"

// ContractQueryTargetは、Database-ownedなContract Queryを実現層から識別するためのtarget identityである。
//
// ApplicationやCapabilityへidentityを変換せず、Database / Contract / Queryの三つ組を正本とする。
// environmentやproviderによるrealization選択条件はこのidentityへ混ぜず、別のBinding / composition責務として扱う。
type ContractQueryTarget struct {
	Database core.CanonicalID `json:"database"`
	Contract core.CanonicalID `json:"contract"`
	Query    core.LocalName   `json:"query"`
}

// Validは、target identityの三要素がすべて明示されていることを確認する。
// 欠けたidentityをfile pathやprovider情報から補完することはしない。
func (t ContractQueryTarget) Valid() bool {
	return t.Database != "" && t.Contract != "" && t.Query != ""
}

// Identityは、diagnosticやruntime registryで利用できる安定したtarget identityを返す。
// file pathやprovider名を含めないため、realization sourceの移動や実装方式の変更ではidentityが変わらない。
func (t ContractQueryTarget) Identity() string {
	return string(t.Database) + "/" + string(t.Contract) + "/" + string(t.Query)
}
