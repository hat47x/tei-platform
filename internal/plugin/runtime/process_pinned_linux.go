//go:build linux

package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"tei/internal/plugin"
)

const maxPinnedLinuxProcessBytes int64 = 64 << 20

// PinnedLinuxProcessは、Linuxのホストが自ら検証した実行内容を起動するための
// 選択式の処理実行器である。通常のStdioProcessLauncherを自動的に置換しない。
// 検証と起動には同一のファイル記述子を用い、署名・来歴・隔離は提供しない。
// ホストと同じ権限の攻撃者に対する保護も、この型だけでは保証しない。
type PinnedLinuxProcess struct {
	mu        sync.RWMutex
	identity  plugin.Identity
	source    string
	digest    string
	image     *os.File
	transport StdioProcessLauncher
}

var _ ProcessLauncher = (*PinnedLinuxProcess)(nil)
var _ ArtifactVerifier = (*PinnedLinuxProcess)(nil)

// boundProcessImage marks a runtime-held image; Registry requires the exact
// same instance to supply both its verifier and its execution launcher.
func (*PinnedLinuxProcess) boundProcessImage() {}

// PinLinuxProcessは実行ファイルをホスト専用領域に複製し、複製した内容の
// SHA-256を照合してから、名前を削除した実行ファイルを読み取り専用で保持する。
// 成果物が未検証の場合は起動可能なオブジェクトを返さない。
// 実行先は/proc/self/fdを利用できるLinuxに限る。
func PinLinuxProcess(ctx context.Context, identity plugin.Identity, source, expectedDigest string, transport StdioProcessLauncher) (*PinnedLinuxProcess, error) {
	if ctx == nil {
		return nil, errors.New("pinned Linux process requires context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if identity.ID == "" || identity.Version == "" {
		return nil, errors.New("pinned Linux process requires plugin identity and version")
	}
	if strings.TrimSpace(source) == "" || strings.ContainsAny(source, " \t\r\n") {
		return nil, errors.New("pinned Linux process requires a single executable path")
	}
	if len(expectedDigest) != len("sha256:")+64 || !strings.HasPrefix(expectedDigest, "sha256:") {
		return nil, errors.New("pinned Linux process requires sha256:<64-hex>")
	}
	if _, err := hex.DecodeString(strings.TrimPrefix(expectedDigest, "sha256:")); err != nil {
		return nil, fmt.Errorf("invalid pinned Linux process digest: %w", err)
	}
	sourceFile, err := os.Open(source)
	if err != nil {
		return nil, fmt.Errorf("open process executable: %w", err)
	}
	defer sourceFile.Close()
	info, err := sourceFile.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxPinnedLinuxProcessBytes {
		return nil, errors.New("pinned Linux process source must be a regular file of at most 64 MiB")
	}

	dir, err := os.MkdirTemp("", "tei-runtime-pinned-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	name := filepath.Join(dir, "image")
	writer, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	hash := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(writer, hash), io.LimitReader(sourceFile, maxPinnedLinuxProcessBytes+1))
	if copyErr == nil && n > maxPinnedLinuxProcessBytes {
		copyErr = errors.New("pinned Linux process source exceeded 64 MiB")
	}
	if copyErr == nil {
		copyErr = ctx.Err()
	}
	if copyErr == nil && "sha256:"+hex.EncodeToString(hash.Sum(nil)) != expectedDigest {
		copyErr = errors.New("pinned Linux process bytes differ from expected SHA-256")
	}
	if copyErr == nil {
		copyErr = writer.Chmod(0500)
	}
	closeErr := writer.Close()
	if copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return nil, copyErr
	}

	// 書込み用の記述子を閉じてから実行用の読取り専用記述子を開く。
	// その後名前を削除するため、元のパスの変更は起動内容に影響しない。
	image, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	if err := os.Remove(name); err != nil {
		image.Close()
		return nil, err
	}
	return &PinnedLinuxProcess{
		identity: identity, source: source, digest: expectedDigest,
		image: image, transport: transport,
	}, nil
}

// Verifyの肯定結果は、保持したファイルの内容と宣言されたSHA-256の一致のみ。
// 署名と来歴を確認済みとして返さず、Hostの別の信頼検査を代用しない。
func (p *PinnedLinuxProcess) Verify(ctx context.Context, identity plugin.Identity, artifact plugin.Artifact) (ArtifactVerification, error) {
	if ctx == nil {
		return ArtifactVerification{}, errors.New("pinned Linux process verification requires context")
	}
	if err := ctx.Err(); err != nil {
		return ArtifactVerification{}, err
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.image == nil || identity != p.identity || artifact.Digest != p.digest {
		return ArtifactVerification{}, errors.New("plugin identity or digest differs from pinned process")
	}
	return ArtifactVerification{VerifierID: "tei-linux-pinned-process-sha256", DigestVerified: true}, nil
}

// Invokeは照合済みの実行内容を子プロセスの記述子3へ引き渡して起動する。
// 読取りロックは子プロセス終了まで保持し、Closeによる早期解放を防ぐ。
// 隔離を必要とするプラグインは別途SandboxLauncherを設定しなければならない。
func (p *PinnedLinuxProcess) Invoke(ctx context.Context, command ProcessCommand, request Request) (Response, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.image == nil || command.Executable != p.source {
		return Response{}, errors.New("process executable is not the pinned image")
	}
	return p.transport.invokeExecutable(ctx, command, request, "/proc/self/fd/3", []*os.File{p.image})
}

// Closeは子プロセスの実行完了を待って、保持した実行内容を解放する。
// 既に解放した場合は成功として扱い、新たな呼出しは拒否する。
func (p *PinnedLinuxProcess) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.image == nil {
		return nil
	}
	err := p.image.Close()
	p.image = nil
	return err
}
