package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 受理の「現在資格」を、どの主張が何に束縛されているかで固定する（#4558）。
//
// ここで固定するのは現行の挙動である。望ましい挙動ではない。特にE2は、却下の記録が後の
// 受理を妨げないことを示しており、持ち主の判断（却下後の受理を保留にするか）が出たら
// その期待を変える。ExpectedはSource・台帳の仕様から手で書き、実行結果から作らない。

func currentEligibilityCandidate(t *testing.T, dir, id, to string) []byte {
	t.Helper()
	c := setAttributeTypeCandidate(revisionOfDir(t, dir), lineIDAttribute, to)
	c.ID = id
	return mustJSON(t, c)
}

func reviewFingerprintOf(t *testing.T, dir string, raw []byte) string {
	t.Helper()
	ev, err := evaluate(dir, raw)
	if err != nil {
		t.Fatal(err)
	}
	fp := buildReviewProjection(ev).ReviewFingerprint
	if fp == "" {
		t.Fatal("review fingerprintが空です")
	}
	return fp
}

func acceptWith(t *testing.T, dir string, raw []byte, fp string) Decision {
	t.Helper()
	d, err := accept(dir, raw, acceptOptions{ExpectedReviewFingerprint: fp})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// cliDecideは、台帳へ記録するCLI経路（review → accept / reject）で決定を出す。
// in-processのaccept・rejectは台帳を書かないため、台帳を見る試験はこちらを使う。
func cliDecide(t *testing.T, dir string, raw []byte, command string) Decision {
	t.Helper()
	path := filepath.Join(t.TempDir(), "candidate.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{command, "-workspace", dir, "-candidate", path}
	switch command {
	case "accept":
		args = append(args, "-expected-review-fingerprint", reviewFingerprintViaCLI(t, dir, path))
	case "reject":
		args = append(args, "-reason", "owner said no")
	}
	var stdout, stderr bytes.Buffer
	runCLI(args, &stdout, &stderr)
	var d Decision
	if err := json.Unmarshal(stdout.Bytes(), &d); err != nil {
		t.Fatalf("%s: %v: %s %s", command, err, stdout.String(), stderr.String())
	}
	return d
}

// E1: A→B→Aの後でも、Aで取ったreviewは受理できる。パッチと承認内容は内容に束縛され、
// 版の履歴には束縛されないためで、過剰な失効を起こさないことを仕様として固定する。
func TestApprovalTakenAtAIsStillValidAfterAToBToA(t *testing.T) {
	dir := scratchWorkspace(t)
	atA := currentEligibilityCandidate(t, dir, "c1-to-order-id", "sales.order_id")
	fp := reviewFingerprintOf(t, dir, atA)
	revisionA := revisionOfDir(t, dir)

	toB := currentEligibilityCandidate(t, dir, "c2-to-identifier", "std.identifier")
	if d := acceptWith(t, dir, toB, reviewFingerprintOf(t, dir, toB)); d.Decision != "accepted" {
		t.Fatalf("A→B=%+v", d.Refusals)
	}
	back := currentEligibilityCandidate(t, dir, "c3-back", "sales.order_line_id")
	if d := acceptWith(t, dir, back, reviewFingerprintOf(t, dir, back)); d.Decision != "accepted" {
		t.Fatalf("B→A=%+v", d.Refusals)
	}
	if got := revisionOfDir(t, dir); got != revisionA {
		t.Fatalf("revisionがAへ戻っていません: %s != %s", got, revisionA)
	}

	d := acceptWith(t, dir, atA, fp)
	if d.Decision != "accepted" || !d.SourceChanged {
		t.Fatalf("Aで取ったreviewが受理されません: %s %+v", d.Decision, d.Refusals)
	}
	if got := attributeTypeOf(t, dir, lineIDAttribute); got != "sales.order_id" {
		t.Fatalf("型=%q", got)
	}
}

// E2: 同じ候補をrejectして記録しても、前のreviewでacceptできる。台帳の却下はacceptの
// 条件になっていない（#4558 S1の判断待ち）。
func TestRecordedRejectionDoesNotBlockLaterAcceptanceOfTheSameCandidate(t *testing.T) {
	dir := scratchWorkspace(t)
	raw := currentEligibilityCandidate(t, dir, "c1-to-order-id", "sales.order_id")

	rejected := cliDecide(t, dir, raw, "reject")
	if rejected.Decision != "rejected" || rejected.RecordPath == "" {
		t.Fatalf("reject=%s record=%q", rejected.Decision, rejected.RecordPath)
	}
	d := cliDecide(t, dir, raw, "accept")
	if d.Decision != "accepted" || !d.SourceChanged {
		t.Fatalf("却下後の受理=%s %+v", d.Decision, d.Refusals)
	}
	if d.CandidateFingerprint != rejected.CandidateFingerprint {
		t.Fatal("却下と受理が別のcandidate fingerprintを持っています")
	}
	index, err := loadDecisionRecords(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	var verdicts []string
	for _, r := range index.Records {
		if r.Decision.CandidateID == "c1-to-order-id" {
			verdicts = append(verdicts, r.Decision.Decision)
		}
	}
	if len(verdicts) != 2 {
		t.Fatalf("台帳の記録=%v。却下と受理の2件が残るはずです", verdicts)
	}
}

// E3: reviewとacceptの間に別ファイルへの無関係な受理が入ると、過剰でも安全側に拒否する。
func TestUnrelatedAcceptedChangeStalesTheCandidate(t *testing.T) {
	dir := scratchWorkspace(t)
	raw := currentEligibilityCandidate(t, dir, "c1-to-order-id", "sales.order_id")
	fp := reviewFingerprintOf(t, dir, raw)

	other := mustJSON(t, unitPriceCandidate(revisionOfDir(t, dir)))
	if d := acceptWith(t, dir, other, reviewFingerprintOf(t, dir, other)); d.Decision != "accepted" {
		t.Fatalf("無関係な受理=%+v", d.Refusals)
	}
	before, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(orderFile)))
	if err != nil {
		t.Fatal(err)
	}

	d := acceptWith(t, dir, raw, fp)
	if d.Decision != "refused" || d.SourceChanged {
		t.Fatalf("拒否されません: %s", d.Decision)
	}
	var codes []string
	for _, r := range d.Refusals {
		codes = append(codes, r.Code)
	}
	if !strings.Contains(strings.Join(codes, ","), "STALE_BASE_MODEL_REVISION") {
		t.Fatalf("refusals=%v", codes)
	}
	after, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(orderFile)))
	if err != nil || string(after) != string(before) {
		t.Fatal("拒否されたのに対象fileが変わりました")
	}
}

// E4: A→B→Aの中間の2件を、台帳が別々の記録として報告し、現在を説明するのは最後の1件だけである。
func TestDecisionLedgerReportsBothIntermediateDecisions(t *testing.T) {
	dir := scratchWorkspace(t)
	toB := currentEligibilityCandidate(t, dir, "c2-to-identifier", "std.identifier")
	if d := cliDecide(t, dir, toB, "accept"); d.Decision != "accepted" {
		t.Fatalf("A→B=%+v", d.Refusals)
	}
	back := currentEligibilityCandidate(t, dir, "c3-back", "sales.order_line_id")
	if d := cliDecide(t, dir, back, "accept"); d.Decision != "accepted" {
		t.Fatalf("B→A=%+v", d.Refusals)
	}

	index, err := loadDecisionRecords(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	status := map[string]string{}
	for _, r := range index.Records {
		status[r.Decision.CandidateID] = r.Status
	}
	// 最後の受理だけが現在のrevisionを説明し、一つ前は置き換えられた記録になる。
	want := map[string]string{"c2-to-identifier": "superseded", "c3-back": "current"}
	for id, wantStatus := range want {
		got, ok := status[id]
		if !ok {
			t.Fatalf("台帳に%sがありません", id)
		}
		if got != wantStatus {
			t.Fatalf("%s: status=%q want %q", id, got, wantStatus)
		}
	}
}
