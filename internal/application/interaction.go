package application

import (
	"sort"

	"tei/internal/core"
)

// InteractionNavigationTargetsは、指定したPatternからの遷移を宣言している
// record Patternを返す。宣言の逆引きであり、新しい意味を作らない。
// 戻り値はPattern IDの順に並べ、呼出し側の表示順を安定させる。
func InteractionNavigationTargets(app Application, from core.CanonicalID) []Pattern {
	var out []Pattern
	for _, pattern := range app.Patterns {
		if pattern.Interaction == nil {
			continue
		}
		for _, navigation := range pattern.Interaction.Navigations {
			if navigation.From == from {
				out = append(out, pattern)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// InteractionItemByAttributeは、対話契約の項目をattributeのidentityで引く。
// 見つからない場合はfalseを返し、既定値や推測で補わない。
func InteractionItemByAttribute(interaction *Interaction, attribute core.CanonicalID) (InteractionItem, bool) {
	if interaction == nil {
		return InteractionItem{}, false
	}
	for _, item := range interaction.Items {
		if item.Attribute == attribute {
			return item, true
		}
	}
	return InteractionItem{}, false
}
