package validation_test

import (
	"testing"

	"tei/internal/plugin"
	"tei/internal/validation"
)

func TestWorldValidationRejectsRepeatedPluginDependencies(t *testing.T) {
	for _, majors := range [][2]int{{1, 1}, {1, 2}} {
		w := referenceWorld()
		w.Plugins[0].Requires = []plugin.Requirement{
			{Plugin: cid("tei.base"), CompatibleMajor: majors[0]},
			{Plugin: cid("tei.base"), CompatibleMajor: majors[1]},
		}
		found := false
		for _, d := range validation.Validate(w) {
			if d.Code == "DUPLICATE_PLUGIN_REQUIREMENT" && d.Severity == validation.Error {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("duplicate requires with majors %v were not rejected", majors)
		}
	}
}
