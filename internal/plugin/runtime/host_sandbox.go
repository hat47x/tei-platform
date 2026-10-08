package runtime

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"tei/internal/resource"
)

type Cleanup func(context.Context) error

type NetworkPolicy struct {
	DenyByDefault bool     `json:"denyByDefault"`
	Allowlist     []string `json:"allowlist,omitempty"`
}

type FilesystemPolicy struct {
	DenyByDefault bool           `json:"denyByDefault"`
	Mounts        []SandboxMount `json:"mounts,omitempty"`
}

// NetworkEnforcer and FilesystemEnforcer are host boundaries. TEI constructs a
// least-authority policy, but only the host can install an OS/container/service-
// mesh rule that actually enforces it.
type NetworkEnforcer interface {
	ApplyNetworkPolicy(context.Context, NetworkPolicy) (Cleanup, error)
}

type NetworkPolicyRefresher interface {
	RefreshNetworkPolicy(context.Context, NetworkPolicy, NetworkPolicy) error
}

type FilesystemEnforcer interface {
	ApplyFilesystemPolicy(context.Context, FilesystemPolicy) (Cleanup, error)
}

type FilesystemPolicyRefresher interface {
	RefreshFilesystemPolicy(context.Context, FilesystemPolicy, FilesystemPolicy) error
}

// SecretInjector resolves opaque handles outside TEI and injects secret
// material according to the target metadata. Raw secret values never enter the
// TEI Request or durable provenance.
type SecretInjector interface {
	InjectSecrets(context.Context, []SecretInjection) (Cleanup, error)
}

type SecretRefresher interface {
	RefreshSecrets(context.Context, []SecretInjection, []SecretInjection) error
}

type SandboxedProcessRunner interface {
	InvokeSandboxed(context.Context, ProcessCommand, Request) (Response, error)
}

type supervisorOwnershipController interface {
	Ownership() (resource.SupervisorOwnership, bool)
	HeartbeatOwnership(context.Context) error
	OwnershipHeartbeatInterval() time.Duration
}

func runOwnershipHeartbeat(ctx context.Context, c supervisorOwnershipController) error {
	if c == nil {
		return nil
	}
	if _, ok := c.Ownership(); !ok {
		return nil
	}
	interval := c.OwnershipHeartbeatInterval()
	if interval <= 0 {
		return fmt.Errorf("supervisor ownership heartbeat interval is not configured")
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			if err := c.HeartbeatOwnership(ctx); err != nil {
				return fmt.Errorf("supervisor ownership heartbeat failed: %w", err)
			}
		}
	}
}

type hostRefreshingLeaseController struct {
	base ResourceLeaseController
	host *HostSandboxLauncher
}

func (c *hostRefreshingLeaseController) Current() resource.Lease { return c.base.Current() }
func (c *hostRefreshingLeaseController) Renew(ctx context.Context) (resource.Lease, error) {
	return c.base.Renew(ctx)
}
func (c *hostRefreshingLeaseController) Refresh(ctx context.Context) (resource.Lease, error) {
	before := c.base.Current()
	after, err := c.base.Refresh(ctx)
	if err != nil {
		return resource.Lease{}, err
	}
	oldProfile, newProfile := sandboxProfile(&before), sandboxProfile(&after)
	fail := func(err error) (resource.Lease, error) {
		if lc, ok := c.base.(*leaseController); ok {
			lc.markFailure(err)
		}
		return resource.Lease{}, err
	}
	oldNet := NetworkPolicy{DenyByDefault: oldProfile.DenyNetworkByDefault, Allowlist: oldProfile.NetworkAllowlist}
	newNet := NetworkPolicy{DenyByDefault: newProfile.DenyNetworkByDefault, Allowlist: newProfile.NetworkAllowlist}
	if !reflect.DeepEqual(oldNet, newNet) {
		r, ok := c.host.Network.(NetworkPolicyRefresher)
		if !ok {
			return fail(fmt.Errorf("live network refresh required but host network enforcer is not refresh-capable"))
		}
		if err := r.RefreshNetworkPolicy(ctx, oldNet, newNet); err != nil {
			return fail(fmt.Errorf("live network refresh failed: %w", err))
		}
	}
	oldFS := FilesystemPolicy{DenyByDefault: oldProfile.DenyFilesystemByDefault, Mounts: oldProfile.Mounts}
	newFS := FilesystemPolicy{DenyByDefault: newProfile.DenyFilesystemByDefault, Mounts: newProfile.Mounts}
	if !reflect.DeepEqual(oldFS, newFS) {
		r, ok := c.host.Filesystem.(FilesystemPolicyRefresher)
		if !ok {
			return fail(fmt.Errorf("live filesystem refresh required but host filesystem enforcer is not refresh-capable"))
		}
		if err := r.RefreshFilesystemPolicy(ctx, oldFS, newFS); err != nil {
			return fail(fmt.Errorf("live filesystem refresh failed: %w", err))
		}
	}
	if !reflect.DeepEqual(oldProfile.SecretInjections, newProfile.SecretInjections) {
		r, ok := c.host.Secrets.(SecretRefresher)
		if !ok {
			return fail(fmt.Errorf("live secret refresh required but host secret injector is not refresh-capable"))
		}
		if err := r.RefreshSecrets(ctx, oldProfile.SecretInjections, newProfile.SecretInjections); err != nil {
			return fail(fmt.Errorf("live secret refresh failed: %w", err))
		}
	}
	return after, nil
}

// HostSandboxLauncher composes explicit host enforcement adapters. It is not a
// sandbox implementation by itself; it refuses to run unless the host provides
// the adapters required to enforce deny-by-default network/filesystem policy and
// any requested secret injection.
type HostSandboxLauncher struct {
	Network        NetworkEnforcer
	Filesystem     FilesystemEnforcer
	Secrets        SecretInjector
	Runner         SandboxedProcessRunner
	RefreshTrigger LeaseRefreshTrigger
	// Now is the host operational clock used by lease supervision. Production
	// hosts normally leave it nil (time.Now); tests and deterministic hosts may
	// inject the same clock used by their Resource Broker.
	Now func() time.Time
}

func (h HostSandboxLauncher) Invoke(ctx context.Context, spec SandboxSpec, req Request) (Response, error) {
	if h.Runner == nil {
		return Response{}, fmt.Errorf("host sandbox has no process runner")
	}
	if spec.Profile.DenyNetworkByDefault && h.Network == nil {
		return Response{}, fmt.Errorf("host sandbox cannot enforce deny-by-default network policy")
	}
	if spec.Profile.DenyFilesystemByDefault && h.Filesystem == nil {
		return Response{}, fmt.Errorf("host sandbox cannot enforce deny-by-default filesystem policy")
	}
	if len(spec.Profile.SecretInjections) > 0 && h.Secrets == nil {
		return Response{}, fmt.Errorf("host sandbox has secret injection requirements but no secret injector")
	}

	var cleanups []Cleanup
	cleanupAll := func() error {
		var errs []error
		for i := len(cleanups) - 1; i >= 0; i-- {
			if cleanups[i] == nil {
				continue
			}
			if err := cleanups[i](context.WithoutCancel(ctx)); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}
	fail := func(err error) (Response, error) {
		return Response{}, errors.Join(err, cleanupAll())
	}

	if h.Network != nil {
		c, err := h.Network.ApplyNetworkPolicy(ctx, NetworkPolicy{DenyByDefault: spec.Profile.DenyNetworkByDefault, Allowlist: append([]string(nil), spec.Profile.NetworkAllowlist...)})
		if err != nil {
			return fail(fmt.Errorf("network policy enforcement failed: %w", err))
		}
		cleanups = append(cleanups, c)
	}
	if h.Filesystem != nil {
		c, err := h.Filesystem.ApplyFilesystemPolicy(ctx, FilesystemPolicy{DenyByDefault: spec.Profile.DenyFilesystemByDefault, Mounts: append([]SandboxMount(nil), spec.Profile.Mounts...)})
		if err != nil {
			return fail(fmt.Errorf("filesystem policy enforcement failed: %w", err))
		}
		cleanups = append(cleanups, c)
	}
	if len(spec.Profile.SecretInjections) > 0 {
		c, err := h.Secrets.InjectSecrets(ctx, append([]SecretInjection(nil), spec.Profile.SecretInjections...))
		if err != nil {
			return fail(fmt.Errorf("secret injection failed: %w", err))
		}
		cleanups = append(cleanups, c)
	}

	invokeCtx, cancelInvoke := context.WithCancel(ctx)
	defer cancelInvoke()
	var supervisorErrCh chan error
	var ownershipErrCh chan error
	// A capability without Permissions has no lease. The caller may pass that as a
	// nil *leaseController wrapped in the interface, which is not == nil.
	if lc, ok := spec.LeaseController.(*leaseController); ok && lc == nil {
		spec.LeaseController = nil
		req.LeaseController = nil
	}
	if spec.LeaseController != nil {
		wrapped := &hostRefreshingLeaseController{base: spec.LeaseController, host: &h}
		spec.LeaseController = wrapped
		req.LeaseController = wrapped
		var recorder *leaseController
		if lc, ok := wrapped.base.(*leaseController); ok {
			recorder = lc
		}
		supervisor := &leaseSupervisor{controller: wrapped, recorder: recorder, policy: spec.LeasePolicy, trigger: h.RefreshTrigger, now: h.Now}
		supervisorErrCh = make(chan error, 1)
		go func() {
			err := supervisor.run(invokeCtx)
			if err != nil {
				cancelInvoke()
			}
			supervisorErrCh <- err
		}()
		if ownerCtl, ok := wrapped.base.(supervisorOwnershipController); ok {
			if _, owned := ownerCtl.Ownership(); owned {
				ownershipErrCh = make(chan error, 1)
				go func() {
					err := runOwnershipHeartbeat(invokeCtx, ownerCtl)
					if err != nil {
						cancelInvoke()
					}
					ownershipErrCh <- err
				}()
			}
		}
	}
	resp, err := h.Runner.InvokeSandboxed(invokeCtx, spec.Command, req)
	cancelInvoke()
	var supervisorErr, ownershipErr error
	if supervisorErrCh != nil {
		supervisorErr = <-supervisorErrCh
	}
	if ownershipErrCh != nil {
		ownershipErr = <-ownershipErrCh
	}
	cleanupErr := cleanupAll()
	if err != nil || supervisorErr != nil || ownershipErr != nil || cleanupErr != nil {
		return resp, errors.Join(err, supervisorErr, ownershipErr, cleanupErr)
	}
	return resp, nil
}
