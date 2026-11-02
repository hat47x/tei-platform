// Package studioplugininventory is a read-only reference for Studio plugin
// data inventory and AI diagnosis (#2045). It keeps the installation state,
// the enabled setting, the current usability, the permission, and the data
// format/version support apart, and never turns an incomplete scan or a past
// count into "0" or "confirmed".
//
// It is not a Studio UI, a plugin loader, a resolver, or an accepted API. It
// never starts a disabled plugin, never reads raw customer data, and never
// grants permission or acceptance. The grid behaviour on a real screen is
// outside this reference (moved to a follow-up issue).
package studioplugininventory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// Schema identifies the read-only inventory report.
const Schema = "tei.reference.studio-plugin-inventory/v0"

// Scan states. A scan that did not finish is never shown as complete.
const (
	ScanComplete = "complete"
	ScanPartial  = "partial"
	ScanNotRun   = "not-run"
)

// Row kinds. A not-installed reference row and an unknown-provider aggregate
// row are distinct from an installed plugin row.
const (
	RowPlugin    = "plugin"
	RowReference = "reference"
	RowAggregate = "aggregate"
)

// Row statuses. Each is a different fact, never one merged value.
const (
	StatusEnabled            = "enabled"
	StatusDisabled           = "disabled"
	StatusNotInstalled       = "not-installed"
	StatusVersionUnsupported = "version-unsupported"
	StatusProviderUnknown    = "provider-unknown"
	StatusMultipleProviders  = "multiple-providers"
	StatusUnreadable         = "unreadable"
	StatusPermissionDenied   = "permission-denied"
)

// Scan is the observation scope and the settings version it belongs to.
type Scan struct {
	State           string `json:"state"`
	ObservedAt      string `json:"observedAt,omitempty"`
	SettingsVersion string `json:"settingsVersion"`
	// AllowedTargets is the explicit allow list for inspection. Empty means
	// nothing is allowed; it is not "all allowed".
	AllowedTargets []string `json:"allowedTargets,omitempty"`
}

// Plugin is one installed or referenced plugin. Installed and Enabled are
// separate facts; a row selection in a grid is neither of them.
type Plugin struct {
	ID                string   `json:"id"`
	Version           string   `json:"version"`
	Installed         bool     `json:"installed"`
	Enabled           bool     `json:"enabled"`
	Permission        string   `json:"permission"` // allowed, denied, unknown
	SupportedVersions []string `json:"supportedVersions,omitempty"`
	Namespaces        []string `json:"namespaces,omitempty"`
	ReadState         string   `json:"readState"`           // readable, unreadable, not-read, out-of-scope
	Detection         string   `json:"detection,omitempty"` // explicit, past, unknown
	Files             int      `json:"files,omitempty"`
	// PastFiles is a previous count. It is kept apart from the current count
	// and is never added to it.
	PastFiles *int `json:"pastFiles,omitempty"`
}

// Area is one data area. Multiple providers may serve the same area; it is
// counted once, not once per provider.
type Area struct {
	Key       string   `json:"key"`
	FormatID  string   `json:"formatId"`
	Namespace string   `json:"namespace"`
	Version   string   `json:"version"`
	Providers []string `json:"providers,omitempty"`
	ReadState string   `json:"readState"` // readable, unreadable, not-read, out-of-scope
}

// Input is the whole synthetic observation.
type Input struct {
	Scan    Scan     `json:"scan"`
	Plugins []Plugin `json:"plugins"`
	Areas   []Area   `json:"areas"`
}

// Counts separates what was observed from the total. A nil total means it is
// unknown; it is never encoded as zero. Areas and files are separate counts.
type Counts struct {
	AreasKnown int  `json:"areasKnown"`
	AreasTotal *int `json:"areasTotal,omitempty"`
	FilesKnown *int `json:"filesKnown,omitempty"`
	FilesTotal *int `json:"filesTotal,omitempty"`
	Confirmed  bool `json:"confirmed"`
}

// Evidence is a reference to a fact, never the raw data itself.
type Evidence struct {
	Kind        string `json:"kind"`
	Ref         string `json:"ref"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

// Row is one grid row or one aggregate row.
type Row struct {
	Kind              string     `json:"kind"`
	PluginID          string     `json:"pluginId,omitempty"`
	AreaKey           string     `json:"areaKey,omitempty"`
	Status            string     `json:"status"`
	Enabled           *bool      `json:"enabled,omitempty"` // the setting, not a row selection
	ReasonCodes       []string   `json:"reasonCodes"`
	Evidence          []Evidence `json:"evidence,omitempty"`
	Target            string     `json:"target,omitempty"`
	Remediation       []string   `json:"remediation,omitempty"`
	AdditionalTargets []string   `json:"additionalTargets,omitempty"`
	Impact            string     `json:"impact,omitempty"`
	SourceFingerprint string     `json:"sourceFingerprint,omitempty"`
	SettingsVersion   string     `json:"settingsVersion,omitempty"`
	Recheck           []string   `json:"recheck,omitempty"`
	Remaining         []string   `json:"remaining,omitempty"`
	Counts            Counts     `json:"counts"`
	// PastFiles is shown as past information; it is not a current count.
	PastFiles *int `json:"pastFiles,omitempty"`
}

// Report is the read-only inventory and diagnosis.
type Report struct {
	Schema          string            `json:"schema"`
	ScanState       string            `json:"scanState"`
	ScanObservedAt  string            `json:"scanObservedAt,omitempty"`
	SettingsVersion string            `json:"settingsVersion"`
	Confirmed       bool              `json:"confirmed"`
	ScanReasons     []string          `json:"scanReasons,omitempty"`
	Remaining       []string          `json:"remaining,omitempty"`
	Rows            []Row             `json:"rows"`
	Totals          Counts            `json:"totals"`
	Messages        map[string]string `json:"messages,omitempty"`
}

// Catalog maps a stable reason code to display text. Changing text never
// changes a code.
type Catalog map[string]string

// DefaultCatalog is the reference display text.
func DefaultCatalog() Catalog {
	return Catalog{
		"plugin-not-installed":     "プラグインが導入されていません。",
		"plugin-disabled":          "プラグインは導入済みですが無効です。",
		"permission-denied":        "この主体には読取り権限がありません。",
		"permission-unknown":       "読取り権限を確認できていません。",
		"version-unsupported":      "保持データの版に対応していません。",
		"version-not-declared":     "対応する版が宣言されていません。",
		"read-unavailable":         "データを読み取れませんでした。",
		"provider-unknown":         "対応する提供元を特定できません。",
		"multiple-providers":       "同じ領域に複数の提供元があります。",
		"no-data-area":             "対応するデータ領域がありません。",
		"scan-incomplete":          "走査が完了していません。",
		"scan-not-run":             "走査はまだ実行されていません。",
		"observation-out-of-scope": "観測の許可対象に含まれていません。",
		"detection-past":           "過去の検出であり、現在の確認ではありません。",
		"counts-unknown":           "件数を確認できていません。",
	}
}

func has(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func sortedCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func fingerprint(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validate(in Input) error {
	switch in.Scan.State {
	case ScanComplete, ScanPartial, ScanNotRun:
	default:
		return fmt.Errorf("unknown scan state %q", in.Scan.State)
	}
	if strings.TrimSpace(in.Scan.SettingsVersion) == "" {
		return fmt.Errorf("scan.settingsVersion is required")
	}
	seenPlugins := map[string]bool{}
	for i, p := range in.Plugins {
		if strings.TrimSpace(p.ID) == "" {
			return fmt.Errorf("plugins[%d].id is required", i)
		}
		if seenPlugins[p.ID] {
			return fmt.Errorf("plugin %q is declared twice", p.ID)
		}
		seenPlugins[p.ID] = true
		switch p.Permission {
		case "allowed", "denied", "unknown":
		default:
			return fmt.Errorf("plugin %q: permission %q must be allowed, denied, or unknown", p.ID, p.Permission)
		}
		switch p.ReadState {
		case "", "readable", "unreadable", "not-read", "out-of-scope":
		default:
			return fmt.Errorf("plugin %q: unknown readState %q", p.ID, p.ReadState)
		}
	}
	seenAreas := map[string]bool{}
	for i, a := range in.Areas {
		if strings.TrimSpace(a.Key) == "" {
			return fmt.Errorf("areas[%d].key is required", i)
		}
		if seenAreas[a.Key] {
			return fmt.Errorf("area %q is declared twice", a.Key)
		}
		seenAreas[a.Key] = true
		if strings.TrimSpace(a.Version) == "" {
			return fmt.Errorf("area %q: version is required", a.Key)
		}
	}
	return nil
}

// Diagnose builds the inventory rows and the diagnosis. The scan state decides
// whether totals are known: a partial or not-run scan leaves them nil and
// Confirmed=false, never zero.
func Diagnose(in Input, catalog Catalog) (Report, error) {
	if err := validate(in); err != nil {
		return Report{}, err
	}
	if catalog == nil {
		catalog = DefaultCatalog()
	}
	complete := in.Scan.State == ScanComplete
	report := Report{
		Schema: Schema, ScanState: in.Scan.State, ScanObservedAt: in.Scan.ObservedAt,
		SettingsVersion: in.Scan.SettingsVersion,
	}
	switch in.Scan.State {
	case ScanPartial:
		report.ScanReasons = append(report.ScanReasons, "scan-incomplete")
		report.Remaining = append(report.Remaining, "rescan-required")
	case ScanNotRun:
		report.ScanReasons = append(report.ScanReasons, "scan-not-run")
		report.Remaining = append(report.Remaining, "run-scan")
	}

	pluginByID := map[string]Plugin{}
	for _, p := range in.Plugins {
		pluginByID[p.ID] = p
	}
	areasByNamespace := map[string][]Area{}
	for _, a := range in.Areas {
		areasByNamespace[a.Namespace] = append(areasByNamespace[a.Namespace], a)
	}
	allowed := map[string]bool{}
	for _, id := range in.Scan.AllowedTargets {
		allowed[id] = true
	}

	// coveredArea is the set of areas represented by an installed plugin row,
	// so an unknown-provider aggregate row is not added twice.
	coveredArea := map[string]bool{}
	codeSet := map[string]bool{}
	addCode := func(row *Row, code string) {
		row.ReasonCodes = append(row.ReasonCodes, code)
		codeSet[code] = true
	}

	var rows []Row
	for _, p := range in.Plugins {
		row := Row{
			Kind: RowPlugin, PluginID: p.ID, SettingsVersion: in.Scan.SettingsVersion,
			SourceFingerprint: fingerprint("plugin", p.ID, p.Version, p.ReadState, strings.Join(sortedCopy(p.Namespaces), ",")),
		}
		enabled := p.Enabled
		row.Enabled = &enabled
		row.Target = strings.Join(sortedCopy(p.Namespaces), ",")
		if p.Detection == "past" {
			addCode(&row, "detection-past")
		}
		// Areas represented by an installed plugin row are covered even when
		// the plugin is disabled: a disabled plugin is not "provider unknown".
		var handled []Area
		if p.Installed {
			handled = handledAreas(p, areasByNamespace)
			for _, area := range handled {
				coveredArea[area.Key] = true
			}
		}
		if !p.Installed {
			row.Kind = RowReference
			row.Status = StatusNotInstalled
			addCode(&row, "plugin-not-installed")
			row.Remediation = []string{"install-plugin"}
			row.Impact = "no-data-access"
			row.Recheck = []string{"after-install"}
			row.Counts.AreasTotal = nil
			row.PastFiles = p.PastFiles
			rows = append(rows, finishRow(row))
			continue
		}
		if !p.Enabled {
			row.Status = StatusDisabled
			addCode(&row, "plugin-disabled")
			row.Remediation = []string{"enable-plugin"}
			row.Impact = "no-data-access"
			row.Recheck = []string{"after-settings-change"}
			row.PastFiles = p.PastFiles
			rows = append(rows, finishRow(row))
			continue
		}
		if p.Permission != "allowed" {
			row.Status = StatusPermissionDenied
			if p.Permission == "unknown" {
				addCode(&row, "permission-unknown")
			} else {
				addCode(&row, "permission-denied")
			}
			row.Remediation = []string{"grant-permission"}
			row.Impact = "no-data-access"
			row.Recheck = []string{"after-permission-change"}
			row.PastFiles = p.PastFiles
			rows = append(rows, finishRow(row))
			continue
		}
		if len(in.Scan.AllowedTargets) > 0 && !allowed[p.ID] {
			addCode(&row, "observation-out-of-scope")
			row.Remaining = append(row.Remaining, "allow-and-rescan")
		}

		var versionUnsupported, unreadable, providerUnknown, multipleProviders []string
		for _, area := range handled {
			if !has(p.SupportedVersions, area.Version) {
				versionUnsupported = append(versionUnsupported, area.Key+":"+area.Version)
			}
			if area.ReadState == "unreadable" || p.ReadState == "unreadable" {
				unreadable = append(unreadable, area.Key)
			}
			if len(area.Providers) == 0 || !has(area.Providers, p.ID) {
				providerUnknown = append(providerUnknown, area.Key)
			}
			if len(area.Providers) > 1 {
				multipleProviders = append(multipleProviders, area.Key)
				for _, provider := range area.Providers {
					if provider != p.ID {
						row.AdditionalTargets = append(row.AdditionalTargets, provider)
					}
				}
			}
		}
		sort.Strings(versionUnsupported)
		sort.Strings(unreadable)
		sort.Strings(providerUnknown)
		sort.Strings(multipleProviders)
		row.AdditionalTargets = sortedCopy(unique(row.AdditionalTargets))

		switch {
		case len(versionUnsupported) > 0:
			row.Status = StatusVersionUnsupported
			for _, item := range versionUnsupported {
				addCode(&row, "version-unsupported:"+item)
			}
			if len(p.SupportedVersions) == 0 {
				addCode(&row, "version-not-declared")
			}
			row.Remediation = []string{"update-plugin", "keep-data-read-only"}
			row.Impact = "no-data-access"
			row.Recheck = []string{"after-plugin-update"}
		case len(unreadable) > 0:
			row.Status = StatusUnreadable
			for _, key := range unreadable {
				addCode(&row, "read-unavailable:"+key)
			}
			addCode(&row, "counts-unknown")
			row.Remediation = []string{"fix-read", "check-permission"}
			row.Impact = "no-data-access"
			row.Recheck = []string{"after-scan-complete"}
		case len(providerUnknown) > 0:
			row.Status = StatusProviderUnknown
			for _, key := range providerUnknown {
				addCode(&row, "provider-unknown:"+key)
			}
			row.Remediation = []string{"assign-provider"}
			row.Impact = "no-data-access"
			row.Recheck = []string{"after-settings-change"}
		case len(multipleProviders) > 0:
			row.Status = StatusMultipleProviders
			for _, key := range multipleProviders {
				addCode(&row, "multiple-providers:"+key)
			}
			row.Remediation = []string{"choose-provider", "verify-same-format"}
			row.Impact = "ambiguous-provider"
			row.Recheck = []string{"after-settings-change"}
		default:
			row.Status = StatusEnabled
			if len(handled) == 0 {
				addCode(&row, "no-data-area")
				row.Impact = "no-data"
			} else {
				row.Impact = "readable"
			}
			row.Recheck = []string{"after-settings-change", "after-plugin-update"}
		}

		// Counts: areas are separated from files, and known from total. A nil
		// total means unknown, never zero.
		readable := row.Status == StatusEnabled || row.Status == StatusMultipleProviders
		if readable && p.ReadState != "unreadable" {
			row.Counts.AreasKnown = len(handled)
			if complete {
				total := len(handled)
				row.Counts.AreasTotal = &total
				files := p.Files
				row.Counts.FilesKnown = &files
				if row.Status == StatusEnabled {
					row.Counts.FilesTotal = &files
				}
			} else {
				files := p.Files
				row.Counts.FilesKnown = &files
			}
		}
		row.Counts.Confirmed = complete && row.Status == StatusEnabled && len(row.ReasonCodes) == 0
		if row.Status == StatusUnreadable || row.Status == StatusProviderUnknown || row.Status == StatusVersionUnsupported || row.Status == StatusPermissionDenied {
			row.Remaining = append(row.Remaining, "resolve-blocker")
		}
		row.PastFiles = p.PastFiles
		for _, area := range handled {
			row.Evidence = append(row.Evidence, Evidence{
				Kind: "area", Ref: area.Key,
				Fingerprint: fingerprint("area", area.Key, area.FormatID, area.Namespace, area.Version),
			})
		}
		rows = append(rows, finishRow(row))
	}

	// Aggregate rows for areas no installed plugin covers, or whose provider is
	// not a known plugin. They are distinct from plugin rows.
	for _, area := range in.Areas {
		if coveredArea[area.Key] {
			continue
		}
		unknown := []string{}
		for _, provider := range area.Providers {
			if _, ok := pluginByID[provider]; !ok {
				unknown = append(unknown, provider)
			}
		}
		row := Row{
			Kind: RowAggregate, AreaKey: area.Key, Status: StatusProviderUnknown,
			SettingsVersion:   in.Scan.SettingsVersion,
			SourceFingerprint: fingerprint("area", area.Key, area.FormatID, area.Namespace, area.Version),
			Impact:            "no-data-access", Remediation: []string{"assign-provider"},
			Recheck: []string{"after-settings-change"},
			Evidence: []Evidence{{Kind: "area", Ref: area.Key,
				Fingerprint: fingerprint("area", area.Key, area.FormatID, area.Namespace, area.Version)}},
		}
		addCode(&row, "provider-unknown:"+area.Key)
		for _, provider := range sortedCopy(unique(unknown)) {
			row.AdditionalTargets = append(row.AdditionalTargets, provider)
		}
		if complete {
			total := 1
			row.Counts.AreasTotal = &total
		}
		rows = append(rows, finishRow(row))
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Kind != rows[j].Kind {
			return kindRank(rows[i].Kind) < kindRank(rows[j].Kind)
		}
		if rows[i].PluginID != rows[j].PluginID {
			return rows[i].PluginID < rows[j].PluginID
		}
		return rows[i].AreaKey < rows[j].AreaKey
	})
	report.Rows = rows

	// Totals: count each area once, regardless of how many providers it has.
	// "Known" means an installed, enabled, permitted plugin whose declared
	// version matches the area and that is the area's provider; an unreadable,
	// version-mismatched, or provider-less area is not counted as known.
	distinctAreas := in.Areas
	knownAreas := 0
	for _, area := range distinctAreas {
		for _, p := range in.Plugins {
			if pluginReadsArea(p, area) {
				knownAreas++
				break
			}
		}
	}
	report.Totals.AreasKnown = knownAreas
	if complete {
		total := len(distinctAreas)
		report.Totals.AreasTotal = &total
	}
	filesKnown := 0
	for _, p := range in.Plugins {
		canRead := false
		for _, area := range handledAreas(p, areasByNamespace) {
			if pluginReadsArea(p, area) {
				canRead = true
				break
			}
		}
		if canRead {
			filesKnown += p.Files
		}
	}
	if complete || filesKnown > 0 {
		value := filesKnown
		report.Totals.FilesKnown = &value
	}
	report.Confirmed = complete && !anyBlocking(rows)
	if report.Confirmed {
		value := filesKnown
		report.Totals.FilesTotal = &value
	}
	report.ScanReasons = sortedCopy(unique(report.ScanReasons))
	report.Remaining = sortedCopy(unique(report.Remaining))
	report.Messages = map[string]string{}
	for code := range codeSet {
		text, ok := catalog[code]
		if !ok {
			if idx := strings.Index(code, ":"); idx > 0 {
				text, ok = catalog[code[:idx]]
			}
		}
		if !ok {
			text = code
		}
		report.Messages[code] = text
	}
	return report, nil
}

// pluginReadsArea is the single definition of "this plugin can read this area
// now". It is used for the totals, not for the row status: a row may still
// exist and describe why it cannot read.
func pluginReadsArea(p Plugin, area Area) bool {
	return p.Installed && p.Enabled && p.Permission == "allowed" &&
		p.ReadState != "unreadable" && area.ReadState == "readable" &&
		has(p.SupportedVersions, area.Version) && has(area.Providers, p.ID)
}

func handledAreas(p Plugin, byNamespace map[string][]Area) []Area {
	var out []Area
	for _, namespace := range p.Namespaces {
		out = append(out, byNamespace[namespace]...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func unique(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func kindRank(kind string) int {
	switch kind {
	case RowPlugin:
		return 0
	case RowReference:
		return 1
	default:
		return 2
	}
}

func anyBlocking(rows []Row) bool {
	for _, row := range rows {
		switch row.Status {
		case StatusDisabled, StatusUnreadable, StatusProviderUnknown, StatusMultipleProviders,
			StatusVersionUnsupported, StatusPermissionDenied:
			return true
		}
	}
	return false
}

func finishRow(row Row) Row {
	row.ReasonCodes = sortedCopy(unique(row.ReasonCodes))
	row.AdditionalTargets = sortedCopy(unique(row.AdditionalTargets))
	row.Remediation = sortedCopy(unique(row.Remediation))
	row.Recheck = sortedCopy(unique(row.Recheck))
	row.Remaining = sortedCopy(unique(row.Remaining))
	sort.Slice(row.Evidence, func(i, j int) bool { return row.Evidence[i].Ref < row.Evidence[j].Ref })
	return row
}

// Skip records why a plugin was not observed.
type Skip struct {
	PluginID string `json:"pluginId"`
	Reason   string `json:"reason"`
}

// Observation records which plugins were actually invoked.
type Observation struct {
	Observed []string `json:"observed"`
	Skipped  []Skip   `json:"skipped"`
}

// Observe invokes fn only for installed, enabled, permitted plugins that are
// in the explicit allow list. A disabled plugin's code is never called. An
// empty allow list allows nothing.
func Observe(in Input, fn func(pluginID string) error) (Observation, error) {
	if err := validate(in); err != nil {
		return Observation{}, err
	}
	allowed := map[string]bool{}
	for _, id := range in.Scan.AllowedTargets {
		allowed[id] = true
	}
	out := Observation{Observed: []string{}, Skipped: []Skip{}}
	for _, p := range in.Plugins {
		switch {
		case !p.Installed:
			out.Skipped = append(out.Skipped, Skip{p.ID, "not-installed"})
		case !p.Enabled:
			out.Skipped = append(out.Skipped, Skip{p.ID, "disabled"})
		case p.Permission != "allowed":
			out.Skipped = append(out.Skipped, Skip{p.ID, "permission-not-granted"})
		case !allowed[p.ID]:
			out.Skipped = append(out.Skipped, Skip{p.ID, "out-of-scope"})
		default:
			if err := fn(p.ID); err != nil {
				return out, err
			}
			out.Observed = append(out.Observed, p.ID)
		}
	}
	sort.Strings(out.Observed)
	sort.Slice(out.Skipped, func(i, j int) bool {
		if out.Skipped[i].PluginID != out.Skipped[j].PluginID {
			return out.Skipped[i].PluginID < out.Skipped[j].PluginID
		}
		return out.Skipped[i].Reason < out.Skipped[j].Reason
	})
	return out, nil
}
