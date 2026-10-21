package orderwebmember

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"

	"tei/internal/core"
	"tei/internal/workspace"
	webapicontract "tei/reference/web-api-contract"
)

// Principal kinds. A projection is derived per kind from one catalog and one
// read plan; the staff projection is never rebuilt from the member one.
const (
	KindMember = "member"
	KindStaff  = "staff"
)

// Disclosure says who may ever see a field. Internal fields are never in a
// member projection, whatever policy a caller passes.
const (
	DisclosureMember   = "member"
	DisclosureInternal = "internal"
)

// Step-1 read plan identity (reference/web-api-contract resolves it from
// reference/workspace through the existing Driver).
var (
	ReadPlanApplication = core.MustCanonicalID("order_entry")
	ReadPlanPattern     = core.MustCanonicalID("order_entry.recent_orders")
)

// CatalogField is one field of the order as this sample stores it. Fields
// with an AttributeID come from the step-1 read plan (the query result
// mapping of order_entry.recent_orders); the others are sample-local
// extensions that the canonical model does not have.
type CatalogField struct {
	AttributeID string // canonical attribute; empty for sample-local fields
	Name        string // public name; for canonical fields it must equal the Semantic symbol
	Column      string // column of owdm.orders
	Disclosure  string
	Member      webapicontract.FieldPolicy
	Staff       webapicontract.FieldPolicy
}

// Catalog is the single declaration of the order fields. Member write access
// is limited to the delivery note; staff change status only through the
// confirm action, not through field writes.
func Catalog() []CatalogField {
	ro := webapicontract.FieldPolicy{Select: true}
	rs := webapicontract.FieldPolicy{Select: true, Sort: true}
	rfs := webapicontract.FieldPolicy{Select: true, Filter: true, Sort: true}
	rf := webapicontract.FieldPolicy{Select: true, Filter: true}
	return []CatalogField{
		{AttributeID: "sales.order.order_id", Name: "orderId", Column: "order_id", Disclosure: DisclosureMember, Member: rfs, Staff: rfs},
		{AttributeID: "sales.order.customer_id", Name: "customerId", Column: "account_id", Disclosure: DisclosureMember, Member: ro, Staff: rfs},
		{AttributeID: "sales.order.ordered_at", Name: "orderedAt", Column: "ordered_at", Disclosure: DisclosureMember, Member: rs, Staff: rs},
		{AttributeID: "sales.order.status", Name: "status", Column: "status", Disclosure: DisclosureMember, Member: rf, Staff: rfs},
		{AttributeID: "sales.order.total", Name: "total", Column: "total_amount", Disclosure: DisclosureMember, Member: ro, Staff: rs},
		{Name: "itemCode", Column: "item_code", Disclosure: DisclosureMember, Member: ro, Staff: rf},
		{Name: "quantity", Column: "quantity", Disclosure: DisclosureMember, Member: ro, Staff: ro},
		{Name: "deliveryNote", Column: "delivery_note", Disclosure: DisclosureMember,
			Member: webapicontract.FieldPolicy{Select: true, Write: true}, Staff: ro},
		{Name: "version", Column: "version", Disclosure: DisclosureMember, Member: ro, Staff: ro},
		{Name: "cost", Column: "cost", Disclosure: DisclosureInternal, Staff: rs},
		{Name: "creditClass", Column: "credit_class", Disclosure: DisclosureInternal, Staff: rf},
		{Name: "staffMemo", Column: "staff_memo", Disclosure: DisclosureInternal, Staff: webapicontract.FieldPolicy{Select: true, Write: true}},
	}
}

// ReadPlanSource is the identity of the step-1 read plan a projection was
// derived from.
type ReadPlanSource struct {
	ApplicationID string `json:"applicationId"`
	PatternID     string `json:"patternId"`
	QueryID       string `json:"queryId"`
	DataID        string `json:"dataId"`
}

// ProjectedField is one field of a derived projection.
type ProjectedField struct {
	Name        string                     `json:"name"`
	Column      string                     `json:"column"`
	AttributeID string                     `json:"attributeId,omitempty"`
	Origin      string                     `json:"origin"` // step1-read-plan | sample-extension
	Policy      webapicontract.FieldPolicy `json:"policy"`
}

// Projection is the allowlist of returned and accepted fields for one kind.
type Projection struct {
	Kind     string           `json:"kind"`
	Source   ReadPlanSource   `json:"source"`
	Fields   []ProjectedField `json:"fields"`
	Writable []string         `json:"writable"`
	Digest   string           `json:"digest"`
}

var ErrProjection = errors.New("projection cannot be derived")

// LoadWorkspace loads the workspace that holds the step-1 read plan.
func LoadWorkspace(root string) (*workspace.Result, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return nil, fmt.Errorf("load workspace %s: %w", root, err)
	}
	return &loaded, nil
}

// DeriveProjection derives the projection of one principal kind. Canonical
// fields go through webapicontract.ResolvePublishedBrowse, so a field that the
// step-1 query does not map, or a symbol that differs from the catalog, stops
// the derivation. Internal fields never enter a member projection.
func DeriveProjection(loaded *workspace.Result, kind string) (Projection, error) {
	if kind != KindMember && kind != KindStaff {
		return Projection{}, fmt.Errorf("%w: unknown kind %q", ErrProjection, kind)
	}
	policyOf := func(f CatalogField) webapicontract.FieldPolicy {
		if kind == KindMember {
			if f.Disclosure != DisclosureMember {
				return webapicontract.FieldPolicy{}
			}
			return f.Member
		}
		return f.Staff
	}
	decisions := map[core.CanonicalID]webapicontract.FieldPolicy{}
	byAttribute := map[string]CatalogField{}
	for _, f := range Catalog() {
		if f.AttributeID == "" {
			continue
		}
		byAttribute[f.AttributeID] = f
		if p := policyOf(f); p.Select || p.Filter || p.Sort {
			decisions[core.MustCanonicalID(f.AttributeID)] = p
		}
	}
	published, err := webapicontract.ResolvePublishedBrowse(loaded.World, ReadPlanApplication, ReadPlanPattern, decisions)
	if err != nil {
		return Projection{}, fmt.Errorf("%w: step-1 read plan: %v", ErrProjection, err)
	}
	p := Projection{Kind: kind, Source: ReadPlanSource{
		ApplicationID: string(published.ApplicationID), PatternID: string(published.PatternID),
		QueryID: string(published.QueryID), DataID: string(published.DataID),
	}}
	seen := map[string]bool{}
	for _, pf := range published.Fields {
		cf, ok := byAttribute[string(pf.AttributeID)]
		if !ok || cf.Name != pf.Name {
			return Projection{}, fmt.Errorf("%w: read plan field %s (%s) does not match the catalog", ErrProjection, pf.Name, pf.AttributeID)
		}
		seen[pf.Name] = true
		p.Fields = append(p.Fields, ProjectedField{Name: pf.Name, Column: cf.Column, AttributeID: string(pf.AttributeID), Origin: "step1-read-plan", Policy: pf.Policy})
		if policyOf(cf).Write {
			p.Writable = append(p.Writable, pf.Name)
		}
	}
	for _, f := range Catalog() {
		pol := policyOf(f)
		if f.AttributeID != "" || seen[f.Name] {
			continue
		}
		if pol.Select || pol.Filter || pol.Sort {
			p.Fields = append(p.Fields, ProjectedField{Name: f.Name, Column: f.Column, Origin: "sample-extension", Policy: pol})
		}
		if pol.Write {
			p.Writable = append(p.Writable, f.Name)
		}
	}
	sort.Strings(p.Writable)
	p.Digest = projectionDigest(p)
	return p, nil
}

func projectionDigest(p Projection) string {
	c := p
	c.Digest = ""
	raw, _ := json.Marshal(c)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// VerifyDigest recomputes the digest; a projection edited after derivation
// does not verify.
func (p Projection) VerifyDigest() bool { return p.Digest != "" && projectionDigest(p) == p.Digest }

func (p Projection) fieldsForBind() []webapicontract.Field {
	out := make([]webapicontract.Field, 0, len(p.Fields))
	for _, f := range p.Fields {
		out = append(out, webapicontract.Field{ID: f.Name, Policy: f.Policy})
	}
	return out
}

// BindRead binds the query options to a read plan restricted to this
// projection (webapicontract.BindReadSubset).
func (p Projection) BindRead(values url.Values, ctx webapicontract.TrustedReadContext) (webapicontract.ReadPlan, error) {
	return webapicontract.BindReadSubset(values, p.fieldsForBind(), ctx)
}

// Column returns the column of a projected field.
func (p Projection) Column(name string) (string, bool) {
	for _, f := range p.Fields {
		if f.Name == name {
			return f.Column, true
		}
	}
	return "", false
}

// Has reports whether a name is in the projection at all.
func (p Projection) Has(name string) bool { _, ok := p.Column(name); return ok }

// NonWritable returns the requested field names that the projection does not
// accept, sorted.
func (p Projection) NonWritable(names []string) []string {
	ok := map[string]bool{}
	for _, w := range p.Writable {
		ok[w] = true
	}
	var bad []string
	for _, n := range names {
		if !ok[n] {
			bad = append(bad, n)
		}
	}
	sort.Strings(bad)
	return bad
}

// BindUpdate checks an update against the accepted-field allowlist with
// webapicontract.BindCommand.
func (p Projection) BindUpdate(actionID, expectedRevision string, changes map[string]any, principal string) (webapicontract.CommandIntent, error) {
	writable := map[string]bool{}
	for _, w := range p.Writable {
		writable[w] = true
	}
	req := webapicontract.CommandRequest{ActionID: actionID, ExpectedRevision: expectedRevision, Changes: map[string]webapicontract.FieldChange{}}
	for k, v := range changes {
		if v == nil {
			req.Changes[k] = webapicontract.FieldChange{Kind: "clear"}
		} else {
			req.Changes[k] = webapicontract.FieldChange{Kind: "set", Value: v}
		}
	}
	return webapicontract.BindCommand(req, webapicontract.CommandSpec{ActionID: actionID, Allowed: true, WritableFields: writable, RequiresRevision: true}, principal)
}

// InternalFieldNames lists the names of fields that a member must never see.
func InternalFieldNames() []string {
	var out []string
	for _, f := range Catalog() {
		if f.Disclosure == DisclosureInternal {
			out = append(out, f.Name)
		}
	}
	return out
}
