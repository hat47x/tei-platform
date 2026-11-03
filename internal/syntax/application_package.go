package syntax

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
)

// ParseApplicationPackageは、Patternを束ねるPackage宣言を読み取る。
// PackageはPatternを複製せず、identityで参照する。
func (p *Parser) ParseApplicationPackage() (application.Package, error) {
	var out application.Package
	if _, err := p.expectText("package"); err != nil {
		return out, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return out, err
	}
	id, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return out, p.errAt(idTok, err.Error())
	}
	if _, err := p.expect(TokenCaret); err != nil {
		return out, err
	}
	versionTok, err := p.expect(TokenIdent)
	if err != nil {
		return out, err
	}
	out = application.Package{ID: id, Version: versionTok.Text}
	if _, err := p.expect(TokenLBrace); err != nil {
		return out, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return out, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "application":
			_, _ = p.next()
			appTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			if _, err := core.ParseCanonicalID(appTok.Text); err != nil {
				return out, p.errAt(appTok, err.Error())
			}
		case "member":
			_, _ = p.next()
			kindTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			patternTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			pattern, err := core.ParseCanonicalID(patternTok.Text)
			if err != nil {
				return out, p.errAt(patternTok, err.Error())
			}
			out.Members = append(out.Members, application.PackageMember{
				Kind:    application.PatternKind(kindTok.Text),
				Pattern: pattern,
			})
		case "requires":
			_, _ = p.next()
			kindTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			if kindTok.Text != "semantic" && kindTok.Text != "relation" {
				return out, p.errAt(kindTok, "package requires semantic or relation")
			}
			refTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			ref, err := core.ParseCanonicalID(refTok.Text)
			if err != nil {
				return out, p.errAt(refTok, err.Error())
			}
			out.Requires = append(out.Requires, application.PackageRequirement{Kind: kindTok.Text, ID: ref})
		case "projection":
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Projection = value.Text
		case "conformance":
			_, _ = p.next()
			value, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Conformance = value.Text
		default:
			return out, p.errAt(t, "expected application, member, requires, projection, or conformance")
		}
	}
	if len(out.Members) == 0 {
		return out, fmt.Errorf("package %s requires at least one member", out.ID)
	}
	return out, nil
}
