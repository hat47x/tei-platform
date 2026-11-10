package operations

import (
	"encoding/json"
	"fmt"
	"strings"

	"tei/internal/database/release"
	"tei/internal/ir"
	"tei/internal/releaseplan"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// Inspection codes distinguish invalid input, a missing or ambiguous canonical
// declaration, and kinds that this initial read-only surface does not support.
const (
	InspectionInvalidSubject  = "INVALID_SUBJECT"
	InspectionUnsupportedKind = "UNSUPPORTED_SUBJECT_KIND"
	InspectionNotFound        = "SUBJECT_NOT_FOUND"
	InspectionAmbiguous       = "SUBJECT_AMBIGUOUS"
)

type InspectionError struct {
	Code    string
	Message string
}

func (e *InspectionError) Error() string { return e.Message }

// SubjectInspection is a transport-neutral projection of one exact Canonical
// Subject from the normalized World, with Source Index evidence. Its World
// fingerprint is descriptive, not a revision token or an acceptance grant.
type SubjectInspection struct {
	SchemaVersion          string              `json:"schemaVersion"`
	Effect                 string              `json:"effect"`
	Subject                sourceindex.Subject `json:"subject"`
	Canonical              json.RawMessage     `json:"canonical"`
	WorldFingerprint       string              `json:"worldFingerprint"`
	SourceIndexFingerprint string              `json:"sourceIndexFingerprint"`
	SourceStatus           string              `json:"sourceStatus"`
	// Workspace names the Workspace this inspection was read from (#3181). The
	// same Canonical ID may exist in several Workspaces with different meaning,
	// and worldFingerprint identifies content, not the Workspace.
	Workspace *workspace.WorkspaceIdentity `json:"workspace,omitempty"`
	Sources   SubjectSources               `json:"sources"`
	// AuthorityLimits states what this answer does not grant. The inspection
	// carries Source paths and reference counts, so a reader must not read the
	// presence of a location as editability, acceptance, or an exposure
	// permission (#1178). trace and catalog carry the same kind of statement.
	AuthorityLimits []string `json:"authorityLimits"`
}

// InspectionAuthorityLimits is the fixed set of statements this read-only answer
// makes about its own authority. It is a constant list, not a per-caller
// decision: naming a Source location never grants write or acceptance authority,
// and a local read is not permission to disclose the answer to another party
// (#1178). An Authorization decision that narrows disclosure is a separate
// question and is not claimed here.
func InspectionAuthorityLimits() []string {
	return []string{
		"inspection-is-a-derived-read-only-projection",
		"source-location-is-not-semantic-acceptance",
		"source-location-is-not-write-authority",
		"local-read-is-not-disclosure-permission",
	}
}

// InspectSubject reads one Workspace, then projects its already-normalized World
// and Source Index. It must not infer a subject from a source path or synthesize
// canonical meaning from a Source Index reference.
func InspectSubject(root string, subject sourceindex.Subject) (SubjectInspection, error) {
	if subject.Kind == "" || subject.ID == "" ||
		subject.Kind != strings.TrimSpace(subject.Kind) ||
		subject.ID != strings.TrimSpace(subject.ID) {
		return SubjectInspection{}, &InspectionError{
			Code: InspectionInvalidSubject, Message: "subject kind and id must be nonempty exact identities",
		}
	}
	if !supportedInspectionKind(subject.Kind) {
		return SubjectInspection{}, &InspectionError{
			Code: InspectionUnsupportedKind, Message: fmt.Sprintf("inspect does not support subject kind %q", subject.Kind),
		}
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return SubjectInspection{}, &ValidationFailure{Stage: StageLoad, Err: err}
	}
	// Resolve the Canonical Subject before deriving provenance. A source
	// reference alone is never proof of a canonical declaration.
	canonical, err := inspectCanonical(loaded.World, subject)
	if err != nil {
		return SubjectInspection{}, err
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return SubjectInspection{}, &ValidationFailure{Stage: StageSourceIndex, Err: err}
	}
	sources, err := TraceSubjectSourcesFromIndex(idx, subject)
	if err != nil {
		return SubjectInspection{}, err
	}
	worldFP, sourceFP, err := workspaceProjectionFingerprints(loaded.World, idx)
	if err != nil {
		return SubjectInspection{}, err
	}
	status := "not-indexed"
	if len(sources.References) > 0 {
		status = "indexed"
	}
	identity := loaded.Identity()
	return SubjectInspection{
		SchemaVersion:          "tei.operations.inspect/v0",
		Effect:                 "read-only",
		Subject:                subject,
		Canonical:              canonical,
		WorldFingerprint:       worldFP,
		SourceIndexFingerprint: sourceFP,
		SourceStatus:           status,
		Workspace:              &identity,
		Sources:                sources,
		AuthorityLimits:        InspectionAuthorityLimits(),
	}, nil
}

// InspectionSubjectKinds is the closed set of subject kinds this read-only
// surface resolves. It is exported so the CLI can advertise the same values it
// accepts, instead of leaving callers to guess or read prose (#3242).
var InspectionSubjectKinds = []string{
	"semantic", "semantic-type", "semantic-data", "semantic-attribute", "semantic-relation",
	"semantic-candidate-key", "semantic-functional-dependency", "semantic-capability", "semantic-validation",
	"application", "application-pattern", "query",
	// The delivery layer. Without these a designer could validate a migration or
	// a release plan and then have no way to read it back (#4128). A release
	// subject is its owner (the database or the plan), and the projection lists
	// every declared version of that owner.
	"migration", "database-release", "solution-release",
}

func supportedInspectionKind(kind string) bool {
	for _, supported := range InspectionSubjectKinds {
		if kind == supported {
			return true
		}
	}
	return false
}

// releaseSet is the Canonical object for a database-release subject. Its
// top-level id is the database, matching the requested id, so the projection
// check in ValidateInspectionProjection still holds.
type releaseSet struct {
	ID       string               `json:"id"`
	Releases []release.Definition `json:"releases"`
}

// solutionReleaseSet is the Canonical object for a solution-release subject.
type solutionReleaseSet struct {
	ID       string                   `json:"id"`
	Releases []releaseplan.Definition `json:"releases"`
}

// inspectCanonical works only with normalized IR objects; every candidate is
// selected by kind AND exact canonical ID. Duplicate declarations are rejected
// rather than resolving to whichever item happens to appear first.
func inspectCanonical(world ir.World, subject sourceindex.Subject) (json.RawMessage, error) {
	var matches []any
	switch subject.Kind {
	case "semantic":
		for _, item := range world.Semantics {
			if string(item.ID) == subject.ID {
				matches = append(matches, item)
			}
		}
	case "semantic-type":
		for _, namespace := range world.Semantics {
			for _, item := range namespace.Types {
				if string(item.ID) == subject.ID {
					matches = append(matches, item)
				}
			}
		}
	case "semantic-capability":
		for _, namespace := range world.Semantics {
			for _, item := range namespace.Capabilities {
				if string(item.ID) == subject.ID {
					matches = append(matches, item)
				}
			}
		}
	case "semantic-validation":
		for _, namespace := range world.Semantics {
			for _, item := range namespace.Validations {
				if string(item.ID) == subject.ID {
					matches = append(matches, item)
				}
			}
		}
	case "semantic-candidate-key":
		for _, namespace := range world.Semantics {
			for _, data := range namespace.Data {
				for _, item := range data.CandidateKeys {
					if string(item.ID) == subject.ID {
						matches = append(matches, item)
					}
				}
			}
		}
	case "semantic-functional-dependency":
		for _, namespace := range world.Semantics {
			for _, data := range namespace.Data {
				for _, item := range data.FunctionalDependencies {
					if string(item.ID) == subject.ID {
						matches = append(matches, item)
					}
				}
			}
		}

	case "semantic-data":
		for _, namespace := range world.Semantics {
			for _, item := range namespace.Data {
				if string(item.ID) == subject.ID {
					matches = append(matches, item)
				}
			}
		}
	case "semantic-attribute":
		for _, namespace := range world.Semantics {
			for _, data := range namespace.Data {
				for _, item := range data.Attributes {
					if string(item.ID) == subject.ID {
						matches = append(matches, item)
					}
				}
			}
		}
	case "semantic-relation":
		for _, namespace := range world.Semantics {
			for _, data := range namespace.Data {
				for _, item := range data.Relations {
					if string(item.ID) == subject.ID {
						matches = append(matches, item)
					}
				}
			}
		}
	case "application":
		for _, item := range world.Applications {
			if string(item.ID) == subject.ID {
				matches = append(matches, item)
			}
		}
	case "application-pattern":
		for _, app := range world.Applications {
			for _, item := range app.Patterns {
				if string(item.ID) == subject.ID {
					matches = append(matches, item)
				}
			}
		}
	case "migration":
		for _, item := range world.Migrations {
			if item.ID == subject.ID {
				matches = append(matches, item)
			}
		}
	case "database-release":
		// A database holds several versions, so the subject is the owner and the
		// projection is the set of its releases. Returning one release per
		// declaration would be ambiguous, and inventing a compound id such as
		// "sales_core@4.2.0" would break the rule that the Canonical object's own
		// top-level id equals the requested id.
		var versions []release.Definition
		for _, item := range world.DatabaseReleases {
			if string(item.Database) == subject.ID {
				versions = append(versions, item)
			}
		}
		if len(versions) > 0 {
			matches = append(matches, releaseSet{ID: subject.ID, Releases: versions})
		}
	case "solution-release":
		var versions []releaseplan.Definition
		for _, item := range world.SolutionReleases {
			if string(item.ID) == subject.ID {
				versions = append(versions, item)
			}
		}
		if len(versions) > 0 {
			matches = append(matches, solutionReleaseSet{ID: subject.ID, Releases: versions})
		}
	case "query":
		for _, item := range world.Queries {
			if string(item.ID) == subject.ID {
				matches = append(matches, item)
			}
		}
	default:
		return nil, &InspectionError{
			Code: InspectionUnsupportedKind, Message: fmt.Sprintf("inspect does not support subject kind %q", subject.Kind),
		}
	}
	if len(matches) == 0 {
		return nil, &InspectionError{
			Code: InspectionNotFound, Message: fmt.Sprintf("canonical subject %s:%s not found", subject.Kind, subject.ID),
		}
	}
	if len(matches) > 1 {
		return nil, &InspectionError{
			Code: InspectionAmbiguous, Message: fmt.Sprintf("canonical subject %s:%s is ambiguous (%d declarations)", subject.Kind, subject.ID, len(matches)),
		}
	}
	raw, err := json.Marshal(matches[0])
	if err != nil {
		return nil, fmt.Errorf("serialize canonical subject: %w", err)
	}
	return raw, nil
}
