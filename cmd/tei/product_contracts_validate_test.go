package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

// 宣言した能力の連鎖が切れていても、現在のvalidateはokを返す(#2410の実測の記録)。
// eki_attempt_executeの出力をdocumentへ替えると、次のeki_result_acceptの入力
// (result_claim)と接続しないが、診断は出ない。
// 能力の接続を検査する契約はOwner判断待ちであり、この試験は「現在は検査しない」ことを固定する。
// 接続の検査を追加した場合は、この試験を検査の試験へ置き換える。
func TestProductContractsBrokenCapabilityChainIsNotDetectedYet(t *testing.T) {
	src := filepath.Join(testRepositoryRoot(t), "reference", "product-contracts")
	root := t.TempDir()
	for _, relative := range []string{
		"tei.workspace.yaml",
		"semantic/contracts.tei",
		"applications/product-flow/application.tei",
	} {
		b, err := os.ReadFile(filepath.Join(src, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		writeWorkspaceFile(t, root, relative, string(b))
	}
	path := filepath.Join(root, "semantic", "contracts.tei")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const before = "    input execution product_contracts.attempt\n    output product_contracts.result_claim\n"
	const after = "    input execution product_contracts.attempt\n    output product_contracts.document\n"
	text := strings.ReplaceAll(string(b), "\r\n", "\n")
	if strings.Count(text, before) != 1 {
		t.Fatal("eki_attempt_execute declaration changed; update this test")
	}
	if err := os.WriteFile(path, []byte(strings.Replace(text, before, after, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	code, err := validateWorkspaceJSON(&out, root)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	if report := readValidationReport(t, &out); !report.OK {
		t.Fatalf("chain check appeared; replace this test: %+v", report)
	}
}
