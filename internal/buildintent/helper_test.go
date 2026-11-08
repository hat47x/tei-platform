package buildintent

import "tei/internal/plugin"

// buildParameterForTest keeps the intent test focused on the name check rather
// than on the plugin parameter shape.
func buildParameterForTest(name string) plugin.BuildParameter {
	return plugin.BuildParameter{Name: name, Value: "x"}
}
