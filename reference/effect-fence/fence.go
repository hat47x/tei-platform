// Package effectfence は、権限失効と状態更新の競合を扱う参照実装である。
// ここでの権限情報は試験用の模擬状態であり、本番の認可基盤ではない。
package effectfence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Scope は、認可結果が許可した操作の範囲を表す。
// 実際の呼出し主体は、認証済み実行文脈から別途確定する必要がある。
type Scope struct {
	Application string
	Capability  string
	Subject     string
	Action      string
	Resource    string
}

func validToken(value string) bool {
	return strings.TrimSpace(value) != "" && !strings.ContainsAny(value, "\x00\r\n")
}

func (s Scope) valid() bool {
	return validToken(s.Application) && validToken(s.Capability) && validToken(s.Subject) &&
		validToken(s.Action) && validToken(s.Resource)
}

type Request struct {
	Scope                   Scope
	OperationID             string
	ExpectedResourceVersion uint64
	Value                   string
}

// Permit は事前検査済みの操作を表す不透明な識別子である。
// 外部からゼロ値を構築しても書込み権限にならない。
type Permit struct {
	id    uint64
	owner *MemoryStore
}

// Disposition は確定、既存結果の再照会、無作用での拒否を区別する。
// 外部サービスで結果が不明な状態をこの模擬実装の「拒否」に読み替えてはならない。
type Disposition string

const (
	Prepared  Disposition = "prepared"
	Committed Disposition = "committed"
	Replayed  Disposition = "replayed"
	Denied    Disposition = "denied"
)

// Receipt に値・認証情報・アクセストークンを含めない。
type Receipt struct {
	OperationID     string
	Application     string
	Resource        string
	ResourceVersion uint64
}

type Result struct {
	Disposition Disposition
	Reason      string
	Receipt     Receipt
}

type authorityState struct {
	version uint64
	allowed bool
}

type resourceState struct {
	value   string
	version uint64
}

type prepared struct {
	request          Request
	authorityVersion uint64
}

type completed struct {
	request Request
	receipt Receipt
}

// MemoryStore は、模擬認可版と保存先の版を一つの排他境界で管理する。
// 別々のDB・外部API・複数プロセス間にこの原子性は及ばない。
type MemoryStore struct {
	mu        sync.Mutex
	sequence  uint64
	authority map[Scope]authorityState
	resources map[string]resourceState
	pending   map[uint64]prepared
	completed map[string]completed
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		authority: make(map[Scope]authorityState),
		resources: make(map[string]resourceState),
		pending:   make(map[uint64]prepared),
		completed: make(map[string]completed),
	}
}

func resourceKey(s Scope) string { return s.Application + "\x00" + s.Resource }

// SetAuthority は試験用の模擬権限を更新する。
// 同一Scopeの版は必ず単調増加させ、失効と版変更を一度に確定する。
func (m *MemoryStore) SetAuthority(s Scope, version uint64, allowed bool) error {
	if !s.valid() || version == 0 {
		return errors.New("権限の範囲または版が不正です")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if prev, exists := m.authority[s]; exists && version <= prev.version {
		return errors.New("権限の版は単調増加が必要です")
	}
	m.authority[s] = authorityState{version: version, allowed: allowed}
	return nil
}

// Prepare は、参照用の権限状態を照合して操作内容を固定する。
// 実Providerでは既存のsecurity.authorize等で検証済みのGrantを受ける必要がある。
func (m *MemoryStore) Prepare(ctx context.Context, req Request) (Permit, Result, error) {
	if err := ctx.Err(); err != nil {
		return Permit{}, Result{}, err
	}
	if !req.Scope.valid() || !validToken(req.OperationID) {
		return Permit{}, Result{Disposition: Denied, Reason: "invalid-request"}, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Permit{}, Result{}, err
	}
	grant, ok := m.authority[req.Scope]
	if !ok || !grant.allowed {
		return Permit{}, Result{Disposition: Denied, Reason: "authority-unavailable"}, nil
	}
	if done, exists := m.completed[req.OperationID]; exists {
		if done.request != req {
			return Permit{}, Result{Disposition: Denied, Reason: "operation-id-conflict"}, nil
		}
		return Permit{}, Result{Disposition: Replayed, Receipt: done.receipt}, nil
	}
	if m.sequence == ^uint64(0) {
		return Permit{}, Result{}, errors.New("操作識別子の内部連番が上限に達しました")
	}
	m.sequence++
	p := Permit{id: m.sequence, owner: m}
	m.pending[p.id] = prepared{request: req, authorityVersion: grant.version}
	return p, Result{Disposition: Prepared}, nil
}

// Commit は、権限版・保存先版の照合と書込みを同一ロック内で行う。
// エラーは失効や業務拒否と異なり、処理の不成立・障害を示す。
func (m *MemoryStore) Commit(ctx context.Context, p Permit) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	entry, ok := m.pending[p.id]
	if !ok || p.id == 0 || p.owner != m {
		return Result{Disposition: Denied, Reason: "unknown-permit"}, nil
	}
	delete(m.pending, p.id)
	req := entry.request
	// 確定済み操作の再照会でも、失効前の許可を流用してはならない。
	grant, exists := m.authority[req.Scope]
	if !exists || !grant.allowed || grant.version != entry.authorityVersion {
		return Result{Disposition: Denied, Reason: "authority-changed"}, nil
	}
	if done, exists := m.completed[req.OperationID]; exists {
		if done.request != req {
			return Result{Disposition: Denied, Reason: "operation-id-conflict"}, nil
		}
		return Result{Disposition: Replayed, Receipt: done.receipt}, nil
	}
	key := resourceKey(req.Scope)
	current := m.resources[key]
	if current.version != req.ExpectedResourceVersion {
		return Result{Disposition: Denied, Reason: "resource-version-changed"}, nil
	}
	if current.version == ^uint64(0) {
		return Result{}, fmt.Errorf("保存先の版が上限に達しました")
	}
	current.version++
	current.value = req.Value
	m.resources[key] = current
	receipt := Receipt{OperationID: req.OperationID, Application: req.Scope.Application, Resource: req.Scope.Resource, ResourceVersion: current.version}
	m.completed[req.OperationID] = completed{request: req, receipt: receipt}
	return Result{Disposition: Committed, Receipt: receipt}, nil
}

// Lookup は、応答が失われた場合に確定済み操作を再照会する。
// 試験用の現在権限で範囲を検査する。実Providerでは照会専用の権限も必要になる。
// プロセス再起動後に照会するには、別途耐久記録が必要となる。
func (m *MemoryStore) Lookup(s Scope, operationID string) (Receipt, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	grant, ok := m.authority[s]
	if !ok || !grant.allowed {
		return Receipt{}, false
	}
	v, ok := m.completed[operationID]
	if !ok || v.request.Scope != s {
		return Receipt{}, false
	}
	return v.receipt, true
}

// Observe は独立した読取りを模した確認口であり、実Providerの外部観測ではない。
func (m *MemoryStore) Observe(s Scope) (string, uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v := m.resources[resourceKey(s)]
	return v.value, v.version
}

// AdvanceResource は別の書き手による更新を模す試験用入口である。
func (m *MemoryStore) AdvanceResource(s Scope, value string) error {
	if !s.valid() {
		return errors.New("保存先の範囲が不正です")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := resourceKey(s)
	v := m.resources[key]
	if v.version == ^uint64(0) {
		return errors.New("保存先の版が上限に達しました")
	}
	v.version++
	v.value = value
	m.resources[key] = v
	return nil
}
