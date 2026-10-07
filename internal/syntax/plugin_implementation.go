package syntax

import (
	"encoding/json"
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/plugin/implementation"
)

// ParsePluginImplementation parses the first constrained TEI-native Plugin
// implementation profile. It is intentionally not a general-purpose language.
func (p *Parser) ParsePluginImplementation() (implementation.Source, error) {
	if _, err := p.expectText("plugin-implementation"); err != nil {
		return implementation.Source{}, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return implementation.Source{}, err
	}
	id, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return implementation.Source{}, p.errAt(idTok, err.Error())
	}
	versionTok, err := p.expect(TokenIdent)
	if err != nil {
		return implementation.Source{}, err
	}
	source := implementation.Source{
		Plugin: plugin.Identity{ID: id, Version: versionTok.Text},
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return source, err
	}

	var profileSeen bool
	var capabilitySeen bool
	for {
		t, err := p.peek()
		if err != nil {
			return source, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "profile":
			if profileSeen {
				return source, p.errAt(t, "profile is declared more than once")
			}
			profileSeen = true
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return source, err
			}
			source.Profile = value.Text
		case "capability":
			if capabilitySeen {
				return source, p.errAt(t, "capability is declared more than once")
			}
			capabilitySeen = true
			_, _ = p.next()
			kindTok, err := p.expect(TokenIdent)
			if err != nil {
				return source, err
			}
			idTok, err := p.expect(TokenIdent)
			if err != nil {
				return source, err
			}
			capabilityID, err := core.ParseCanonicalID(idTok.Text)
			if err != nil {
				return source, p.errAt(idTok, err.Error())
			}
			source.CapabilityKind = plugin.CapabilityKind(kindTok.Text)
			source.CapabilityID = capabilityID
		case "operation":
			_, _ = p.next()
			nameTok, err := p.expect(TokenIdent)
			if err != nil {
				return source, err
			}
			if _, err := p.expectText("json"); err != nil {
				return source, err
			}
			if _, err := p.expect(TokenLBrace); err != nil {
				return source, err
			}
			raw, _, err := p.lx.ReadRawBlock()
			if err != nil {
				return source, err
			}
			p.peeked = nil
			payload := json.RawMessage(strings.TrimSpace(raw))
			source.Operations = append(source.Operations, implementation.Operation{
				Name:    nameTok.Text,
				Payload: payload,
			})
		default:
			return source, p.errAt(t, fmt.Sprintf("expected profile, capability, or operation, got %q", t.Text))
		}
	}
	if err := p.expectEOF(); err != nil {
		return source, err
	}
	if err := source.Validate(); err != nil {
		return source, fmt.Errorf("plugin implementation: %w", err)
	}
	return source, nil
}
