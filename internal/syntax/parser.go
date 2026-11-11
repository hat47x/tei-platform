package syntax

import (
	"fmt"
	"strconv"
	"strings"

	"tei/internal/application"
	apprelease "tei/internal/application/release"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/migration"
	"tei/internal/database/persistence"
	"tei/internal/database/release"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/permission"
	"tei/internal/plugin"
	"tei/internal/query"
	"tei/internal/realization"
	"tei/internal/releaseplan"
	"tei/internal/resource"
	"tei/internal/semantic"
)

type ParseError struct {
	Pos Position
	End Position
	Msg string
}

func (e *ParseError) Error() string { return fmt.Sprintf("%d:%d: %s", e.Pos.Line, e.Pos.Column, e.Msg) }

type Parser struct {
	lx     *Lexer
	peeked *Token
}

func NewParser(src string) *Parser { return &Parser{lx: NewLexer(src)} }

// ParseWorkspace reads the legacy `tei.workspace.tei` manifest. The name is
// the declared workspace identity; it is returned so callers can report which
// Workspace a read came from instead of inferring it from a path (#3181).
func (p *Parser) ParseWorkspace() (name string, roots []string, err error) {
	if _, err := p.expectText("workspace"); err != nil {
		return "", nil, err
	}
	nameTok, err := p.expect(TokenIdent)
	if err != nil {
		return "", nil, err
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return "", nil, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return "", nil, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		if _, err := p.expectText("source"); err != nil {
			return "", nil, err
		}
		v, err := p.expect(TokenIdent)
		if err != nil {
			return "", nil, err
		}
		roots = append(roots, strings.TrimSuffix(v.Text, "/**"))
	}
	return nameTok.Text, roots, p.expectEOF()
}

func (p *Parser) ParseApplication() (application.Application, error) {
	if _, err := p.expectText("application"); err != nil {
		return application.Application{}, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return application.Application{}, err
	}
	id, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return application.Application{}, p.errAt(idTok, err.Error())
	}
	a := application.Application{ID: id}
	if _, err := p.expect(TokenLBrace); err != nil {
		return a, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return a, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "database":
			r, err := p.parseDatabaseRequirement()
			if err != nil {
				return a, err
			}
			a.Databases = append(a.Databases, r)
		case "uses":
			_, _ = p.next()
			kind, err := p.expect(TokenIdent)
			if err != nil {
				return a, err
			}
			v, err := p.expect(TokenIdent)
			if err != nil {
				return a, err
			}
			ref, err := core.ParseCanonicalID(v.Text)
			if err != nil {
				return a, p.errAt(v, err.Error())
			}
			switch kind.Text {
			case "semantic":
				a.SemanticUses = append(a.SemanticUses, ref)
			case "validation":
				a.Validations = append(a.Validations, ref)
			default:
				return a, p.errAt(kind, "expected semantic or validation after uses")
			}
		case "binds":
			_, _ = p.next()
			kind, err := p.expect(TokenIdent)
			if err != nil {
				return a, err
			}
			if kind.Text != "package" {
				return a, p.errAt(kind, "expected package after binds")
			}
			idTok, err := p.expect(TokenIdent)
			if err != nil {
				return a, err
			}
			packageID, err := core.ParseCanonicalID(idTok.Text)
			if err != nil {
				return a, p.errAt(idTok, err.Error())
			}
			if _, err := p.expect(TokenCaret); err != nil {
				return a, err
			}
			versionTok, err := p.expect(TokenIdent)
			if err != nil {
				return a, err
			}
			a.PackageBindings = append(a.PackageBindings, application.PackageBinding{ID: packageID, Version: versionTok.Text})
		default:
			// A pattern kind is the most likely reason someone writes a stray
			// block here. Naming the placement is the difference between "this
			// token is wrong" and "patterns live in their own file", which the
			// generic message cannot express. application.PatternKinds is the same
			// list the pattern parser accepts, so the two cannot drift (#4128).
			if isApplicationPatternKind(t.Text) {
				return a, p.errAt(t, fmt.Sprintf(
					"%s starts an application pattern, which is not declared in the application body; declare it as \"pattern <application>.<id> %s { ... }\" in its own file",
					strconv.Quote(t.Text), t.Text))
			}
			if misplaced := p.misplacedDeclaration(t, "in the application body"); misplaced != nil {
				return a, misplaced
			}
			return a, p.errAt(t, "expected database, uses, or binds")
		}
	}
	return a, p.expectEOF()
}

// isApplicationPatternKind reports whether the token starts a pattern block.
func isApplicationPatternKind(text string) bool {
	for _, kind := range application.PatternKinds {
		if text == kind {
			return true
		}
	}
	return false
}

func (p *Parser) parseDatabaseRequirement() (application.DatabaseRequirement, error) {
	_, _ = p.next() // database
	aliasTok, err := p.expect(TokenIdent)
	if err != nil {
		return application.DatabaseRequirement{}, err
	}
	alias, err := core.ParseLocalName(aliasTok.Text)
	if err != nil {
		return application.DatabaseRequirement{}, p.errAt(aliasTok, err.Error())
	}
	r := application.DatabaseRequirement{Alias: alias}
	if _, err := p.expect(TokenLBrace); err != nil {
		return r, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return r, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "target":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return r, err
			}
			r.TargetDatabase, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return r, p.errAt(v, err.Error())
			}
		case "access":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return r, err
			}
			r.Access = database.Access(v.Text)
		case "requires":
			_, _ = p.next()
			if _, err := p.expect(TokenLBrace); err != nil {
				return r, err
			}
			for {
				z, err := p.peek()
				if err != nil {
					return r, err
				}
				if z.Kind == TokenRBrace {
					_, _ = p.next()
					break
				}
				if _, err := p.expectText("contract"); err != nil {
					return r, err
				}
				c, err := p.expect(TokenIdent)
				if err != nil {
					return r, err
				}
				cid, err := core.ParseCanonicalID(c.Text)
				if err != nil {
					return r, p.errAt(c, err.Error())
				}
				if _, err := p.expect(TokenCaret); err != nil {
					return r, err
				}
				major, err := p.expect(TokenIdent)
				if err != nil {
					return r, err
				}
				n, err := strconv.Atoi(major.Text)
				if err != nil {
					return r, p.errAt(major, "contract major version must be an integer")
				}
				r.RequiredContracts = append(r.RequiredContracts, database.ContractRequirement{ContractID: cid, CompatibleMajor: n})
			}
		default:
			return r, p.errAt(t, "expected target, access, or requires")
		}
	}
	return r, nil
}

func (p *Parser) ParseDatabase() (database.Database, error) {
	if _, err := p.expectText("database"); err != nil {
		return database.Database{}, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return database.Database{}, err
	}
	id, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return database.Database{}, p.errAt(idTok, err.Error())
	}
	d := database.Database{ID: id}
	if _, err := p.expect(TokenLBrace); err != nil {
		return d, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return d, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "kind":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return d, err
			}
			d.Kind = v.Text
		case "ownership":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return d, err
			}
			d.Ownership = database.Ownership(v.Text)
		case "uses":
			_, _ = p.next()
			if _, err := p.expectText("semantic"); err != nil {
				return d, err
			}
			v, err := p.expect(TokenIdent)
			if err != nil {
				return d, err
			}
			ref, err := core.ParseCanonicalID(v.Text)
			if err != nil {
				return d, p.errAt(v, err.Error())
			}
			d.SemanticUses = append(d.SemanticUses, ref)
		default:
			if misplaced := p.misplacedDeclaration(t, "in the database body"); misplaced != nil {
				return d, misplaced
			}
			return d, p.errAt(t, "expected kind, ownership, or uses")
		}
	}
	return d, p.expectEOF()
}

func (p *Parser) ParseContract() (database.Contract, error) {
	if _, err := p.expectText("contract"); err != nil {
		return database.Contract{}, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return database.Contract{}, err
	}
	id, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return database.Contract{}, p.errAt(idTok, err.Error())
	}
	// A missing version is the mistake a first-time author makes here: the id and
	// the version are both bare tokens, so it looks like the brace can follow the
	// id. The generic `unexpected token "{"` left the author to guess (#4169).
	// The check has to happen BEFORE expect(), which consumes the offending token.
	if next, err := p.peek(); err == nil && next.Kind != TokenIdent {
		return database.Contract{}, p.errAt(next, fmt.Sprintf(
			"expected a version after the contract id %q: write it as \"contract %s v4 { ... }\" (the id and the version are separate tokens)",
			idTok.Text, idTok.Text))
	}
	verTok, err := p.expect(TokenIdent)
	if err != nil {
		return database.Contract{}, err
	}
	ver, err := parseVersion(verTok.Text)
	if err != nil {
		return database.Contract{}, p.errAt(verTok, err.Error())
	}
	c := database.Contract{ID: id, Version: ver}
	if _, err := p.expect(TokenLBrace); err != nil {
		return c, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return c, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "read", "write":
			kind := t.Text
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return c, err
			}
			ref, err := core.ParseCanonicalID(v.Text)
			if err != nil {
				return c, p.errAt(v, err.Error())
			}
			if kind == "read" {
				c.Reads = append(c.Reads, ref)
			} else {
				c.Writes = append(c.Writes, ref)
			}
		case "query":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return c, err
			}
			queryContract, err := p.parseContractQuerySignature(v.Text)
			if err != nil {
				return c, err
			}
			c.Queries = append(c.Queries, queryContract)
		default:
			return c, p.errAt(t, "expected read, write, or query")
		}
	}
	return c, p.expectEOF()
}

func (p *Parser) ParseEnvironment() (environment.Environment, error) {
	if _, err := p.expectText("environment"); err != nil {
		return environment.Environment{}, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return environment.Environment{}, err
	}
	id, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return environment.Environment{}, p.errAt(idTok, err.Error())
	}
	e := environment.Environment{ID: id}
	if _, err := p.expect(TokenLBrace); err != nil {
		return e, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return e, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "plugin", "bundle":
			kind := t.Text
			_, _ = p.next()
			idTok, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			id, err := core.ParseCanonicalID(idTok.Text)
			if err != nil {
				return e, p.errAt(idTok, err.Error())
			}
			if _, err := p.expect(TokenCaret); err != nil {
				return e, err
			}
			majorTok, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			major, err := strconv.Atoi(majorTok.Text)
			if err != nil || major < 0 {
				return e, p.errAt(majorTok, "plugin/bundle major version must be a non-negative integer")
			}
			req := environment.ExtensionRequirement{ID: id, CompatibleMajor: major}
			if kind == "plugin" {
				e.Plugins = append(e.Plugins, req)
			} else {
				e.Bundles = append(e.Bundles, req)
			}
		case "feature":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			fid, err := core.ParseCanonicalID(v.Text)
			if err != nil {
				return e, p.errAt(v, err.Error())
			}
			e.Features = append(e.Features, environment.Feature(fid))
		case "database-instance":
			_, _ = p.next()
			iidTok, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			iid, err := core.ParseCanonicalID(iidTok.Text)
			if err != nil {
				return e, p.errAt(iidTok, err.Error())
			}
			inst := environment.DatabaseInstance{ID: iid}
			if _, err := p.expect(TokenLBrace); err != nil {
				return e, err
			}
			for {
				z, err := p.peek()
				if err != nil {
					return e, err
				}
				if z.Kind == TokenRBrace {
					_, _ = p.next()
					break
				}
				switch z.Text {
				case "definition":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return e, err
					}
					inst.Definition, err = core.ParseCanonicalID(v.Text)
					if err != nil {
						return e, p.errAt(v, err.Error())
					}
				case "provider":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return e, err
					}
					inst.Provider = environment.DatabaseProvider(v.Text)
				default:
					return e, p.errAt(z, "expected definition or provider")
				}
			}
			e.Instances = append(e.Instances, inst)
		case "deployment-executor":
			_, _ = p.next()
			action, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			binding := environment.DeploymentExecutorBinding{Action: deployment.ActionKind(action.Text), Scope: environment.ExecutorScopeGlobal}
			next, err := p.peek()
			if err != nil {
				return e, err
			}
			if next.Text == "for" {
				_, _ = p.next()
				scopeTok, err := p.expect(TokenIdent)
				if err != nil {
					return e, err
				}
				targetTok, err := p.expect(TokenIdent)
				if err != nil {
					return e, err
				}
				binding.Scope = environment.ExecutorScope(scopeTok.Text)
				binding.Target, err = core.ParseCanonicalID(targetTok.Text)
				if err != nil {
					return e, p.errAt(targetTok, err.Error())
				}
			}
			if _, err := p.expect(TokenArrow); err != nil {
				return e, err
			}
			execTok, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			binding.Executor, err = core.ParseCanonicalID(execTok.Text)
			if err != nil {
				return e, p.errAt(execTok, err.Error())
			}
			e.DeploymentExecutors = append(e.DeploymentExecutors, binding)
		case "observation-provider":
			_, _ = p.next()
			kind, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			if _, err := p.expect(TokenArrow); err != nil {
				return e, err
			}
			providerTok, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			providerID, err := core.ParseCanonicalID(providerTok.Text)
			if err != nil {
				return e, p.errAt(providerTok, err.Error())
			}
			e.ObservationProviders = append(e.ObservationProviders, environment.ObservationProviderBinding{Kind: deployment.ObservationKind(kind.Text), Provider: providerID})
		case "evidence-provider":
			_, _ = p.next()
			kind, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			if _, err := p.expect(TokenArrow); err != nil {
				return e, err
			}
			providerTok, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			providerID, err := core.ParseCanonicalID(providerTok.Text)
			if err != nil {
				return e, p.errAt(providerTok, err.Error())
			}
			e.EvidenceProviders = append(e.EvidenceProviders, environment.EvidenceProviderBinding{Kind: deployment.EvidenceKind(kind.Text), Provider: providerID})
		case "resource":
			_, _ = p.next()
			kindTok, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			scopeTok, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			if _, err := p.expect(TokenArrow); err != nil {
				return e, err
			}
			resourceTok, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			resourceID, err := core.ParseCanonicalID(resourceTok.Text)
			if err != nil {
				return e, p.errAt(resourceTok, err.Error())
			}
			binding := environment.ResourceBinding{Kind: permission.Kind(kindTok.Text), Scope: scopeTok.Text, ResourceID: resourceID}
			next, err := p.peek()
			if err != nil {
				return e, err
			}
			if next.Text == "inject" {
				_, _ = p.next()
				deliveryTok, err := p.expect(TokenIdent)
				if err != nil {
					return e, err
				}
				binding.SecretDelivery = resource.SecretDelivery(deliveryTok.Text)
				if binding.SecretDelivery == resource.SecretDeliveryEnvironment || binding.SecretDelivery == resource.SecretDeliveryFile {
					targetTok, err := p.expect(TokenString)
					if err != nil {
						return e, err
					}
					binding.SecretTarget = targetTok.Text
				}
				next, err = p.peek()
				if err != nil {
					return e, err
				}
			}
			if next.Text == "refresh" {
				_, _ = p.next()
				modeTok, err := p.expect(TokenIdent)
				if err != nil {
					return e, err
				}
				binding.Refresh = resource.RefreshMode(modeTok.Text)
			}
			e.ResourceBindings = append(e.ResourceBindings, binding)
		case "lease-policy":
			_, _ = p.next()
			capKind, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			capIDTok, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			capID, err := core.ParseCanonicalID(capIDTok.Text)
			if err != nil {
				return e, p.errAt(capIDTok, err.Error())
			}
			pol := resource.LeasePolicy{}
			for {
				next, err := p.peek()
				if err != nil {
					return e, err
				}
				if next.Kind == TokenRBrace || next.Text == "plugin" || next.Text == "bundle" || next.Text == "feature" || next.Text == "database-instance" || next.Text == "bind" || next.Text == "resource" || next.Text == "grant" || next.Text == "lease-policy" || next.Text == "deployment-executor" || next.Text == "observation-provider" || next.Text == "evidence-provider" {
					break
				}
				key, err := p.expect(TokenIdent)
				if err != nil {
					return e, err
				}
				val, err := p.expect(TokenIdent)
				if err != nil {
					return e, err
				}
				switch key.Text {
				case "max-lifetime":
					pol.MaxLifetime = val.Text
				case "renew-before":
					pol.RenewBefore = val.Text
				case "revoke-timeout":
					pol.RevokeTimeout = val.Text
				case "renewal-failure":
					pol.RenewalFailure = resource.RenewalFailurePolicy(val.Text)
				default:
					return e, p.errAt(key, "unknown lease-policy field "+key.Text)
				}
			}
			e.LeasePolicies = append(e.LeasePolicies, environment.CapabilityLeasePolicy{CapabilityKind: capKind.Text, CapabilityID: capID, Policy: pol})
		case "grant":
			_, _ = p.next()
			capKind, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			capIDTok, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			capID, err := core.ParseCanonicalID(capIDTok.Text)
			if err != nil {
				return e, p.errAt(capIDTok, err.Error())
			}
			if _, err := p.expectText("permission"); err != nil {
				return e, err
			}
			kindTok, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			accessTok, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			scopeTok, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			e.PermissionGrants = append(e.PermissionGrants, permission.Grant{
				CapabilityKind: capKind.Text, CapabilityID: capID,
				Permission: permission.Spec{Kind: permission.Kind(kindTok.Text), Access: permission.Access(accessTok.Text), Scope: scopeTok.Text},
			})
		case "bind":
			_, _ = p.next()
			lhs, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			dot := strings.LastIndex(lhs.Text, ".")
			if dot <= 0 || dot == len(lhs.Text)-1 {
				return e, p.errAt(lhs, "binding must be application.alias")
			}
			appID, err := core.ParseCanonicalID(lhs.Text[:dot])
			if err != nil {
				return e, p.errAt(lhs, err.Error())
			}
			alias, err := core.ParseLocalName(lhs.Text[dot+1:])
			if err != nil {
				return e, p.errAt(lhs, err.Error())
			}
			if _, err := p.expect(TokenArrow); err != nil {
				return e, err
			}
			rhs, err := p.expect(TokenIdent)
			if err != nil {
				return e, err
			}
			instID, err := core.ParseCanonicalID(rhs.Text)
			if err != nil {
				return e, p.errAt(rhs, err.Error())
			}
			e.Bindings = append(e.Bindings, environment.ApplicationDatabaseBinding{Application: appID, Alias: alias, Instance: instID})
		default:
			return e, p.errAt(t, "expected plugin, bundle, feature, database-instance, bind, resource, grant, lease-policy, deployment-executor, observation-provider, or evidence-provider")
		}
	}
	return e, p.expectEOF()
}

func (p *Parser) ParseQuery(owner core.CanonicalID) (query.EmbeddedSQL, error) {
	out, err := p.ParseQueryAuthoring(owner)
	if err != nil {
		return query.EmbeddedSQL{}, err
	}
	if err := nativeQueryRequiresWorkspace(out); err != nil {
		return out.Value, err
	}
	return out.Value, nil
}

func (p *Parser) ParseRealization() (realization.CustomGoBinding, error) {
	if _, err := p.expectText("realize"); err != nil {
		return realization.CustomGoBinding{}, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return realization.CustomGoBinding{}, err
	}
	id, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return realization.CustomGoBinding{}, p.errAt(idTok, err.Error())
	}
	if _, err := p.expectText("with"); err != nil {
		return realization.CustomGoBinding{}, err
	}
	if _, err := p.expectText("custom"); err != nil {
		return realization.CustomGoBinding{}, err
	}
	if _, err := p.expectText("go"); err != nil {
		return realization.CustomGoBinding{}, err
	}
	if err := p.expectEOF(); err != nil {
		return realization.CustomGoBinding{}, err
	}
	return realization.CustomGoBinding{Capability: id}, nil
}

// ParseMigrationProvider parses the database-local integration point for an
// external migration/versioning tool such as Atlas, Flyway or Liquibase.
func (p *Parser) ParseMigrationProvider() (migration.ProviderConfig, error) {
	if _, err := p.expectText("migration-provider"); err != nil {
		return migration.ProviderConfig{}, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return migration.ProviderConfig{}, err
	}
	db, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return migration.ProviderConfig{}, p.errAt(idTok, err.Error())
	}
	c := migration.ProviderConfig{Database: db}
	if _, err := p.expect(TokenLBrace); err != nil {
		return c, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return c, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "tool":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return c, err
			}
			c.Tool = migration.Tool(v.Text)
		case "mode":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return c, err
			}
			c.Mode = migration.Mode(v.Text)
		case "source":
			_, _ = p.next()
			v, err := p.expect(TokenString)
			if err != nil {
				return c, err
			}
			c.Source = v.Text
		default:
			return c, p.errAt(t, "expected tool, mode, or source")
		}
	}
	return c, p.expectEOF()
}

// ParseMigration parses a versioned database migration asset. DDL/DML SQL is
// preserved verbatim as a provider-specific realization; TEI does not treat
// SQL text as semantic domain truth.
func (p *Parser) ParseMigration() (migration.Definition, error) {
	if _, err := p.expectText("migration"); err != nil {
		return migration.Definition{}, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return migration.Definition{}, err
	}
	m := migration.Definition{ID: idTok.Text}
	if _, err := p.expect(TokenLBrace); err != nil {
		return m, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return m, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "database":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return m, err
			}
			m.Database, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return m, p.errAt(v, err.Error())
			}
		case "parent":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return m, err
			}
			m.Parents = append(m.Parents, v.Text)
		case "recovery":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return m, err
			}
			m.Recovery = migration.RecoveryStrategy(v.Text)
		case "schema", "data", "verify":
			kindTok := t.Text
			_, _ = p.next()
			if _, err := p.expectText("sql"); err != nil {
				return m, err
			}
			pv, err := p.expect(TokenIdent)
			if err != nil {
				return m, err
			}
			provider := environment.DatabaseProvider(pv.Text)
			if _, err := p.expect(TokenLBrace); err != nil {
				return m, err
			}
			body, _, err := p.lx.ReadRawBlock()
			if err != nil {
				return m, err
			}
			p.peeked = nil
			kind := migration.StepSchema
			if kindTok == "data" {
				kind = migration.StepData
			} else if kindTok == "verify" {
				kind = migration.StepVerify
			}
			m.Steps = append(m.Steps, migration.Step{Kind: kind, SQL: &migration.SQLStep{Provider: provider, Source: strings.TrimSpace(body)}})
		default:
			return m, p.errAt(t, "expected database, parent, recovery, schema, data, or verify")
		}
	}
	return m, p.expectEOF()
}

// ParseSolutionRelease parses a release-orchestration plan. It references
// independently versioned Application/Database releases rather than owning
// their definitions.
func (p *Parser) ParseSolutionRelease() (releaseplan.Definition, error) {
	if _, err := p.expectText("solution-release"); err != nil {
		return releaseplan.Definition{}, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return releaseplan.Definition{}, err
	}
	id, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return releaseplan.Definition{}, p.errAt(idTok, err.Error())
	}
	vTok, err := p.expect(TokenIdent)
	if err != nil {
		return releaseplan.Definition{}, err
	}
	r := releaseplan.Definition{ID: id, Version: vTok.Text}
	if _, err := p.expect(TokenLBrace); err != nil {
		return r, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return r, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "step":
			_, _ = p.next()
			sidTok, err := p.expect(TokenIdent)
			if err != nil {
				return r, err
			}
			sid, err := core.ParseLocalName(sidTok.Text)
			if err != nil {
				return r, p.errAt(sidTok, err.Error())
			}
			st := releaseplan.Step{ID: sid, Phase: releaseplan.PhaseOther}
			if _, err := p.expect(TokenLBrace); err != nil {
				return r, err
			}
			for {
				z, err := p.peek()
				if err != nil {
					return r, err
				}
				if z.Kind == TokenRBrace {
					_, _ = p.next()
					break
				}
				switch z.Text {
				case "kind":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return r, err
					}
					st.Kind = releaseplan.StepKind(v.Text)
				case "phase":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return r, err
					}
					st.Phase = releaseplan.Phase(v.Text)
				case "database":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return r, err
					}
					st.Database, err = core.ParseCanonicalID(v.Text)
					if err != nil {
						return r, p.errAt(v, err.Error())
					}
				case "application":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return r, err
					}
					st.Application, err = core.ParseCanonicalID(v.Text)
					if err != nil {
						return r, p.errAt(v, err.Error())
					}
				case "version":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return r, err
					}
					st.Version = v.Text
				case "strategy":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return r, err
					}
					st.RolloutStrategy = releaseplan.RolloutStrategy(v.Text)
				case "traffic":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return r, err
					}
					n, err := strconv.Atoi(v.Text)
					if err != nil {
						return r, p.errAt(v, "traffic basis points must be an integer")
					}
					st.TrafficSteps = append(st.TrafficSteps, n)
				case "contract":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return r, err
					}
					st.Contract, err = core.ParseCanonicalID(v.Text)
					if err != nil {
						return r, p.errAt(v, err.Error())
					}
					if _, err := p.expect(TokenCaret); err != nil {
						return r, err
					}
					m, err := p.expect(TokenIdent)
					if err != nil {
						return r, err
					}
					n, err := strconv.Atoi(m.Text)
					if err != nil {
						return r, p.errAt(m, "contract major version must be an integer")
					}
					st.CompatibleMajor = n
				default:
					return r, p.errAt(z, "expected kind, phase, database, application, version, strategy, traffic, or contract")
				}
			}
			r.Steps = append(r.Steps, st)
		case "after":
			_, _ = p.next()
			stepTok, err := p.expect(TokenIdent)
			if err != nil {
				return r, err
			}
			step, err := core.ParseLocalName(stepTok.Text)
			if err != nil {
				return r, p.errAt(stepTok, err.Error())
			}
			depTok, err := p.expect(TokenIdent)
			if err != nil {
				return r, err
			}
			dep, err := core.ParseLocalName(depTok.Text)
			if err != nil {
				return r, p.errAt(depTok, err.Error())
			}
			r.After = append(r.After, releaseplan.Dependency{Step: step, After: dep})
		default:
			return r, p.errAt(t, "expected step or after")
		}
	}
	return r.Normalize(), p.expectEOF()
}

func parsePluginVersionToken(tok Token) (string, error) {
	v := strings.TrimPrefix(tok.Text, "v")
	if !core.ValidNumericVersion(v) {
		return "", fmt.Errorf("plugin/bundle version must be vMAJOR.MINOR.PATCH or MAJOR.MINOR.PATCH")
	}
	return v, nil
}

func (p *Parser) ParsePlugin() (plugin.Definition, error) {
	if _, err := p.expectText("plugin"); err != nil {
		return plugin.Definition{}, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return plugin.Definition{}, err
	}
	id, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return plugin.Definition{}, p.errAt(idTok, err.Error())
	}
	verTok, err := p.expect(TokenIdent)
	if err != nil {
		return plugin.Definition{}, err
	}
	version, err := parsePluginVersionToken(verTok)
	if err != nil {
		return plugin.Definition{}, p.errAt(verTok, err.Error())
	}
	d := plugin.Definition{ID: id, Version: version}
	if _, err := p.expect(TokenLBrace); err != nil {
		return d, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return d, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "runtime":
			_, _ = p.next()
			kindTok, err := p.expect(TokenIdent)
			if err != nil {
				return d, err
			}
			d.Runtime.Kind = plugin.RuntimeKind(kindTok.Text)
			if d.Runtime.Kind == plugin.RuntimeProcess {
				cmd, err := p.expect(TokenString)
				if err != nil {
					return d, err
				}
				if strings.ContainsAny(cmd.Text, " \t\r\n") {
					return d, p.errAt(cmd, "process command must be a single executable name/path; use args for arguments")
				}
				d.Runtime.Command = cmd.Text
				next, err := p.peek()
				if err != nil {
					return d, err
				}
				if next.Text == "args" {
					argsTok, _ := p.next()
					for {
						arg, err := p.peek()
						if err != nil {
							return d, err
						}
						if arg.Kind != TokenString {
							break
						}
						v, _ := p.next()
						d.Runtime.Args = append(d.Runtime.Args, v.Text)
					}
					if len(d.Runtime.Args) == 0 {
						return d, p.errAt(argsTok, "args requires at least one quoted argument")
					}
					next, err = p.peek()
					if err != nil {
						return d, err
					}
				}
				if next.Text == "sandbox" {
					_, _ = p.next()
					pol, err := p.expect(TokenIdent)
					if err != nil {
						return d, err
					}
					d.Runtime.Sandbox = plugin.SandboxPolicy(pol.Text)
				}
			}
		case "artifact":
			_, _ = p.next()
			if _, err := p.expect(TokenLBrace); err != nil {
				return d, err
			}
			for {
				z, err := p.peek()
				if err != nil {
					return d, err
				}
				if z.Kind == TokenRBrace {
					_, _ = p.next()
					break
				}
				key := z.Text
				_, _ = p.next()
				val, err := p.expect(TokenString)
				if err != nil {
					return d, err
				}
				switch key {
				case "digest":
					d.Artifact.Digest = val.Text
				case "signature":
					d.Artifact.SignatureRef = val.Text
				case "provenance":
					d.Artifact.Provenance = val.Text
				default:
					return d, p.errAt(z, "expected digest, signature, or provenance")
				}
			}
		case "requires":
			_, _ = p.next()
			if _, err := p.expectText("plugin"); err != nil {
				return d, err
			}
			refTok, err := p.expect(TokenIdent)
			if err != nil {
				return d, err
			}
			ref, err := core.ParseCanonicalID(refTok.Text)
			if err != nil {
				return d, p.errAt(refTok, err.Error())
			}
			if _, err := p.expect(TokenCaret); err != nil {
				return d, err
			}
			mTok, err := p.expect(TokenIdent)
			if err != nil {
				return d, err
			}
			major, err := strconv.Atoi(mTok.Text)
			if err != nil || major < 0 {
				return d, p.errAt(mTok, "plugin major version must be a non-negative integer")
			}
			d.Requires = append(d.Requires, plugin.Requirement{Plugin: ref, CompatibleMajor: major})
		case "external":
			_, _ = p.next()
			scopeTok, err := p.expect(TokenIdent)
			if err != nil {
				return d, err
			}
			kindTok, err := p.expect(TokenIdent)
			if err != nil {
				return d, err
			}
			nameTok, err := p.expect(TokenIdent)
			if err != nil {
				return d, err
			}
			d.External = append(d.External, plugin.ExternalRequirement{Scope: scopeTok.Text, Kind: kindTok.Text, Name: nameTok.Text})
		case "provides":
			_, _ = p.next()
			kindTok, err := p.expect(TokenIdent)
			if err != nil {
				return d, err
			}
			idTok, err := p.expect(TokenIdent)
			if err != nil {
				return d, err
			}
			capID, err := core.ParseCanonicalID(idTok.Text)
			if err != nil {
				return d, p.errAt(idTok, err.Error())
			}
			cap := plugin.Capability{Kind: plugin.CapabilityKind(kindTok.Text), ID: capID}
			if _, err := p.expect(TokenLBrace); err != nil {
				return d, err
			}
			for {
				z, err := p.peek()
				if err != nil {
					return d, err
				}
				if z.Kind == TokenRBrace {
					_, _ = p.next()
					break
				}
				switch z.Text {
				case "action":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return d, err
					}
					cap.Actions = append(cap.Actions, deployment.ActionKind(v.Text))
				case "observation":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return d, err
					}
					cap.Observations = append(cap.Observations, deployment.ObservationKind(v.Text))
				case "evidence":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return d, err
					}
					cap.Evidence = append(cap.Evidence, deployment.EvidenceKind(v.Text))
				case "provider":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return d, err
					}
					cap.DatabaseProviders = append(cap.DatabaseProviders, environment.DatabaseProvider(v.Text))
				case "tool":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return d, err
					}
					cap.MigrationTools = append(cap.MigrationTools, migration.Tool(v.Text))
				case "feature":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return d, err
					}
					cap.Features = append(cap.Features, environment.Feature(v.Text))
				case "permission":
					_, _ = p.next()
					k, err := p.expect(TokenIdent)
					if err != nil {
						return d, err
					}
					a, err := p.expect(TokenIdent)
					if err != nil {
						return d, err
					}
					sc, err := p.expect(TokenIdent)
					if err != nil {
						return d, err
					}
					cap.Permissions = append(cap.Permissions, permission.Spec{Kind: permission.Kind(k.Text), Access: permission.Access(a.Text), Scope: sc.Text})
				default:
					return d, p.errAt(z, "expected action, observation, evidence, provider, tool, feature, or permission")
				}
			}
			d.Provides = append(d.Provides, cap)
		default:
			return d, p.errAt(t, "expected runtime, artifact, requires, external, or provides")
		}
	}
	if err := p.expectEOF(); err != nil {
		return d, err
	}
	return plugin.NormalizeDefinition(d), nil
}

func (p *Parser) ParseBundle() (plugin.Bundle, error) {
	if _, err := p.expectText("bundle"); err != nil {
		return plugin.Bundle{}, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return plugin.Bundle{}, err
	}
	id, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return plugin.Bundle{}, p.errAt(idTok, err.Error())
	}
	verTok, err := p.expect(TokenIdent)
	if err != nil {
		return plugin.Bundle{}, err
	}
	version, err := parsePluginVersionToken(verTok)
	if err != nil {
		return plugin.Bundle{}, p.errAt(verTok, err.Error())
	}
	b := plugin.Bundle{ID: id, Version: version}
	if _, err := p.expect(TokenLBrace); err != nil {
		return b, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return b, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		if _, err := p.expectText("plugin"); err != nil {
			return b, err
		}
		pidTok, err := p.expect(TokenIdent)
		if err != nil {
			return b, err
		}
		pid, err := core.ParseCanonicalID(pidTok.Text)
		if err != nil {
			return b, p.errAt(pidTok, err.Error())
		}
		if _, err := p.expect(TokenCaret); err != nil {
			return b, err
		}
		mTok, err := p.expect(TokenIdent)
		if err != nil {
			return b, err
		}
		major, err := strconv.Atoi(mTok.Text)
		if err != nil || major < 0 {
			return b, p.errAt(mTok, "plugin major version must be a non-negative integer")
		}
		b.Plugins = append(b.Plugins, plugin.Requirement{Plugin: pid, CompatibleMajor: major})
	}
	return b, p.expectEOF()
}

// ParseApplicationRelease parses an immutable Application release snapshot.
// Database requirements are copied into the release so compatibility of old
// releases does not change when the current Application definition evolves.
func (p *Parser) ParseApplicationRelease() (apprelease.Definition, error) {
	if _, err := p.expectText("application-release"); err != nil {
		return apprelease.Definition{}, err
	}
	aTok, err := p.expect(TokenIdent)
	if err != nil {
		return apprelease.Definition{}, err
	}
	aid, err := core.ParseCanonicalID(aTok.Text)
	if err != nil {
		return apprelease.Definition{}, p.errAt(aTok, err.Error())
	}
	vTok, err := p.expect(TokenIdent)
	if err != nil {
		return apprelease.Definition{}, err
	}
	r := apprelease.Definition{Application: aid, Version: vTok.Text}
	if _, err := p.expect(TokenLBrace); err != nil {
		return r, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return r, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "source-revision":
			_, _ = p.next()
			v, err := p.expect(TokenString)
			if err != nil {
				return r, err
			}
			r.SourceRevision = v.Text
		case "database":
			db, err := p.parseDatabaseRequirement()
			if err != nil {
				return r, err
			}
			r.Databases = append(r.Databases, db)
		default:
			return r, p.errAt(t, "expected source-revision or database")
		}
	}
	return r.Normalize(), p.expectEOF()
}

// ParseDatabaseRelease parses an independently versioned Database release.
// It snapshots the delivery boundary (migrations/contracts/readiness) without
// coupling it to any Application release.
func (p *Parser) ParseDatabaseRelease() (release.Definition, error) {
	if _, err := p.expectText("database-release"); err != nil {
		return release.Definition{}, err
	}
	dbTok, err := p.expect(TokenIdent)
	if err != nil {
		return release.Definition{}, err
	}
	db, err := core.ParseCanonicalID(dbTok.Text)
	if err != nil {
		return release.Definition{}, p.errAt(dbTok, err.Error())
	}
	verTok, err := p.expect(TokenIdent)
	if err != nil {
		return release.Definition{}, err
	}
	r := release.Definition{Database: db, Version: verTok.Text, DataReadiness: release.DataUnknown}
	if _, err := p.expect(TokenLBrace); err != nil {
		return r, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return r, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "schema-fingerprint":
			_, _ = p.next()
			v, err := p.expect(TokenString)
			if err != nil {
				return r, err
			}
			r.SchemaFingerprint = v.Text
		case "includes":
			_, _ = p.next()
			if _, err := p.expectText("migration"); err != nil {
				return r, err
			}
			v, err := p.expect(TokenIdent)
			if err != nil {
				return r, err
			}
			r.Migrations = append(r.Migrations, v.Text)
		case "provides":
			_, _ = p.next()
			if _, err := p.expectText("contract"); err != nil {
				return r, err
			}
			c, err := p.expect(TokenIdent)
			if err != nil {
				return r, err
			}
			cid, err := core.ParseCanonicalID(c.Text)
			if err != nil {
				return r, p.errAt(c, err.Error())
			}
			v, err := p.expect(TokenIdent)
			if err != nil {
				return r, err
			}
			cv, err := parseVersion(v.Text)
			if err != nil {
				return r, p.errAt(v, err.Error())
			}
			r.Provides = append(r.Provides, release.ProvidedContract{ID: cid, Version: cv})
		case "data-readiness":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return r, err
			}
			r.DataReadiness = release.DataReadiness(v.Text)
		default:
			return r, p.errAt(t, "expected schema-fingerprint, includes, provides, or data-readiness")
		}
	}
	return r, p.expectEOF()
}

// ParsePersistence parses a database-owned relational persistence mapping.
// M0.6 intentionally keeps physical optimization small while preserving a
// provider-neutral storage type that can be realized for multiple DBMSs.
func (p *Parser) ParsePersistence() (persistence.Mapping, error) {
	if _, err := p.expectText("persist"); err != nil {
		return persistence.Mapping{}, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return persistence.Mapping{}, err
	}
	id, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return persistence.Mapping{}, p.errAt(idTok, err.Error())
	}
	m := persistence.Mapping{ID: id}
	if _, err := p.expect(TokenLBrace); err != nil {
		return m, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return m, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "database":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return m, err
			}
			m.Database, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return m, p.errAt(v, err.Error())
			}
		case "semantic":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return m, err
			}
			m.SemanticData, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return m, p.errAt(v, err.Error())
			}
		case "table":
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return m, err
			}
			m.Table = v.Text
		case "column":
			_, _ = p.next()
			a, err := p.expect(TokenIdent)
			if err != nil {
				return m, err
			}
			attr, err := core.ParseCanonicalID(a.Text)
			if err != nil {
				return m, p.errAt(a, err.Error())
			}
			if _, err := p.expect(TokenArrow); err != nil {
				return m, err
			}
			c, err := p.expect(TokenIdent)
			if err != nil {
				return m, err
			}
			if _, err := p.expectText("storage"); err != nil {
				return m, err
			}
			st, err := p.expect(TokenIdent)
			if err != nil {
				return m, err
			}
			n, err := p.expect(TokenIdent)
			if err != nil {
				return m, err
			}
			m.Columns = append(m.Columns, persistence.ColumnMapping{Attribute: attr, Column: c.Text, Storage: persistence.StorageType(st.Text), Nullability: persistence.Nullability(n.Text)})
		case "primary-key":
			_, _ = p.next()
			a, err := p.expect(TokenIdent)
			if err != nil {
				return m, err
			}
			attr, err := core.ParseCanonicalID(a.Text)
			if err != nil {
				return m, p.errAt(a, err.Error())
			}
			m.PrimaryKey = append(m.PrimaryKey, attr)
		case "index", "unique-index":
			unique := t.Text == "unique-index"
			_, _ = p.next()
			name, err := p.expect(TokenIdent)
			if err != nil {
				return m, err
			}
			next, err := p.peek()
			if err != nil {
				return m, err
			}
			if next.Text == "on" {
				_, _ = p.next()
				a, err := p.expect(TokenIdent)
				if err != nil {
					return m, err
				}
				attr, err := core.ParseCanonicalID(a.Text)
				if err != nil {
					return m, p.errAt(a, err.Error())
				}
				m.Indexes = append(m.Indexes, persistence.IndexMapping{Name: name.Text, Attributes: []core.CanonicalID{attr}, Unique: unique})
				break
			}
			if next.Kind != TokenLBrace {
				return m, p.errAt(next, "expected on or advanced index block")
			}
			index, err := p.parseAdvancedIndexBody(name, unique)
			if err != nil {
				return m, err
			}
			m.Indexes = append(m.Indexes, index)
		case "unique":
			_, _ = p.next()
			name, err := p.expect(TokenIdent)
			if err != nil {
				return m, err
			}
			constraint := persistence.UniqueConstraintMapping{Name: name.Text}
			if _, err := p.expect(TokenLBrace); err != nil {
				return m, err
			}
			for {
				z, err := p.peek()
				if err != nil {
					return m, err
				}
				if z.Kind == TokenRBrace {
					_, _ = p.next()
					break
				}
				if _, err := p.expectText("attribute"); err != nil {
					return m, err
				}
				a, err := p.expect(TokenIdent)
				if err != nil {
					return m, err
				}
				attr, err := core.ParseCanonicalID(a.Text)
				if err != nil {
					return m, p.errAt(a, err.Error())
				}
				constraint.Attributes = append(constraint.Attributes, attr)
			}
			if len(constraint.Attributes) == 0 {
				return m, p.errAt(name, "unique requires at least one attribute")
			}
			m.UniqueConstraints = append(m.UniqueConstraints, constraint)
		case "foreign-key":
			_, _ = p.next()
			name, err := p.expect(TokenIdent)
			if err != nil {
				return m, err
			}
			fk := persistence.ForeignKeyMapping{Name: name.Text}
			if _, err := p.expect(TokenLBrace); err != nil {
				return m, err
			}
			for {
				z, err := p.peek()
				if err != nil {
					return m, err
				}
				if z.Kind == TokenRBrace {
					_, _ = p.next()
					break
				}
				switch z.Text {
				case "source":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return m, err
					}
					id, err := core.ParseCanonicalID(v.Text)
					if err != nil {
						return m, p.errAt(v, err.Error())
					}
					fk.Attributes = append(fk.Attributes, id)
				case "references":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return m, err
					}
					if fk.ReferencedMapping != "" {
						return m, p.errAt(v, "foreign-key references may be declared only once")
					}
					fk.ReferencedMapping, err = core.ParseCanonicalID(v.Text)
					if err != nil {
						return m, p.errAt(v, err.Error())
					}
				case "target":
					_, _ = p.next()
					v, err := p.expect(TokenIdent)
					if err != nil {
						return m, err
					}
					id, err := core.ParseCanonicalID(v.Text)
					if err != nil {
						return m, p.errAt(v, err.Error())
					}
					fk.ReferencedAttributes = append(fk.ReferencedAttributes, id)
				default:
					return m, p.errAt(z, "expected source, references, or target")
				}
			}
			if len(fk.Attributes) == 0 {
				return m, p.errAt(name, "foreign-key requires at least one source attribute")
			}
			if fk.ReferencedMapping == "" {
				return m, p.errAt(name, "foreign-key requires references mapping")
			}
			if len(fk.Attributes) != len(fk.ReferencedAttributes) {
				return m, p.errAt(name, "foreign-key source/target attribute count mismatch")
			}
			m.ForeignKeys = append(m.ForeignKeys, fk)
		default:
			return m, p.errAt(t, "expected database, semantic, table, column, primary-key, unique, index, unique-index, or foreign-key")
		}
	}
	return m, p.expectEOF()
}

// ParseSemanticFile parses a semantic source file. The syntax remains intentionally
// small: a namespace followed by type, data, capability and typed validation declarations.
// parseOptionalSemanticFormatVersion consumes an optional leading
// "formatVersion N" header. Zero means the file did not declare a version and
// inherits the Workspace manifest version. This describes the syntax
// generation; it is not the content version of a contract.
func (p *Parser) parseOptionalSemanticFormatVersion() (int, error) {
	t, err := p.peek()
	if err != nil {
		return 0, err
	}
	if t.Kind != TokenIdent || t.Text != "formatVersion" {
		return 0, nil
	}
	_, _ = p.next()
	valueTok, err := p.expect(TokenIdent)
	if err != nil {
		return 0, err
	}
	version, convErr := strconv.Atoi(valueTok.Text)
	if convErr != nil || version < 1 {
		return 0, p.errAt(valueTok, "formatVersion must be a positive integer")
	}
	return version, nil
}

func (p *Parser) ParseSemanticFile() (semantic.Namespace, error) {
	formatVersion, err := p.parseOptionalSemanticFormatVersion()
	if err != nil {
		return semantic.Namespace{}, err
	}
	if _, err := p.expectText("namespace"); err != nil {
		return semantic.Namespace{}, err
	}
	nsTok, err := p.expect(TokenIdent)
	if err != nil {
		return semantic.Namespace{}, err
	}
	nsID, err := core.ParseCanonicalID(nsTok.Text)
	if err != nil {
		return semantic.Namespace{}, p.errAt(nsTok, err.Error())
	}
	if strings.Contains(string(nsID), ".") {
		return semantic.Namespace{}, p.errAt(nsTok, "semantic namespace must be a top-level canonical id")
	}
	ns := semantic.Namespace{ID: nsID, FormatVersion: formatVersion}

	for {
		t, err := p.peek()
		if err != nil {
			return ns, err
		}
		if t.Kind == TokenEOF {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "type":
			typ, err := p.parseSemanticType(nsID)
			if err != nil {
				return ns, err
			}
			ns.Types = append(ns.Types, typ)
		case "data":
			d, err := p.parseSemanticData(nsID)
			if err != nil {
				return ns, err
			}
			ns.Data = append(ns.Data, d)
		case "capability":
			c, err := p.parseSemanticCapability(nsID)
			if err != nil {
				return ns, err
			}
			ns.Capabilities = append(ns.Capabilities, c)
		case "validation":
			v, err := p.parseSemanticValidation(nsID)
			if err != nil {
				return ns, err
			}
			ns.Validations = append(ns.Validations, v)
		case "calendar":
			cal, err := p.parseSemanticCalendar(nsID)
			if err != nil {
				return ns, err
			}
			ns.Calendars = append(ns.Calendars, cal)
		case "period":
			prd, err := p.parseSemanticPeriod(nsID)
			if err != nil {
				return ns, err
			}
			ns.Periods = append(ns.Periods, prd)
		case "effect-rule":
			rule, err := p.parseSemanticEffectRule(nsID)
			if err != nil {
				return ns, err
			}
			ns.EffectRules = append(ns.EffectRules, rule)
		case "run-plan":
			plan, err := p.parseSemanticRunPlan(nsID)
			if err != nil {
				return ns, err
			}
			ns.RunPlans = append(ns.RunPlans, plan)
		default:
			// A top-level keyword from another domain (an application, a
			// migration) is a placement mistake, not a typo. Name it, then fall
			// back to the declaration list this namespace actually accepts (#4169).
			if misplaced := p.misplacedDeclaration(t, "in a semantic namespace"); misplaced != nil {
				return ns, misplaced
			}
			return ns, p.errAt(t, "expected type, data, capability, validation, calendar, period, effect-rule, or run-plan")
		}
	}
	return ns, nil
}

func (p *Parser) parseSemanticType(namespace core.CanonicalID) (semantic.Type, error) {
	_, _ = p.next() // type
	symTok, err := p.expect(TokenIdent)
	if err != nil {
		return semantic.Type{}, err
	}
	seg, err := core.SymbolToSegment(symTok.Text)
	if err != nil {
		return semantic.Type{}, p.errAt(symTok, err.Error())
	}
	id, _ := core.ParseCanonicalID(string(namespace) + "." + seg)
	if _, err := p.expectText("from"); err != nil {
		return semantic.Type{}, err
	}
	underTok, err := p.expect(TokenIdent)
	if err != nil {
		return semantic.Type{}, err
	}
	under, err := core.ParseCanonicalID(underTok.Text)
	if err != nil {
		return semantic.Type{}, p.errAt(underTok, err.Error())
	}
	var values []string
	if nt, _ := p.peek(); nt.Text == "values" {
		_, _ = p.next()
		if _, err := p.expect(TokenLBrace); err != nil {
			return semantic.Type{}, err
		}
		for {
			t, err := p.peek()
			if err != nil {
				return semantic.Type{}, err
			}
			if t.Kind == TokenRBrace {
				_, _ = p.next()
				break
			}
			value, err := p.expect(TokenString)
			if err != nil {
				return semantic.Type{}, err
			}
			values = append(values, value.Text)
		}
	}
	return semantic.Type{ID: id, Symbol: symTok.Text, Kind: semantic.TypeAlias, Underlying: under, Values: values}, nil
}

func (p *Parser) parseSemanticData(namespace core.CanonicalID) (semantic.Data, error) {
	_, _ = p.next() // data
	symTok, err := p.expect(TokenIdent)
	if err != nil {
		return semantic.Data{}, err
	}
	seg, err := core.SymbolToSegment(symTok.Text)
	if err != nil {
		return semantic.Data{}, p.errAt(symTok, err.Error())
	}
	id, err := core.ParseCanonicalID(string(namespace) + "." + seg)
	if err != nil {
		return semantic.Data{}, p.errAt(symTok, err.Error())
	}
	d := semantic.Data{ID: id, Symbol: symTok.Text}
	if _, err := p.expect(TokenLBrace); err != nil {
		return d, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return d, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "attribute":
			_, _ = p.next()
			aSym, err := p.expect(TokenIdent)
			if err != nil {
				return d, err
			}
			if _, err := p.expectText("type"); err != nil {
				return d, err
			}
			typeTok, err := p.expect(TokenIdent)
			if err != nil {
				return d, err
			}
			typeRef, err := core.ParseCanonicalID(typeTok.Text)
			if err != nil {
				return d, p.errAt(typeTok, err.Error())
			}
			desc := ""
			disp := ""
			required := false
			immutable := false
			role := ""
			roleDeclared := false
			var declaredValue *string
		attributeOptions:
			for {
				nt, err := p.peek()
				if err != nil {
					return d, err
				}
				switch nt.Text {
				case "description":
					_, _ = p.next()
					descTok, err := p.expect(TokenString)
					if err != nil {
						return d, err
					}
					desc = descTok.Text
				case "display":
					_, _ = p.next()
					dispTok, err := p.expect(TokenString)
					if err != nil {
						return d, err
					}
					disp = dispTok.Text
				case "value":
					_, _ = p.next()
					valueTok, err := p.expect(TokenString)
					if err != nil {
						return d, err
					}
					value := valueTok.Text
					declaredValue = &value
				case "required":
					_, _ = p.next()
					required = true
				case "immutable":
					_, _ = p.next()
					immutable = true
				case "role":
					if roleDeclared {
						return d, p.errAt(nt, "duplicate role option for attribute "+aSym.Text)
					}
					_, _ = p.next()
					roleTok, err := p.expect(TokenIdent)
					if err != nil {
						return d, err
					}
					role = roleTok.Text
					roleDeclared = true
				default:
					break attributeOptions
				}
			}
			aseg, err := core.SymbolToSegment(aSym.Text)
			if err != nil {
				return d, p.errAt(aSym, err.Error())
			}
			aid, _ := core.ParseCanonicalID(string(id) + "." + aseg)
			d.Attributes = append(d.Attributes, semantic.Attribute{ID: aid, Symbol: aSym.Text, Description: desc, Display: disp, TypeRef: typeRef, Required: required, Immutable: immutable, Role: role, Value: declaredValue})
		case "description":
			_, _ = p.next()
			descTok, err := p.expect(TokenString)
			if err != nil {
				return d, err
			}
			d.Description = descTok.Text
		case "display":
			_, _ = p.next()
			dispTok, err := p.expect(TokenString)
			if err != nil {
				return d, err
			}
			d.Display = dispTok.Text
		case "script":
			_, _ = p.next()
			scriptTok, err := p.expect(TokenString)
			if err != nil {
				return d, err
			}
			d.Script = scriptTok.Text
		case "relation":
			_, _ = p.next()
			rSym, err := p.expect(TokenIdent)
			if err != nil {
				return d, err
			}
			if _, err := p.expectText("to"); err != nil {
				return d, err
			}
			targetTok, err := p.expect(TokenIdent)
			if err != nil {
				return d, err
			}
			target, err := core.ParseCanonicalID(targetTok.Text)
			if err != nil {
				return d, p.errAt(targetTok, err.Error())
			}
			cardTok, err := p.expect(TokenIdent)
			if err != nil {
				return d, err
			}
			rseg, err := core.SymbolToSegment(rSym.Text)
			if err != nil {
				return d, p.errAt(rSym, err.Error())
			}
			rid, _ := core.ParseCanonicalID(string(id) + "." + rseg)
			onDelete := ""
			if nt, _ := p.peek(); nt.Text == "on-delete" {
				_, _ = p.next()
				policyTok, err := p.expect(TokenIdent)
				if err != nil {
					return d, err
				}
				onDelete = policyTok.Text
			}
			keys, err := p.parseSemanticRelationKeys()
			if err != nil {
				return d, err
			}
			d.Relations = append(d.Relations, semantic.Relation{ID: rid, Symbol: rSym.Text, Target: target, Cardinality: semantic.Cardinality(cardTok.Text), Keys: keys, OnDelete: onDelete})
		case "candidate-key":
			key, err := p.parseSemanticCandidateKey(id)
			if err != nil {
				return d, err
			}
			d.CandidateKeys = append(d.CandidateKeys, key)
		case "functional-dependency":
			fd, err := p.parseSemanticFunctionalDependency(id)
			if err != nil {
				return d, err
			}
			d.FunctionalDependencies = append(d.FunctionalDependencies, fd)
		default:
			return d, p.errAt(t, "expected attribute, relation, candidate-key, or functional-dependency")
		}
	}
	return d, nil
}

func (p *Parser) parseSemanticCapability(namespace core.CanonicalID) (semantic.Capability, error) {
	_, _ = p.next() // capability
	symTok, err := p.expect(TokenIdent)
	if err != nil {
		return semantic.Capability{}, err
	}
	var id core.CanonicalID
	symbol := symTok.Text
	if strings.Contains(symTok.Text, ".") {
		id, err = core.ParseCanonicalID(symTok.Text)
		if err != nil {
			return semantic.Capability{}, p.errAt(symTok, err.Error())
		}
		if !strings.HasPrefix(string(id), string(namespace)+".") {
			return semantic.Capability{}, p.errAt(symTok, "capability canonical id must belong to the declared semantic namespace")
		}
		parts := strings.Split(string(id), ".")
		symbol = parts[len(parts)-1]
	} else {
		seg, err := core.SymbolToSegment(symTok.Text)
		if err != nil {
			return semantic.Capability{}, p.errAt(symTok, err.Error())
		}
		id, _ = core.ParseCanonicalID(string(namespace) + "." + seg)
	}
	c := semantic.Capability{ID: id, Symbol: symbol}
	if _, err := p.expect(TokenLBrace); err != nil {
		return c, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return c, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "effect":
			if c.Effect != nil {
				return c, p.errAt(t, "capability data effect is declared more than once")
			}
			_, _ = p.next()
			kindTok, err := p.expect(TokenIdent)
			if err != nil {
				return c, err
			}
			kind := semantic.DataEffectKind(kindTok.Text)
			if !kind.Valid() {
				return c, p.errAt(kindTok, fmt.Sprintf("unsupported data effect kind %q", kindTok.Text))
			}
			dataTok, err := p.expect(TokenIdent)
			if err != nil {
				return c, err
			}
			data, err := core.ParseCanonicalID(dataTok.Text)
			if err != nil {
				return c, p.errAt(dataTok, err.Error())
			}
			c.Effect = &semantic.DataEffect{Kind: kind, Data: data}
		case "input":
			_, _ = p.next()
			nameTok, err := p.expect(TokenIdent)
			if err != nil {
				return c, err
			}
			name, err := core.ParseLocalName(nameTok.Text)
			if err != nil {
				return c, p.errAt(nameTok, err.Error())
			}
			typeTok, err := p.expect(TokenIdent)
			if err != nil {
				return c, err
			}
			typeRef, err := core.ParseCanonicalID(typeTok.Text)
			if err != nil {
				return c, p.errAt(typeTok, err.Error())
			}
			c.Inputs = append(c.Inputs, semantic.Parameter{Name: name, Type: typeRef})
		case "output":
			_, _ = p.next()
			typeTok, err := p.expect(TokenIdent)
			if err != nil {
				return c, err
			}
			c.Output, err = core.ParseCanonicalID(typeTok.Text)
			if err != nil {
				return c, p.errAt(typeTok, err.Error())
			}
		case "outcome":
			_, _ = p.next()
			nameTok, err := p.expect(TokenIdent)
			if err != nil {
				return c, err
			}
			name, err := core.ParseLocalName(nameTok.Text)
			if err != nil {
				return c, p.errAt(nameTok, err.Error())
			}
			payloadTok, err := p.expect(TokenIdent)
			if err != nil {
				return c, err
			}
			payload, err := core.ParseCanonicalID(payloadTok.Text)
			if err != nil {
				return c, p.errAt(payloadTok, err.Error())
			}
			c.Outcomes = append(c.Outcomes, semantic.Outcome{Name: name, Payload: payload})
		case "emptyPolicy":
			if c.EmptyPolicy != "" {
				return c, p.errAt(t, "capability emptyPolicy is declared more than once")
			}
			_, _ = p.next()
			valueTok, err := p.expect(TokenIdent)
			if err != nil {
				return c, err
			}
			policy := semantic.EmptyPolicy(valueTok.Text)
			if !policy.Valid() {
				return c, p.errAt(valueTok, fmt.Sprintf("unsupported emptyPolicy %q (allowed: allow, requires-evidence, deny)", valueTok.Text))
			}
			c.EmptyPolicy = policy
		default:
			return c, p.errAt(t, "expected input, output, outcome, effect, or emptyPolicy")
		}
	}
	return c, nil
}

func (p *Parser) parseSemanticValidation(namespace core.CanonicalID) (semantic.Validation, error) {
	_, _ = p.next() // validation
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return semantic.Validation{}, err
	}
	var id core.CanonicalID
	symbol := idTok.Text
	if strings.Contains(idTok.Text, ".") {
		id, err = core.ParseCanonicalID(idTok.Text)
		if err != nil {
			return semantic.Validation{}, p.errAt(idTok, err.Error())
		}
		if !strings.HasPrefix(string(id), string(namespace)+".") {
			return semantic.Validation{}, p.errAt(idTok, "validation canonical id must belong to the declared semantic namespace")
		}
		parts := strings.Split(string(id), ".")
		symbol = parts[len(parts)-1]
	} else {
		seg, err := core.SymbolToSegment(idTok.Text)
		if err != nil {
			return semantic.Validation{}, p.errAt(idTok, err.Error())
		}
		id, _ = core.ParseCanonicalID(string(namespace) + "." + seg)
	}

	kindTok, err := p.expect(TokenIdent)
	if err != nil {
		return semantic.Validation{}, err
	}
	v := semantic.Validation{ID: id, Symbol: symbol, Kind: semantic.ValidationKind(kindTok.Text)}
	if !v.Kind.Valid() {
		return v, p.errAt(kindTok, fmt.Sprintf("unsupported semantic validation kind %q", kindTok.Text))
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return v, err
	}

	var hasAttribute, hasRelation bool
	for {
		t, err := p.peek()
		if err != nil {
			return v, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "attribute":
			if hasAttribute {
				return v, p.errAt(t, "attribute is declared more than once")
			}
			hasAttribute = true
			_, _ = p.next()
			a, err := p.expect(TokenIdent)
			if err != nil {
				return v, err
			}
			v.Attribute, err = core.ParseCanonicalID(a.Text)
			if err != nil {
				return v, p.errAt(a, err.Error())
			}
		case "relation":
			if hasRelation {
				return v, p.errAt(t, "relation is declared more than once")
			}
			hasRelation = true
			_, _ = p.next()
			r, err := p.expect(TokenIdent)
			if err != nil {
				return v, err
			}
			v.Relation, err = core.ParseCanonicalID(r.Text)
			if err != nil {
				return v, p.errAt(r, err.Error())
			}
		default:
			return v, p.errAt(t, "expected attribute or relation")
		}
	}

	if v.Attribute == "" {
		return v, fmt.Errorf("semantic validation %s requires attribute", v.ID)
	}
	switch v.Kind {
	case semantic.ValidationPositive:
		if v.Relation != "" {
			return v, fmt.Errorf("positive semantic validation %s does not accept relation", v.ID)
		}
	case semantic.ValidationRelationUnique:
		if v.Relation == "" {
			return v, fmt.Errorf("relation-unique semantic validation %s requires relation", v.ID)
		}
	}
	return v, nil
}

func parseVersion(s string) (database.ContractVersion, error) {
	if !strings.HasPrefix(s, "v") {
		return database.ContractVersion{}, fmt.Errorf("version must start with v")
	}
	parts := strings.Split(strings.TrimPrefix(s, "v"), ".")
	if len(parts) < 1 || len(parts) > 2 {
		return database.ContractVersion{}, fmt.Errorf("invalid version %q", s)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return database.ContractVersion{}, fmt.Errorf("invalid version %q", s)
	}
	minor := 0
	if len(parts) == 2 {
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return database.ContractVersion{}, fmt.Errorf("invalid version %q", s)
		}
	}
	return database.ContractVersion{Major: major, Minor: minor}, nil
}

func (p *Parser) next() (Token, error) {
	if p.peeked != nil {
		t := *p.peeked
		p.peeked = nil
		return t, nil
	}
	return p.lx.Next()
}
func (p *Parser) peek() (Token, error) {
	if p.peeked != nil {
		return *p.peeked, nil
	}
	t, err := p.lx.Next()
	if err != nil {
		return Token{}, err
	}
	p.peeked = &t
	return t, nil
}
func (p *Parser) expect(k TokenKind) (Token, error) {
	t, err := p.next()
	if err != nil {
		return Token{}, err
	}
	if t.Kind != k {
		return Token{}, p.errAt(t, fmt.Sprintf("unexpected token %q", t.Text))
	}
	return t, nil
}
func (p *Parser) expectText(s string) (Token, error) {
	t, err := p.next()
	if err != nil {
		return Token{}, err
	}
	if t.Kind != TokenIdent || t.Text != s {
		return Token{}, p.errAt(t, fmt.Sprintf("expected %q", s))
	}
	return t, nil
}

// topLevelDeclaration names the declaration a keyword starts, so a block written
// in the wrong file is told what it is and where it belongs instead of being
// reported as an unknown token (#4169). Every keyword here is an entry point the
// parser itself reads (ParseApplication, ParseMigration, ...), so the table
// cannot promise a syntax this package does not accept.
type topLevelDeclaration struct {
	keyword     string
	description string
}

var topLevelDeclarations = []topLevelDeclaration{
	{"application", "an application"},
	{"pattern", "an application pattern"},
	{"application-release", "an application release"},
	{"environment", "an environment"},
	{"database", "a database"},
	{"database-release", "a database release"},
	{"migration", "a migration"},
	{"migration-provider", "a migration provider"},
	{"solution-release", "a solution release"},
	{"plugin", "a plugin"},
	{"bundle", "a bundle"},
	{"persist", "a persistence mapping"},
	{"realize", "a realization"},
	{"query", "a query"},
	{"contract", "a contract"},
	{"namespace", "a semantic namespace"},
	{"type", "a semantic type"},
	{"data", "a semantic data declaration"},
	{"capability", "a semantic capability"},
	{"validation", "a semantic validation"},
	{"calendar", "a semantic calendar"},
	{"period", "a semantic period"},
	{"effect-rule", "a semantic effect rule"},
	{"run-plan", "a semantic run plan"},
}

// declarationDescription returns a human phrase for the declaration the token
// starts, or "" when the token is not a top-level keyword.
func declarationDescription(text string) string {
	for _, declaration := range topLevelDeclarations {
		if text == declaration.keyword {
			return declaration.description
		}
	}
	return ""
}

// misplacedDeclaration explains that a top-level declaration keyword cannot be
// used here, and names the declaration. It returns nil when the token is not a
// top-level keyword, so the caller keeps its own narrower message.
func (p *Parser) misplacedDeclaration(t Token, scope string) error {
	description := declarationDescription(t.Text)
	if description == "" {
		return nil
	}
	return p.errAt(t, fmt.Sprintf(
		"%q starts %s, which cannot be declared %s; declare it in its own file",
		t.Text, description, scope))
}

func (p *Parser) expectEOF() error {
	t, err := p.next()
	if err != nil {
		return err
	}
	if t.Kind != TokenEOF {
		// A stray top-level keyword usually means a declaration went into the
		// wrong file. Naming it is the difference between "this token is wrong"
		// and "migrations live in their own file". An unknown token keeps the
		// old message (#4169).
		if misplaced := p.misplacedDeclaration(t, "after the end of this declaration"); misplaced != nil {
			return misplaced
		}
		return p.errAt(t, fmt.Sprintf("unexpected trailing token %q", t.Text))
	}
	return nil
}
func (p *Parser) errAt(t Token, msg string) error {
	return &ParseError{Pos: t.Start, End: t.End, Msg: msg}
}

func (p *Parser) parseSemanticPeriod(namespace core.CanonicalID) (semantic.Period, error) {
	_, _ = p.next() // period
	symTok, err := p.expect(TokenIdent)
	if err != nil {
		return semantic.Period{}, err
	}
	var id core.CanonicalID
	if strings.Contains(symTok.Text, ".") {
		id, err = core.ParseCanonicalID(symTok.Text)
		if err != nil {
			return semantic.Period{}, p.errAt(symTok, err.Error())
		}
		if !strings.HasPrefix(string(id), string(namespace)+".") {
			return semantic.Period{}, p.errAt(symTok, "period canonical id must belong to the declared semantic namespace")
		}
	} else {
		seg, err := core.SymbolToSegment(symTok.Text)
		if err != nil {
			return semantic.Period{}, p.errAt(symTok, err.Error())
		}
		id, _ = core.ParseCanonicalID(string(namespace) + "." + seg)
	}
	period := semantic.Period{ID: id}
	if _, err := p.expect(TokenLBrace); err != nil {
		return period, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return period, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "data":
			_, _ = p.next()
			value, err := p.expectCanonicalReference()
			if err != nil {
				return period, err
			}
			period.Data = value
		case "business-date":
			_, _ = p.next()
			value, err := p.expectCanonicalReference()
			if err != nil {
				return period, err
			}
			period.BusinessDate = value
		case "states":
			_, _ = p.next()
			values, err := p.readPeriodNames()
			if err != nil {
				return period, err
			}
			period.States = values
		case "closed-rejects":
			_, _ = p.next()
			values, err := p.readPeriodNames()
			if err != nil {
				return period, err
			}
			period.ClosedRejects = values
		case "calendar":
			_, _ = p.next()
			value, err := p.expectCanonicalReference()
			if err != nil {
				return period, err
			}
			period.Calendar = value
		case "close-requires":
			_, _ = p.next()
			value, err := p.expectCanonicalReference()
			if err != nil {
				return period, err
			}
			period.CloseRequires = value
		case "correction":
			_, _ = p.next()
			if _, err := p.expectText("via"); err != nil {
				return period, err
			}
			value, err := p.expectCanonicalReference()
			if err != nil {
				return period, err
			}
			period.Correction = value
		case "closes":
			_, _ = p.next()
			if _, err := p.expectText("on"); err != nil {
				return period, err
			}
			if _, err := p.expectText("day"); err != nil {
				return period, err
			}
			dayTok, err := p.expect(TokenIdent)
			if err != nil {
				return period, err
			}
			day, convErr := strconv.Atoi(dayTok.Text)
			if convErr != nil {
				return period, p.errAt(dayTok, "closing day is not a whole number")
			}
			period.ClosingDay = day
		default:
			return period, p.errAt(t, "expected data, business-date, calendar, states, close-requires, closed-rejects, correction, or closes")
		}
	}
	return period, nil
}

// expectCanonicalReference reads one canonical identifier.
func (p *Parser) expectCanonicalReference() (core.CanonicalID, error) {
	tok, err := p.expect(TokenIdent)
	if err != nil {
		return "", err
	}
	value, err := core.ParseCanonicalID(tok.Text)
	if err != nil {
		return "", p.errAt(tok, err.Error())
	}
	return value, nil
}

// readPeriodNames reads a space-separated list of names and stops at the next
// period block keyword or at the closing brace.
func (p *Parser) readPeriodNames() ([]string, error) {
	stop := map[string]bool{
		"data": true, "business-date": true, "states": true,
		"close-requires": true, "closed-rejects": true, "correction": true,
		// closes starts the closing-day declaration, so a states list must end
		// before it; otherwise the day number is read as another state name.
		"closes": true,
	}
	var values []string
	for {
		t, err := p.peek()
		if err != nil {
			return nil, err
		}
		if t.Kind == TokenRBrace || t.Kind == TokenEOF {
			return values, nil
		}
		if t.Kind != TokenIdent || stop[t.Text] {
			return values, nil
		}
		_, _ = p.next()
		values = append(values, t.Text)
	}
}

func (p *Parser) parseSemanticEffectRule(namespace core.CanonicalID) (semantic.EffectRule, error) {
	_, _ = p.next() // effect-rule
	symTok, err := p.expect(TokenIdent)
	if err != nil {
		return semantic.EffectRule{}, err
	}
	var id core.CanonicalID
	if strings.Contains(symTok.Text, ".") {
		id, err = core.ParseCanonicalID(symTok.Text)
		if err != nil {
			return semantic.EffectRule{}, p.errAt(symTok, err.Error())
		}
		if !strings.HasPrefix(string(id), string(namespace)+".") {
			id, err = core.ParseCanonicalID(string(namespace) + "." + symTok.Text)
			if err != nil {
				return semantic.EffectRule{}, p.errAt(symTok, err.Error())
			}
		}
	} else {
		seg, err := core.SymbolToSegment(symTok.Text)
		if err != nil {
			return semantic.EffectRule{}, p.errAt(symTok, err.Error())
		}
		id, _ = core.ParseCanonicalID(string(namespace) + "." + seg)
	}
	rule := semantic.EffectRule{ID: id}
	if _, err := p.expect(TokenLBrace); err != nil {
		return rule, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return rule, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "on":
			_, _ = p.next()
			if _, err := p.expectText("capability"); err != nil {
				return rule, err
			}
			value, err := p.expectCanonicalReference()
			if err != nil {
				return rule, err
			}
			rule.OnCapability = value
		case "stage":
			_, _ = p.next()
			stageTok, err := p.expect(TokenIdent)
			if err != nil {
				return rule, err
			}
			rule.Stage = stageTok.Text
		case "effect":
			_, _ = p.next()
			kindTok, err := p.expect(TokenIdent)
			if err != nil {
				return rule, err
			}
			kind := semantic.DataEffectKind(kindTok.Text)
			if !kind.Valid() {
				return rule, p.errAt(kindTok, fmt.Sprintf("unsupported data effect kind %q", kindTok.Text))
			}
			value, err := p.expectCanonicalReference()
			if err != nil {
				return rule, err
			}
			rule.EffectKind = kind
			rule.EffectData = value
		case "via":
			_, _ = p.next()
			if _, err := p.expectText("capability"); err != nil {
				return rule, err
			}
			value, err := p.expectCanonicalReference()
			if err != nil {
				return rule, err
			}
			rule.ViaCapability = value
		case "on-failure":
			_, _ = p.next()
			valueTok, err := p.expect(TokenIdent)
			if err != nil {
				return rule, err
			}
			rule.OnFailure = valueTok.Text
		default:
			return rule, p.errAt(t, "expected on capability, stage, effect, via capability, or on-failure")
		}
	}
	return rule, nil
}

func (p *Parser) parseSemanticRunPlan(namespace core.CanonicalID) (semantic.RunPlan, error) {
	_, _ = p.next() // run-plan
	symTok, err := p.expect(TokenIdent)
	if err != nil {
		return semantic.RunPlan{}, err
	}
	id, err := core.ParseCanonicalID(symTok.Text)
	if err != nil {
		return semantic.RunPlan{}, p.errAt(symTok, err.Error())
	}
	if !strings.HasPrefix(string(id), string(namespace)+".") {
		id, err = core.ParseCanonicalID(string(namespace) + "." + symTok.Text)
		if err != nil {
			return semantic.RunPlan{}, p.errAt(symTok, err.Error())
		}
	}
	plan := semantic.RunPlan{ID: id}
	if _, err := p.expect(TokenLBrace); err != nil {
		return plan, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return plan, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "operation":
			_, _ = p.next()
			value, err := p.expectCanonicalReference()
			if err != nil {
				return plan, err
			}
			plan.Operations = append(plan.Operations, value)
		case "key":
			_, _ = p.next()
			key, err := p.expectCanonicalReference()
			if err != nil {
				return plan, err
			}
			plan.Key = append(plan.Key, key)
		case "concurrency":
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return plan, err
			}
			plan.Concurrency = value.Text
		case "resume":
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return plan, err
			}
			plan.Resume = value.Text
		case "output":
			_, _ = p.next()
			value, err := p.expectCanonicalReference()
			if err != nil {
				return plan, err
			}
			plan.Outputs = append(plan.Outputs, value)
		case "autonomy":
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return plan, err
			}
			plan.Autonomy = value.Text
		default:
			return plan, p.errAt(t, "expected operation, key, concurrency, resume, output, or autonomy")
		}
	}
	return plan, nil
}

// ParseRelationQueryは、Relation Queryブロックを読み取るための簡易wrapperである。
func (p *Parser) ParseRelationQuery(owner core.CanonicalID) (query.RelationQuery, error) {
	authoring, err := p.ParseRelationQueryAuthoring(owner)
	if err != nil {
		return query.RelationQuery{}, err
	}
	return authoring.Value, nil
}
