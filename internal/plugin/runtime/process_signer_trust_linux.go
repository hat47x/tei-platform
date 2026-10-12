//go:build linux

package runtime

import (
	"crypto/ed25519"
	"errors"
	"sync"
)

// LinuxHostSignerTrustは、プラグインの署名データから独立してホストが管理する
// 署名検証鍵の集合である。署名参照を別の鍵へ交換したときは世代も更新する。
// 生成・更新・失効はホストの管理経路からのみ呼び出すこと。
type LinuxHostSignerTrust struct {
	mu      sync.RWMutex
	serial  uint64
	signers map[string]linuxTrustedSigner
}

type linuxTrustedSigner struct {
	key   ed25519.PublicKey
	epoch uint64
}

func NewLinuxHostSignerTrust() *LinuxHostSignerTrust {
	return &LinuxHostSignerTrust{signers: make(map[string]linuxTrustedSigner)}
}

// Allowはホストが許可した鍵を登録または交換する。鍵の元データを複製し、
// 呼出し元が後から公開鍵を変更しても信頼集合には影響させない。
// 同じ鍵で登録し直した場合も旧世代の承認を失効させる。
func (t *LinuxHostSignerTrust) Allow(ref string, public ed25519.PublicKey) error {
	if t == nil {
		return errors.New("Linux signer trust policy is missing")
	}
	if err := validateLinuxSignatureRef(ref); err != nil {
		return err
	}
	if len(public) != ed25519.PublicKeySize {
		return errors.New("Linux signer trust requires an Ed25519 public key")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.signers == nil {
		t.signers = make(map[string]linuxTrustedSigner)
	}
	t.serial++
	t.signers[ref] = linuxTrustedSigner{
		key:   append(ed25519.PublicKey(nil), public...),
		epoch: t.serial,
	}
	return nil
}

// Revokeは失効した参照に対応する鍵を削除する。直前に完了した処理へは
// 遡及しないが、完了後に開始する検証と起動では旧承認を使用できない。
func (t *LinuxHostSignerTrust) Revoke(ref string) bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.signers[ref]; !ok {
		return false
	}
	t.serial++
	delete(t.signers, ref)
	return true
}

func (t *LinuxHostSignerTrust) resolve(ref string) (ed25519.PublicKey, uint64, error) {
	if t == nil {
		return nil, 0, errors.New("Linux signer trust policy is missing")
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	entry, ok := t.signers[ref]
	if !ok {
		return nil, 0, errors.New("Linux signer reference is not trusted by Host")
	}
	return append(ed25519.PublicKey(nil), entry.key...), entry.epoch, nil
}

func (t *LinuxHostSignerTrust) current(ref string, epoch uint64) bool {
	if t == nil {
		return false
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	entry, ok := t.signers[ref]
	return ok && entry.epoch == epoch
}

// startIfCurrent serializes the authorization decision with the OS process
// start. Revoke and Allow use the exclusive lock; once either returns, a stale
// epoch cannot start a new child. The read lock is released immediately after
// Start, not after the child finishes, so existing work need not be drained
// before revocation is recorded.
//
// This provides Host-local start/revoke ordering. It does not stop children
// that are already started or synchronize trust across multiple Host processes.
func (t *LinuxHostSignerTrust) startIfCurrent(ref string, epoch uint64, start func() error) error {
	if t == nil {
		return errors.New("Linux signer trust policy is missing")
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	entry, ok := t.signers[ref]
	if !ok || entry.epoch != epoch {
		return errors.New("Host signer trust was revoked or changed")
	}
	return start()
}
