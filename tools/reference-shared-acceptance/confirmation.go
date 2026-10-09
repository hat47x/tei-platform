package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tei/internal/sourceindex"
)

// 確認記録（ISSUE-000074、RQ-016）。
//
// 利用者やAIが出した確認と、その答えを記録し、同じ確認を理由なく繰り返さないための派生物である。
// 決定記録（.tei/decisions/）と同じく、managed sourceではないのでmodel revisionに入らず、
// 消してもaccepted meaningは変わらない。消したときに起きるのは、確認がもう一度出ることだけである。
//
// 再び確認するかどうかは、慣れではなく、答えた時点の前提から何が変わったかで決める。
// 疑わしいとき（読めない記録、履歴の分岐、前提の解決不能）は、確認を出す側へ倒す。
const (
	confirmationSchema      = "tei.reference.confirmation/v0"
	confirmationIndexSchema = "tei.reference.confirmation-index/v0"
	confirmationCheckSchema = "tei.reference.confirmation-check/v0"
	confirmationDir         = ".tei/confirmations"

	outcomeAnswered           = "answered"
	outcomeCategorySuppressed = "category-suppressed"
	outcomeCategoryReinstated = "category-reinstated"

	actorHuman = "human"
	actorAI    = "ai"
)

// SubjectRefは、確認の前提になるCanonical subjectである（Source Indexのsubject kind / ID）。
type SubjectRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

// ConfirmationIdentityは確認の同一性である。文面の言い回しからは決めない。
// category-*の記録では、Subjectを空にして、その種類の確認すべてを指す。
type ConfirmationIdentity struct {
	Kind    string `json:"kind"`
	Subject string `json:"subject,omitempty"`
}

// ConfirmationActorは答えた主体である。provenanceであり、authorityではない。
type ConfirmationActor struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

// BasisEntryは、答えた時点の前提を一つ固定する。現行のSource Indexはfile単位のfingerprintしか持たないので、
// subjectを宣言しているfileのfingerprintで固定する。
type BasisEntry struct {
	Subject     SubjectRef `json:"subject"`
	Artifact    string     `json:"artifact"`
	Fingerprint string     `json:"fingerprint"`
}

// ConfirmationRecordは保存される記録である。後から書き換えず、答えの変更や解除は新しい記録として足す。
type ConfirmationRecord struct {
	Schema         string               `json:"schema"`
	Outcome        string               `json:"outcome"`
	Identity       ConfirmationIdentity `json:"identity"`
	Basis          []BasisEntry         `json:"basis,omitempty"`
	ModelRevision  string               `json:"modelRevision"`
	Answer         string               `json:"answer,omitempty"`
	Reason         string               `json:"reason,omitempty"`
	DecidedBy      ConfirmationActor    `json:"decidedBy"`
	LinkedDecision string               `json:"linkedDecision,omitempty"`
	Supersedes     string               `json:"supersedes,omitempty"`
}

// ConfirmationInputは、記録を足すときの入力である。前提のfingerprintは入力せず、workspaceから取る。
type ConfirmationInput struct {
	Outcome        string               `json:"outcome"`
	Identity       ConfirmationIdentity `json:"identity"`
	BasisSubjects  []SubjectRef         `json:"basisSubjects,omitempty"`
	Answer         string               `json:"answer,omitempty"`
	Reason         string               `json:"reason,omitempty"`
	DecidedBy      ConfirmationActor    `json:"decidedBy"`
	LinkedDecision string               `json:"linkedDecision,omitempty"`
	Supersedes     string               `json:"supersedes,omitempty"`
}

// ConfirmationQueryは、確認を出す前に「出すべきか」を尋ねるときの入力である。
// Questionは表示用で、判定には使わない。言い回しが変わっても同じ確認として扱うためである。
type ConfirmationQuery struct {
	Kind     string `json:"kind"`
	Subject  string `json:"subject"`
	Question string `json:"question,omitempty"`
}

// ConfirmationRefusalは、記録を足せない理由である。
type ConfirmationRefusal struct {
	Code    string
	Message string
}

func (r ConfirmationRefusal) Error() string { return r.Code + ": " + r.Message }

func confirmationRefusal(code, format string, args ...any) error {
	return ConfirmationRefusal{Code: code, Message: fmt.Sprintf(format, args...)}
}

type BasisChange struct {
	Subject  SubjectRef `json:"subject"`
	Artifact string     `json:"artifact,omitempty"`
	Before   string     `json:"before,omitempty"`
	After    string     `json:"after,omitempty"`
}

type UnreadableConfirmation struct {
	Path    string `json:"path"`
	Problem string `json:"problem"`
}

// ConfirmationCheckは、確認を出すべきかの判定である。
type ConfirmationCheck struct {
	Schema         string                   `json:"schema"`
	Ask            bool                     `json:"ask"`
	Reason         string                   `json:"reason"`
	Message        string                   `json:"message"`
	Identity       ConfirmationIdentity     `json:"identity"`
	Record         string                   `json:"record,omitempty"`
	PreviousAnswer string                   `json:"previousAnswer,omitempty"`
	PreviousReason string                   `json:"previousReason,omitempty"`
	DecidedBy      *ConfirmationActor       `json:"decidedBy,omitempty"`
	Changed        []BasisChange            `json:"changed,omitempty"`
	Heads          []string                 `json:"heads,omitempty"`
	Unreadable     []UnreadableConfirmation `json:"unreadable,omitempty"`
}

type confirmationEntry struct {
	name string
	rel  string
	rec  ConfirmationRecord
}

func decodeStrict(raw []byte, into any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode(into)
}

func identityKey(id ConfirmationIdentity) string { return id.Kind + "\x00" + id.Subject }

func isCategory(outcome string) bool {
	return outcome == outcomeCategorySuppressed || outcome == outcomeCategoryReinstated
}

// validateRecordShapeは、保存された記録が、このtoolの書く形になっているかを検査する。
func validateRecordShape(rec ConfirmationRecord) error {
	if rec.Schema != confirmationSchema {
		return fmt.Errorf("schema %q は未対応です", rec.Schema)
	}
	switch rec.Outcome {
	case outcomeAnswered, outcomeCategorySuppressed, outcomeCategoryReinstated:
	default:
		return fmt.Errorf("outcome %q は未対応です", rec.Outcome)
	}
	if strings.TrimSpace(rec.Identity.Kind) == "" {
		return fmt.Errorf("identity.kindが空です")
	}
	if isCategory(rec.Outcome) != (rec.Identity.Subject == "") {
		return fmt.Errorf("identity.subjectは、answeredでは必須、category-*では空である必要があります")
	}
	if rec.DecidedBy.Kind != actorHuman && rec.DecidedBy.Kind != actorAI {
		return fmt.Errorf("decidedBy.kindは%sまたは%sです", actorHuman, actorAI)
	}
	if strings.TrimSpace(rec.DecidedBy.ID) == "" {
		return fmt.Errorf("decidedBy.idが空です")
	}
	return nil
}

// readConfirmationsは保存された記録を読む。読めない記録は、全体を止めずに列挙して返す。
// 呼び出し側が、疑わしいときに確認を出す側へ倒すかどうかを決める。
func readConfirmations(root string) ([]confirmationEntry, []UnreadableConfirmation, error) {
	dir := filepath.Join(root, filepath.FromSlash(confirmationDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	var out []confirmationEntry
	var bad []UnreadableConfirmation
	for _, name := range names {
		rel := confirmationDir + "/" + name
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			bad = append(bad, UnreadableConfirmation{Path: rel, Problem: err.Error()})
			continue
		}
		sum := sha256.Sum256(content)
		if expected := hex.EncodeToString(sum[:]) + ".json"; name != expected {
			bad = append(bad, UnreadableConfirmation{Path: rel, Problem: "content hashがfile名と一致しません"})
			continue
		}
		var rec ConfirmationRecord
		decoder := json.NewDecoder(bytes.NewReader(content))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&rec); err != nil {
			bad = append(bad, UnreadableConfirmation{Path: rel, Problem: err.Error()})
			continue
		}
		if err := validateRecordShape(rec); err != nil {
			bad = append(bad, UnreadableConfirmation{Path: rel, Problem: err.Error()})
			continue
		}
		out = append(out, confirmationEntry{name: name, rel: rel, rec: rec})
	}
	return out, bad, nil
}

// confirmationHeadsは、identityごとの「他の記録に置き換えられていない記録」を返す。
// supersedesが別のidentityの記録を指すなど、履歴として成り立たないものはinconsistentとして返す。
func confirmationHeads(entries []confirmationEntry) (map[string][]confirmationEntry, []UnreadableConfirmation) {
	byName := map[string]confirmationEntry{}
	for _, e := range entries {
		byName[e.name] = e
	}
	superseded := map[string]bool{}
	var bad []UnreadableConfirmation
	for _, e := range entries {
		if e.rec.Supersedes == "" {
			continue
		}
		target, ok := byName[e.rec.Supersedes]
		if !ok {
			continue // 置き換え元が消えていても、この記録自体は有効な先頭として扱う
		}
		if identityKey(target.rec.Identity) != identityKey(e.rec.Identity) {
			bad = append(bad, UnreadableConfirmation{Path: e.rel, Problem: "supersedesが別のidentityの記録を指しています"})
			continue
		}
		superseded[target.name] = true
	}
	heads := map[string][]confirmationEntry{}
	for _, e := range entries {
		if !superseded[e.name] {
			key := identityKey(e.rec.Identity)
			heads[key] = append(heads[key], e)
		}
	}
	return heads, bad
}

// resolveBasisは、前提のsubjectを、今のworkspaceでそれを宣言しているfileとそのfingerprintへ引く。
func resolveBasis(s snapshot, subjects []SubjectRef) ([]BasisEntry, []SubjectRef) {
	fingerprints := map[string]string{}
	for _, a := range s.index.Artifacts {
		fingerprints[a.Path] = a.Fingerprint
	}
	seen := map[SubjectRef]bool{}
	var out []BasisEntry
	var unresolved []SubjectRef
	for _, subject := range subjects {
		if seen[subject] {
			continue
		}
		seen[subject] = true
		paths := map[string]bool{}
		for _, ref := range s.index.ReferencesFor(sourceindex.Subject{Kind: subject.Kind, ID: subject.ID}) {
			paths[ref.ArtifactPath] = true
		}
		if len(paths) == 0 {
			unresolved = append(unresolved, subject)
			continue
		}
		for path := range paths {
			out = append(out, BasisEntry{Subject: subject, Artifact: path, Fingerprint: fingerprints[path]})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Subject.Kind != b.Subject.Kind {
			return a.Subject.Kind < b.Subject.Kind
		}
		if a.Subject.ID != b.Subject.ID {
			return a.Subject.ID < b.Subject.ID
		}
		return a.Artifact < b.Artifact
	})
	return out, unresolved
}

func normalizeConfirmationInput(in ConfirmationInput) (ConfirmationInput, error) {
	in.Identity.Kind = strings.TrimSpace(in.Identity.Kind)
	in.Identity.Subject = strings.TrimSpace(in.Identity.Subject)
	in.Answer = strings.TrimSpace(in.Answer)
	in.Reason = strings.TrimSpace(in.Reason)
	in.DecidedBy.ID = strings.TrimSpace(in.DecidedBy.ID)
	switch in.Outcome {
	case outcomeAnswered:
		if in.Identity.Subject == "" {
			return in, confirmationRefusal("IDENTITY_SUBJECT_REQUIRED", "answeredの記録にはidentity.subjectが必要です")
		}
		if in.Answer == "" {
			return in, confirmationRefusal("ANSWER_REQUIRED", "answeredの記録にはanswerが必要です")
		}
		if len(in.BasisSubjects) == 0 {
			return in, confirmationRefusal("BASIS_REQUIRED", "answeredの記録には前提のsubjectが一つ以上必要です。前提が無い答えは、何が変わっても確認を出さなくなります")
		}
	case outcomeCategorySuppressed, outcomeCategoryReinstated:
		if in.Identity.Subject != "" {
			return in, confirmationRefusal("IDENTITY_SUBJECT_FORBIDDEN", "category-*の記録は種類全体を指すため、identity.subjectを空にしてください")
		}
		if len(in.BasisSubjects) != 0 {
			return in, confirmationRefusal("BASIS_FORBIDDEN", "category-*の記録は前提のsubjectを持ちません")
		}
		if in.Outcome == outcomeCategorySuppressed && in.Reason == "" {
			return in, confirmationRefusal("REASON_REQUIRED", "抑制には理由が必要です。理由が無い抑制は、利用者が見ても解除すべきか判断できません")
		}
	default:
		return in, confirmationRefusal("OUTCOME_UNSUPPORTED", "outcome %q は未対応です", in.Outcome)
	}
	if in.Identity.Kind == "" {
		return in, confirmationRefusal("IDENTITY_KIND_REQUIRED", "identity.kindが必要です")
	}
	if in.DecidedBy.Kind != actorHuman && in.DecidedBy.Kind != actorAI {
		return in, confirmationRefusal("DECIDED_BY_INVALID", "decidedBy.kindは%sまたは%sです", actorHuman, actorAI)
	}
	if in.DecidedBy.ID == "" {
		return in, confirmationRefusal("DECIDED_BY_INVALID", "decidedBy.idが必要です")
	}
	return in, nil
}

// recordConfirmationは、確認の答えを派生物として保存し、保存先のworkspace相対pathを返す。
// 同じidentityに有効な記録が既にある場合は、supersedesでそれを置き換える必要がある。
// 黙って履歴を分岐させず、前の記録は消さない。
func recordConfirmation(root string, in ConfirmationInput) (string, error) {
	in, err := normalizeConfirmationInput(in)
	if err != nil {
		return "", err
	}
	s, err := takeSnapshot(root)
	if err != nil {
		return "", err
	}
	entries, unreadable, err := readConfirmations(root)
	if err != nil {
		return "", err
	}
	if len(unreadable) != 0 {
		return "", confirmationRefusal("STORE_UNREADABLE", "読めない確認記録があるため、記録を足しません: %s（%s）", unreadable[0].Path, unreadable[0].Problem)
	}

	heads, inconsistent := confirmationHeads(entries)
	if len(inconsistent) != 0 {
		return "", confirmationRefusal("STORE_INCONSISTENT", "履歴として成り立たない確認記録があるため、記録を足しません: %s（%s）", inconsistent[0].Path, inconsistent[0].Problem)
	}
	current := heads[identityKey(in.Identity)]
	switch {
	case len(current) > 1:
		return "", confirmationRefusal("HISTORY_FORKED", "identityの履歴が分岐しています（先頭が%d件）。置き換えを先に整理してください", len(current))
	case len(current) == 1 && in.Supersedes == "":
		return "", confirmationRefusal("MUST_SUPERSEDE", "このidentityには有効な記録 %s があります。答えを変える場合はsupersedesにその名前を指定してください", current[0].name)
	case len(current) == 1 && in.Supersedes != current[0].name:
		return "", confirmationRefusal("SUPERSEDES_NOT_HEAD", "supersedes %q は先頭の記録ではありません。先頭は %s です", in.Supersedes, current[0].name)
	case len(current) == 0 && in.Supersedes != "":
		return "", confirmationRefusal("SUPERSEDES_NOT_FOUND", "supersedes %q に対応する有効な記録がありません", in.Supersedes)
	}

	if in.LinkedDecision != "" {
		if !strings.HasPrefix(in.LinkedDecision, decisionDir+"/") {
			return "", confirmationRefusal("LINKED_DECISION_INVALID", "linkedDecisionは%s/配下の決定記録を指す必要があります", decisionDir)
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(in.LinkedDecision))); err != nil {
			return "", confirmationRefusal("LINKED_DECISION_NOT_FOUND", "linkedDecision %q が存在しません", in.LinkedDecision)
		}
	}

	var basis []BasisEntry
	if in.Outcome == outcomeAnswered {
		var unresolved []SubjectRef
		basis, unresolved = resolveBasis(s, in.BasisSubjects)
		if len(unresolved) != 0 {
			return "", confirmationRefusal("BASIS_SUBJECT_UNRESOLVED", "前提のsubject %s:%s を宣言しているsourceが見つかりません。名前の類似から補完しません", unresolved[0].Kind, unresolved[0].ID)
		}
	}

	rec := ConfirmationRecord{
		Schema: confirmationSchema, Outcome: in.Outcome, Identity: in.Identity, Basis: basis,
		ModelRevision: s.revision, Answer: in.Answer, Reason: in.Reason, DecidedBy: in.DecidedBy,
		LinkedDecision: in.LinkedDecision, Supersedes: in.Supersedes,
	}
	encoded, err := encodeJSON(rec)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	rel := confirmationDir + "/" + hex.EncodeToString(sum[:]) + ".json"
	target := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	if err := writeFileAtomic(target, encoded); err != nil {
		return "", err
	}
	return rel, nil
}

// checkConfirmationは、確認を出すべきかを返す。判定に使うのは、identity（種類と対象）と、
// 答えた時点の前提が今も同じかだけである。確認の文面と、記録の古さは使わない。
func checkConfirmation(root string, q ConfirmationQuery) (ConfirmationCheck, error) {
	q.Kind, q.Subject = strings.TrimSpace(q.Kind), strings.TrimSpace(q.Subject)
	if q.Kind == "" || q.Subject == "" {
		return ConfirmationCheck{}, confirmationRefusal("QUERY_INVALID", "kindとsubjectが必要です")
	}
	identity := ConfirmationIdentity{Kind: q.Kind, Subject: q.Subject}
	result := ConfirmationCheck{Schema: confirmationCheckSchema, Identity: identity}
	ask := func(reason, message string) (ConfirmationCheck, error) {
		result.Ask, result.Reason, result.Message = true, reason, message
		return result, nil
	}

	s, err := takeSnapshot(root)
	if err != nil {
		return result, err
	}
	entries, unreadable, err := readConfirmations(root)
	if err != nil {
		return result, err
	}
	heads, inconsistent := confirmationHeads(entries)
	result.Unreadable = append(unreadable, inconsistent...)
	if len(result.Unreadable) != 0 {
		return ask("record-unreadable", "確認記録に読めないものがあるため、抑制せずに確認を出します")
	}

	category := heads[identityKey(ConfirmationIdentity{Kind: q.Kind})]
	if len(category) > 1 {
		return ask("ambiguous-history", "この種類の抑制の履歴が分岐しているため、確認を出します")
	}
	if len(category) == 1 && category[0].rec.Outcome == outcomeCategorySuppressed {
		result.Record, result.PreviousReason = category[0].rel, category[0].rec.Reason
		result.DecidedBy = &category[0].rec.DecidedBy
		result.Ask, result.Reason = false, "category-suppressed"
		result.Message = "この種類の確認は抑制されています。解除するにはcategory-reinstatedの記録を足してください"
		return result, nil
	}

	own := heads[identityKey(identity)]
	switch {
	case len(own) == 0:
		return ask("no-record", "この確認への答えの記録がありません")
	case len(own) > 1:
		for _, e := range own {
			result.Heads = append(result.Heads, e.rel)
		}
		return ask("ambiguous-history", "この確認の履歴が分岐しているため、確認を出します")
	}

	head := own[0]
	result.Record, result.PreviousAnswer, result.PreviousReason = head.rel, head.rec.Answer, head.rec.Reason
	result.DecidedBy = &head.rec.DecidedBy
	if head.rec.DecidedBy.Kind != actorHuman {
		return ask("answer-not-by-human", "この答えは人が決めたものではないため、確認の抑制には使いません。人の答えが必要です")
	}

	var subjects []SubjectRef
	for _, b := range head.rec.Basis {
		subjects = append(subjects, b.Subject)
	}
	current, unresolved := resolveBasis(s, subjects)
	if len(unresolved) != 0 {
		for _, u := range unresolved {
			result.Changed = append(result.Changed, BasisChange{Subject: u})
		}
		return ask("basis-unresolved", "答えた時点の前提のsubjectが、今は見つかりません")
	}
	type key struct{ kind, id, artifact string }
	now := map[key]string{}
	for _, b := range current {
		now[key{b.Subject.Kind, b.Subject.ID, b.Artifact}] = b.Fingerprint
	}
	recorded := map[key]bool{}
	for _, b := range head.rec.Basis {
		k := key{b.Subject.Kind, b.Subject.ID, b.Artifact}
		recorded[k] = true
		if now[k] != b.Fingerprint {
			result.Changed = append(result.Changed, BasisChange{Subject: b.Subject, Artifact: b.Artifact, Before: b.Fingerprint, After: now[k]})
		}
	}
	for _, b := range current {
		if !recorded[key{b.Subject.Kind, b.Subject.ID, b.Artifact}] {
			result.Changed = append(result.Changed, BasisChange{Subject: b.Subject, Artifact: b.Artifact, After: b.Fingerprint})
		}
	}
	if len(result.Changed) != 0 {
		return ask("basis-changed", "答えた時点から前提が変わっています。前回の答えと変化点を添えて確認を出します")
	}
	result.Ask, result.Reason = false, "answered-basis-unchanged"
	result.Message = "答えた時点の前提が変わっていないため、確認を出しません"
	return result, nil
}

// ConfirmationListEntryは、一覧に出す記録と、その現在の状態である。
type ConfirmationListEntry struct {
	Path   string             `json:"path"`
	Status string             `json:"status"` // current / superseded
	Record ConfirmationRecord `json:"record"`
}

type ConfirmationIndex struct {
	Schema               string                  `json:"schema"`
	ModelRevision        string                  `json:"modelRevision"`
	Records              []ConfirmationListEntry `json:"records"`
	SuppressedCategories []string                `json:"suppressedCategories"`
}

// listConfirmationsは、利用者が抑制と答えを見て、解除や変更の判断をするための一覧を返す。
// 読めない記録があれば、黙って飛ばさずerrorにする（decisionsと同じ）。
func listConfirmations(root string) (ConfirmationIndex, error) {
	index := ConfirmationIndex{Schema: confirmationIndexSchema, Records: []ConfirmationListEntry{}, SuppressedCategories: []string{}}
	s, err := takeSnapshot(root)
	if err != nil {
		return index, err
	}
	index.ModelRevision = s.revision
	entries, unreadable, err := readConfirmations(root)
	if err != nil {
		return index, err
	}
	heads, inconsistent := confirmationHeads(entries)
	if bad := append(unreadable, inconsistent...); len(bad) != 0 {
		return index, fmt.Errorf("確認記録 %s を読めません: %s", bad[0].Path, bad[0].Problem)
	}
	isHead := map[string]bool{}
	for key, list := range heads {
		for _, e := range list {
			isHead[e.name] = true
		}
		if len(list) == 1 && list[0].rec.Outcome == outcomeCategorySuppressed {
			index.SuppressedCategories = append(index.SuppressedCategories, strings.SplitN(key, "\x00", 2)[0])
		}
	}
	sort.Strings(index.SuppressedCategories)
	for _, e := range entries {
		status := "superseded"
		if isHead[e.name] {
			status = "current"
		}
		index.Records = append(index.Records, ConfirmationListEntry{Path: e.rel, Status: status, Record: e.rec})
	}
	return index, nil
}
