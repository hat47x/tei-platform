package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"tei/internal/validation"
)

// reference/product-contractsは、TAI配下の製品契約を型として並べた参照workspaceである。
// どの検査からも参照されない状態では、変更で壊れても検出できないため、
// 構文・読込み・意味検証が通ることだけをここで確認する。
// 通ることは、型宣言が読めることを示すだけで、各製品の実装・権限・受理を示さない(#2409)。
// worldFingerprintは変更のたびに更新が要るため、固定値は埋め込まず、存在だけを確認する。
func TestProductContractsReferenceWorkspaceValidates(t *testing.T) {
	root := filepath.Join(testRepositoryRoot(t), "reference", "product-contracts")
	var out bytes.Buffer
	code, err := validateWorkspaceJSON(&out, root)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	report := readValidationReport(t, &out)
	if !report.OK || report.Error != nil {
		t.Fatalf("product-contracts is not valid: %+v", report)
	}
	for _, item := range report.Diagnostics {
		if item.Diagnostic.Severity == validation.Error {
			t.Fatalf("unexpected error diagnostic: %+v", item)
		}
	}
	if report.WorldFingerprint == "" || report.SourceIndexFingerprint == "" {
		t.Fatalf("fingerprints are missing: %+v", report)
	}
}
