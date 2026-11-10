package inputprofile

import "testing"

// The three identifiers below are wire values: a Host reads them to know which
// evidence it is holding, so a rename is a contract change and has to show up in
// a test. These assertions pin the constant to its literal; they do not exercise
// the producing paths (PreparePackageInputs and ObservePackageOutput need a
// BuildPlan and an evaluation result), which is noted where those paths are
// tested (#4032).
func TestPackageEvidenceSchemasKeepTheirWireValues(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"plugin definition profile", PluginDefinitionJSONV2, "tei.plugin-definition-json/v2"},
		{"input binding evidence", PackageInputBindingSchema, "tei.plugin-package-input-binding/v1"},
		{"output observation", PackageOutputObservationSchema, "tei.plugin-package-output-observation/v1"},
	}
	for _, testCase := range cases {
		if testCase.got != testCase.want {
			t.Errorf("%s = %q, want %q", testCase.name, testCase.got, testCase.want)
		}
	}
}
