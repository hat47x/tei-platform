// Package webapicontract contains deliberately non-production API contract probes.
package webapicontract

import (
	"net/url"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/ir"
)

// PublishedBrowse is a typed, read-only projection of an existing Application
// browse Pattern. No new query, SQL realization or business meaning is created.
// It is internal binding metadata and MUST NOT be returned as public HTTP data.
type PublishedBrowse struct {
	ApplicationID core.CanonicalID
	PatternID     core.CanonicalID
	QueryID       core.CanonicalID
	DataID        core.CanonicalID
	Fields        []PublishedField
}

type PublishedField struct {
	Name        string
	AttributeID core.CanonicalID
	TypeID      core.CanonicalID
	Policy      FieldPolicy
}

// ResolvePublishedBrowse uses the existing Driver to resolve authoritative
// Pattern/Query/Data identities, then binds externally supplied authorization
// decisions by exact canonical attribute identity. Decisions must be obtained
// from a trusted caller at invocation time; this function is not an authorizer.
//
// No-gateway rights mean no publication; result mappings cannot grant access.
// An allowed field which is not mapped by the Query is rejected rather than
// silently providing a different or inferred field.
func ResolvePublishedBrowse(
	world ir.World,
	applicationID, patternID core.CanonicalID,
	decisions map[core.CanonicalID]FieldPolicy,
) (PublishedBrowse, error) {
	resolved, err := driver.NewResolver(world).ResolvePattern(applicationID, patternID)
	if err != nil || resolved.Pattern.Kind != application.PatternBrowse ||
		resolved.Data == nil || resolved.Query == nil || resolved.Datasource == nil {
		return PublishedBrowse{}, ErrRejected
	}
	if len(resolved.Query.ResultColumns) == 0 {
		return PublishedBrowse{}, ErrRejected
	}
	attributes := make(map[core.CanonicalID]struct {
		Name string
		Type core.CanonicalID
	}, len(resolved.Data.Attributes))
	for _, attribute := range resolved.Data.Attributes {
		if attribute.ID == "" || attribute.Symbol == "" || attribute.TypeRef == "" {
			return PublishedBrowse{}, ErrRejected
		}
		if _, duplicate := attributes[attribute.ID]; duplicate {
			return PublishedBrowse{}, ErrRejected
		}
		attributes[attribute.ID] = struct {
			Name string
			Type core.CanonicalID
		}{Name: attribute.Symbol, Type: attribute.TypeRef}
	}
	published := PublishedBrowse{
		ApplicationID: applicationID, PatternID: patternID,
		QueryID: resolved.Query.ID, DataID: resolved.Data.ID,
	}
	seenNames := map[string]bool{}
	seenIDs := map[core.CanonicalID]bool{}
	for _, mapping := range resolved.Query.ResultColumns {
		attribute, found := attributes[mapping.Attribute]
		if !found || seenIDs[mapping.Attribute] || mapping.Column == "" {
			return PublishedBrowse{}, ErrRejected
		}
		seenIDs[mapping.Attribute] = true
		if seenNames[attribute.Name] {
			return PublishedBrowse{}, ErrRejected
		}
		seenNames[attribute.Name] = true
		if decision, granted := decisions[mapping.Attribute]; granted &&
			(decision.Select || decision.Filter || decision.Sort) {
			published.Fields = append(published.Fields, PublishedField{
				Name: attribute.Name, AttributeID: mapping.Attribute,
				TypeID: attribute.Type, Policy: decision,
			})
		}
	}
	for id, policy := range decisions {
		if (policy.Select || policy.Filter || policy.Sort) && !seenIDs[id] {
			return PublishedBrowse{}, ErrRejected
		}
	}
	// A publication with only hidden or filterable data has no safe default
	// projection, and therefore cannot serve a normal list.
	selectable := false
	for _, field := range published.Fields {
		selectable = selectable || field.Policy.Select
	}
	if !selectable {
		return PublishedBrowse{}, ErrRejected
	}
	return published, nil
}

// BindRead binds the narrow synthetic query expression to the exact field
// grant projection. It does not execute any Query or establish row security.
func (published PublishedBrowse) BindRead(values url.Values, ctx TrustedReadContext) (ReadPlan, error) {
	if published.ApplicationID == "" || published.PatternID == "" ||
		published.QueryID == "" || published.DataID == "" {
		return ReadPlan{}, ErrRejected
	}
	fields := make([]Field, 0, len(published.Fields))
	for _, f := range published.Fields {
		fields = append(fields, Field{ID: f.Name, Policy: f.Policy})
	}
	return BindReadSubset(values, fields, ctx)
}
