package operations

import (
	"encoding/json"
	"fmt"
	"strings"

	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// Inspection codes distinguish invalid input, a missing or ambiguous canonical
// declaration, and kinds that this initial read-only surface does not support.
const (
	InspectionInvalidSubject   = "INVALID_SUBJECT"
	InspectionUnsupportedKind = "UNSUPPORTED_SUBJECT_KIND"
	InspectionNotFound         = "SUBJECT_NOT_FOUND"
	InspectionAmbiguous        = "SUBJECT_AMBIGUOUS"
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
	Sources                SubjectSources      `json:"sources"`
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
	return SubjectInspection{
		SchemaVersion: "tei.operations.inspect/v0",
		Effect: "read-only",
		Subject: subject,
		Canonical: canonical,
		WorldFingerprint: worldFP,
		SourceIndexFingerprint: sourceFP,
		SourceStatus: status,
		Sources: sources,
	}, nil
}

func supportedInspectionKind(kind string) bool {
	switch kind {
	case "semantic-data", "semantic-relation", "application", "application-pattern", "query":
		return true
	default:
		return false
	}
}

// inspectCanonical works only with normalized IR objects; every candidate is
// selected by kind AND exact canonical ID. Duplicate declarations are rejected
// rather than resolving to whichever item happens to appear first.
func inspectCanonical(world ir.World, subject sourceindex.Subject) (json.RawMessage, error) {
	var matches []any
	switch subject.Kind {
	case "semantic-data":
		for _, namespace := range world.Semantics {
			for _, item := range namespace.Data {
				if string(item.ID) == subject.ID {
					matches = append(matches, item)
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
