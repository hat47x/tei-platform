package resource

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"tei/internal/core"
	"tei/internal/permission"
)

type SecretDelivery string

type RefreshMode string

const (
	RefreshNone RefreshMode = "none"
	RefreshLive RefreshMode = "live"
)

func (m RefreshMode) Valid() bool { return m == "" || m == RefreshNone || m == RefreshLive }

type RenewalFailurePolicy string

const (
	RenewalFailClosed RenewalFailurePolicy = "fail-closed"
)

func (p RenewalFailurePolicy) Valid() bool { return p == "" || p == RenewalFailClosed }

// LeasePolicy is an environment/runtime policy, not a plugin-owned default.
// Duration fields use Go duration syntax (for example 30m, 10s).
// M0.19 supports fail-closed renewal semantics only.
type LeasePolicy struct {
	MaxLifetime    string               `json:"maxLifetime,omitempty"`
	RenewBefore    string               `json:"renewBefore,omitempty"`
	RevokeTimeout  string               `json:"revokeTimeout,omitempty"`
	RenewalFailure RenewalFailurePolicy `json:"renewalFailure,omitempty"`
}

func DefaultLeasePolicy() LeasePolicy {
	return LeasePolicy{MaxLifetime: "30m", RenewBefore: "5m", RevokeTimeout: "10s", RenewalFailure: RenewalFailClosed}
}

func (p LeasePolicy) WithDefaults() LeasePolicy {
	d := DefaultLeasePolicy()
	if p.MaxLifetime != "" {
		d.MaxLifetime = p.MaxLifetime
	}
	if p.RenewBefore != "" {
		d.RenewBefore = p.RenewBefore
	}
	if p.RevokeTimeout != "" {
		d.RevokeTimeout = p.RevokeTimeout
	}
	if p.RenewalFailure != "" {
		d.RenewalFailure = p.RenewalFailure
	}
	return d
}

func (p LeasePolicy) Validate() error {
	p = p.WithDefaults()
	for name, raw := range map[string]string{"max-lifetime": p.MaxLifetime, "renew-before": p.RenewBefore, "revoke-timeout": p.RevokeTimeout} {
		d, err := time.ParseDuration(raw)
		if err != nil || d <= 0 {
			return fmt.Errorf("lease policy %s must be a positive duration, got %q", name, raw)
		}
	}
	max, _ := time.ParseDuration(p.MaxLifetime)
	rb, _ := time.ParseDuration(p.RenewBefore)
	if rb >= max {
		return fmt.Errorf("lease policy renew-before %s must be shorter than max-lifetime %s", p.RenewBefore, p.MaxLifetime)
	}
	if !p.RenewalFailure.Valid() {
		return fmt.Errorf("unknown lease renewal failure policy %q", p.RenewalFailure)
	}
	return nil
}

func (p LeasePolicy) MaxLifetimeDuration() time.Duration {
	d, _ := time.ParseDuration(p.WithDefaults().MaxLifetime)
	return d
}
func (p LeasePolicy) RenewBeforeDuration() time.Duration {
	d, _ := time.ParseDuration(p.WithDefaults().RenewBefore)
	return d
}
func (p LeasePolicy) RevokeTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(p.WithDefaults().RevokeTimeout)
	return d
}

const (
	SecretDeliveryHandle      SecretDelivery = "handle"
	SecretDeliveryEnvironment SecretDelivery = "environment"
	SecretDeliveryFile        SecretDelivery = "file"
)

func (d SecretDelivery) Valid() bool {
	return d == "" || d == SecretDeliveryHandle || d == SecretDeliveryEnvironment || d == SecretDeliveryFile
}

// Binding maps a logical permission scope used by TEI definitions to a host-
// resolvable resource identity. It deliberately does not contain secret values,
// host filesystem paths, or network credentials. SecretDelivery/SecretTarget
// describe only how an opaque secret handle is to be injected by the host.
type Binding struct {
	Kind           permission.Kind  `json:"kind"`
	Scope          string           `json:"scope"`
	ResourceID     core.CanonicalID `json:"resourceId"`
	SecretDelivery SecretDelivery   `json:"secretDelivery,omitempty"`
	SecretTarget   string           `json:"secretTarget,omitempty"`
	Refresh        RefreshMode      `json:"refresh,omitempty"`
}

func (b Binding) Key() string { return string(b.Kind) + "|" + b.Scope }

func (b Binding) Validate() error {
	if b.Scope == "" {
		return fmt.Errorf("resource binding scope is required")
	}
	switch b.Kind {
	case permission.KindFilesystem, permission.KindNetwork:
		if b.SecretDelivery != "" || b.SecretTarget != "" {
			return fmt.Errorf("resource binding %s may not declare secret injection", b.Key())
		}
	case permission.KindSecret:
		if !b.SecretDelivery.Valid() {
			return fmt.Errorf("resource binding %s has unknown secret delivery %q", b.Key(), b.SecretDelivery)
		}
		d := b.SecretDelivery
		if d == "" {
			d = SecretDeliveryHandle
		}
		if (d == SecretDeliveryEnvironment || d == SecretDeliveryFile) && b.SecretTarget == "" {
			return fmt.Errorf("resource binding %s secret delivery %s requires a target", b.Key(), d)
		}
		if d == SecretDeliveryHandle && b.SecretTarget != "" {
			return fmt.Errorf("resource binding %s handle delivery may not declare a target", b.Key())
		}
	default:
		return fmt.Errorf("unknown resource binding kind %q", b.Kind)
	}
	if b.ResourceID == "" {
		return fmt.Errorf("resource binding %s requires resource id", b.Key())
	}
	if !b.Refresh.Valid() {
		return fmt.Errorf("resource binding %s has unknown refresh mode %q", b.Key(), b.Refresh)
	}
	return nil
}

func NormalizeBindings(in []Binding) []Binding {
	out := append([]Binding(nil), in...)
	for i := range out {
		if out[i].Kind == permission.KindSecret && out[i].SecretDelivery == "" {
			out[i].SecretDelivery = SecretDeliveryHandle
		}
		if out[i].Refresh == "" {
			out[i].Refresh = RefreshNone
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Key() != out[j].Key() {
			return out[i].Key() < out[j].Key()
		}
		if out[i].ResourceID != out[j].ResourceID {
			return out[i].ResourceID < out[j].ResourceID
		}
		if out[i].SecretDelivery != out[j].SecretDelivery {
			return out[i].SecretDelivery < out[j].SecretDelivery
		}
		if out[i].SecretTarget != out[j].SecretTarget {
			return out[i].SecretTarget < out[j].SecretTarget
		}
		return out[i].Refresh < out[j].Refresh
	})
	return out
}

// Resolved is the least-authority resource view created by a host Resource
// Broker for one invocation lease. A secret is represented only by an opaque
// handle; a filesystem resource only by its sandbox mount point/source handle;
// and a network resource only by the endpoint admitted for this lease.
type Resolved struct {
	Kind              permission.Kind   `json:"kind"`
	Access            permission.Access `json:"access"`
	Scope             string            `json:"scope"`
	ResourceID        core.CanonicalID  `json:"resourceId"`
	SecretHandle      string            `json:"secretHandle,omitempty"`
	SecretDelivery    SecretDelivery    `json:"secretDelivery,omitempty"`
	SecretTarget      string            `json:"secretTarget,omitempty"`
	NetworkEndpoint   string            `json:"networkEndpoint,omitempty"`
	MountSourceHandle string            `json:"mountSourceHandle,omitempty"`
	MountPath         string            `json:"mountPath,omitempty"`
}

func (r Resolved) Key() string {
	return string(r.Kind) + "|" + string(r.Access) + "|" + r.Scope + "|" + string(r.ResourceID)
}

func NormalizeResolved(in []Resolved) []Resolved {
	out := append([]Resolved(nil), in...)
	for i := range out {
		if out[i].Kind == permission.KindSecret && out[i].SecretDelivery == "" {
			out[i].SecretDelivery = SecretDeliveryHandle
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out
}

// Lease is intentionally short-lived and invocation-scoped. Revision starts at
// one and increases only through a validated renewal of the same logical lease.
type Lease struct {
	ID                       string     `json:"id"`
	Revision                 int        `json:"revision,omitempty"`
	FencingToken             uint64     `json:"fencingToken,omitempty"`
	IssuedAt                 string     `json:"issuedAt,omitempty"`
	ExpiresAt                string     `json:"expiresAt,omitempty"`
	RuntimePolicyFingerprint string     `json:"runtimePolicyFingerprint,omitempty"`
	Resources                []Resolved `json:"resources,omitempty"`
}

// AcquireRequest contains only logical identities. The broker owns the mapping
// from ResourceID to concrete credentials, mounts, or endpoints.
type AcquireRequest struct {
	Environment              core.CanonicalID  `json:"environment"`
	CapabilityKind           string            `json:"capabilityKind"`
	CapabilityID             core.CanonicalID  `json:"capabilityId"`
	RuntimePolicyFingerprint string            `json:"runtimePolicyFingerprint,omitempty"`
	Permissions              []permission.Spec `json:"permissions"`
	Bindings                 []Binding         `json:"bindings"`
	LeasePolicy              LeasePolicy       `json:"leasePolicy,omitempty"`
}

func NormalizeAcquireRequest(req AcquireRequest) AcquireRequest {
	req.Permissions = permission.Normalize(req.Permissions)
	req.Bindings = NormalizeBindings(req.Bindings)
	req.LeasePolicy = req.LeasePolicy.WithDefaults()
	return req
}

func AcquireRequestFingerprint(req AcquireRequest) string {
	req = NormalizeAcquireRequest(req)
	raw, _ := json.Marshal(req)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// Broker is supplied by the TEI host. Implementations may front a secret
// manager, namespace/mount manager, network policy controller, or test fixture.
type Broker interface {
	Acquire(context.Context, AcquireRequest) (Lease, error)
	Release(context.Context, Lease) error
}

// RenewableBroker is optional. Renewal is explicit and constrained to the
// original AcquireRequest. It must never broaden permission, resource identity,
// capability, environment, or runtime policy.
type RenewableBroker interface {
	Broker
	Renew(context.Context, AcquireRequest, Lease) (Lease, error)
}

// RefreshableBroker changes the concrete resource view of bindings explicitly
// marked refresh=live. Refresh never extends lease lifetime.
type RefreshableBroker interface {
	Broker
	Refresh(context.Context, AcquireRequest, Lease) (Lease, error)
}

// FenceableBroker installs a monotonic supervisor fencing token on an active
// lease. Authority-bearing Renew/Refresh/Release operations must reject stale
// lease copies after a newer supervisor has taken ownership.
type FenceableBroker interface {
	Broker
	Fence(context.Context, Lease, uint64) (Lease, error)
}

type RenewalRecord struct {
	Revision  int    `json:"revision"`
	RenewedAt string `json:"renewedAt,omitempty"`
	ExpiresAt string `json:"expiresAt,omitempty"`
}

type RefreshRecord struct {
	Revision    int      `json:"revision"`
	RefreshedAt string   `json:"refreshedAt,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`
}

// SupervisionState describes host-side lifecycle observations for an invocation
// resource lease. These are operational audit states, not authority-bearing
// resource values.
type SupervisionState string

const (
	SupervisionActive                SupervisionState = "active"
	SupervisionRenewalDue            SupervisionState = "renewal-due"
	SupervisionRenewed               SupervisionState = "renewed"
	SupervisionRenewalFailed         SupervisionState = "renewal-failed"
	SupervisionRenewalDeadlineMissed SupervisionState = "renewal-deadline-missed"
	SupervisionRefreshDue            SupervisionState = "refresh-due"
	SupervisionRefreshed             SupervisionState = "refreshed"
	SupervisionRefreshFailed         SupervisionState = "refresh-failed"
	SupervisionRevokePending         SupervisionState = "revoke-pending"
	SupervisionRevoked               SupervisionState = "revoked"
	SupervisionRevokeFailed          SupervisionState = "revoke-failed"
)

// SupervisionRecord intentionally contains no concrete secret handle, endpoint
// or mount path. It records only the lease lifecycle transition and revision.
type SupervisionRecord struct {
	State      SupervisionState `json:"state"`
	ObservedAt string           `json:"observedAt,omitempty"`
	Revision   int              `json:"revision,omitempty"`
	ExpiresAt  string           `json:"expiresAt,omitempty"`
	Message    string           `json:"message,omitempty"`
}

func MatchBindings(perms []permission.Spec, bindings []Binding) ([]Binding, error) {
	byKey := map[string]Binding{}
	for _, b := range NormalizeBindings(bindings) {
		if err := b.Validate(); err != nil {
			return nil, err
		}
		if _, dup := byKey[b.Key()]; dup {
			return nil, fmt.Errorf("duplicate resource binding for %s", b.Key())
		}
		byKey[b.Key()] = b
	}
	var out []Binding
	for _, p := range permission.Normalize(perms) {
		key := string(p.Kind) + "|" + p.Scope
		b, ok := byKey[key]
		if !ok {
			return nil, fmt.Errorf("permission %s has no resource binding", p.Key())
		}
		out = append(out, b)
	}
	return NormalizeBindings(out), nil
}

// Record is the non-sensitive lease provenance retained by TEI. Concrete
// endpoints, mount paths/source handles, and secret handles intentionally do not
// enter durable audit artifacts.
type Record struct {
	Kind           permission.Kind   `json:"kind"`
	Access         permission.Access `json:"access"`
	Scope          string            `json:"scope"`
	ResourceID     core.CanonicalID  `json:"resourceId"`
	SecretDelivery SecretDelivery    `json:"secretDelivery,omitempty"`
	SecretTarget   string            `json:"secretTarget,omitempty"`
}

func (r Record) Key() string {
	return string(r.Kind) + "|" + string(r.Access) + "|" + r.Scope + "|" + string(r.ResourceID)
}

func Records(in []Resolved) []Record {
	var out []Record
	for _, r := range NormalizeResolved(in) {
		out = append(out, Record{Kind: r.Kind, Access: r.Access, Scope: r.Scope, ResourceID: r.ResourceID, SecretDelivery: r.SecretDelivery, SecretTarget: r.SecretTarget})
	}
	return out
}

func ValidateLease(req AcquireRequest, lease Lease) error {
	req = NormalizeAcquireRequest(req)
	if err := req.LeasePolicy.Validate(); err != nil {
		return err
	}
	if lease.ID == "" {
		return fmt.Errorf("resource broker returned empty lease id")
	}
	if lease.Revision < 0 {
		return fmt.Errorf("resource broker lease revision may not be negative")
	}
	if req.RuntimePolicyFingerprint != "" && lease.RuntimePolicyFingerprint != req.RuntimePolicyFingerprint {
		return fmt.Errorf("resource broker lease runtime policy fingerprint differs from acquire request")
	}
	if lease.IssuedAt != "" && lease.ExpiresAt != "" {
		issued, ierr := time.Parse(time.RFC3339, lease.IssuedAt)
		exp, eerr := time.Parse(time.RFC3339, lease.ExpiresAt)
		if ierr != nil || eerr != nil {
			return fmt.Errorf("resource broker lease has invalid issuedAt/expiresAt")
		}
		if !exp.After(issued) {
			return fmt.Errorf("resource broker lease expiry must be after issue time")
		}
		if exp.After(issued.Add(req.LeasePolicy.MaxLifetimeDuration())) {
			return fmt.Errorf("resource broker lease exceeds max lifetime %s", req.LeasePolicy.MaxLifetime)
		}
	}
	if len(lease.Resources) != len(req.Permissions) {
		return fmt.Errorf("resource broker lease contains %d resources for %d permissions", len(lease.Resources), len(req.Permissions))
	}
	bindings, err := MatchBindings(req.Permissions, req.Bindings)
	if err != nil {
		return err
	}
	byScope := map[string]Binding{}
	for _, b := range bindings {
		byScope[b.Key()] = b
	}
	seen := map[string]struct{}{}
	for _, r := range lease.Resources {
		key := string(r.Kind) + "|" + r.Scope
		if _, dup := seen[key]; dup {
			return fmt.Errorf("resource broker lease contains duplicate resource %s", key)
		}
		seen[key] = struct{}{}
		b, ok := byScope[key]
		if !ok {
			return fmt.Errorf("resource broker lease returned unrequested resource %s", key)
		}
		if r.ResourceID != b.ResourceID {
			return fmt.Errorf("resource broker lease resource %s resolved id %s, expected %s", key, r.ResourceID, b.ResourceID)
		}
		var expectedAccess permission.Access
		for _, p := range req.Permissions {
			if p.Kind == r.Kind && p.Scope == r.Scope {
				expectedAccess = p.Access
				break
			}
		}
		if r.Access != expectedAccess {
			return fmt.Errorf("resource broker lease resource %s access %s, expected %s", key, r.Access, expectedAccess)
		}
		switch r.Kind {
		case permission.KindSecret:
			d := r.SecretDelivery
			if d == "" {
				d = SecretDeliveryHandle
			}
			bd := b.SecretDelivery
			if bd == "" {
				bd = SecretDeliveryHandle
			}
			if r.SecretHandle == "" || r.NetworkEndpoint != "" || r.MountPath != "" || r.MountSourceHandle != "" {
				return fmt.Errorf("secret resource %s must expose only an opaque secret handle", key)
			}
			if d != bd || r.SecretTarget != b.SecretTarget {
				return fmt.Errorf("secret resource %s injection target differs from binding", key)
			}
		case permission.KindNetwork:
			if r.NetworkEndpoint == "" || r.SecretHandle != "" || r.MountPath != "" || r.MountSourceHandle != "" || r.SecretTarget != "" {
				return fmt.Errorf("network resource %s must expose only an allowed endpoint", key)
			}
		case permission.KindFilesystem:
			if r.MountPath == "" || r.SecretHandle != "" || r.NetworkEndpoint != "" || r.SecretTarget != "" {
				return fmt.Errorf("filesystem resource %s must expose only a mount target", key)
			}
		default:
			return fmt.Errorf("resource broker lease returned unsupported kind %s", r.Kind)
		}
	}
	return nil
}

func ValidateRenewal(req AcquireRequest, previous, renewed Lease) error {
	if err := ValidateLease(req, previous); err != nil {
		return fmt.Errorf("previous lease is invalid: %w", err)
	}
	if err := ValidateLease(req, renewed); err != nil {
		return fmt.Errorf("renewed lease is invalid: %w", err)
	}
	if renewed.ID != previous.ID {
		return fmt.Errorf("resource lease renewal changed lease id from %s to %s", previous.ID, renewed.ID)
	}
	prevRev := previous.Revision
	if prevRev == 0 {
		prevRev = 1
	}
	if renewed.Revision != prevRev+1 {
		return fmt.Errorf("resource lease renewal revision %d, expected %d", renewed.Revision, prevRev+1)
	}
	prevRecords, renewedRecords := Records(previous.Resources), Records(renewed.Resources)
	if len(prevRecords) != len(renewedRecords) {
		return fmt.Errorf("resource lease renewal changed resource count")
	}
	for i := range prevRecords {
		if prevRecords[i] != renewedRecords[i] {
			return fmt.Errorf("resource lease renewal broadened or changed logical resource identity")
		}
	}
	if previous.ExpiresAt != "" && renewed.ExpiresAt != "" {
		p, perr := time.Parse(time.RFC3339, previous.ExpiresAt)
		n, nerr := time.Parse(time.RFC3339, renewed.ExpiresAt)
		if perr == nil && nerr == nil && !n.After(p) {
			return fmt.Errorf("resource lease renewal expiry %s does not advance previous expiry %s", renewed.ExpiresAt, previous.ExpiresAt)
		}
	}
	return nil
}

// ValidateInPlaceRenewal is the M0.18 enforcement rule for a live invocation.
// Renewal may extend time/revision only. Concrete secret handles, network
// endpoints and mount views must remain unchanged because a generic host cannot
// safely hot-swap injected secrets or sandbox policies inside an already running
// process. Credential/resource rotation therefore requires a new invocation
// until a refresh-capable sandbox contract is introduced.
func ValidateInPlaceRenewal(req AcquireRequest, previous, renewed Lease) error {
	if err := ValidateRenewal(req, previous, renewed); err != nil {
		return err
	}
	a, b := NormalizeResolved(previous.Resources), NormalizeResolved(renewed.Resources)
	if len(a) != len(b) {
		return fmt.Errorf("in-place resource lease renewal changed concrete resource count")
	}
	for i := range a {
		if a[i] != b[i] {
			return fmt.Errorf("in-place resource lease renewal changed concrete resource view for %s", a[i].Key())
		}
	}
	return nil
}

// ValidateRefresh enforces M0.19 refresh semantics: the same logical authority
// and expiry are retained, revision advances by one, and only bindings marked
// refresh=live may change their concrete resource view.
func ValidateRefresh(req AcquireRequest, previous, refreshed Lease) error {
	if err := ValidateRenewal(req, previous, refreshed); err != nil {
		// ValidateRenewal requires expiry advancement; refresh explicitly must not.
		if err := ValidateLease(req, previous); err != nil {
			return fmt.Errorf("previous lease is invalid: %w", err)
		}
		if err := ValidateLease(req, refreshed); err != nil {
			return fmt.Errorf("refreshed lease is invalid: %w", err)
		}
		if refreshed.ID != previous.ID {
			return fmt.Errorf("resource lease refresh changed lease id")
		}
		prevRev := previous.Revision
		if prevRev == 0 {
			prevRev = 1
		}
		if refreshed.Revision != prevRev+1 {
			return fmt.Errorf("resource lease refresh revision %d, expected %d", refreshed.Revision, prevRev+1)
		}
	}
	if refreshed.ExpiresAt != previous.ExpiresAt {
		return fmt.Errorf("resource lease refresh may not extend or shorten expiry")
	}
	if refreshed.IssuedAt != previous.IssuedAt {
		return fmt.Errorf("resource lease refresh may not change issuedAt")
	}
	a, b := NormalizeResolved(previous.Resources), NormalizeResolved(refreshed.Resources)
	if len(a) != len(b) {
		return fmt.Errorf("resource lease refresh changed resource count")
	}
	bindingByKey := map[string]Binding{}
	for _, x := range NormalizeBindings(req.Bindings) {
		bindingByKey[x.Key()] = x
	}
	changed := false
	for i := range a {
		if a[i].Key() != b[i].Key() {
			return fmt.Errorf("resource lease refresh changed logical resource identity")
		}
		bind := bindingByKey[string(a[i].Kind)+"|"+a[i].Scope]
		if a[i] != b[i] {
			if bind.Refresh != RefreshLive {
				return fmt.Errorf("resource %s changed but binding is not refresh live", a[i].Key())
			}
			changed = true
		}
	}
	if !changed {
		return fmt.Errorf("resource lease refresh did not change any live resource")
	}
	return nil
}
