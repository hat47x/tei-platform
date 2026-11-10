package validation

import (
	"strings"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/subject"
)

// validatePackageConformance checks the contract fields a Package declares about
// itself: its conformance suite name and the projection it produces.
//
// RQ-C1 asks for packages that carry a contract, a version, and a conformance test,
// and for two independent packages to be compared under the same conformance
// framework. `Conformance` and `Projection` were parsed and stored but nothing read
// them, so a package could claim any suite name and any projection without the
// claim being checked — the D-9 shape again (a declared value that means nothing at
// run time).
//
// The check is a contract check, not a proof: it requires the fields to be present
// and well formed, and it reports two packages in one application that claim
// different suites, because comparing them under one framework then is not possible.
// Running the suite itself is a separate provider's job.
func validatePackageConformance(w ir.World) []Diagnostic {
	var out []Diagnostic
	for _, app := range w.Applications {
		// firstSuite records, per pattern kind, the first package and suite that
		// claims it. A later package claiming the same kind with a different suite is
		// reported: the two cannot be compared under one framework.
		firstSuite := map[application.PatternKind]application.Package{}
		for _, pkg := range app.Packages {
			pkgSubject := subject.Subject{Kind: "application-package", ID: string(pkg.ID)}
			switch {
			case strings.TrimSpace(pkg.Conformance) == "":
				out = append(out, errFor(pkgSubject, "APPLICATION_PACKAGE_CONFORMANCE_REQUIRED",
					"package %s does not declare a conformance suite", pkg.ID))
			case strings.ContainsAny(pkg.Conformance, " \t"):
				out = append(out, errFor(pkgSubject, "APPLICATION_PACKAGE_CONFORMANCE_INVALID",
					"package %s declares conformance %q with whitespace", pkg.ID, pkg.Conformance))
			}
			if strings.TrimSpace(pkg.Projection) == "" {
				out = append(out, errFor(pkgSubject, "APPLICATION_PACKAGE_PROJECTION_REQUIRED",
					"package %s does not declare the projection it produces", pkg.ID))
			}
			for _, member := range pkg.Members {
				first, seen := firstSuite[member.Kind]
				if !seen {
					firstSuite[member.Kind] = pkg
					continue
				}
				if first.Conformance == pkg.Conformance {
					continue
				}
				out = append(out, errFor(pkgSubject, "APPLICATION_PACKAGE_CONFORMANCE_MISMATCH",
					"packages %s and %s both declare a %s member but name different conformance suites (%q and %q), so they cannot be compared under one framework",
					first.ID, pkg.ID, member.Kind, first.Conformance, pkg.Conformance))
			}
		}
	}
	return out
}
