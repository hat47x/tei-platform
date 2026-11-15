package management

import "tei/internal/plugin"

// DocumentReadPreflight projects the existing inventory's selection evidence
// into the first prerequisite for a source-backed public document read.
//
// It does not discover a Document, read its bytes, check Audience/Disclosure,
// or grant publication. Until a Plugin -> Document binding is accepted, the
// result always says that the document cannot be read.
type DocumentReadPreflight struct {
	PluginID         string `json:"pluginId"`
	Environment      string `json:"environment"`
	SelectionStatus  string `json:"selectionStatus"`
	ResolvedVersion  string `json:"resolvedVersion,omitempty"`
	LockPinned       *bool  `json:"lockPinned"`
	SourceBinding    string `json:"sourceBinding"`
	Disclosure       string `json:"disclosure"`
	ReadEligibility  string `json:"readEligibility"`
}

const (
	DocumentSelectionScanIncomplete   = "scan-incomplete"
	DocumentSelectionNotInInventory   = "not-in-inventory"
	DocumentSelectionNotDeclared      = "not-declared"
	DocumentSelectionNotSelected      = "not-selected"
	DocumentSelectionResolverUnbound  = "resolver-unbound"
	DocumentSelectionVersionUnknown   = "version-unknown"
	DocumentSelectionLockUnknown      = "lock-unknown"
	DocumentSelectionLockMismatch     = "lock-mismatch"
	DocumentSelectionExactPinned      = "exact-version-pinned"
)

// InspectDocumentReadPreflight NEVER resolves a Plugin a second time.
// The input is the same Report already derived for plugin-inventory and its
// exact Row. This projection is intentionally non-authorizing, including when
// SelectionStatus is exact-version-pinned.
func InspectDocumentReadPreflight(report Report, pluginID string) DocumentReadPreflight {
	out := DocumentReadPreflight{
		PluginID:         pluginID,
		Environment:      report.Environment,
		SelectionStatus:  DocumentSelectionScanIncomplete,
		SourceBinding:    "unbound",
		Disclosure:       "not-checked",
		ReadEligibility:   "not-assessed",
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
