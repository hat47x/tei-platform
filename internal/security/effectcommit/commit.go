// Package effectcommit defines the transaction boundary required by authorized
// application writes. Its interfaces are not evidence that any Provider has
// actually supplied an atomic or durable implementation.
package effectcommit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"tei/internal/security/legacyauthorization"
)

var (
	ErrInvalidRequest    = errors.New("副作用の要求または認可根拠が不正です")
	ErrResourceConflict  = errors.New("更新対象の版が一致しません")
	ErrMissingProvider   = errors.New("確定境界を持つ保存先がありません")
)

// CommitUncertainError は確定要求の応答を確認できないことを示す。
// このエラーを「書込み失敗」「業務拒否」として扱ってはならない。
// 新規の別OperationIDで再送せず、同じOperationIDで記録照会する。
type CommitUncertainError struct{ Cause error }

func (e *CommitUncertainError) Error() string {
	return fmt.Sprintf("保存先の確定結果を確認できません: %v", e.Cause)
}
func (e *CommitUncertainError) Unwrap() error { return e.Cause }

type Scope struct {
	Application string
	Capability  string
	Subject     string
	Action      string
	Resource    string
}

// Requestは一回の副作用の不変の入力を保持する。
// OperationIDは保存先の一意制約で守り、同じIDで違う内容を再送してはならない。
type Request struct {
	Scope           Scope
	OperationID     string
	ExpectedVersion uint64
	Payload         []byte
}

type Receipt struct {
	Scope           Scope
	OperationID     string
	RequestHash     string
	ResourceVersion uint64
}

type Status string

const (
	StatusCommitted   Status = "committed"
	StatusReplayed    Status = "replayed"
	StatusDenied      Status = "denied"
	StatusConflict    Status = "resource-conflict"
	StatusNotRecorded Status = "not-recorded"
)

type Result struct {
	Status  Status
	Receipt Receipt
}

// Transactionは同じ物理トランザクション上で以下の順序を実装する。
// CheckAuthorityは対象の認可正本版と主体リンクを検証して確定まで変更を
// 直列化する。FindReceiptはOperationIDの一意制約を使用する。
// WriteResourceはExpectedVersionをCASで比較する。
// SaveReceiptはWriteResourceと同じトランザクションに書き込む。
// どれかを別接続・別サービスに逃がす実装はこの契約に適合しない。
type Transaction interface {
	CheckAuthority(context.Context, Scope, legacyauthorization.AuthorityEpochs) (bool, error)
	FindReceipt(context.Context, string) (Receipt, bool, error)
	WriteResource(context.Context, Request) (uint64, error)
	SaveReceipt(context.Context, Receipt) error
	Commit(context.Context) error
	Rollback() error
}

type Provider interface {
	Begin(context.Context) (Transaction, error)
}

// Runnerは同一トランザクション内の検査順序を統一する。
// 「認可正本と副作用が本当に同じDBにある」ことは別の適合性証拠が必要。
// grantは呼出側が直接生成してはならず、信頼されたsecurity.authorizeの
// 成功結果から受け取る。Application/Capabilityは保存側がScopeで検証する。
type Runner struct{ Provider Provider }

func validScope(s Scope) bool {
	return validText(s.Application) && validText(s.Capability) &&
		validText(s.Subject) && validText(s.Action) && validText(s.Resource)
}
func validText(s string) bool {
	return s != "" && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\x00\r\n")
}
func validate(req Request, grant legacyauthorization.AuthorizationGrant) error {
	if !validScope(req.Scope) || !validText(req.OperationID) ||
		req.ExpectedVersion == math.MaxUint64 {
		return ErrInvalidRequest
	}
	if grant.SubjectID != req.Scope.Subject || grant.Action != req.Scope.Action ||
		grant.Resource != req.Scope.Resource || grant.Epochs == nil ||
		!validText(grant.Epochs.Source) || grant.Epochs.SubjectLink == 0 ||
		grant.Epochs.Membership == 0 || grant.Epochs.Policy == 0 {
		return ErrInvalidRequest
	}
	return nil
}

func digest(req Request) (string, error) {
	// JSONによる固定された構造の直列化で、Scope・操作ID・対象版・内容を結ぶ。
	data, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

func matches(rec Receipt, req Request, hash string) bool {
	return rec.OperationID == req.OperationID && rec.Scope == req.Scope &&
		rec.RequestHash == hash && rec.ResourceVersion != 0
}

// Executeは権限版確認→確定記録再送確認→版付きデータ更新→確定記録→Commit。
// 失効・版衝突では記録を開示せず、Commitでの障害は結果不明として返す。
func (r Runner) Execute(ctx context.Context, req Request, grant legacyauthorization.AuthorizationGrant) (Result, error) {
	if err := ctx.Err(); err != nil { return Result{}, err }
	if err := validate(req, grant); err != nil { return Result{}, err }
	if r.Provider == nil { return Result{}, ErrMissingProvider }
	hash, err := digest(req)
	if err != nil { return Result{}, err }
	tx, err := r.Provider.Begin(ctx)
	if err != nil { return Result{}, err }
	if tx == nil { return Result{}, ErrMissingProvider }
	defer tx.Rollback()

	allowed, err := tx.CheckAuthority(ctx, req.Scope, *grant.Epochs)
	if err != nil { return Result{}, err }
	if !allowed { return Result{Status: StatusDenied}, nil }
	old, exists, err := tx.FindReceipt(ctx, req.OperationID)
	if err != nil { return Result{}, err }
	if exists {
		if !matches(old, req, hash) { return Result{Status: StatusConflict}, nil }
		// 実装がロックを確保するまで認可版を保持している必要がある。
		if err := tx.Commit(ctx); err != nil {
			return Result{}, &CommitUncertainError{Cause: err}
		}
		return Result{Status: StatusReplayed, Receipt: old}, nil
	}

	next, err := tx.WriteResource(ctx, cloneRequest(req))
	if errors.Is(err, ErrResourceConflict) { return Result{Status: StatusConflict}, nil }
	if err != nil { return Result{}, err }
	if next != req.ExpectedVersion+1 { return Result{}, fmt.Errorf("保存先が期待した対象版を確定できません") }
	rec := Receipt{Scope: req.Scope, OperationID: req.OperationID, RequestHash: hash, ResourceVersion: next}
	if err := tx.SaveReceipt(ctx, rec); err != nil { return Result{}, err }
	if err := tx.Commit(ctx); err != nil {
		return Result{}, &CommitUncertainError{Cause: err}
	}
	return Result{Status: StatusCommitted, Receipt: rec}, nil
}

// Lookupは認可を現在の正本で再検査してから、同一のDB境界で記録を照会する。
// not-recordedは同DBで記録がないという意味だけで、外部副作用の不存在ではない。
func (r Runner) Lookup(ctx context.Context, req Request, grant legacyauthorization.AuthorizationGrant) (Result, error) {
	if err := ctx.Err(); err != nil { return Result{}, err }
	if err := validate(req, grant); err != nil { return Result{}, err }
	if r.Provider == nil { return Result{}, ErrMissingProvider }
	hash, err := digest(req)
	if err != nil { return Result{}, err }
	tx, err := r.Provider.Begin(ctx)
	if err != nil { return Result{}, err }
	if tx == nil { return Result{}, ErrMissingProvider }
	defer tx.Rollback()

	allowed, err := tx.CheckAuthority(ctx, req.Scope, *grant.Epochs)
	if err != nil { return Result{}, err }
	if !allowed { return Result{Status: StatusDenied}, nil }
	rec, found, err := tx.FindReceipt(ctx, req.OperationID)
	if err != nil { return Result{}, err }
	if found && !matches(rec, req, hash) { return Result{Status: StatusConflict}, nil }
	if err := tx.Commit(ctx); err != nil { return Result{}, err }
	if !found { return Result{Status: StatusNotRecorded}, nil }
	return Result{Status: StatusReplayed, Receipt: rec}, nil
}

func cloneRequest(req Request) Request {
	req.Payload = append([]byte(nil), req.Payload...)
	return req
}
