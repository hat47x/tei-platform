package studioplugindatalifecycle

import (
	"fmt"
	"sort"
	"strings"
)

// Area mappings. A confirmed area has one known target; a shared area has
// several; an unknown area has no known provider.
const (
	MappingConfirmed = "confirmed"
	MappingShared    = "shared"
	MappingUnknown   = "unknown"
)

// Segment kinds. Only owned and shared segments may be deleted for a selected
// area. Extensions and unknown attributes/elements are never cut out.
const (
	SegmentOwned            = "owned"
	SegmentShared           = "shared"
	SegmentExtension        = "extension"
	SegmentUnknownAttribute = "unknown-attribute"
	SegmentUnknownElement   = "unknown-element"
	SegmentUnknown          = "unknown"
)

// Actions for a data lifecycle plan.
const (
	ActionRemovePlugin = "remove-plugin"
	ActionDeleteData   = "delete-data"
)

// Area is one data area and its mapping.
type Area struct {
	Key          string   `json:"key"`
	Mapping      string   `json:"mapping"`
	Owners       []string `json:"owners,omitempty"`
	ReferencedBy []string `json:"referencedBy,omitempty"`
}

// Segment is one byte range of a source file.
type Segment struct {
	Key   string `json:"key"`
	Area  string `json:"area,omitempty"`
	Kind  string `json:"kind"`
	Bytes string `json:"bytes"`
}

// SourceFile is one design source and its byte ranges.
type SourceFile struct {
	Path        string    `json:"path"`
	Segments    []Segment `json:"segments"`
	Fingerprint string    `json:"fingerprint,omitempty"`
}

// DataState is the current plugin and data mapping.
type DataState struct {
	PluginID string       `json:"pluginId"`
	Areas    []Area       `json:"areas"`
	Files    []SourceFile `json:"files"`
}

// DeleteRequest names the operation and the explicitly selected areas.
type DeleteRequest struct {
	Action               string   `json:"action"`
	SelectedAreas        []string `json:"selectedAreas,omitempty"`
	ConfirmShared        bool     `json:"confirmShared,omitempty"`
	ConfirmReferenced    bool     `json:"confirmReferenced,omitempty"`
	DeleteAllUnconfirmed bool     `json:"deleteAllUnconfirmed,omitempty"`
}

// RetainedArea records an area the plan does not delete and why.
type RetainedArea struct {
	Key    string `json:"key"`
	Reason string `json:"reason"`
}

// DeleteDiff is the removal summary. Files and areas are separate counts.
type DeleteDiff struct {
	Files        int `json:"files"`
	Areas        int `json:"areas"`
	RemovedBytes int `json:"removedBytes"`
	KeptBytes    int `json:"keptBytes"`
}

// DeletePlan is the read-only deletion proposal.
type DeletePlan struct {
	Schema             string         `json:"schema"`
	Action             string         `json:"action"`
	PluginID           string         `json:"pluginId"`
	Confirmed          []string       `json:"confirmed"`
	RequiresSelection  []string       `json:"requiresSelection,omitempty"`
	Selected           []string       `json:"selected,omitempty"`
	Retained           []RetainedArea `json:"retained,omitempty"`
	ReferenceImpact    []string       `json:"referenceImpact,omitempty"`
	Diff               DeleteDiff     `json:"diff"`
	OperationAllowed   bool           `json:"operationAllowed"`
	DataDeleted        bool           `json:"dataDeleted"`
	ManualEditAllowed  bool           `json:"manualEditAllowed"`
	Reasons            []string       `json:"reasons,omitempty"`
	FilesAfter         []SourceFile   `json:"filesAfter,omitempty"`
	SourceUnchanged    bool           `json:"sourceUnchanged"`
	ProtectedPreserved bool           `json:"protectedPreserved"`
}

func validateDataState(state DataState) error {
	if strings.TrimSpace(state.PluginID) == "" {
		return fmt.Errorf("pluginId is required")
	}
	seen := map[string]bool{}
	for i, area := range state.Areas {
		if strings.TrimSpace(area.Key) == "" {
			return fmt.Errorf("areas[%d].key is required", i)
		}
		if seen[area.Key] {
			return fmt.Errorf("area %q is declared twice", area.Key)
		}
		seen[area.Key] = true
		switch area.Mapping {
		case MappingConfirmed, MappingShared, MappingUnknown:
		default:
			return fmt.Errorf("area %q: mapping %q must be confirmed, shared, or unknown", area.Key, area.Mapping)
		}
	}
	for i, file := range state.Files {
		if strings.TrimSpace(file.Path) == "" {
			return fmt.Errorf("files[%d].path is required", i)
		}
		for j, segment := range file.Segments {
			switch segment.Kind {
			case SegmentOwned, SegmentShared, SegmentExtension, SegmentUnknownAttribute,
				SegmentUnknownElement, SegmentUnknown:
			default:
				return fmt.Errorf("files[%d].segments[%d]: unknown kind %q", i, j, segment.Kind)
			}
			if segment.Area != "" && !seen[segment.Area] {
				return fmt.Errorf("files[%d].segments[%d]: area %q is not declared", i, j, segment.Area)
			}
		}
	}
	return nil
}

func copyFiles(files []SourceFile) []SourceFile {
	out := make([]SourceFile, len(files))
	for i, file := range files {
		out[i] = file
		out[i].Segments = append([]Segment(nil), file.Segments...)
	}
	return out
}

func selectedSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		out[value] = true
	}
	return out
}

func retainedSort(values []RetainedArea) []RetainedArea {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Key != values[j].Key {
			return values[i].Key < values[j].Key
		}
		return values[i].Reason < values[j].Reason
	})
	return values
}

// PlanDeletion separates "remove the plugin" from "delete the related data".
// Plugin removal never deletes data. Data deletion touches only confirmed or
// explicitly confirmed shared areas, keeps unknown areas, and refuses rather
// than partially rewriting a source when a selected area contains bytes it
// cannot preserve.
func PlanDeletion(state DataState, req DeleteRequest) (DeletePlan, error) {
	if err := validateDataState(state); err != nil {
		return DeletePlan{}, err
	}
	plan := DeletePlan{
		Schema: Schema, Action: req.Action, PluginID: state.PluginID,
		Selected: sortedUnique(req.SelectedAreas), ManualEditAllowed: true,
	}
	switch req.Action {
	case ActionRemovePlugin:
		plan.OperationAllowed = true
		plan.DataDeleted = false
		plan.FilesAfter = copyFiles(state.Files)
		plan.SourceUnchanged = true
		plan.ProtectedPreserved = true
		for _, area := range state.Areas {
			plan.Retained = append(plan.Retained, RetainedArea{Key: area.Key, Reason: "plugin-removal-keeps-data"})
		}
		plan.Retained = retainedSort(plan.Retained)
		return plan, nil
	case ActionDeleteData:
	default:
		return DeletePlan{}, fmt.Errorf("action %q must be remove-plugin or delete-data", req.Action)
	}

	if req.DeleteAllUnconfirmed {
		plan.Reasons = append(plan.Reasons, "unknown-area-not-guessable")
	}
	selected := selectedSet(req.SelectedAreas)
	deletable := map[string]bool{}
	for _, area := range state.Areas {
		switch area.Mapping {
		case MappingConfirmed:
			plan.Confirmed = append(plan.Confirmed, area.Key)
			if !selected[area.Key] {
				plan.Retained = append(plan.Retained, RetainedArea{Key: area.Key, Reason: "not-selected"})
				continue
			}
			if len(area.Owners) > 0 && !contains(area.Owners, state.PluginID) {
				plan.Retained = append(plan.Retained, RetainedArea{Key: area.Key, Reason: "not-owned-by-plugin"})
				continue
			}
			if len(area.ReferencedBy) > 0 && !req.ConfirmReferenced {
				plan.Retained = append(plan.Retained, RetainedArea{Key: area.Key, Reason: "referenced-by-others:" + strings.Join(sortedUnique(area.ReferencedBy), ",")})
				continue
			}
			deletable[area.Key] = true
		case MappingShared:
			plan.RequiresSelection = append(plan.RequiresSelection, area.Key)
			if !selected[area.Key] {
				plan.Retained = append(plan.Retained, RetainedArea{Key: area.Key, Reason: "shared-not-selected"})
				continue
			}
			if !req.ConfirmShared {
				plan.Retained = append(plan.Retained, RetainedArea{Key: area.Key, Reason: "shared-requires-explicit-selection"})
				continue
			}
			if len(area.ReferencedBy) > 0 && !req.ConfirmReferenced {
				plan.Retained = append(plan.Retained, RetainedArea{Key: area.Key, Reason: "referenced-by-others:" + strings.Join(sortedUnique(area.ReferencedBy), ",")})
				continue
			}
			deletable[area.Key] = true
		case MappingUnknown:
			plan.Retained = append(plan.Retained, RetainedArea{Key: area.Key, Reason: "unknown-provider"})
			if selected[area.Key] {
				plan.Reasons = append(plan.Reasons, "unknown-area-not-deletable:"+area.Key)
			}
		}
		if selected[area.Key] && len(area.ReferencedBy) > 0 {
			plan.ReferenceImpact = append(plan.ReferenceImpact,
				"area:"+area.Key+":referenced-by:"+strings.Join(sortedUnique(area.ReferencedBy), ","))
		}
	}
	plan.Confirmed = sortedUnique(plan.Confirmed)
	plan.RequiresSelection = sortedUnique(plan.RequiresSelection)
	plan.ReferenceImpact = sortedUnique(plan.ReferenceImpact)
	if req.DeleteAllUnconfirmed {
		plan.OperationAllowed = false
		plan.SourceUnchanged = true
		plan.Retained = retainedSort(plan.Retained)
		return plan, nil
	}

	if len(deletable) == 0 {
		plan.Reasons = append(plan.Reasons, "no-selected-confirmed-area")
		plan.Retained = retainedSort(plan.Retained)
		return plan, nil
	}

	// Build the new bytes, refusing the whole operation if a selected area
	// contains a segment that cannot be preserved.
	after := make([]SourceFile, 0, len(state.Files))
	removedBytes, keptBytes, changedFiles := 0, 0, 0
	conflict := ""
	for _, file := range state.Files {
		kept := make([]Segment, 0, len(file.Segments))
		removedHere := 0
		for _, segment := range file.Segments {
			if segment.Area != "" && deletable[segment.Area] {
				switch segment.Kind {
				case SegmentOwned, SegmentShared:
					removedBytes += len(segment.Bytes)
					removedHere += len(segment.Bytes)
					continue
				default:
					conflict = "preservation-conflict:" + segment.Key
				}
			}
			kept = append(kept, segment)
			keptBytes += len(segment.Bytes)
		}
		if conflict != "" {
			break
		}
		if removedHere > 0 {
			changedFiles++
		}
		after = append(after, SourceFile{Path: file.Path, Segments: kept, Fingerprint: Fingerprint(segmentsText(kept))})
	}
	if conflict != "" {
		plan.Reasons = append(plan.Reasons, conflict)
		plan.SourceUnchanged = true
		plan.OperationAllowed = false
		plan.DataDeleted = false
		plan.Retained = retainedSort(plan.Retained)
		return plan, nil
	}

	plan.OperationAllowed = true
	plan.DataDeleted = true
	plan.FilesAfter = after
	plan.SourceUnchanged = false
	plan.ProtectedPreserved = true
	plan.Diff = DeleteDiff{Files: changedFiles, Areas: len(deletable), RemovedBytes: removedBytes, KeptBytes: keptBytes}
	plan.Retained = retainedSort(plan.Retained)
	plan.Reasons = sortedUnique(plan.Reasons)
	return plan, nil
}

func segmentsText(segments []Segment) string {
	var b strings.Builder
	for _, segment := range segments {
		b.WriteString(segment.Bytes)
	}
	return b.String()
}

// ConvertChange is one declared conversion. A change that cannot be preserved
// is listed separately and never silently dropped.
type ConvertChange struct {
	Area        string `json:"area"`
	Kind        string `json:"kind"`
	Before      string `json:"before,omitempty"`
	After       string `json:"after,omitempty"`
	Preservable bool   `json:"preservable"`
}

// ConvertRequest asks to change the format version and the data it carries.
type ConvertRequest struct {
	FromVersion  string          `json:"fromVersion"`
	ToVersion    string          `json:"toVersion"`
	Convert      bool            `json:"convert"`
	AllowPartial bool            `json:"allowPartial,omitempty"`
	Changes      []ConvertChange `json:"changes,omitempty"`
}

// ConvertPlan is the read-only conversion proposal.
type ConvertPlan struct {
	Schema           string          `json:"schema"`
	FromVersion      string          `json:"fromVersion"`
	ToVersion        string          `json:"toVersion"`
	Allowed          bool            `json:"allowed"`
	Converted        bool            `json:"converted"`
	Partial          bool            `json:"partial"`
	Changes          []ConvertChange `json:"changes,omitempty"`
	Unpreservable    []string        `json:"unpreservable,omitempty"`
	Reasons          []string        `json:"reasons,omitempty"`
	FilesAfter       []SourceFile    `json:"filesAfter,omitempty"`
	SourceUnchanged  bool            `json:"sourceUnchanged"`
	PreservedUnknown bool            `json:"preservedUnknown"`
}

// PlanConversion refuses a version-only update and requires explicit
// conversion rules. Unknown and extension bytes are kept.
func PlanConversion(state DataState, req ConvertRequest) (ConvertPlan, error) {
	if err := validateDataState(state); err != nil {
		return ConvertPlan{}, err
	}
	if strings.TrimSpace(req.FromVersion) == "" || strings.TrimSpace(req.ToVersion) == "" {
		return ConvertPlan{}, fmt.Errorf("fromVersion and toVersion are required")
	}
	plan := ConvertPlan{Schema: Schema, FromVersion: req.FromVersion, ToVersion: req.ToVersion,
		Changes: append([]ConvertChange(nil), req.Changes...)}
	if req.FromVersion == req.ToVersion {
		plan.Reasons = append(plan.Reasons, "no-change")
		plan.SourceUnchanged = true
		return finishConvert(plan), nil
	}
	if !req.Convert {
		plan.Reasons = append(plan.Reasons, "conversion-required")
		plan.SourceUnchanged = true
		for _, change := range req.Changes {
			if !change.Preservable {
				plan.Unpreservable = append(plan.Unpreservable, change.Area+":"+change.Kind)
			}
		}
		return finishConvert(plan), nil
	}
	if len(req.Changes) == 0 {
		plan.Reasons = append(plan.Reasons, "no-conversion-rules")
		plan.SourceUnchanged = true
		return finishConvert(plan), nil
	}
	for _, change := range req.Changes {
		if !change.Preservable {
			plan.Unpreservable = append(plan.Unpreservable, change.Area+":"+change.Kind)
		}
	}
	plan.Unpreservable = sortedUnique(plan.Unpreservable)
	if len(plan.Unpreservable) > 0 && !req.AllowPartial {
		plan.Reasons = append(plan.Reasons, "unpreservable-change")
		plan.SourceUnchanged = true
		return finishConvert(plan), nil
	}

	byArea := map[string]ConvertChange{}
	for _, change := range req.Changes {
		if change.Preservable {
			byArea[change.Area] = change
		}
	}
	after := make([]SourceFile, 0, len(state.Files))
	preservedUnknown := true
	for _, file := range state.Files {
		segments := make([]Segment, 0, len(file.Segments))
		for _, original := range file.Segments {
			segment := original
			change, ok := byArea[segment.Area]
			if ok && (segment.Kind == SegmentOwned || segment.Kind == SegmentShared) {
				segment.Bytes = change.After
			}
			if segment.Kind != SegmentOwned && segment.Kind != SegmentShared && segment.Bytes != original.Bytes {
				preservedUnknown = false
			}
			segments = append(segments, segment)
		}
		after = append(after, SourceFile{Path: file.Path, Segments: segments, Fingerprint: Fingerprint(segmentsText(segments))})
	}
	plan.Allowed = true
	plan.Converted = true
	plan.Partial = len(plan.Unpreservable) > 0
	plan.FilesAfter = after
	plan.SourceUnchanged = false
	plan.PreservedUnknown = preservedUnknown
	return finishConvert(plan), nil
}

func finishConvert(plan ConvertPlan) ConvertPlan {
	plan.Reasons = sortedUnique(plan.Reasons)
	plan.Unpreservable = sortedUnique(plan.Unpreservable)
	sort.Slice(plan.Changes, func(i, j int) bool {
		if plan.Changes[i].Area != plan.Changes[j].Area {
			return plan.Changes[i].Area < plan.Changes[j].Area
		}
		return plan.Changes[i].Kind < plan.Changes[j].Kind
	})
	return plan
}
