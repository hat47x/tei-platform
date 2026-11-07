package hook

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
)

// CommandPatternForは、Applicationの`command` Patternのうち、その能力を宣言する一つを
// 解決する。0件は未宣言、2件以上は曖昧として、どちらも拒否する。宣言順や名前の類似で
// 選ばない。
//
// `hook.CommandCaller`は能力からPatternへの対応を呼出し側から受け取る。この関数を
// 使えば、対応を推測せずに済む。戻り値のエラーは`hook.Error`で、符号は
// `HOOK_PATTERN_NOT_DECLARED`（0件）と`HOOK_PATTERN_AMBIGUOUS`（2件以上）である。
func CommandPatternFor(world ir.World, applicationID, capability core.CanonicalID) (core.CanonicalID, error) {
	if applicationID == "" {
		return "", &Error{Code: "HOOK_APPLICATION_REQUIRED", Message: "the command pattern lookup needs the application id"}
	}
	var found core.CanonicalID
	var foundBy core.CanonicalID
	for i := range world.Applications {
		if world.Applications[i].ID != applicationID {
			continue
		}
		for _, pattern := range world.Applications[i].Patterns {
			if pattern.Kind != application.PatternCommand || pattern.Capability != capability {
				continue
			}
			if found != "" {
				return "", &Error{
					Code:    "HOOK_PATTERN_AMBIGUOUS",
					Message: fmt.Sprintf("capability %s is declared by more than one command pattern (%s and %s)", capability, foundBy, pattern.ID),
				}
			}
			found = pattern.ID
			foundBy = pattern.ID
		}
	}
	if found == "" {
		return "", &Error{
			Code:    "HOOK_PATTERN_NOT_DECLARED",
			Message: "no command pattern of application " + string(applicationID) + " declares capability " + string(capability),
		}
	}
	return found, nil
}
