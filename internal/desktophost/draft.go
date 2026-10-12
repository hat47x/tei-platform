package desktophost

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const MarkerVersion = "tei.desktop.draft-marker/v0"

var ErrInvalidDraftMarker = errors.New("invalid Desktop Draft marker")

// DraftMarker contains only references and a read-only observation. It does
// not contain candidate bytes, capabilities, authorization, or executable steps.
// Saving/retrieving this marker is the host's separate, protected storage task.
// It must not be placed into Canonical Source or treated as an authority token.
type DraftMarker struct {
	Version           string            `json:"version"`
	CandidateRef      string            `json:"candidateRef"`
	BaseModelRevision string            `json:"baseModelRevision"`
	Baseline          SourceObservation `json:"baseline"`
}

// RecoveryStatus never contains a state that authorizes automatic acceptance.
type RecoveryStatus string

const (
	RecoveryNeedsReview       RecoveryStatus = "needs-review"
	RecoverySourceChanged     RecoveryStatus = "source-changed"
	RecoverySourceUnavailable RecoveryStatus = "source-unavailable"
)

type Recovery struct {
	Marker DraftMarker
	Status RecoveryStatus
}

func validateMarker(m DraftMarker) error {
	if m.Version != MarkerVersion || strings.TrimSpace(m.CandidateRef) == "" || strings.TrimSpace(m.BaseModelRevision) == "" || m.Baseline.WorkspaceRoot == "" {
		return ErrInvalidDraftMarker
	}
	if err := CompareObservations(m.Baseline, m.Baseline); err != nil {
		return ErrInvalidDraftMarker
	}
	return nil
}

// EncodeDraftMarker only serializes metadata; callers must separately decide
// whether saving it is allowed and protect storage and later disclosure.
func EncodeDraftMarker(m DraftMarker) ([]byte, error) {
	if err := validateMarker(m); err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

// RecoverDraftMarker parses a local marker and *only* classifies a fresh
// observation. It performs no side effects, invokes no proposal acceptance,
// and never resumes an unknown external command.
func RecoverDraftMarker(raw []byte, authorizedWorkspaceRoot string, observe func(string) (SourceObservation, error)) (Recovery, error) {
	if len(raw) == 0 || len(raw) > 1<<20 || observe == nil || authorizedWorkspaceRoot == "" {
		return Recovery{}, ErrInvalidDraftMarker
	}
	var marker DraftMarker
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&marker); err != nil {
		return Recovery{}, fmt.Errorf("decode Desktop Draft marker: %w", ErrInvalidDraftMarker)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Recovery{}, ErrInvalidDraftMarker
	}
	if err := validateMarker(marker); err != nil {
		return Recovery{}, err
	}
	// The persisted path is untrusted metadata, not permission to open a Workspace.
	if marker.Baseline.WorkspaceRoot != authorizedWorkspaceRoot {
		return Recovery{}, ErrInvalidDraftMarker
	}
	current, err := observe(authorizedWorkspaceRoot)
	if err != nil {
		return Recovery{Marker: marker, Status: RecoverySourceUnavailable}, nil
	}
	if CompareObservations(marker.Baseline, current) != nil {
		return Recovery{Marker: marker, Status: RecoverySourceChanged}, nil
	}
	return Recovery{Marker: marker, Status: RecoveryNeedsReview}, nil
}
