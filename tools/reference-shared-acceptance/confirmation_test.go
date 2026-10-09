package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var orderEntry = SubjectRef{Kind: "application", ID: "order_entry"}

func humanInput(subject string, basis ...SubjectRef) ConfirmationInput {
	return ConfirmationInput{
		Outcome:       outcomeAnswered,
		Identity:      ConfirmationIdentity{Kind: "scope-question", Subject: subject},
		BasisSubjects: basis,
		Answer:        "keep",
		DecidedBy:     ConfirmationActor{Kind: actorHuman, ID: "owner"},
	}
}

func mustRecord(t *testing.T, root string, in ConfirmationInput) string {
	t.Helper()
	rel, err := recordConfirmation(root, in)
	if err != nil {
		t.Fatal(err)
	}
	return rel
}

func mustCheck(t *testing.T, root, subject string) ConfirmationCheck {
	t.Helper()
	c, err := checkConfirmation(root, ConfirmationQuery{Kind: "scope-question", Subject: subject})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func refusalCode(err error) string {
	var r ConfirmationRefusal
	if errors.As(err, &r) {
		return r.Code
	}
	return ""
}

func touchOrderEntry(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "applications", "order-entry", "application.tei")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString("\n"); err != nil {
		t.Fatal(err)
	}
}

func TestNoRecordAsks(t *testing.T) {
	c := mustCheck(t, scratchWorkspace(t), "q1")
	if !c.Ask || c.Reason != "no-record" {
		t.Fatalf("記録が無いのに確認を出していません: %+v", c)
	}
}

func TestAnsweredWithUnchangedBasisDoesNotAsk(t *testing.T) {
	root := scratchWorkspace(t)
	mustRecord(t, root, humanInput("q1", orderEntry))
	c := mustCheck(t, root, "q1")
	if c.Ask || c.Reason != "answered-basis-unchanged" {
		t.Fatalf("前提が同じなのに確認を出しています: %+v", c)
	}
}

func TestChangedBasisAsksAgainWithPreviousAnswer(t *testing.T) {
	root := scratchWorkspace(t)
	mustRecord(t, root, humanInput("q1", orderEntry))
	touchOrderEntry(t, root)
	c := mustCheck(t, root, "q1")
	if !c.Ask || c.Reason != "basis-changed" || len(c.Changed) != 1 || c.PreviousAnswer != "keep" {
		t.Fatalf("前提が変わったのに前回の答え付きで確認を出していません: %+v", c)
	}
	if c.Changed[0].Before == c.Changed[0].After || c.Changed[0].Before == "" {
		t.Fatalf("変化点のfingerprintが不正です: %+v", c.Changed[0])
	}
}

func TestUnrelatedChangeDoesNotAsk(t *testing.T) {
	root := scratchWorkspace(t)
	mustRecord(t, root, humanInput("q1", orderEntry))
	path := filepath.Join(root, "applications", "repair-intake", "application.tei")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("\n")
	f.Close()
	if c := mustCheck(t, root, "q1"); c.Ask {
		t.Fatalf("無関係なsourceの変更で確認を出しています: %+v", c)
	}
}

func TestQuestionWordingIsNotPartOfIdentity(t *testing.T) {
	root := scratchWorkspace(t)
	mustRecord(t, root, humanInput("q1", orderEntry))
	c, err := checkConfirmation(root, ConfirmationQuery{Kind: "scope-question", Subject: "q1", Question: "言い回しを変えた確認"})
	if err != nil || c.Ask {
		t.Fatalf("言い回しの違いで確認を出しています: %+v %v", c, err)
	}
}

func TestAIAnswerNeverSuppresses(t *testing.T) {
	root := scratchWorkspace(t)
	in := humanInput("q1", orderEntry)
	in.DecidedBy = ConfirmationActor{Kind: actorAI, ID: "assistant"}
	mustRecord(t, root, in)
	if c := mustCheck(t, root, "q1"); !c.Ask || c.Reason != "answer-not-by-human" {
		t.Fatalf("AIの答えで確認を抑制しています: %+v", c)
	}
}

func TestUnresolvedBasisIsRefusedAtRecordTime(t *testing.T) {
	root := scratchWorkspace(t)
	_, err := recordConfirmation(root, humanInput("q1", SubjectRef{Kind: "application", ID: "no_such_app"}))
	if refusalCode(err) != "BASIS_SUBJECT_UNRESOLVED" {
		t.Fatalf("解決できない前提を拒否していません: %v", err)
	}
}

func TestBasisIsRequired(t *testing.T) {
	_, err := recordConfirmation(scratchWorkspace(t), humanInput("q1"))
	if refusalCode(err) != "BASIS_REQUIRED" {
		t.Fatalf("前提の無い答えを拒否していません: %v", err)
	}
}

func TestChangingAnAnswerRequiresSupersedes(t *testing.T) {
	root := scratchWorkspace(t)
	first := mustRecord(t, root, humanInput("q1", orderEntry))
	_, err := recordConfirmation(root, humanInput("q1", orderEntry))
	if refusalCode(err) != "MUST_SUPERSEDE" {
		t.Fatalf("置き換え指定の無い上書きを拒否していません: %v", err)
	}
	in := humanInput("q1", orderEntry)
	in.Answer = "change"
	in.Supersedes = filepath.Base(first)
	second := mustRecord(t, root, in)
	if c := mustCheck(t, root, "q1"); c.Record != second || c.PreviousAnswer != "change" {
		t.Fatalf("置き換え後の先頭を使っていません: %+v", c)
	}
	in.Supersedes = filepath.Base(first) // すでに先頭ではない
	if _, err := recordConfirmation(root, in); refusalCode(err) != "SUPERSEDES_NOT_HEAD" {
		t.Fatalf("先頭でない記録の置き換えを拒否していません: %v", err)
	}
}

func TestForkedHistoryAsks(t *testing.T) {
	root := scratchWorkspace(t)
	first := mustRecord(t, root, humanInput("q1", orderEntry))
	a := humanInput("q1", orderEntry)
	a.Answer, a.Supersedes = "a", filepath.Base(first)
	mustRecord(t, root, a)
	// 別の経路で同じ記録を置き換えた分岐を、直接書いて作る
	rec := ConfirmationRecord{Schema: confirmationSchema, Outcome: outcomeAnswered,
		Identity: ConfirmationIdentity{Kind: "scope-question", Subject: "q1"}, Answer: "b",
		DecidedBy: ConfirmationActor{Kind: actorHuman, ID: "owner"}, Supersedes: filepath.Base(first),
		Basis: []BasisEntry{{Subject: orderEntry, Artifact: "applications/order-entry/application.tei", Fingerprint: "x"}}}
	writeRawRecord(t, root, rec)
	// 分岐: 先頭が1件(a)+その他。先に置き換え済みのfirstは先頭でなく、aとbが先頭で2件になる
	if c := mustCheck(t, root, "q1"); !c.Ask || c.Reason != "ambiguous-history" {
		t.Fatalf("分岐した履歴で確認を出していません: %+v", c)
	}
}

func writeRawRecord(t *testing.T, root string, rec ConfirmationRecord) string {
	t.Helper()
	encoded, err := encodeJSON(rec)
	if err != nil {
		t.Fatal(err)
	}
	rel := confirmationDir + "/" + sha256Hex(string(encoded)) + ".json"
	target := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	return rel
}

func TestTamperedOrUnknownRecordAsksAndBlocksWrites(t *testing.T) {
	root := scratchWorkspace(t)
	rel := mustRecord(t, root, humanInput("q1", orderEntry))
	path := filepath.Join(root, filepath.FromSlash(rel))
	content, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(content), "keep", "skip", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	c := mustCheck(t, root, "q1")
	if !c.Ask || c.Reason != "record-unreadable" || len(c.Unreadable) != 1 {
		t.Fatalf("改ざんされた記録で確認を出していません: %+v", c)
	}
	if _, err := recordConfirmation(root, humanInput("q2", orderEntry)); refusalCode(err) != "STORE_UNREADABLE" {
		t.Fatalf("読めない記録がある間の追記を拒否していません: %v", err)
	}
	if _, err := listConfirmations(root); err == nil {
		t.Fatal("読めない記録を黙って一覧から飛ばしています")
	}
}

func TestCategorySuppressionAndReinstatement(t *testing.T) {
	root := scratchWorkspace(t)
	sup := ConfirmationInput{Outcome: outcomeCategorySuppressed, Identity: ConfirmationIdentity{Kind: "scope-question"},
		DecidedBy: ConfirmationActor{Kind: actorHuman, ID: "owner"}}
	if _, err := recordConfirmation(root, sup); refusalCode(err) != "REASON_REQUIRED" {
		t.Fatalf("理由の無い抑制を拒否していません: %v", err)
	}
	sup.Reason = "この種類は毎回同じ答えで、見る価値がない"
	first := mustRecord(t, root, sup)
	if c := mustCheck(t, root, "anything"); c.Ask || c.Reason != "category-suppressed" || c.PreviousReason == "" {
		t.Fatalf("種類の抑制が効いていません: %+v", c)
	}
	idx, err := listConfirmations(root)
	if err != nil || len(idx.SuppressedCategories) != 1 || idx.SuppressedCategories[0] != "scope-question" {
		t.Fatalf("一覧が抑制中の種類を示していません: %+v %v", idx, err)
	}
	re := ConfirmationInput{Outcome: outcomeCategoryReinstated, Identity: ConfirmationIdentity{Kind: "scope-question"},
		DecidedBy: ConfirmationActor{Kind: actorHuman, ID: "owner"}, Supersedes: filepath.Base(first)}
	mustRecord(t, root, re)
	if c := mustCheck(t, root, "anything"); !c.Ask {
		t.Fatalf("解除後も確認が抑制されています: %+v", c)
	}
}

func TestRecordsDoNotChangeModelRevision(t *testing.T) {
	root := scratchWorkspace(t)
	before, _ := takeSnapshot(root)
	mustRecord(t, root, humanInput("q1", orderEntry))
	after, _ := takeSnapshot(root)
	if before.revision != after.revision {
		t.Fatal("確認記録がmodel revisionに入っています")
	}
}

func TestDeletingTheStoreOnlyBringsBackTheQuestion(t *testing.T) {
	root := scratchWorkspace(t)
	mustRecord(t, root, humanInput("q1", orderEntry))
	if err := os.RemoveAll(filepath.Join(root, ".tei", "confirmations")); err != nil {
		t.Fatal(err)
	}
	if c := mustCheck(t, root, "q1"); !c.Ask || c.Reason != "no-record" {
		t.Fatalf("記録を消したのに確認が戻っていません: %+v", c)
	}
}

func TestConfirmCLIRefusesReferenceBaseline(t *testing.T) {
	input := filepath.Join(t.TempDir(), "in.json")
	_ = os.WriteFile(input, []byte(`{"outcome":"answered"}`), 0o600)
	var out, errOut strings.Builder
	code := runCLI([]string{"confirm-record", "-workspace", filepath.Join(repositoryRoot(t), "reference", "workspace"), "-input", input}, &out, &errOut)
	if code != 2 {
		t.Fatalf("baselineへの記録を拒否していません: code=%d %s", code, errOut.String())
	}
}

func TestSupersedesAcrossIdentitiesIsInconsistent(t *testing.T) {
	root := scratchWorkspace(t)
	first := mustRecord(t, root, humanInput("q1", orderEntry))
	writeRawRecord(t, root, ConfirmationRecord{Schema: confirmationSchema, Outcome: outcomeAnswered,
		Identity: ConfirmationIdentity{Kind: "scope-question", Subject: "q2"}, Answer: "x",
		DecidedBy: ConfirmationActor{Kind: actorHuman, ID: "owner"}, Supersedes: filepath.Base(first),
		Basis: []BasisEntry{{Subject: orderEntry, Artifact: "applications/order-entry/application.tei", Fingerprint: "x"}}})
	if c := mustCheck(t, root, "q1"); !c.Ask || c.Reason != "record-unreadable" {
		t.Fatalf("別identityへのsupersedesで確認を出していません: %+v", c)
	}
}
