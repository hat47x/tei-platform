package validation

import (
	"strings"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/subject"
)

// validateApplicationPackagesは、Packageが既存PatternとSemanticだけを参照しているかを検査する。
// Packageは意味を複製しないため、未知の参照や種別の食い違いは推測で補わずエラーにする。
func validateApplicationPackages(w ir.World) []Diagnostic {
	var out []Diagnostic

	patterns := map[core.CanonicalID]application.Pattern{}
	patternApplications := map[core.CanonicalID]core.CanonicalID{}
	semanticData := map[core.CanonicalID]core.CanonicalID{}
	semanticRelations := map[core.CanonicalID]core.CanonicalID{}
	for _, namespace := range w.Semantics {
		for _, data := range namespace.Data {
			semanticData[data.ID] = namespace.ID
			for _, relation := range data.Relations {
				semanticRelations[relation.ID] = namespace.ID
			}
		}
	}
	for _, app := range w.Applications {
		for _, pattern := range app.Patterns {
			if _, exists := patterns[pattern.ID]; !exists {
				patterns[pattern.ID] = pattern
				patternApplications[pattern.ID] = app.ID
			}
		}
	}

	for _, app := range w.Applications {
		used := map[core.CanonicalID]bool{}
		for _, namespace := range app.SemanticUses {
			used[namespace] = true
		}
		declaredPackages := map[core.CanonicalID]bool{}
		memberOwners := map[core.CanonicalID]core.CanonicalID{}
		for _, pkg := range app.Packages {
			packageSubject := subject.Subject{Kind: "application-package", ID: string(pkg.ID)}
			if declaredPackages[pkg.ID] {
				out = append(out, errFor(packageSubject, "APPLICATION_PACKAGE_DUPLICATE", "application %s declares package %s more than once", app.ID, pkg.ID))
			}
			declaredPackages[pkg.ID] = true
			if !strings.HasPrefix(string(pkg.ID), string(app.ID)+".") {
				out = append(out, errFor(packageSubject, "APPLICATION_PACKAGE_NAMESPACE_MISMATCH", "package %s is declared under application %s", pkg.ID, app.ID))
			}
			if pkg.Version == "" {
				out = append(out, errFor(packageSubject, "APPLICATION_PACKAGE_VERSION_REQUIRED", "package %s requires a version", pkg.ID))
			}
			seen := map[core.CanonicalID]bool{}
			for _, member := range pkg.Members {
				pattern, ok := patterns[member.Pattern]
				if !ok {
					out = append(out, errFor(packageSubject, "UNKNOWN_APPLICATION_PACKAGE_MEMBER", "package %s refers to unknown pattern %s", pkg.ID, member.Pattern))
					continue
				}
				if patternApplications[member.Pattern] != app.ID {
					out = append(out, errFor(packageSubject, "APPLICATION_PACKAGE_MEMBER_APPLICATION_MISMATCH", "package %s refers to pattern %s owned by application %s", pkg.ID, member.Pattern, patternApplications[member.Pattern]))
				}
				if pattern.Kind != member.Kind {
					out = append(out, errFor(packageSubject, "APPLICATION_PACKAGE_MEMBER_KIND_MISMATCH", "package %s declares member %s as %s but the pattern is %s", pkg.ID, member.Pattern, member.Kind, pattern.Kind))
				}
				if seen[member.Pattern] {
					out = append(out, errFor(packageSubject, "APPLICATION_PACKAGE_MEMBER_DUPLICATE", "package %s declares member %s more than once", pkg.ID, member.Pattern))
				}
				if owner, exists := memberOwners[member.Pattern]; exists && owner != pkg.ID {
					out = append(out, errFor(packageSubject, "APPLICATION_PACKAGE_MEMBER_COLLISION", "pattern %s is declared by packages %s and %s; resolve the collision explicitly", member.Pattern, owner, pkg.ID))
				} else {
					memberOwners[member.Pattern] = pkg.ID
				}
				seen[member.Pattern] = true
			}
			for _, requirement := range pkg.Requires {
				var namespace core.CanonicalID
				switch requirement.Kind {
				case "semantic":
					namespace = semanticData[requirement.ID]
				case "relation":
					namespace = semanticRelations[requirement.ID]
				default:
					out = append(out, errFor(packageSubject, "UNKNOWN_APPLICATION_PACKAGE_REQUIREMENT", "package %s requires %s %s with an unsupported kind", pkg.ID, requirement.Kind, requirement.ID))
					continue
				}
				if namespace == "" {
					out = append(out, errFor(packageSubject, "UNKNOWN_APPLICATION_PACKAGE_REQUIREMENT", "package %s requires unknown %s %s", pkg.ID, requirement.Kind, requirement.ID))
					continue
				}
				if !used[namespace] {
					out = append(out, errFor(packageSubject, "APPLICATION_PACKAGE_REQUIREMENT_SEMANTIC_NOT_USED", "package %s requires %s %s from namespace %s that application %s does not use", pkg.ID, requirement.Kind, requirement.ID, namespace, app.ID))
				}
			}
		}
	}
	return out
}
