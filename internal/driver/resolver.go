package driver

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/semantic"
)

// Resolverは、Canonical WorldからApplication上の意味を解決するDriverの最小入口である。
//
// この段階では実行方法を決めない。Patternが参照するData、Relation、Query、Capabilityと、
// Canonicalに明示されたlogical datasourceだけを解決し、Driver固有の推測を持ち込まない。
type Resolver struct {
	world ir.World
}

func NewResolver(world ir.World) Resolver {
	return Resolver{world: world}
}

// ResolvedRelationは、Relation自身に加えて、そのRelationを所有するsource Dataとtarget Dataを表す。
// RelationのsourceをID文字列の規則から推測せず、Semantic定義上の所属関係から解決する。
type ResolvedRelation struct {
	Source   semantic.Data
	Relation semantic.Relation
	Target   semantic.Data
}

// PatternResolutionは、一つのApplication PatternからCanonicalに解決できた意味を表す。
// nilのfieldは「未解決」ではなく、そのPattern Kindがこの段階ではその情報を要求しないことを示す。
type PatternResolution struct {
	Application application.Application
	Pattern     application.Pattern
	Data        *semantic.Data
	Relation    *ResolvedRelation
	Query       *query.EmbeddedSQL
	Datasource  *application.DatabaseRequirement
	Capability  *semantic.Capability
}

// ResolvePatternは、ApplicationとPatternのCanonical IDを起点として、
// Pattern Kindごとに必要な既存資産だけを解決する。
//
// recordやheader-detailについて、Persistence MappingやDatasourceを推測で補うことはしない。
// browseのDatasourceはQueryを正本とし、referenceのDatasourceはPatternに明示されたaliasを利用する。
func (r Resolver) ResolvePattern(applicationID, patternID core.CanonicalID) (PatternResolution, error) {
	app, err := r.applicationByID(applicationID)
	if err != nil {
		return PatternResolution{}, err
	}
	pattern, err := patternByID(app, patternID)
	if err != nil {
		return PatternResolution{}, err
	}

	resolved := PatternResolution{Application: app, Pattern: pattern}

	switch pattern.Kind {
	case application.PatternBrowse:
		data, err := r.dataByID(pattern.Data)
		if err != nil {
			return PatternResolution{}, fmt.Errorf("browse Pattern %sのDataを解決できません: %w", pattern.ID, err)
		}
		q, err := r.queryByID(pattern.Query)
		if err != nil {
			return PatternResolution{}, fmt.Errorf("browse Pattern %sのQueryを解決できません: %w", pattern.ID, err)
		}
		if q.Application != app.ID {
			return PatternResolution{}, fmt.Errorf("browse Pattern %sはApplication %sに属しますが、Query %sはApplication %sに属しています", pattern.ID, app.ID, q.ID, q.Application)
		}
		if q.Datasource == "" {
			return PatternResolution{}, fmt.Errorf("browse Pattern %sが参照するQuery %sにDatasourceがありません", pattern.ID, q.ID)
		}
		if pattern.Datasource != "" && pattern.Datasource != q.Datasource {
			return PatternResolution{}, fmt.Errorf("browse Pattern %sのDatasource %sとQuery %sのDatasource %sが一致しません", pattern.ID, pattern.Datasource, q.ID, q.Datasource)
		}
		datasource, err := readableDatasource(app, q.Datasource)
		if err != nil {
			return PatternResolution{}, fmt.Errorf("browse Pattern %sのDatasourceを解決できません: %w", pattern.ID, err)
		}
		resolved.Data = &data
		resolved.Query = &q
		resolved.Datasource = &datasource

	case application.PatternRecord:
		data, err := r.dataByID(pattern.Data)
		if err != nil {
			return PatternResolution{}, fmt.Errorf("record Pattern %sのDataを解決できません: %w", pattern.ID, err)
		}
		resolved.Data = &data

	case application.PatternReference:
		relation, err := r.relationByID(pattern.Relation)
		if err != nil {
			return PatternResolution{}, fmt.Errorf("reference Pattern %sのRelationを解決できません: %w", pattern.ID, err)
		}
		if pattern.Datasource == "" {
			return PatternResolution{}, fmt.Errorf("reference Pattern %sにDatasourceがありません", pattern.ID)
		}
		datasource, err := readableDatasource(app, pattern.Datasource)
		if err != nil {
			return PatternResolution{}, fmt.Errorf("reference Pattern %sのDatasourceを解決できません: %w", pattern.ID, err)
		}
		resolved.Relation = &relation
		resolved.Datasource = &datasource

	case application.PatternCommand:
		capability, err := r.capabilityByID(pattern.Capability)
		if err != nil {
			return PatternResolution{}, fmt.Errorf("command Pattern %sのCapabilityを解決できません: %w", pattern.ID, err)
		}
		resolved.Capability = &capability

	case application.PatternHeaderDetail:
		relation, err := r.relationByID(pattern.Relation)
		if err != nil {
			return PatternResolution{}, fmt.Errorf("header-detail Pattern %sのRelationを解決できません: %w", pattern.ID, err)
		}
		if relation.Relation.Cardinality != semantic.CardinalityMany {
			return PatternResolution{}, fmt.Errorf("header-detail Pattern %sのRelation %sはmanyではありません: %s", pattern.ID, relation.Relation.ID, relation.Relation.Cardinality)
		}
		resolved.Relation = &relation

	default:
		return PatternResolution{}, fmt.Errorf("Pattern %sのKind %qはDriverで解決できません", pattern.ID, pattern.Kind)
	}

	return resolved, nil
}

func (r Resolver) applicationByID(id core.CanonicalID) (application.Application, error) {
	var found application.Application
	count := 0
	for _, app := range r.world.Applications {
		if app.ID != id {
			continue
		}
		found = app
		count++
	}
	if count == 0 {
		return application.Application{}, fmt.Errorf("Application %sが見つかりません", id)
	}
	if count > 1 {
		return application.Application{}, fmt.Errorf("Application %sが複数定義されています: %d件", id, count)
	}
	return found, nil
}

func patternByID(app application.Application, id core.CanonicalID) (application.Pattern, error) {
	var found application.Pattern
	count := 0
	for _, pattern := range app.Patterns {
		if pattern.ID != id {
			continue
		}
		found = pattern
		count++
	}
	if count == 0 {
		return application.Pattern{}, fmt.Errorf("Pattern %sはApplication %sに定義されていません", id, app.ID)
	}
	if count > 1 {
		return application.Pattern{}, fmt.Errorf("Pattern %sがApplication %sに複数定義されています: %d件", id, app.ID, count)
	}
	return found, nil
}

func (r Resolver) queryByID(id core.CanonicalID) (query.EmbeddedSQL, error) {
	var found query.EmbeddedSQL
	count := 0
	for _, q := range r.world.Queries {
		if q.ID != id {
			continue
		}
		found = q
		count++
	}
	if count == 0 {
		return query.EmbeddedSQL{}, fmt.Errorf("Query %sが見つかりません", id)
	}
	if count > 1 {
		return query.EmbeddedSQL{}, fmt.Errorf("Query %sが複数定義されています: %d件", id, count)
	}
	return found, nil
}

func (r Resolver) dataByID(id core.CanonicalID) (semantic.Data, error) {
	var found semantic.Data
	count := 0
	for _, namespace := range r.world.Semantics {
		for _, data := range namespace.Data {
			if data.ID != id {
				continue
			}
			found = data
			count++
		}
	}
	if count == 0 {
		return semantic.Data{}, fmt.Errorf("Semantic Data %sが見つかりません", id)
	}
	if count > 1 {
		return semantic.Data{}, fmt.Errorf("Semantic Data %sが複数定義されています: %d件", id, count)
	}
	return found, nil
}

func (r Resolver) relationByID(id core.CanonicalID) (ResolvedRelation, error) {
	var source semantic.Data
	var relation semantic.Relation
	count := 0
	for _, namespace := range r.world.Semantics {
		for _, data := range namespace.Data {
			for _, candidate := range data.Relations {
				if candidate.ID != id {
					continue
				}
				source = data
				relation = candidate
				count++
			}
		}
	}
	if count == 0 {
		return ResolvedRelation{}, fmt.Errorf("Semantic Relation %sが見つかりません", id)
	}
	if count > 1 {
		return ResolvedRelation{}, fmt.Errorf("Semantic Relation %sが複数定義されています: %d件", id, count)
	}
	target, err := r.dataByID(relation.Target)
	if err != nil {
		return ResolvedRelation{}, fmt.Errorf("Relation %sのtarget %sを解決できません: %w", relation.ID, relation.Target, err)
	}
	return ResolvedRelation{Source: source, Relation: relation, Target: target}, nil
}

func (r Resolver) capabilityByID(id core.CanonicalID) (semantic.Capability, error) {
	var found semantic.Capability
	count := 0
	for _, namespace := range r.world.Semantics {
		for _, capability := range namespace.Capabilities {
			if capability.ID != id {
				continue
			}
			found = capability
			count++
		}
	}
	if count == 0 {
		return semantic.Capability{}, fmt.Errorf("Semantic Capability %sが見つかりません", id)
	}
	if count > 1 {
		return semantic.Capability{}, fmt.Errorf("Semantic Capability %sが複数定義されています: %d件", id, count)
	}
	return found, nil
}

func readableDatasource(app application.Application, alias core.LocalName) (application.DatabaseRequirement, error) {
	if !app.HasDatabaseAlias(alias) {
		return application.DatabaseRequirement{}, fmt.Errorf("Datasource %sはApplication %sに定義されていません", alias, app.ID)
	}
	datasource, ok := app.DatabaseByAliasExact(alias)
	if !ok {
		return application.DatabaseRequirement{}, fmt.Errorf("Datasource %sがApplication %sに複数定義されています", alias, app.ID)
	}
	if datasource.Access != database.AccessRead && datasource.Access != database.AccessReadWrite {
		return application.DatabaseRequirement{}, fmt.Errorf("Datasource %sのaccess %sでは読み取りできません", alias, datasource.Access)
	}
	return datasource, nil
}
