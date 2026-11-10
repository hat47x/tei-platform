package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
)

// A Package declares its own identity and the semantic elements it requires.
// Each rule answers with its own code, so the codes are produced here by
// breaking one declaration of an accepted world (#4032).
func TestApplicationPackageCodesAreProduced(t *testing.T) {
	t.Run("package namespace mismatch", func(t *testing.T) {
		world := packageWorld()
		world.Applications[0].Packages[0].ID = "other_app.list_to_record"
		assertHasDiagnostic(t, Validate(world), "APPLICATION_PACKAGE_NAMESPACE_MISMATCH")
	})

	t.Run("package version required", func(t *testing.T) {
		world := packageWorld()
		world.Applications[0].Packages[0].Version = ""
		assertHasDiagnostic(t, Validate(world), "APPLICATION_PACKAGE_VERSION_REQUIRED")
	})

	t.Run("member belongs to another application", func(t *testing.T) {
		world := packageWorld()
		world.Applications = append(world.Applications, application.Application{
			ID: "other_app",
			Patterns: []application.Pattern{
				{ID: "other_app.record", Kind: application.PatternRecord, Data: "sales.order"},
			},
		})
		world.Applications[0].Packages[0].Members = append(world.Applications[0].Packages[0].Members,
			application.PackageMember{Kind: application.PatternRecord, Pattern: "other_app.record"})
		assertHasDiagnostic(t, Validate(world), "APPLICATION_PACKAGE_MEMBER_APPLICATION_MISMATCH")
	})

	t.Run("unknown package requirement", func(t *testing.T) {
		world := packageWorld()
		world.Applications[0].Packages[0].Requires = append(world.Applications[0].Packages[0].Requires,
			application.PackageRequirement{Kind: "semantic", ID: core.CanonicalID("sales.missing")})
		assertHasDiagnostic(t, Validate(world), "UNKNOWN_APPLICATION_PACKAGE_REQUIREMENT")
	})
}
