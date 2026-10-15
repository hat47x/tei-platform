// Package derivationdisclosure is a D5 read-only reference boundary.
// It delegates all policy decisions to the existing security.authorize
// capability; it does not implement authorization, SQL, or derivation rules.
package derivationdisclosure

import (
	"context"
	"errors"
	"math/big"
	"reflect"
	"strings"

	"tei/internal/core"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/semantic/derivationref"
)

var (
	ErrInvalidPlan        = errors.New("invalid derivation disclosure plan")
	ErrDenied             = errors.New("derivation disclosure denied")
	ErrAuthorityFailure   = errors.New("derivation authorization unavailable")
	ErrEvaluationFailure  = errors.New("derivation evaluation unavailable")
)

type Mode string

const (
	Detail    Mode = "detail"
	Aggregate Mode = "aggregate"
)

type Surface string

const (
	Select   Surface = "select"
	Filter   Surface = "filter"
	Sort     Surface = "sort"
	Count    Surface = "count"
	Group    Surface = "group"
	Page     Surface = "page"
	Metadata Surface = "metadata"
	Trace    Surface = "trace"
)

// TrustedContext and QueryPlan must come from the trusted Host and the
// accepted D2/HTTP query boundary, NOT client payload or Plugin assertions.
// This reference cannot attest the Host's identity or QueryPlan completeness.
type TrustedContext struct {
	SubjectID       string
	Tenant          core.CanonicalID
	Purpose         core.CanonicalID
	PolicyPartition string
	// For an aggregate only: Host-selected executor identity. It must be
	// independently resolved/locked, not specified by the requesting user.
	ProviderSubjectID string
}

// AuthorizedRow is a Host/D2-scoped row occurrence. Actual row filtering,
// joins and aggregation order belong to D2, not to this gate.
type AuthorizedRow struct {
	ID     core.CanonicalID
	Tenant core.CanonicalID
}

type QueryPlan struct {
	Definition derivationref.Definition
	Target     core.CanonicalID
	Mode       Mode
	Rows       []AuthorizedRow
	Surfaces   []Surface
}

// Authority is the existing security.authorize Capability adapter shape.
// The D5 gate never loads roles, issues grants, or parses credentials.
type Authority interface {
	Invoke(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error)
}

type Evaluator func(context.Context) (derivationref.Input, error)

// Evaluate rechecks the current authority both before and after the supplied
// evaluator (including a cache read). No partial/secret values or provider
// errors cross a failed gate. Its authorization and provider operations are
// not atomic; it is NOT an execution grant, row filter, or privacy proof.
func Evaluate(ctx context.Context, trusted TrustedContext, plan QueryPlan, authority Authority, evaluator Evaluator) (derivationref.Input, error) {
	if err := validate(trusted, plan); err != nil {
		return derivationref.Input{}, err
	}
	if authority == nil || evaluator == nil || ctx == nil {
		return derivationref.Input{}, ErrAuthorityFailure
	}
	requests := requiredPermissions(trusted, plan)
	for _, request := range requests {
		if err := check(ctx, authority, request); err != nil {
			return derivationref.Input{}, err
		}
	}
	if ctx.Err() != nil {
		return derivationref.Input{}, ErrAuthorityFailure
	}
	value, err := evaluator(ctx)
	// Deliberately discard the provider's raw error, which might contain
	// protected input values, subject identity, or row existence.
	if err != nil || !validValue(value) {
		return derivationref.Input{}, ErrEvaluationFailure
	}
	for _, request := range requests {
		if err := check(ctx, authority, request); err != nil {
			return derivationref.Input{}, err
		}
	}
	if ctx.Err() != nil {
		return derivationref.Input{}, ErrAuthorityFailure
	}
	return value, nil
}

func validValue(v derivationref.Input) bool {
	switch v.State {
	case derivationref.Null:
		return v.Decimal == ""
	case derivationref.Present:
		if strings.TrimSpace(v.Decimal) != v.Decimal || v.Decimal == "" {
			return false
		}
		_, ok := new(big.Rat).SetString(v.Decimal)
		return ok
	default:
		// Missing, absent relation, multiple relations and unknown evaluation
		// states are not valid values for an authorized projection.
		return false
	}
}

func canonical(id core.CanonicalID) bool {
	parsed, err := core.ParseCanonicalID(string(id))
	return err == nil && parsed == id
}

func validate(t TrustedContext, p QueryPlan) error {
	if strings.TrimSpace(t.SubjectID) == "" || t.SubjectID != strings.TrimSpace(t.SubjectID) ||
		!canonical(t.Tenant) || !canonical(t.Purpose) ||
		strings.TrimSpace(t.PolicyPartition) == "" || !canonical(p.Target) ||
		p.Definition.ID == "" || len(p.Definition.Inputs) == 0 ||
		derivationref.ValidateGraph([]derivationref.Definition{p.Definition}, nil) != nil {
		return ErrInvalidPlan
	}
	for _, id := range p.Definition.Inputs {
		if !canonical(core.CanonicalID(id)) {
			return ErrInvalidPlan
		}
	}
	if p.Mode != Detail && p.Mode != Aggregate {
		return ErrInvalidPlan
	}
	if p.Mode == Aggregate && strings.TrimSpace(t.ProviderSubjectID) == "" {
		return ErrInvalidPlan
	}
	if p.Mode == Detail && t.ProviderSubjectID != "" {
		return ErrInvalidPlan
	}
	if len(p.Surfaces) == 0 || len(p.Rows) == 0 {
		// No implicit "select" or unrestricted/unknown row scan.
		return ErrInvalidPlan
	}
	seenRows := make(map[core.CanonicalID]bool, len(p.Rows))
	for _, row := range p.Rows {
		if !canonical(row.ID) || !canonical(row.Tenant) ||
			row.Tenant != t.Tenant || seenRows[row.ID] {
			// Missing row-scope or cross-tenant scans fail BEFORE evaluation.
			return ErrInvalidPlan
		}
		seenRows[row.ID] = true
	}
	seenSurfaces := make(map[Surface]bool, len(p.Surfaces))
	for _, surface := range p.Surfaces {
		switch surface {
		case Select, Filter, Sort, Count, Group, Page, Metadata, Trace:
		default:
			return ErrInvalidPlan
		}
		if seenSurfaces[surface] {
			return ErrInvalidPlan
		}
		seenSurfaces[surface] = true
	}
	if p.Mode == Aggregate && !seenSurfaces[Group] {
		// Aggregate-only disclosure cannot be disguised as direct select.
		return ErrInvalidPlan
	}
	if p.Mode == Detail && seenSurfaces[Group] {
		return ErrInvalidPlan
	}
	return nil
}

type permissionCheck struct {
	subject  string
	action   string
	resource string
}

func requiredPermissions(t TrustedContext, p QueryPlan) []permissionCheck {
	// The resource combines exact Host-owned tenant/purpose and typed
	// declaration IDs. No physical column names or display labels are used.
	base := "tenant/" + string(t.Tenant) + "/purpose/" + string(t.Purpose) + "/"
	add := func(subject, action, resource string) permissionCheck {
		return permissionCheck{subject: subject, action: action, resource: base + resource}
	}
	checks := make([]permissionCheck, 0, len(p.Rows)+len(p.Definition.Inputs)+len(p.Surfaces)+3)
	for _, row := range p.Rows {
		checks = append(checks, add(t.SubjectID, "derive.row.include", "row/"+string(row.ID)))
	}
	checks = append(checks, add(t.SubjectID, "derive.rule.execute", "rule/"+p.Definition.ID))
	for _, input := range p.Definition.Inputs {
		if p.Mode == Detail {
			checks = append(checks, add(t.SubjectID, "derive.input.read", "input/"+input))
		} else {
			// This is an executor's own permission, not the caller's read grant.
			checks = append(checks, add(t.ProviderSubjectID, "derive.input.internal", "input/"+input))
		}
	}
	if p.Mode == Aggregate {
		checks = append(checks, add(t.SubjectID, "derive.aggregate.run", "rule/"+p.Definition.ID))
		checks = append(checks, add(t.SubjectID, "derive.aggregate.disclose", "target/"+string(p.Target)))
	} else {
		checks = append(checks, add(t.SubjectID, "derive.detail.disclose", "target/"+string(p.Target)))
	}
	for _, surface := range p.Surfaces {
		checks = append(checks, add(t.SubjectID, "derive.surface."+string(surface), "target/"+string(p.Target)))
	}
	return checks
}

func check(ctx context.Context, authority Authority, p permissionCheck) error {
	if ctx.Err() != nil {
		return ErrAuthorityFailure
	}
	result, err := authority.Invoke(ctx, legacyauthorization.AuthorizationCapability, legacyauthorization.Input{
		SubjectID: p.subject, Action: p.action, Resource: p.resource,
	})
	// Do not wrap or format an untrusted error or denial reason.
	if err != nil {
		return ErrAuthorityFailure
	}
	if result.Capability != legacyauthorization.AuthorizationCapability ||
		result.Mode != capabilityinvoke.ResultOutcome {
		return ErrAuthorityFailure
	}
	if result.Outcome == "denied" {
		if _, ok := result.Payload.(legacyauthorization.AuthorizationDenial); ok {
			return ErrDenied
		}
		return ErrAuthorityFailure
	}
	if result.Outcome != "allowed" {
		return ErrAuthorityFailure
	}
	grant, ok := result.Payload.(legacyauthorization.AuthorizationGrant)
	if !ok || grant.SubjectID != p.subject || grant.Action != p.action || grant.Resource != p.resource {
		return ErrAuthorityFailure
	}
	// Epochs are optional in the legacy Adapter. The D5 gate does not
	// manufacture a version guarantee or turn this into a grant for effects.
	if grant.Epochs != nil && reflect.ValueOf(grant.Epochs).IsNil() {
		return ErrAuthorityFailure
	}
	return nil
}
