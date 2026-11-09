package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
)

type changeRow struct {
	Category string `json:"category"`
	ID       string `json:"id"`
	Action   string `json:"action"`
	FromJSON string `json:"fromJSON,omitempty"`
	ToJSON   string `json:"toJSON,omitempty"`
}
type reviewPreview struct {
	SchemaVersion         string       `json:"schemaVersion"`
	Authority             string       `json:"authority"`
	ProcedureID           string       `json:"procedureId"`
	BaseReviewSHA256      string       `json:"baseReviewSHA256"`
	CandidateReviewSHA256 string       `json:"candidateReviewSHA256"`
	ProcedureSourceSHA256 string       `json:"procedureSourceSHA256"`
	HandoffSourceSHA256   string       `json:"handoffSourceSHA256"`
	WorldRevisionStatus   string       `json:"worldRevisionStatus"`
	AcceptanceStatus      string       `json:"acceptanceStatus"`
	SourceWrites          []string     `json:"sourceWrites"`
	Changes               []changeRow  `json:"changes"`
	Diagnostics           []diagnostic `json:"diagnostics"`
	ReadyForAcceptance    bool         `json:"readyForAcceptance"`
}

// The input is two unaccepted review hypotheses against exactly the same
// authoring fixture. This is not #1124's Shared Acceptance Preview.
// It must not modify the Procedure fixture or claim a write location.
func previewReview(base, next review, previous, proposed projection, baseSnapshots, nextSnapshots []navigationSnapshot) (reviewPreview, error) {
	if len(baseSnapshots) != 3 || len(nextSnapshots) != 3 {
		return reviewPreview{}, fmt.Errorf("three source snapshots required for each DFD review")
	}
	if baseSnapshots[0] != nextSnapshots[0] || baseSnapshots[2] != nextSnapshots[2] {
		return reviewPreview{}, fmt.Errorf("DFD_PROCEDURE_OR_HANDOFF_CHANGED_BETWEEN_REVIEWS")
	}
	if base.Procedure != next.Procedure || previous.ProcedureID != proposed.ProcedureID ||
		base.Procedure != previous.ProcedureID {
		return reviewPreview{}, fmt.Errorf("DFD_PROCEDURE_IDENTITY_CHANGED")
	}
	if base.Authority != "proposal" || next.Authority != "proposal" ||
		base.SchemaVersion != "business-dfd-review-v0" || next.SchemaVersion != "business-dfd-review-v0" {
		return reviewPreview{}, fmt.Errorf("DFD_REVIEW_AUTHORITY_OR_SCHEMA_INVALID")
	}
	out := reviewPreview{
		SchemaVersion:         "business-dfd-review-preview-v0",
		Authority:             "derived-read-only",
		ProcedureID:           base.Procedure,
		BaseReviewSHA256:      baseSnapshots[1].SHA256,
		CandidateReviewSHA256: nextSnapshots[1].SHA256,
		ProcedureSourceSHA256: baseSnapshots[0].SHA256,
		HandoffSourceSHA256:   baseSnapshots[2].SHA256,
		WorldRevisionStatus:   "unverified-no-model-revision-binding",
		AcceptanceStatus:      "not-accepted-not-a-write-candidate",
		SourceWrites:          []string{}, Changes: []changeRow{}, Diagnostics: []diagnostic{},
		ReadyForAcceptance: false,
	}
	out.Diagnostics = append(out.Diagnostics, previous.Diagnostics...)
	out.Diagnostics = append(out.Diagnostics, proposed.Diagnostics...)
	// Display-only changes in the proposal file are not accepted meaning.
	// The exact-ID keys below avoid label-based reconciliation.
	collect := func(category string, oldRows, newRows map[string]string) {
		keys := map[string]bool{}
		for id := range oldRows {
			keys[id] = true
		}
		for id := range newRows {
			keys[id] = true
		}
		for id := range keys {
			o, existsOld := oldRows[id]
			n, existsNew := newRows[id]
			if existsOld && existsNew && o == n {
				continue
			}
			change := changeRow{Category: category, ID: id}
			switch {
			case !existsOld:
				change.Action = "proposed-add"
			case !existsNew:
				change.Action = "proposed-remove"
			default:
				change.Action = "proposed-update"
			}
			if existsOld {
				change.FromJSON = o
			}
			if existsNew {
				change.ToJSON = n
			}
			out.Changes = append(out.Changes, change)
		}
	}
	serialize := func(v any) string {
		bytes, err := json.Marshal(v)
		if err != nil {
			panic(fmt.Sprintf("internal fixed DFD review serialization: %v", err))
		}
		return string(bytes)
	}
	triggerRows := func(rows []reviewTrigger) map[string]string {
		out := map[string]string{}
		for _, x := range rows {
			out[x.ID] = serialize(x)
		}
		return out
	}
	flowRows := func(rows []reviewFlow) map[string]string {
		out := map[string]string{}
		for _, x := range rows {
			// No logical flow ID exists in v0; edits are remove/add.
			id := fmt.Sprintf("flow-sha256:%x", sha256.Sum256([]byte(serialize(x))))
			out[id] = serialize(x)
		}
		return out
	}
	roleRows := func(rows []businessRole) map[string]string {
		out := map[string]string{}
		for _, x := range rows {
			out[x.ID] = serialize(x)
		}
		return out
	}
	partyRows := func(rows []externalParty) map[string]string {
		out := map[string]string{}
		for _, x := range rows {
			out[x.ID] = serialize(x)
		}
		return out
	}
	storeRows := func(rows []informationStore) map[string]string {
		out := map[string]string{}
		for _, x := range rows {
			out[x.ID] = serialize(x)
		}
		return out
	}
	responsibilityRows := func(rows []stepResponsibility) map[string]string {
		out := map[string]string{}
		for _, x := range rows {
			out[x.Step+"\x00"+x.Role] = serialize(x)
		}
		return out
	}
	infoRows := func(rows []informationFlow) map[string]string {
		out := map[string]string{}
		for _, x := range rows {
			out[x.ID] = serialize(x)
		}
		return out
	}
	collect("trigger", triggerRows(base.Triggers), triggerRows(next.Triggers))
	collect("control-transition", flowRows(base.Flows), flowRows(next.Flows))
	collect("business-role", roleRows(base.BusinessRoles), roleRows(next.BusinessRoles))
	collect("external-party", partyRows(base.ExternalParties), partyRows(next.ExternalParties))
	collect("conceptual-store", storeRows(base.Stores), storeRows(next.Stores))
	collect("responsibility", responsibilityRows(base.Responsibilities), responsibilityRows(next.Responsibilities))
	collect("information-flow", infoRows(base.InformationFlows), infoRows(next.InformationFlows))
	sort.Slice(out.Changes, func(i, j int) bool {
		a, b := out.Changes[i], out.Changes[j]
		if a.Category != b.Category {
			return a.Category < b.Category
		}
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		return a.Action < b.Action
	})
	// Reject success claims for diagnostics in either review version.
	for _, d := range out.Diagnostics {
		if d.Severity == "error" {
			out.AcceptanceStatus = "blocked-unresolved-diagnostic"
			break
		}
	}
	return out, nil
}
