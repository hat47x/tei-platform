package driver

import (
	"errors"
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// ErrApplicationValidationSelectionNotDeclaredは、Applicationが`uses validation`を一件も
// 宣言していないことを表す。
//
// 宣言が無いことを「評価すべきValidationが無い」とは読まない。呼び出し側が選択を要求した
// 時点で、これは業務上の拒否ではなく、実行できない状態（Execution Error側）である（ADR-0029）。
var ErrApplicationValidationSelectionNotDeclared = errors.New("APPLICATION_VALIDATION_SELECTION_NOT_DECLARED")

// ResolveApplicationValidationsは、Applicationが評価するBusiness Validationを、そのApplicationの
// `uses validation`宣言から、宣言順に解決する（ADR-0029）。
//
// 評価対象の選択はCanonicalな意味であり、Target・Scenario・API・AIエージェントの呼び出し側は
// 自分で一覧を持たず、この入口だけを使う。
//
//   - 宣言が一件も無いときは、空集合を「全て満たされた」と読ませず、
//     ErrApplicationValidationSelectionNotDeclaredを返す。
//   - 宣言したIDがWorldに無いときは、評価を省略せずerrorを返す。
//   - 同じIDを二回宣言していても、二重に評価せずerrorを返す。評価順の意味があいまいになるため。
//   - 順序は宣言順のまま返す。Loaderは並べ替えない。
func (r Resolver) ResolveApplicationValidations(applicationID core.CanonicalID) ([]semantic.Validation, error) {
	app, err := r.applicationByID(applicationID)
	if err != nil {
		return nil, err
	}
	if len(app.Validations) == 0 {
		return nil, fmt.Errorf(
			"%w: Application %sは`uses validation`を宣言していないため、評価するValidationを選べません",
			ErrApplicationValidationSelectionNotDeclared,
			applicationID,
		)
	}

	seen := make(map[core.CanonicalID]struct{}, len(app.Validations))
	resolved := make([]semantic.Validation, 0, len(app.Validations))
	for _, id := range app.Validations {
		if _, duplicate := seen[id]; duplicate {
			return nil, fmt.Errorf("Application %sが`uses validation %s`を重複して宣言しています", applicationID, id)
		}
		seen[id] = struct{}{}

		rule, err := r.validationByID(id)
		if err != nil {
			return nil, fmt.Errorf("Application %sが宣言したValidationを解決できません: %w", applicationID, err)
		}
		resolved = append(resolved, rule)
	}
	return resolved, nil
}

func (r Resolver) validationByID(id core.CanonicalID) (semantic.Validation, error) {
	var found semantic.Validation
	count := 0
	for _, namespace := range r.world.Semantics {
		for _, rule := range namespace.Validations {
			if rule.ID != id {
				continue
			}
			found = rule
			count++
		}
	}
	if count == 0 {
		return semantic.Validation{}, fmt.Errorf("Semantic Validation %sが見つかりません", id)
	}
	if count > 1 {
		return semantic.Validation{}, fmt.Errorf("Semantic Validation %sが複数定義されています: %d件", id, count)
	}
	return found, nil
}
