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
	signers map[string]*linuxTrustedSigner
}

type linuxTrustedSigner struct {
	key   ed25519.PublicKey
	epoch uint64
	gate  sync.RWMutex
}

func NewLinuxHostSignerTrust() *LinuxHostSignerTrust {
	return &LinuxHostSignerTrust{signers: make(map[string]*linuxTrustedSigner)}
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
	if t.signers == nil {
		t.signers = make(map[string]*linuxTrustedSigner)
	}
	previous := t.signers[ref]
	t.serial++
	t.signers[ref] = &linuxTrustedSigner{
		key:   append(ed25519.PublicKey(nil), public...),
		epoch: t.serial,
	}
	t.mu.Unlock()

	// map交換から旧世代への新規入場を拒否する。旧世代の処理が
	// 完了するまで待つが、無関係な鍵の更新は妨げない。
	if previous != nil {
		previous.gate.Lock()
		previous.gate.Unlock()
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
	previous, ok := t.signers[ref]
	if ok {
		t.serial++
		delete(t.signers, ref)
	}
	t.mu.Unlock()
	if !ok {
		return false
	}

	// 失効の完了時点には、旧世代で入場済みの処理も終了している。
	// 稼働中の子プロセスを強制停止する方式ではない。
	previous.gate.Lock()
	previous.gate.Unlock()
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

// invokeCurrentは署名鍵の現行世代に対する実行権限を保持して呼び出す。
// 失効・交換の完了は、この権限で入場済みの処理が終わるまで待つ。
// 信頼集合の共通ロックは実行中に保持せず、他の署名参照は更新できる。
// 呼び出した処理が終了しなければ失効も待つため、時間制限はHostの責務。
func (t *LinuxHostSignerTrust) invokeCurrent(ref string, epoch uint64, run func() (Response, error)) (Response, error) {
	if t == nil {
		return Response{}, errors.New("Linux signer trust policy is missing")
	}
	t.mu.RLock()
	entry, ok := t.signers[ref]
	if !ok || entry.epoch != epoch {
		t.mu.RUnlock()
		return Response{}, errors.New("Host signer trust was revoked or changed")
	}
	// mapを読取りロックしたまま、この世代だけの実行権限を確保する。
	entry.gate.RLock()
	t.mu.RUnlock()
	defer entry.gate.RUnlock()
	return run()
}
