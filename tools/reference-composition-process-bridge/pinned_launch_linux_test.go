package compositionprocessbridge

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

const pinnedReferenceMaxBytes = 64 << 20

// pinnedReferenceHost is a Linux-only proof that verification and process launch
// can refer to one private, unlinked executable inode, not a mutable source path.
// It is intentionally not a production sandbox, signature verifier or Host ABI.
type pinnedReferenceHost struct {
	source     string
	digest     string
	executable *os.File
	launches   int
}

func pinReferenceExecutable(ctx context.Context, source, expected string) (*pinnedReferenceHost, error) {
	if len(expected) != len("sha256:")+64 || !strings.HasPrefix(expected, "sha256:") {
		return nil, fmt.Errorf("invalid expected SHA-256")
	}
	if _, err := hex.DecodeString(strings.TrimPrefix(expected, "sha256:")); err != nil {
		return nil, fmt.Errorf("invalid SHA-256 hex: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	src, err := os.Open(source)
	if err != nil {
		return nil, err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > pinnedReferenceMaxBytes {
		return nil, fmt.Errorf("source is not a bounded regular executable")
	}
	dir, err := os.MkdirTemp("", "tei-pinned-reference-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "executable")
	dst, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(dst, h), io.LimitReader(src, pinnedReferenceMaxBytes+1))
	if copyErr == nil && n > pinnedReferenceMaxBytes {
		copyErr = fmt.Errorf("source exceeds copy limit")
	}
	if copyErr == nil {
		copyErr = ctx.Err()
	}
	if copyErr == nil && expected != "sha256:"+hex.EncodeToString(h.Sum(nil)) {
		copyErr = errors.New("copied executable differs from expected digest")
	}
	if copyErr == nil {
		copyErr = dst.Chmod(0500)
	}
	if closeErr := dst.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return nil, copyErr
	}
	// Close the writer before opening a read-only descriptor: an executable
	// with an outstanding writable descriptor is rejected by Linux (ETXTBSY).
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if err := os.Remove(path); err != nil {
		fd.Close()
		return nil, err
	}
	return &pinnedReferenceHost{source: source, digest: expected, executable: fd}, nil
}

func (h *pinnedReferenceHost) Close() error {
	if h.executable == nil {
		return nil
	}
	err := h.executable.Close()
	h.executable = nil
	return err
}

func (h *pinnedReferenceHost) Verify(ctx context.Context, _ plugin.Identity, artifact plugin.Artifact) (pruntime.ArtifactVerification, error) {
	if err := ctx.Err(); err != nil {
		return pruntime.ArtifactVerification{}, err
	}
	if h.executable == nil || artifact.Digest != h.digest {
		return pruntime.ArtifactVerification{}, errors.New("locked artifact is not the pinned executable")
	}
	return pruntime.ArtifactVerification{VerifierID: "reference-pinned-inode-sha256", DigestVerified: true}, nil
}

func (h *pinnedReferenceHost) Invoke(ctx context.Context, command pruntime.ProcessCommand, req pruntime.Request) (pruntime.Response, error) {
	if h.executable == nil || command.Executable != h.source {
		return pruntime.Response{}, errors.New("process path does not match pinned executable")
	}
	input, err := json.Marshal(req)
	if err != nil {
		return pruntime.Response{}, err
	}
	// ExtraFiles installs fd 3 before execve. On Linux, /proc/self/fd/3 then
	// refers to the already-verified, unlinked inode rather than source path.
	cmd := exec.CommandContext(ctx, "/proc/self/fd/3", command.Args...)
	cmd.ExtraFiles = []*os.File{h.executable}
	cmd.Stdin = bytes.NewReader(append(input, '\n'))
	cmd.Stderr = io.Discard // never copy untrusted stderr or secrets into test errors
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	h.launches++
	if err := cmd.Run(); err != nil {
		return pruntime.Response{}, fmt.Errorf("pinned executable failed: %w", err)
	}
	if stdout.Len() > 1<<20 {
		return pruntime.Response{}, errors.New("reference response exceeds 1 MiB")
	}
	dec := json.NewDecoder(&stdout)
	var response pruntime.Response
	if err := dec.Decode(&response); err != nil {
		return pruntime.Response{}, err
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		return pruntime.Response{}, errors.New("reference response contains extra JSON")
	}
	return response, nil
}

func buildPinnedFixture(t *testing.T, ctx context.Context, dir, name string) (string, string) {
	t.Helper()
	path := filepath.Join(dir, name)
	build := exec.CommandContext(ctx, "go", "build", "-ldflags", "-X main.referenceRevision="+name, "-o", path, "../../plugins/reference-process-executor")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", name, err, out)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	return path, "sha256:" + hex.EncodeToString(sum[:])
}

func runPinnedReference(t *testing.T, ctx context.Context, h *pinnedReferenceHost, path, digest string) executor.Result {
	t.Helper()
	identity := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}
	capabilityID := core.MustCanonicalID("process.reference")
	plan := pruntime.EnvironmentPlan{
		Environment: core.MustCanonicalID("reference"), PluginLockFingerprint: "reference-pinned-lock",
		Plugins: []pruntime.PluginRuntime{{
			Plugin: identity, DescriptorFingerprint: "reference-pinned-descriptor",
			Boundary: pruntime.BoundaryProcess, Trust: pruntime.TrustExternalProcess,
			Command: path, Args: []string{referenceArgvMarker}, SandboxPolicy: plugin.SandboxNone,
			Artifact: plugin.Artifact{Digest: digest},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: identity, Capability: plugin.Capability{Kind: plugin.CapabilityDeploymentExecutor, ID: capabilityID,
				Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease}},
			Boundary: pruntime.BoundaryProcess,
		}},
	}
	plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
	adapter := pruntime.DeploymentExecutor{
		Plan: plan, Registry: pruntime.Registry{Verifier: h, Processes: h}, CapabilityID: capabilityID,
	}
	command := executor.Command{ID: "pinned-reference", ActionID: "sales_expand/apply", IdempotencyKey: "pinned-once",
		Kind: string(deployment.ActionDatabaseRelease), Database: core.MustCanonicalID("sales_core"), Version: "1.0.0"}
	result, err := adapter.Execute(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != executor.OutcomeAccepted || result.PluginInvocation == nil ||
		!result.PluginInvocation.DigestVerified || result.PluginInvocation.SignatureVerified || result.PluginInvocation.Sandboxed {
		t.Fatalf("unexpected pinned process evidence: %+v", result)
	}
	return result
}

func TestLinuxPinnedProcessSourceReplacement(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	dir := t.TempDir()
	original, expected := buildPinnedFixture(t, ctx, dir, "approved")
	replacement, replacementDigest := buildPinnedFixture(t, ctx, dir, "replaced")
	if expected == replacementDigest {
		t.Fatal("fixtures must have different bytes")
	}
	h, err := pinReferenceExecutable(ctx, original, expected)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	if err := os.Rename(replacement, original); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(original)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	if "sha256:"+hex.EncodeToString(sum[:]) != replacementDigest {
		t.Fatal("source replacement did not occur")
	}
	result := runPinnedReference(t, ctx, h, original, expected)
	if !strings.Contains(result.Message, "revision=approved") || strings.Contains(result.Message, "revision=replaced") || h.launches != 1 {
		t.Fatalf("source path substitution changed pinned executable: %+v launches=%d", result, h.launches)
	}
	if _, err := h.Verify(ctx, plugin.Identity{}, plugin.Artifact{Digest: replacementDigest}); err == nil {
		t.Fatal("host accepted changed declared artifact as the pinned bytes")
	}
}

func TestLinuxPinnedProcessSymlinkRetargetAndRevocation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	dir := t.TempDir()
	original, expected := buildPinnedFixture(t, ctx, dir, "approved")
	replacement, _ := buildPinnedFixture(t, ctx, dir, "replaced")
	link := filepath.Join(dir, "selected")
	if err := os.Symlink(original, link); err != nil {
		t.Fatal(err)
	}
	h, err := pinReferenceExecutable(ctx, link, expected)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(replacement, link); err != nil {
		t.Fatal(err)
	}
	result := runPinnedReference(t, ctx, h, link, expected)
	if !strings.Contains(result.Message, "revision=approved") {
		t.Fatalf("symlink retarget bypassed pin: %+v", result)
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Invoke(ctx, pruntime.ProcessCommand{Executable: link}, pruntime.Request{}); err == nil {
		t.Fatal("closed pinned executable still dispatched")
	}
	if h.launches != 1 {
		t.Fatal("closed host dispatched another process")
	}
}

func TestLinuxPinnedProcessRejectsWrongDigestAndSource(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	dir := t.TempDir()
	original, expected := buildPinnedFixture(t, ctx, dir, "approved")
	for _, bad := range []string{"", "sha256:123", "sha256:" + strings.Repeat("0", 64)} {
		if h, err := pinReferenceExecutable(ctx, original, bad); err == nil {
			h.Close()
			t.Fatalf("accepted mismatched digest %q", bad)
		}
	}
	h, err := pinReferenceExecutable(ctx, original, expected)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	if _, err := h.Invoke(ctx, pruntime.ProcessCommand{Executable: "/tmp/other"}, pruntime.Request{}); err == nil {
		t.Fatal("host launched a different source")
	}
	if h.launches != 0 {
		t.Fatal("denied launch reached process")
	}
	stopped, stop := context.WithCancel(ctx)
	stop()
	if _, err := pinReferenceExecutable(stopped, original, expected); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled pin: %v", err)
	}
}
