package applicationview

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// ApplicationViewは、ApplicationのPattern・Package・意味の版を一覧する投影である。
type ApplicationView struct {
	SchemaVersion string                  `json:"schemaVersion"`
	Application   core.CanonicalID        `json:"application"`
	SemanticUses  []core.CanonicalID      `json:"semanticUses,omitempty"`
	Patterns      []ApplicationPatternRef `json:"patterns,omitempty"`
	Packages      []ApplicationPackageRef `json:"packages,omitempty"`
	Periods       []ApplicationPeriodRef  `json:"periods,omitempty"`
	// StandardBusiness lists the standard business Data (tax, exchange rate,
	// numbering, holiday) the used semantics declare, with the attributes each
	// binder reads. It is a reading aid; the roles are checked by validation.
	StandardBusiness    []StandardBusinessRef `json:"standardBusiness,omitempty"`
	SemanticFingerprint string                `json:"semanticFingerprint,omitempty"`
}

// ApplicationPatternRefは、Applicationが持つPatternの種別とIDである。
type ApplicationPatternRef struct {
	ID   core.CanonicalID        `json:"id"`
	Kind application.PatternKind `json:"kind"`
}

// ApplicationPackageRefは、Applicationが持つPackageと版束縛である。
type ApplicationPackageRef struct {
	ID      core.CanonicalID `json:"id"`
	Version string           `json:"version"`
	Bound   bool             `json:"bound"`
}

// ApplicationPeriodRefは、Applicationが使う期間の宣言である。
type ApplicationPeriodRef struct {
	ID           core.CanonicalID `json:"id"`
	Data         core.CanonicalID `json:"data"`
	BusinessDate core.CanonicalID `json:"businessDate"`
	States       []string         `json:"states,omitempty"`
	// Calendar names the business calendar the period follows. An empty value
	// means the declaration names none, and a reader must not assume a default.
	Calendar core.CanonicalID `json:"calendar,omitempty"`
	// ClosingDay splits every month into two segments, 1..28. Zero means the
	// period is the whole month, so a reader sees "no closing" as an absence
	// rather than a default day.
	ClosingDay int `json:"closingDay,omitempty"`
}

// BuildApplicationは、ApplicationのPattern・Packageと、使用する意味の版を解決する。
func BuildApplication(world ir.World, applicationID core.CanonicalID) (ApplicationView, error) {
	var app *application.Application
	for i := range world.Applications {
		if world.Applications[i].ID == applicationID {
			app = &world.Applications[i]
			break
		}
	}
	if app == nil {
		return ApplicationView{}, fmt.Errorf("application %sがありません", applicationID)
	}
	view := ApplicationView{
		SchemaVersion: SchemaVersion,
		Application:   app.ID,
		SemanticUses:  app.SemanticUses,
	}
	for _, pattern := range app.Patterns {
		view.Patterns = append(view.Patterns, ApplicationPatternRef{ID: pattern.ID, Kind: pattern.Kind})
	}
	for _, pkg := range app.Packages {
		bound := false
		for _, binding := range app.PackageBindings {
			if binding.ID == pkg.ID && binding.Version == pkg.Version {
				bound = true
			}
		}
		view.Packages = append(view.Packages, ApplicationPackageRef{ID: pkg.ID, Version: pkg.Version, Bound: bound})
	}
	used := map[core.CanonicalID]bool{}
	for _, id := range app.SemanticUses {
		used[id] = true
	}
	var namespaces []semantic.Namespace
	for _, namespace := range world.Semantics {
		if used[namespace.ID] {
			namespaces = append(namespaces, namespace)
		}
	}
	fingerprint, err := semantic.Fingerprint(namespaces)
	if err != nil {
		return ApplicationView{}, err
	}
	for _, namespace := range namespaces {
		for _, period := range namespace.Periods {
			view.Periods = append(view.Periods, ApplicationPeriodRef{
				ID:           period.ID,
				Data:         period.Data,
				BusinessDate: period.BusinessDate,
				States:       period.States,
				Calendar:     period.Calendar,
				ClosingDay:   period.ClosingDay,
			})
		}
	}
	view.StandardBusiness = standardBusiness(namespaces)
	view.SemanticFingerprint = fingerprint
	return view, nil
}
