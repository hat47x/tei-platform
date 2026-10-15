//go:build linux

package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"

	"tei/internal/plugin"
)

// LinuxHostStartAdmission is provided by the trusted Host for ONE selected
// Environment, exact Runtime Capability and current Authority generation.
// Its implementation must serialize its current Authority check with Host
// revocation, and call start exactly once while the admission is held.
//
// This is a start-time mechanism, NOT a new source of Permission Grants.
// A process/plugin/request must never manufacture or select this function.
// It is optional for the Linux pinned-process host; other transports require
// their own target-specific start/admission integration.
type LinuxHostStartAdmission func(context.Context, func() error) error

// HostAdmittedPinnedLinuxProcess is an opt-in adapter around the EXISTING
// verified Linux process image and stdio transport. It supplies the same
// object as ArtifactVerifier and SandboxedProcessRunner, while introducing a
// Host-owned admission check at the physical OS Start boundary.
//
// The adapter does not validate or issue Authority evidence, and does not
// replace a signed image's independently enforced Host signer-trust gate.
// A trusted Host must bind the admission callback to the exact Plan and scope.
type HostAdmittedPinnedLinuxProcess struct {
	image *PinnedLinuxProcess
	admit LinuxHostStartAdmission
}

var _ ArtifactVerifier = (*HostAdmittedPinnedLinuxProcess)(nil)
var _ ProcessLauncher = (*HostAdmittedPinnedLinuxProcess)(nil)
var _ SandboxedProcessRunner = (*HostAdmittedPinnedLinuxProcess)(nil)

// Registry recognizes this as a held image. A Registry verifier and Sandbox
// runner must be the SAME adapter object, not independent wrappers around
// the same image. That rule cannot be bypassed through the marker.
func (*HostAdmittedPinnedLinuxProcess) boundProcessImage() {}

// NewHostAdmittedPinnedLinuxProcess binds a Host-supplied Start admission
// callback. The image is still owned/closed by its original Host handle.
func NewHostAdmittedPinnedLinuxProcess(
	image *PinnedLinuxProcess, admit LinuxHostStartAdmission,
) (*HostAdmittedPinnedLinuxProcess, error) {
	if image == nil || admit == nil {
		return nil, errors.New("Linux Host Start admission requires pinned image and trusted Host gate")
	}
	image.mu.RLock()
	defer image.mu.RUnlock()
	if image.image == nil {
		return nil, errors.New("Linux Host Start admission requires live pinned image")
	}
	return &HostAdmittedPinnedLinuxProcess{image: image, admit: admit}, nil
}

func (r *HostAdmittedPinnedLinuxProcess) Verify(
	ctx context.Context, identity plugin.Identity, artifact plugin.Artifact,
) (ArtifactVerification, error) {
	if r == nil || r.image == nil {
		return ArtifactVerification{}, errors.New("Linux Host Start admission verifier is missing")
	}
	return r.image.Verify(ctx, identity, artifact)
}

// InvokeSandboxed is called after the existing HostSandboxLauncher applies
// host-enforced policy adapters. The adapter does not itself provide sandbox
// isolation; no policy is inferred from a successful Start admission.
func (r *HostAdmittedPinnedLinuxProcess) InvokeSandboxed(
	ctx context.Context, command ProcessCommand, request Request,
) (Response, error) {
	return r.Invoke(ctx, command, request)
}

func (r *HostAdmittedPinnedLinuxProcess) Invoke(
	ctx context.Context, command ProcessCommand, request Request,
) (Response, error) {
	if r == nil || r.image == nil || r.admit == nil {
		return Response{}, errors.New("Linux Host Start admission runner is missing")
	}
	if ctx == nil {
		return Response{}, errors.New("Linux Host Start admission requires context")
	}
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}
	p := r.image
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.image == nil || command.Executable != p.source {
		return Response{}, errors.New("process executable is not the pinned image")
	}
	startGate := func(start func() error) error {
		return r.admit(ctx, func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			// Nest existing signer trust INSIDE Environment admission.
			// The signer trust gate, if present, stays held through OS Start.
			// Both gates finish before cmd.Wait; neither holds a Grant lock
			// for the whole child lifetime.
			if p.signatureVerified {
				return p.signerTrust.startIfCurrent(p.signatureRef, p.signerEpoch, start)
			}
			return start()
		})
	}
	response, err := p.transport.invokeExecutableWithStartGate(
		ctx, command, request, "/proc/self/fd/3", []*os.File{p.image}, startGate,
	)
	if err != nil {
		return response, fmt.Errorf("Host-admitted pinned Linux process: %w", err)
	}
	return response, nil
}
