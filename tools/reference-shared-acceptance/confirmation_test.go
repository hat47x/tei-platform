package main

import (
	"encoding/json"
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

// proposalBasisは、unitPriceCandidateが触る既存のCanonical subjectである（semantic/sales/order.teiで宣言）。
var proposalBasis = SubjectRef{Kind: "semantic-attribute", ID: "sales.order_line.unit_price"}

// rejectViaCLIは、CLIのrejectで候補を退け、書かれた決定記録を返す。
func rejectViaCLI(t *testing.T, root string, c Candidate, reason string) Decision {
	t.Helper()
	code, out, errOut := cliRun(t, "reject", "-workspace", root, "-candidate", candidateFile(t, c), "-reason", reason)
	if code != 0 {
		t.Fatalf("reject code=%d err=%s out=%s", code, errOut, out)
	}
	var decision Decision
	if err := json.Unmarshal([]byte(out), &decision); err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "rejected" || decision.CandidateFingerprint == "" || decision.RecordPath == "" {
		t.Fatalf("rejectの決定にcandidateFingerprintかrecordPathがありません: %+v", decision)
	}
	return decision
}

// rejectedProposalInputは、rejectが書いた決定記録へ結んだ、人による退けの答えを返す。
func rejectedProposalInput(d Decision, basis ...SubjectRef) ConfirmationInput {
	return ConfirmationInput{
		Outcome:        outcomeRejectedProposal,
		Identity:       ConfirmationIdentity{Kind: identityKindProposal, Subject: d.CandidateFingerprint},
		BasisSubjects:  basis,
		Answer:         answerRejected,
		Reason:         "今回は見送る",
		DecidedBy:      ConfirmationActor{Kind: actorHuman, ID: "owner"},
		LinkedDecision: d.RecordPath,
	}
}

func checkProposal(t *testing.T, root, fingerprint string) ConfirmationCheck {
	t.Helper()
	c, err := checkConfirmation(root, ConfirmationQuery{Kind: identityKindProposal, Subject: fingerprint})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// appendNewlineは、末尾に改行を一つ足す。意味は変えず、fileのfingerprintだけを変える。
func appendNewline(t *testing.T, root, rel string) {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(root, filepath.FromSlash(rel)), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString("\n"); err != nil {
		t.Fatal(err)
	}
}

func TestRejectedProposalByHumanWithUnchangedBasisIsNotAskedAgain(t *testing.T) {
	root := scratchWorkspace(t)
	d := rejectViaCLI(t, root, unitPriceCandidate(revisionOfDir(t, root)), "今回は見送る")
	mustRecord(t, root, rejectedProposalInput(d, proposalBasis))
	c := checkProposal(t, root, d.CandidateFingerprint)
	if c.Ask || c.Reason != "rejected-proposal-basis-unchanged" || c.PreviousAnswer != "rejected" ||
		c.PreviousReason != "今回は見送る" || c.DecidedBy == nil || c.DecidedBy.Kind != actorHuman ||
		!strings.Contains(c.Message, "退けており") {
		t.Fatalf("人が退けた提案を、前提が同じなのに再び確認しようとしています: %+v", c)
	}
}

func TestRejectedProposalAsksAgainWhenBasisChanges(t *testing.T) {
	root := scratchWorkspace(t)
	d := rejectViaCLI(t, root, unitPriceCandidate(revisionOfDir(t, root)), "今回は見送る")
	mustRecord(t, root, rejectedProposalInput(d, proposalBasis))
	appendNewline(t, root, "semantic/sales/order.tei")
	c := checkProposal(t, root, d.CandidateFingerprint)
	if !c.Ask || c.Reason != "basis-changed" || len(c.Changed) == 0 || c.PreviousAnswer != "rejected" {
		t.Fatalf("前提が変わった退けた提案を、前回の答え付きで確認していません: %+v", c)
	}
}

func TestAIRejectionNeverSuppressesAProposal(t *testing.T) {
	root := scratchWorkspace(t)
	d := rejectViaCLI(t, root, unitPriceCandidate(revisionOfDir(t, root)), "今回は見送る")
	in := rejectedProposalInput(d, proposalBasis)
	in.DecidedBy = ConfirmationActor{Kind: actorAI, ID: "assistant"}
	mustRecord(t, root, in)
	if c := checkProposal(t, root, d.CandidateFingerprint); !c.Ask || c.Reason != "answer-not-by-human" || c.PreviousAnswer != "rejected" {
		t.Fatalf("AIの退けで提案を抑制しています: %+v", c)
	}
}

func TestRejectedProposalRefusals(t *testing.T) {
	root := scratchWorkspace(t)
	d := rejectViaCLI(t, root, unitPriceCandidate(revisionOfDir(t, root)), "今回は見送る")
	cases := []struct {
		name string
		edit func(in *ConfirmationInput)
		code string
	}{
		{"理由が無い", func(in *ConfirmationInput) { in.Reason = "" }, "REASON_REQUIRED"},
		{"答えがrejectedでない", func(in *ConfirmationInput) { in.Answer = "accepted" }, "ANSWER_INVALID"},
		{"答えが無い", func(in *ConfirmationInput) { in.Answer = "" }, "ANSWER_INVALID"},
		{"linkedDecisionが無い", func(in *ConfirmationInput) { in.LinkedDecision = "" }, "LINKED_DECISION_REQUIRED"},
		{"identity.kindがproposalでない", func(in *ConfirmationInput) { in.Identity.Kind = "scope-question" }, "IDENTITY_KIND_INVALID"},
		{"identity.subjectが無い", func(in *ConfirmationInput) { in.Identity.Subject = "" }, "IDENTITY_SUBJECT_REQUIRED"},
		{"前提が無い", func(in *ConfirmationInput) { in.BasisSubjects = nil }, "BASIS_REQUIRED"},
	}
	for _, tc := range cases {
		in := rejectedProposalInput(d, proposalBasis)
		tc.edit(&in)
		if _, err := recordConfirmation(root, in); refusalCode(err) != tc.code {
			t.Errorf("%s: %s を期待しましたが %v", tc.name, tc.code, err)
		}
	}
	if entries, _, err := readConfirmations(root); err != nil || len(entries) != 0 {
		t.Fatalf("拒否された退けた提案の記録が書かれています: %d件 %v", len(entries), err)
	}
}

func TestRejectedProposalLinkedDecisionMustExist(t *testing.T) {
	root := scratchWorkspace(t)
	d := rejectViaCLI(t, root, unitPriceCandidate(revisionOfDir(t, root)), "今回は見送る")
	in := rejectedProposalInput(d, proposalBasis)
	in.LinkedDecision = decisionDir + "/" + strings.Repeat("0", 64) + ".json"
	if _, err := recordConfirmation(root, in); refusalCode(err) != "LINKED_DECISION_NOT_FOUND" {
		t.Fatalf("存在しない決定記録を指すlinkedDecisionを拒否していません: %v", err)
	}
}

func TestProposalFingerprintDoesNotCollideWithScopeQuestionSubject(t *testing.T) {
	root := scratchWorkspace(t)
	d := rejectViaCLI(t, root, unitPriceCandidate(revisionOfDir(t, root)), "今回は見送る")
	fp := d.CandidateFingerprint
	mustRecord(t, root, rejectedProposalInput(d, proposalBasis))
	if c := mustCheck(t, root, fp); !c.Ask || c.Reason != "no-record" {
		t.Fatalf("提案の記録が、同じ文字列のscope-questionを抑制しています: %+v", c)
	}
	mustRecord(t, root, humanInput(fp, orderEntry))
	if c := checkProposal(t, root, fp); c.Ask || c.Reason != "rejected-proposal-basis-unchanged" || c.PreviousAnswer != "rejected" {
		t.Fatalf("scope-questionの答えで、提案の記録が置き換わっています: %+v", c)
	}
	if c := mustCheck(t, root, fp); c.Ask || c.Reason != "answered-basis-unchanged" || c.PreviousAnswer != "keep" {
		t.Fatalf("提案の記録で、scope-questionの答えが置き換わっています: %+v", c)
	}
}

func TestRejectingAProposalDoesNotChangeModelRevision(t *testing.T) {
	root := scratchWorkspace(t)
	before, err := takeSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	d := rejectViaCLI(t, root, unitPriceCandidate(before.revision), "今回は見送る")
	mustRecord(t, root, rejectedProposalInput(d, proposalBasis))
	after, err := takeSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	if before.revision != after.revision || d.SourceChanged || d.ModelRevisionBefore != d.ModelRevisionAfter {
		t.Fatalf("退けた提案の記録がmodel revisionを動かしています: before=%s after=%s decision=%+v", before.revision, after.revision, d)
	}
}

func TestRejectedProposalKeepsTheSupersedesRules(t *testing.T) {
	root := scratchWorkspace(t)
	d := rejectViaCLI(t, root, unitPriceCandidate(revisionOfDir(t, root)), "今回は見送る")
	first := mustRecord(t, root, rejectedProposalInput(d, proposalBasis))
	if _, err := recordConfirmation(root, rejectedProposalInput(d, proposalBasis)); refusalCode(err) != "MUST_SUPERSEDE" {
		t.Fatalf("置き換え指定の無い退けた提案の上書きを拒否していません: %v", err)
	}
	in := rejectedProposalInput(d, proposalBasis)
	in.Reason = "理由を書き直す"
	in.Supersedes = filepath.Base(first)
	second := mustRecord(t, root, in)
	if c := checkProposal(t, root, d.CandidateFingerprint); c.Record != second || c.PreviousReason != "理由を書き直す" {
		t.Fatalf("置き換え後の退けた提案の先頭を使っていません: %+v", c)
	}
}

func TestCategorySuppressionStillAppliesToProposals(t *testing.T) {
	root := scratchWorkspace(t)
	mustRecord(t, root, ConfirmationInput{
		Outcome:   outcomeCategorySuppressed,
		Identity:  ConfirmationIdentity{Kind: identityKindProposal},
		Reason:    "提案の種類ごと、今は見ない",
		DecidedBy: ConfirmationActor{Kind: actorHuman, ID: "owner"},
	})
	if c := checkProposal(t, root, "sha256:"+strings.Repeat("a", 64)); c.Ask || c.Reason != "category-suppressed" {
		t.Fatalf("提案の種類の抑制が効いていません: %+v", c)
	}
}

// 同じdeltaでも、baseModelRevisionが変わるとcandidateFingerprintも変わり、別のidentityになる。
// 無関係なsourceの変更でそうなると、新しい提案には記録が無いため確認を出す（安全側）。古い記録は消えずに残る。
func TestRegeneratedProposalOnANewRevisionIsAskedAgain(t *testing.T) {
	root := scratchWorkspace(t)
	old := rejectViaCLI(t, root, unitPriceCandidate(revisionOfDir(t, root)), "今回は見送る")
	mustRecord(t, root, rejectedProposalInput(old, proposalBasis))
	appendNewline(t, root, "applications/repair-intake/application.tei")
	regenerated := rejectViaCLI(t, root, unitPriceCandidate(revisionOfDir(t, root)), "今回は見送る")
	if regenerated.CandidateFingerprint == old.CandidateFingerprint {
		t.Fatal("baseModelRevisionが変わった候補のfingerprintが同じです")
	}
	if c := checkProposal(t, root, regenerated.CandidateFingerprint); !c.Ask || c.Reason != "no-record" {
		t.Fatalf("別identityになった提案を、記録が無いのに抑制しています: %+v", c)
	}
	if c := checkProposal(t, root, old.CandidateFingerprint); c.Ask || c.Reason != "rejected-proposal-basis-unchanged" {
		t.Fatalf("前提が変わっていないのに、古い退けた記録が効かなくなっています: %+v", c)
	}
}

// 保存された退けた提案が必須項目を欠くときは、読めない記録として確認を出す側へ倒す。
// 内容のhashは書き換え後に付け直せるため、読む側でも書く側と同じ項目を確かめる。欠けていない控えで、検査が効いていることも確かめる。
func TestStoredRejectedProposalMissingRequiredFieldsIsUnreadable(t *testing.T) {
	cases := []struct {
		name   string
		edit   func(rec *ConfirmationRecord)
		ask    bool
		reason string
	}{
		{"欠けた項目が無い（控え）", func(rec *ConfirmationRecord) {}, false, "rejected-proposal-basis-unchanged"},
		{"前提が無い", func(rec *ConfirmationRecord) { rec.Basis = nil }, true, "record-unreadable"},
		{"理由が無い", func(rec *ConfirmationRecord) { rec.Reason = "" }, true, "record-unreadable"},
		{"答えがrejectedでない", func(rec *ConfirmationRecord) { rec.Answer = "keep" }, true, "record-unreadable"},
		{"linkedDecisionが無い", func(rec *ConfirmationRecord) { rec.LinkedDecision = "" }, true, "record-unreadable"},
		{"identity.kindがproposalでない", func(rec *ConfirmationRecord) { rec.Identity.Kind = "scope-question" }, true, "record-unreadable"},
	}
	for _, tc := range cases {
		root := scratchWorkspace(t)
		d := rejectViaCLI(t, root, unitPriceCandidate(revisionOfDir(t, root)), "今回は見送る")
		s, err := takeSnapshot(root)
		if err != nil {
			t.Fatal(err)
		}
		basis, _ := resolveBasis(s, []SubjectRef{proposalBasis})
		rec := ConfirmationRecord{
			Schema:         confirmationSchema,
			Outcome:        outcomeRejectedProposal,
			Identity:       ConfirmationIdentity{Kind: identityKindProposal, Subject: d.CandidateFingerprint},
			Basis:          basis,
			ModelRevision:  s.revision,
			Answer:         answerRejected,
			Reason:         "今回は見送る",
			DecidedBy:      ConfirmationActor{Kind: actorHuman, ID: "owner"},
			LinkedDecision: d.RecordPath,
		}
		tc.edit(&rec)
		writeRawRecord(t, root, rec)
		if c := checkProposal(t, root, d.CandidateFingerprint); c.Ask != tc.ask || c.Reason != tc.reason {
			t.Errorf("%s: ask=%v reason=%s を期待しましたが %+v", tc.name, tc.ask, tc.reason, c)
		}
	}
}

// linkedDecisionは、.tei/decisions/で始まるだけでは足りず、決定記録の外へ出るpathを拒否する。
func TestLinkedDecisionCannotEscapeTheDecisionsDirectory(t *testing.T) {
	root := scratchWorkspace(t)
	d := rejectViaCLI(t, root, unitPriceCandidate(revisionOfDir(t, root)), "今回は見送る")
	in := rejectedProposalInput(d, proposalBasis)
	in.LinkedDecision = decisionDir + "/../../tei.workspace.yaml"
	if _, err := recordConfirmation(root, in); refusalCode(err) != "LINKED_DECISION_INVALID" {
		t.Fatalf("決定記録の外を指すlinkedDecisionを拒否していません: %v", err)
	}
}

// reject → confirm-record → confirm-check を、CLIの入力JSONだけで通す。docsの例と同じ形を検査する。
func TestRejectedProposalFlowRunsThroughTheCLI(t *testing.T) {
	root := scratchWorkspace(t)
	d := rejectViaCLI(t, root, unitPriceCandidate(revisionOfDir(t, root)), "今回は見送る")
	input := `{
  "outcome": "rejected-proposal",
  "identity": {"kind": "proposal", "subject": "FINGERPRINT"},
  "basisSubjects": [{"kind": "semantic-attribute", "id": "sales.order_line.unit_price"}],
  "answer": "rejected",
  "reason": "今回は見送る",
  "decidedBy": {"kind": "human", "id": "owner"},
  "linkedDecision": "RECORD"
}`
	input = strings.NewReplacer("FINGERPRINT", d.CandidateFingerprint, "RECORD", d.RecordPath).Replace(input)
	inputPath := filepath.Join(t.TempDir(), "rejected.json")
	writeFile(t, inputPath, input)
	if code, out, errOut := cliRun(t, "confirm-record", "-workspace", root, "-input", inputPath); code != 0 {
		t.Fatalf("confirm-record code=%d out=%s err=%s", code, out, errOut)
	}
	queryPath := filepath.Join(t.TempDir(), "query.json")
	writeFile(t, queryPath, `{"kind": "proposal", "subject": "`+d.CandidateFingerprint+`"}`)
	code, out, errOut := cliRun(t, "confirm-check", "-workspace", root, "-input", queryPath)
	if code != 0 {
		t.Fatalf("confirm-check code=%d out=%s err=%s", code, out, errOut)
	}
	var check ConfirmationCheck
	if err := json.Unmarshal([]byte(out), &check); err != nil {
		t.Fatal(err)
	}
	if check.Ask || check.Reason != "rejected-proposal-basis-unchanged" {
		t.Fatalf("CLIで記録した退けた提案を再び確認しようとしています: %+v", check)
	}
}

// writeRawDecisionは、決定記録の形で一件書く。reject以外の経路で結び先を作り、結び先の検査を試すためである。
func writeRawDecision(t *testing.T, root string, d Decision) string {
	t.Helper()
	encoded, err := encodeJSON(d)
	if err != nil {
		t.Fatal(err)
	}
	rel := decisionDir + "/" + sha256Hex(string(encoded)) + ".json"
	target := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	return rel
}

// removeUnitPriceDeclarationは、unitPriceの宣言を消す。前提のsubjectが解決できなくなる変化を作る。
func removeUnitPriceDeclaration(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "semantic", "sales", "order.tei")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	line := "    attribute unitPrice type std.money\n"
	if strings.Count(string(raw), line) != 1 {
		t.Fatal("unitPriceの宣言が一つだけ見つかりません")
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), line, "", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
}

// 答えの記録（answered）でidentity.kindをproposalにすると、rejected-proposalの理由とlinkedDecisionの要件を迂回できた。
// 提案の種類は退けた提案の記録だけが使い、読む側も同じ規則で確かめる。
func TestAnsweredOutcomeCannotRecordAProposalRejection(t *testing.T) {
	root := scratchWorkspace(t)
	d := rejectViaCLI(t, root, unitPriceCandidate(revisionOfDir(t, root)), "今回は見送る")
	in := ConfirmationInput{
		Outcome:       outcomeAnswered,
		Identity:      ConfirmationIdentity{Kind: identityKindProposal, Subject: d.CandidateFingerprint},
		BasisSubjects: []SubjectRef{proposalBasis},
		Answer:        answerRejected,
		DecidedBy:     ConfirmationActor{Kind: actorHuman, ID: "owner"},
	}
	if _, err := recordConfirmation(root, in); refusalCode(err) != "IDENTITY_KIND_RESERVED" {
		t.Fatalf("answeredで提案を退けた記録を拒否していません: %v", err)
	}
	if entries, _, err := readConfirmations(root); err != nil || len(entries) != 0 {
		t.Fatalf("拒否された記録が書かれています: %d件 %v", len(entries), err)
	}
	// 書く側の拒否を回避して手で書いた同じ形の記録も、読む側で確認を出す側へ倒す。
	s, err := takeSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	basis, _ := resolveBasis(s, []SubjectRef{proposalBasis})
	writeRawRecord(t, root, ConfirmationRecord{
		Schema:        confirmationSchema,
		Outcome:       outcomeAnswered,
		Identity:      in.Identity,
		Basis:         basis,
		ModelRevision: s.revision,
		Answer:        answerRejected,
		DecidedBy:     ConfirmationActor{Kind: actorHuman, ID: "owner"},
	})
	if c := checkProposal(t, root, d.CandidateFingerprint); !c.Ask || c.Reason != "record-unreadable" {
		t.Fatalf("answeredで書かれた提案の退けで抑制しています: %+v", c)
	}
}

// 退けた提案の結び先は、identity.subjectの候補を退けたreject決定でなければならない。
// 別の候補の退け、まだ退けていない候補、採用の決定、決定記録の形を外れたfile（directoryや内容とずれた名前）は結べない。
func TestRejectedProposalLinkMustBeTheRejectionOfThisCandidate(t *testing.T) {
	root := scratchWorkspace(t)
	rev := revisionOfDir(t, root)
	dA := rejectViaCLI(t, root, unitPriceCandidate(rev), "今回は見送る")
	candB := unitPriceCandidate(rev)
	candB.Delta[0].ID = "sales.order_line.unit_price_checked"
	_, fpB, err := normalizeCandidate(candB)
	if err != nil || fpB == dA.CandidateFingerprint {
		t.Fatalf("テスト用の別候補を作れません: %s %v", fpB, err)
	}
	accepted := writeRawDecision(t, root, Decision{Schema: decisionSchema, Decision: "accepted", CandidateFingerprint: dA.CandidateFingerprint})
	dirName := strings.Repeat("1", 64) + ".json"
	if err := os.MkdirAll(filepath.Join(root, decisionDir, dirName), 0o755); err != nil {
		t.Fatal(err)
	}
	misnamed := strings.Repeat("2", 64) + ".json"
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(dA.RecordPath)))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, decisionDir, misnamed), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		subject string
		linked  string
		code    string
	}{
		{"別の候補の退けを、まだ退けていない候補の答えに結ぶ", fpB, dA.RecordPath, "LINKED_DECISION_INVALID"},
		{"候補と一致しないfingerprint", "sha256:" + strings.Repeat("0", 64), dA.RecordPath, "LINKED_DECISION_INVALID"},
		{"候補の形ではないsubject", "not-a-fingerprint", dA.RecordPath, "IDENTITY_SUBJECT_INVALID"},
		{"採用の決定", dA.CandidateFingerprint, accepted, "LINKED_DECISION_INVALID"},
		{"決定記録の名前をしたdirectory", dA.CandidateFingerprint, decisionDir + "/" + dirName, "LINKED_DECISION_INVALID"},
		{"内容のhashと名前がずれた決定記録", dA.CandidateFingerprint, decisionDir + "/" + misnamed, "LINKED_DECISION_INVALID"},
	}
	for _, tc := range cases {
		in := rejectedProposalInput(dA, proposalBasis)
		in.Identity.Subject = tc.subject
		in.LinkedDecision = tc.linked
		if _, err := recordConfirmation(root, in); refusalCode(err) != tc.code {
			t.Errorf("%s: %s を期待しましたが %v", tc.name, tc.code, err)
		}
	}
	if entries, _, err := readConfirmations(root); err != nil || len(entries) != 0 {
		t.Fatalf("結べない退けた提案の記録が書かれています: %d件 %v", len(entries), err)
	}
}

// 空白だけのlinkedDecisionは、空のlinkedDecisionと同じく必須の欠落として拒否する。
func TestRejectedProposalBlankLinkedDecisionIsRequired(t *testing.T) {
	root := scratchWorkspace(t)
	d := rejectViaCLI(t, root, unitPriceCandidate(revisionOfDir(t, root)), "今回は見送る")
	for _, blank := range []string{"", "   ", "\t"} {
		in := rejectedProposalInput(d, proposalBasis)
		in.LinkedDecision = blank
		if _, err := recordConfirmation(root, in); refusalCode(err) != "LINKED_DECISION_REQUIRED" {
			t.Errorf("linkedDecision %q: LINKED_DECISION_REQUIRED を期待しましたが %v", blank, err)
		}
	}
}

// 前提のsubjectの宣言が消えたときも、前提が変わったこととして確認を出す（spec: basis-changed）。
// 答えた時点のfingerprintを変化点に添える。
func TestRejectedProposalBasisRemovedAsksAsBasisChanged(t *testing.T) {
	root := scratchWorkspace(t)
	d := rejectViaCLI(t, root, unitPriceCandidate(revisionOfDir(t, root)), "今回は見送る")
	mustRecord(t, root, rejectedProposalInput(d, proposalBasis))
	removeUnitPriceDeclaration(t, root)
	c := checkProposal(t, root, d.CandidateFingerprint)
	if !c.Ask || c.Reason != "basis-changed" || c.PreviousAnswer != "rejected" || len(c.Changed) == 0 || c.Changed[0].Before == "" {
		t.Fatalf("前提のsubjectが消えた退けた提案を、basis-changedとして確認していません: %+v", c)
	}
}

// 保存された退けた提案の結び先が書く側の規則に反するとき、読む側は読めない記録として確認を出す側へ倒す。
// 決定記録が消されただけの場合は問題にしない（決定記録は消せる派生物であり、抑制をその有無に依らせない）。
func TestStoredRejectedProposalWithAForgedLinkIsUnreadable(t *testing.T) {
	type forgedCase struct {
		name   string
		forge  func(t *testing.T, root string, d Decision) (subject, linked string)
		ask    bool
		reason string
	}
	cases := []forgedCase{
		{"決定記録ではないfile", func(t *testing.T, root string, d Decision) (string, string) {
			return d.CandidateFingerprint, "README.md"
		}, true, "record-unreadable"},
		{"決定記録の外へ出るpath", func(t *testing.T, root string, d Decision) (string, string) {
			return d.CandidateFingerprint, decisionDir + "/../../tei.workspace.yaml"
		}, true, "record-unreadable"},
		{"決定記録の形ではないdirectory", func(t *testing.T, root string, d Decision) (string, string) {
			return d.CandidateFingerprint, decisionDir
		}, true, "record-unreadable"},
		{"採用の決定を指す", func(t *testing.T, root string, d Decision) (string, string) {
			accepted := writeRawDecision(t, root, Decision{Schema: decisionSchema, Decision: "accepted", CandidateFingerprint: d.CandidateFingerprint})
			return d.CandidateFingerprint, accepted
		}, true, "record-unreadable"},
		{"別の候補の退けを指す", func(t *testing.T, root string, d Decision) (string, string) {
			return "sha256:" + strings.Repeat("0", 64), d.RecordPath
		}, true, "record-unreadable"},
		{"候補の形ではないsubject", func(t *testing.T, root string, d Decision) (string, string) {
			return "not-a-fingerprint", d.RecordPath
		}, true, "record-unreadable"},
		{"決定記録が消されている（抑制は続く）", func(t *testing.T, root string, d Decision) (string, string) {
			return d.CandidateFingerprint, decisionDir + "/" + strings.Repeat("0", 64) + ".json"
		}, false, "rejected-proposal-basis-unchanged"},
	}
	for _, tc := range cases {
		root := scratchWorkspace(t)
		d := rejectViaCLI(t, root, unitPriceCandidate(revisionOfDir(t, root)), "今回は見送る")
		subject, linked := tc.forge(t, root, d)
		s, err := takeSnapshot(root)
		if err != nil {
			t.Fatal(err)
		}
		basis, _ := resolveBasis(s, []SubjectRef{proposalBasis})
		writeRawRecord(t, root, ConfirmationRecord{
			Schema:         confirmationSchema,
			Outcome:        outcomeRejectedProposal,
			Identity:       ConfirmationIdentity{Kind: identityKindProposal, Subject: subject},
			Basis:          basis,
			ModelRevision:  s.revision,
			Answer:         answerRejected,
			Reason:         "今回は見送る",
			DecidedBy:      ConfirmationActor{Kind: actorHuman, ID: "owner"},
			LinkedDecision: linked,
		})
		if c := checkProposal(t, root, d.CandidateFingerprint); c.Ask != tc.ask || c.Reason != tc.reason {
			t.Errorf("%s: ask=%v reason=%s を期待しましたが %+v", tc.name, tc.ask, tc.reason, c)
		}
	}
}
