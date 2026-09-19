package syntax

import (
	"tei/internal/core"
	"tei/internal/database/persistence"
)

// RelationStorageAuthoringは、Database-ownedなRelation Storage定義を
// 既存Persistence Mappingへ合流させる前のauthoring結果を表す。
// Mappingは合流先Persistence MappingのCanonical IDであり、Storageは
// Semantic Relationと物理support columnの対応だけを保持する。
type RelationStorageAuthoring struct {
	Mapping core.CanonicalID
	Storage persistence.RelationStorageMapping
}

// ParseRelationStorageは、Semantic Relationを物理的に保存するsupport columnを、
// target Semantic Dataの通常属性へ偽装せずにauthoringする。
//
//	relation-storage <persistence-mapping-id> {
//	    relation <semantic-relation-id>
//	    component <source-attribute-id> -> <column> storage <storage-type> <nullability>
//	}
//
// componentの順序は複合source identityを保持するため、そのままCanonical modelへ渡す。
func (p *Parser) ParseRelationStorage() (RelationStorageAuthoring, error) {
	if _, err := p.expectText("relation-storage"); err != nil {
		return RelationStorageAuthoring{}, err
	}

	mappingTok, err := p.expect(TokenIdent)
	if err != nil {
		return RelationStorageAuthoring{}, err
	}
	mappingID, err := core.ParseCanonicalID(mappingTok.Text)
	if err != nil {
		return RelationStorageAuthoring{}, p.errAt(mappingTok, err.Error())
	}
	out := RelationStorageAuthoring{Mapping: mappingID}

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
		case "relation":
			if out.Storage.Relation != "" {
				return out, p.errAt(t, "relation may be declared only once")
			}
			_, _ = p.next()
			relationTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Storage.Relation, err = core.ParseCanonicalID(relationTok.Text)
			if err != nil {
				return out, p.errAt(relationTok, err.Error())
			}

		case "component":
			_, _ = p.next()
			sourceTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			sourceAttribute, err := core.ParseCanonicalID(sourceTok.Text)
			if err != nil {
				return out, p.errAt(sourceTok, err.Error())
			}
			if _, err := p.expect(TokenArrow); err != nil {
				return out, err
			}
			columnTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			if _, err := p.expectText("storage"); err != nil {
				return out, err
			}
			storageTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			nullabilityTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Storage.Components = append(out.Storage.Components, persistence.RelationStorageComponentMapping{
				SourceAttribute: sourceAttribute,
				Column:          columnTok.Text,
				Storage:         persistence.StorageType(storageTok.Text),
				Nullability:     persistence.Nullability(nullabilityTok.Text),
			})

		default:
			return out, p.errAt(t, "expected relation or component")
		}
	}

	if out.Storage.Relation == "" {
		return out, p.errAt(mappingTok, "relation-storage requires relation")
	}
	if len(out.Storage.Components) == 0 {
		return out, p.errAt(mappingTok, "relation-storage requires at least one component")
	}
	if err := p.expectEOF(); err != nil {
		return out, err
	}
	return out, nil
}
