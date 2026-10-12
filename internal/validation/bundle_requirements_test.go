package validation_test

import (
	"strings"
	"testing"

	"tei/internal/environment"
	"tei/internal/plugin"
	"tei/internal/validation"
)

func TestBundleWorldValidationRejectsRepeatedPluginID(t *testing.T) {
	for _, majors := range [][2]int{{1, 1}, {1, 2}, {2, 1}} {
		w := referenceWorld()
		w.Bundles = []plugin.Bundle{{
			ID: cid("tei.bundle"), Version: "1.0.0",
			Plugins: []plugin.Requirement{
				{Plugin: cid("tei.test.reference"), CompatibleMajor: majors[0]},
				{Plugin: cid("tei.test.reference"), CompatibleMajor: majors[1]},
			},
		}}
		count := 0
		for _, d := range validation.Validate(w) {
			if d.Code != "DUPLICATE_BUNDLE_PLUGIN_REQUIREMENT" {
				continue
			}
			count++
			if d.Severity != validation.Error ||
				!strings.Contains(d.Message, "bundle tei.bundle version 1.0.0") ||
				!strings.Contains(d.Message, "plugin requirement tei.test.reference appears more than once") {
				t.Fatalf("majors %v: incomplete bundle duplicate diagnostic: %+v", majors, d)
			}
		}
		if count != 1 {
			t.Fatalf("majors %v: expected exactly one bundle duplicate diagnostic, got %d", majors, count)
		}
	}
}

func TestBundleWorldValidationAllowsSharedRequirementsAcrossBundles(t *testing.T) {
	w := referenceWorld()
	requirement := plugin.Requirement{Plugin: cid("tei.test.reference"), CompatibleMajor: 1}
	w.Bundles = []plugin.Bundle{
		{ID: cid("tei.first"), Version: "1.0.0", Plugins: []plugin.Requirement{requirement}},
		{ID: cid("tei.second"), Version: "1.0.0", Plugins: []plugin.Requirement{requirement}},
	}
	w.Environments[0].Bundles = []environment.ExtensionRequirement{
		{ID: cid("tei.first"), CompatibleMajor: 1},
		{ID: cid("tei.second"), CompatibleMajor: 1},
	}
	for _, d := range validation.Validate(w) {
		if d.Severity == validation.Error {
			t.Fatalf("shared requirement must not be rejected: %+v", d)
		}
	}
}

func TestBundleWorldValidationDuplicateDiagnosticIsOrderIndependent(t *testing.T) {
	first := []plugin.Requirement{
		{Plugin: cid("tei.z"), CompatibleMajor: 1},
		{Plugin: cid("tei.a"), CompatibleMajor: 2},
		{Plugin: cid("tei.z"), CompatibleMajor: 2},
		{Plugin: cid("tei.a"), CompatibleMajor: 1},
	}
	second := []plugin.Requirement{first[3], first[2], first[1], first[0]}
	var expected string
	for _, requirements := range [][]plugin.Requirement{first, second} {
		w := referenceWorld()
		w.Bundles = []plugin.Bundle{{
			ID: cid("tei.bundle"), Version: "1.0.0", Plugins: requirements,
		}}
		var message string
		for _, diagnostic := range validation.Validate(w) {
			if diagnostic.Code != "DUPLICATE_BUNDLE_PLUGIN_REQUIREMENT" {
				continue
			}
			if message != "" {
				t.Fatal("duplicate diagnostic must occur once per invalid bundle")
			}
			message = diagnostic.Message
		}
		if !strings.Contains(message, "plugin requirement tei.a appears more than once") {
			t.Fatalf("expected stable lowest duplicate ID, got %q", message)
		}
		if expected == "" {
			expected = message
		} else if message != expected {
			t.Fatalf("order changed the diagnostic: first=%q second=%q", expected, message)
		}
	}
}
