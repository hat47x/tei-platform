package applicationdriver

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/realization"
	"tei/internal/semantic"
)

// PatternResolutionは、Scenario内の一つのPatternから解決した既存Canonical modelを保持する。
// Query、Relation、Capability、Datasource、Bindingの意味をDriver専用の別modelへ写さず、
// それぞれを所有する既存の型をそのまま利用する。
type PatternResolution struct {
	Pattern        application.Pattern
	Query          *query.EmbeddedSQL
	Datasource     *application.DatabaseRequirement
	RelationSource *semantic.Data
	Relation       *semantic.Relation
	Capability     *semantic.Capability
	Binding        *realization.CustomGoBinding
}

// ResolveScenarioResourcesは、ScenarioPlanのPattern順序を保ったまま、
// Patternが実行時に参照する既存資産をCanonical identityから一意に解決する。
//
// この段階では資産を実行しない。解決できないidentityや曖昧なidentityは、
// 列挙順で補わず実行準備上のerrorとして返す。
func ResolveScenarioResources(world ir.World, plan ScenarioPlan) ([]PatternResolution, error) {
	out := make([]PatternResolution, 0, len(plan.Patterns))
	for _, pattern := range plan.Patterns {
		resolved := PatternResolution{Pattern: pattern}

		switch pattern.Kind {
		case application.PatternBrowse:
			q, err := resolveQuery(world, plan.Application.ID, pattern.Query)
			if err != nil {
				return nil, fmt.Errorf("browse pattern %s: %w", pattern.ID, err)
			}
			resolved.Query = &q

			datasourceAlias := pattern.Datasource
			if datasourceAlias == "" {
				datasourceAlias = q.Datasource
			} else if datasourceAlias != q.Datasource {
				return nil, fmt.Errorf("browse pattern %sのdatasource %sはquery %sのdatasource %sと一致しません", pattern.ID, datasourceAlias, q.ID, q.Datasource)
			}
			datasource, err := resolveDatasource(plan.Application, datasourceAlias)
			if err != nil {
				return nil, fmt.Errorf("browse pattern %s: %w", pattern.ID, err)
			}
			resolved.Datasource = &datasource

		case application.PatternReference:
			source, relation, err := resolveRelation(world, pattern.Relation)
			if err != nil {
				return nil, fmt.Errorf("reference pattern %s: %w", pattern.ID, err)
			}
			resolved.RelationSource = &source
			resolved.Relation = &relation

			datasource, err := resolveDatasource(plan.Application, pattern.Datasource)
			if err != nil {
				return nil, fmt.Errorf("reference pattern %s: %w", pattern.ID, err)
			}
			resolved.Datasource = &datasource

		case application.PatternHeaderDetail:
			source, relation, err := resolveRelation(world, pattern.Relation)
			if err != nil {
				return nil, fmt.Errorf("header-detail pattern %s: %w", pattern.ID, err)
			}
			resolved.RelationSource = &source
			resolved.Relation = &relation

		case application.PatternCommand:
			capability, err := resolveCapability(world, pattern.Capability)
			if err != nil {
				return nil, fmt.Errorf("command pattern %s: %w", pattern.ID, err)
			}
			resolved.Capability = &capability

			binding, err := realization.ResolveCustomGoBinding(world.Bindings, plan.Application.ID, capability.ID)
			if err != nil {
				return nil, fmt.Errorf("command pattern %s: %w", pattern.ID, err)
			}
			resolved.Binding = &binding

		case application.PatternRecord:
			// recordは現時点ではData identityだけで成立している。
			// QueryやDatasourceをDriver側から推測して追加しない。

		default:
			return nil, fmt.Errorf("pattern %sは未対応のkind %qを持っています", pattern.ID, pattern.Kind)
		}

		out = append(out, resolved)
	}
	return out, nil
}

func resolveQuery(world ir.World, applicationID, queryID core.CanonicalID) (query.EmbeddedSQL, error) {
	var found query.EmbeddedSQL
	count := 0
	for _, q := range world.Queries {
		if q.ID != queryID {
			continue
		}
		found = q
		count++
	}

	switch count {
	case 0:
		return query.EmbeddedSQL{}, fmt.Errorf("query %sがCanonical Worldに見つかりません", queryID)
	case 1:
		if found.Application != applicationID {
			return query.EmbeddedSQL{}, fmt.Errorf("query %sはapplication %sに属しており、application %sから利用できません", queryID, found.Application, applicationID)
		}
		return found, nil
	default:
		return query.EmbeddedSQL{}, fmt.Errorf("query %sがCanonical Worldに%d件あり、一意に解決できません", queryID, count)
	}
}

func resolveDatasource(app application.Application, alias core.LocalName) (application.DatabaseRequirement, error) {
	var found application.DatabaseRequirement
	count := 0
	for _, datasource := range app.Databases {
		if datasource.Alias != alias {
			continue
		}
		found = datasource
		count++
	}

	switch count {
	case 0:
		return application.DatabaseRequirement{}, fmt.Errorf("application %sにdatasource %sが見つかりません", app.ID, alias)
	case 1:
		return found, nil
	default:
		return application.DatabaseRequirement{}, fmt.Errorf("application %sにdatasource %sが%d件あり、一意に解決できません", app.ID, alias, count)
	}
}

func resolveRelation(world ir.World, relationID core.CanonicalID) (semantic.Data, semantic.Relation, error) {
	var source semantic.Data
	var found semantic.Relation
	count := 0
	for _, namespace := range world.Semantics {
		for _, data := range namespace.Data {
			for _, relation := range data.Relations {
				if relation.ID != relationID {
					continue
				}
				source = data
				found = relation
				count++
			}
		}
	}

	switch count {
	case 0:
		return semantic.Data{}, semantic.Relation{}, fmt.Errorf("semantic relation %sがCanonical Worldに見つかりません", relationID)
	case 1:
		return source, found, nil
	default:
		return semantic.Data{}, semantic.Relation{}, fmt.Errorf("semantic relation %sがCanonical Worldに%d件あり、一意に解決できません", relationID, count)
	}
}

func resolveCapability(world ir.World, capabilityID core.CanonicalID) (semantic.Capability, error) {
	var found semantic.Capability
	count := 0
	for _, namespace := range world.Semantics {
		for _, capability := range namespace.Capabilities {
			if capability.ID != capabilityID {
				continue
			}
			found = capability
			count++
		}
	}

	switch count {
	case 0:
		return semantic.Capability{}, fmt.Errorf("semantic capability %sがCanonical Worldに見つかりません", capabilityID)
	case 1:
		return found, nil
	default:
		return semantic.Capability{}, fmt.Errorf("semantic capability %sがCanonical Worldに%d件あり、一意に解決できません", capabilityID, count)
	}
}
