package management

import (
	"sort"

	"tei/internal/plugin"
)

// DocumentUpstreamSelection reports the *existing resolver's* selected exact
// identity for a directly required Plugin. This is not an upstream document
// reference, authorization, or a choice of which version to use.
type DocumentUpstreamSelection struct {
	PluginID        string `json:"pluginId"`
	CompatibleMajor int    `json:"compatibleMajor"`
	SelectionStatus string `json:"selectionStatus"`
	ResolvedVersion string `json:"resolvedVersion,omitempty"`
	LockPinned      *bool  `json:"lockPinned"`
}

// DocumentReadPreflight projects the existing inventory's selection evidence
// into the first prerequisite for a source-backed public document read.
//
// It does not discover a Document, read its bytes, check Audience/Disclosure,
// or grant publication. Until a Plugin -> Document binding is accepted, the
// result does not assess document eligibility.
type DocumentReadPreflight struct {
	PluginID        string                      `json:"pluginId"`
	Environment     string                      `json:"environment"`
	SelectionStatus string                      `json:"selectionStatus"`
	ResolvedVersion string                      `json:"resolvedVersion,omitempty"`
	LockPinned      *bool                       `json:"lockPinned"`
	SourceBinding   string                      `json:"sourceBinding"`
	Disclosure      string                      `json:"disclosure"`
	ReadEligibility string                      `json:"readEligibility"`
	Upstream        []DocumentUpstreamSelection `json:"upstreamSelection,omitempty"`
}

const (
	DocumentSelectionScanIncomplete  = "scan-incomplete"
	DocumentSelectionNotInInventory  = "not-in-inventory"
	DocumentSelectionNotDeclared     = "not-declared"
	DocumentSelectionNotSelected     = "not-selected"
	DocumentSelectionResolverUnbound = "resolver-unbound"
	DocumentSelectionVersionUnknown  = "version-unknown"
	DocumentSelectionVersionMismatch = "version-mismatch"
	DocumentSelectionLockUnknown     = "lock-unknown"
	DocumentSelectionLockMismatch    = "lock-mismatch"
	DocumentSelectionExactPinned     = "exact-version-pinned"
)

// InspectDocumentReadPreflight NEVER resolves a Plugin a second time.
// The input is the same Report already derived for plugin-inventory and its
// exact Row. This projection is intentionally non-authorizing, including when
// SelectionStatus is exact-version-pinned.
func InspectDocumentReadPreflight(report Report, pluginID string) DocumentReadPreflight {
	out := DocumentReadPreflight{
		PluginID:        pluginID,
		Environment:     report.Environment,
		SelectionStatus: DocumentSelectionScanIncomplete,
		SourceBinding:   "unbound",
		Disclosure:      "not-checked",
		ReadEligibility: "not-assessed",
	}
	if report.ScanState != ScanComplete {
		return out
	}
	row, ok := FindRow(report, pluginID)
	if !ok {
		out.SelectionStatus = DocumentSelectionNotInInventory
		return out
	}
	if !row.Declared {
		out.SelectionStatus = DocumentSelectionNotDeclared
		return out
	}
	if !row.EnvSelected {
		out.SelectionStatus = DocumentSelectionNotSelected
		return out
	}
	if !report.Resolver.Resolved || !row.Resolved {
		out.SelectionStatus = DocumentSelectionResolverUnbound
		return out
	}
	if _, err := plugin.ParseVersion(row.Version); err != nil {
		out.SelectionStatus = DocumentSelectionVersionUnknown
		return out
	}
	out.ResolvedVersion = row.Version
	out.Upstream = inspectDocumentUpstreamSelections(report, row.Requires)
	out.LockPinned = row.LockPinned
	if row.LockPinned == nil {
		out.SelectionStatus = DocumentSelectionLockUnknown
		return out
	}
	if !*row.LockPinned {
		out.SelectionStatus = DocumentSelectionLockMismatch
		return out
	}
	out.SelectionStatus = DocumentSelectionExactPinned
	return out
}

// inspectDocumentUpstreamSelections projects only *direct* Requires entries
// from the already-derived inventory. It never re-resolves, looks up Source,
// searches for a Document, or assumes a compatibleMajor chooses an exact
// version. The selection facts are useful for B -> A document provenance,
// but must never be treated as publication eligibility.
func inspectDocumentUpstreamSelections(report Report, requires []Requirement) []DocumentUpstreamSelection {
	if len(requires) == 0 {
		return nil
	}
	out := make([]DocumentUpstreamSelection, 0, len(requires))
	for _, req := range requires {
		upstream := DocumentUpstreamSelection{
			PluginID:        req.Plugin,
			CompatibleMajor: req.CompatibleMajor,
			SelectionStatus: DocumentSelectionNotInInventory,
		}
		row, ok := FindRow(report, req.Plugin)
		switch {
		case !ok:
			// The requested dependency is not a selected Plugin identity.
		case !row.Declared:
			upstream.SelectionStatus = DocumentSelectionNotDeclared
		case !row.EnvSelected:
			upstream.SelectionStatus = DocumentSelectionNotSelected
		case !report.Resolver.Resolved || !row.Resolved:
			upstream.SelectionStatus = DocumentSelectionResolverUnbound
		default:
			if _, err := plugin.ParseVersion(row.Version); err != nil {
				upstream.SelectionStatus = DocumentSelectionVersionUnknown
			} else if !plugin.CompatibleMajor(row.Version, req.CompatibleMajor) {
				upstream.SelectionStatus = DocumentSelectionVersionMismatch
			} else {
				upstream.ResolvedVersion = row.Version
				upstream.LockPinned = row.LockPinned
				switch {
				case row.LockPinned == nil:
					upstream.SelectionStatus = DocumentSelectionLockUnknown
				case !*row.LockPinned:
					upstream.SelectionStatus = DocumentSelectionLockMismatch
				default:
					upstream.SelectionStatus = DocumentSelectionExactPinned
				}
			}
		}
		out = append(out, upstream)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PluginID != out[j].PluginID {
			return out[i].PluginID < out[j].PluginID
		}
		return out[i].CompatibleMajor < out[j].CompatibleMajor
	})
	return out
}
