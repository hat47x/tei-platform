package main

import (
	"fmt"
	"strings"
)

// pinReproducibleArtifactSetは、独立した二回の実行が同じ内容の集合を返した場合に限り、
// その集合の内容同一性（指紋）を正準の出力として返す。
//
// compareArtifactSetsの`Reproducible`は比較の結果であり、それ自体は何も拒否しない。
// 再現しない集合（欠落・余分・変更・指紋の不一致があるもの）を正準の出力として固定すると、
// 後続のLockが一回の実行の偶然の内容を拠り所にしてしまう。このため、再現しない場合は
// 指紋を返さずに失敗し、どのRoleが原因かをエラーに残す。
//
// locatorは内容同一性に含まれないので、locatorだけが異なる二つの集合は再現したものとして扱う。
// この関数はBuild ProvenanceやAcceptanceを与えない。同じ内容が二回得られたことだけを示す。
func pinReproducibleArtifactSet(first, second ArtifactSet) (string, error) {
	comparison, err := compareArtifactSets(first, second)
	if err != nil {
		return "", err
	}
	if !comparison.Reproducible {
		return "", fmt.Errorf("artifact set is not reproducible and cannot be pinned: %s", describeArtifactSetDrift(comparison))
	}
	return comparison.FirstFingerprint, nil
}

func describeArtifactSetDrift(comparison ArtifactSetComparison) string {
	var parts []string
	if len(comparison.MissingFromSecond) > 0 {
		parts = append(parts, "missing from second: "+strings.Join(comparison.MissingFromSecond, ", "))
	}
	if len(comparison.ExtraInSecond) > 0 {
		parts = append(parts, "extra in second: "+strings.Join(comparison.ExtraInSecond, ", "))
	}
	if len(comparison.Changed) > 0 {
		roles := make([]string, 0, len(comparison.Changed))
		for _, change := range comparison.Changed {
			roles = append(roles, change.Role)
		}
		parts = append(parts, "changed: "+strings.Join(roles, ", "))
	}
	if len(parts) == 0 {
		// 指紋だけが異なる場合。メンバー単位の差が無いのに指紋が違うことは通常起きないが、
		// 起きたときに「再現した」と誤って読まれないよう、理由を明示して拒否する。
		parts = append(parts, "fingerprints differ")
	}
	return strings.Join(parts, "; ")
}
