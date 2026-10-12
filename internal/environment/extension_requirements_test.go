package environment_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
)

func TestExtensionRequirementRejectsMalformedIDAndMajor(t *testing.T) {
	tests := []struct {
		name string
		req  environment.ExtensionRequirement
		want string
	}{
		{"empty id", environment.ExtensionRequirement{ID: "", CompatibleMajor: 1}, "invalid canonical id"},
		{"malformed id", environment.ExtensionRequirement{ID: core.CanonicalID("bad id"), CompatibleMajor: 1}, "invalid canonical id"},
		{"padded id", environment.ExtensionRequirement{ID: core.CanonicalID(" tei.bundle "), CompatibleMajor: 1}, "invalid canonical id"},
		{"zero major", environment.ExtensionRequirement{ID: core.MustCanonicalID("tei.bundle"), CompatibleMajor: 0}, "compatible major"},
		{"negative major", environment.ExtensionRequirement{ID: core.MustCanonicalID("tei.bundle"), CompatibleMajor: -1}, "compatible major"},
		{"valid", environment.ExtensionRequirement{ID: core.MustCanonicalID("tei.bundle"), CompatibleMajor: 1}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.want == "" {
				if err != nil {
					t.Fatalf("valid selection rejected: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %s: got %v", tc.want, err)
			}
		})
	}
}

func TestExtensionRequirementValidationOrderIndependent(t *testing.T) {
	a := environment.ExtensionRequirement{ID: core.MustCanonicalID("tei.a"), CompatibleMajor: -1}
	z := environment.ExtensionRequirement{ID: core.MustCanonicalID("tei.z"), CompatibleMajor: 0}
	first := []environment.ExtensionRequirement{z, a}
	second := []environment.ExtensionRequirement{a, z}
	err1 := environment.ValidateExtensionRequirements(first)
	err2 := environment.ValidateExtensionRequirements(second)
	if err1 == nil || err2 == nil || err1.Error() != err2.Error() {
		t.Fatalf("declaration order changed error: %v / %v", err1, err2)
	}
	if !strings.Contains(err1.Error(), "tei.a") {
		t.Fatalf("lexicographically first invalid ID missing: %v", err1)
	}
	if err := environment.ValidateExtensionRequirements(nil); err != nil {
		t.Fatalf("empty selection should be valid: %v", err)
	}
}
