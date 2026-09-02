package applicationdriver

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
)

// ScenarioPlanは、Driverが一つのApplication Scenarioで利用する
// Canonical ApplicationとPatternの解決結果を保持する。
//
// Patternが参照するData、Query、Relation、Capability、Datasourceの意味は
// 既存のCanonical modelを正本とし、Driver専用の定義へ複製しない。
type ScenarioPlan struct {
	Application application.Application
	Patterns    []application.Pattern
}

// AssemblePatternScenarioは、Applicationと順序付きPattern IDをCanonical Worldから
// 一意に解決してScenarioPlanを組み立てる。
//
// この段階ではPatternの列を汎用Workflowとして解釈しない。呼び出し側が明示した
// Scenario上の順序を保ち、各Patternの意味は既存Canonical modelから取得する。
func AssemblePatternScenario(
	world ir.World,
	applicationID core.CanonicalID,
	patternIDs []core.CanonicalID,
) (ScenarioPlan, error) {
	app, err := resolveApplication(world, applicationID)
	if err != nil {
		return ScenarioPlan{}, err
	}
	if len(patternIDs) == 0 {
		return ScenarioPlan{}, fmt.Errorf("application %sのScenarioにPatternが指定されていません", applicationID)
	}

	patterns := make([]application.Pattern, 0, len(patternIDs))
	seen := make(map[core.CanonicalID]struct{}, len(patternIDs))
	for _, patternID := range patternIDs {
		if _, duplicated := seen[patternID]; duplicated {
			return ScenarioPlan{}, fmt.Errorf("application %sのScenarioでPattern %sが重複指定されています", applicationID, patternID)
		}
		seen[patternID] = struct{}{}

		pattern, err := resolvePattern(app, patternID)
		if err != nil {
			return ScenarioPlan{}, err
		}
		if !pattern.Kind.Valid() {
			return ScenarioPlan{}, fmt.Errorf("application %sのPattern %sは未対応のkind %qを持っています", applicationID, patternID, pattern.Kind)
		}
		patterns = append(patterns, pattern)
	}

	return ScenarioPlan{
		Application: app,
		Patterns:    patterns,
	}, nil
}

func resolveApplication(world ir.World, id core.CanonicalID) (application.Application, error) {
	var found application.Application
	count := 0
	for _, app := range world.Applications {
		if app.ID != id {
			continue
		}
		found = app
		count++
	}

	switch count {
	case 0:
		return application.Application{}, fmt.Errorf("application %sがCanonical Worldに見つかりません", id)
	case 1:
		return found, nil
	default:
		return application.Application{}, fmt.Errorf("application %sがCanonical Worldに%d件あり、一意に解決できません", id, count)
	}
}

func resolvePattern(app application.Application, id core.CanonicalID) (application.Pattern, error) {
	var found application.Pattern
	count := 0
	for _, pattern := range app.Patterns {
		if pattern.ID != id {
			continue
		}
		found = pattern
		count++
	}

	switch count {
	case 0:
		return application.Pattern{}, fmt.Errorf("application %sにPattern %sが見つかりません", app.ID, id)
	case 1:
		return found, nil
	default:
		return application.Pattern{}, fmt.Errorf("application %sにPattern %sが%d件あり、一意に解決できません", app.ID, id, count)
	}
}
