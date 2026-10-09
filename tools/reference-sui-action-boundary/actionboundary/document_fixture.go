package actionboundary

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
)

// JSONDocumentStore is a reference-only, in-memory SUI-shaped document adapter.
// The authoritative SUI DocumentV1 implementation remains in the SUI project.
// In particular, this fixture does not implement native drag/drop island joining,
// undo/redo, validation, or durable persistence.
type JSONDocumentStore struct {
	mu         sync.Mutex
	revision   string
	baseRevision string
	documentID string
	document   []byte
	writes     int
}

// MovePayload carries the final absolute world coordinates after a native UI
// preview. It must not be emitted for intermediate pointer movements.
type MovePayload struct {
	CardID string  `json:"cardId"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

// NewScopedJSONDocumentStore binds the document identity for the lifetime of
// the fixture. An empty identity is rejected at commit time.
func NewScopedJSONDocumentStore(documentID, revision string, document []byte) *JSONDocumentStore {
	return &JSONDocumentStore{
		documentID: documentID,
		revision:   revision,
		baseRevision: revision,
		document:   append([]byte(nil), document...),
	}
}

// Snapshot returns a copy so callers cannot mutate the stored document.
func (s *JSONDocumentStore) Snapshot() (string, []byte, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.revision, append([]byte(nil), s.document...), s.writes
}

func (s *JSONDocumentStore) Commit(_ context.Context, in Intent) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.documentID == "" || in.DocumentID != s.documentID {
		return "", ErrDenied
	}
	if in.ExpectedRevision != s.revision {
		return "", ErrConflict
	}
	payload, ok := in.Payload.(MovePayload)
	if !ok || payload.CardID == "" || math.IsNaN(payload.X) || math.IsNaN(payload.Y) ||
		math.IsInf(payload.X, 0) || math.IsInf(payload.Y, 0) {
		return "", ErrInvalid
	}
	if in.ActionID != "sui.move" {
		return "", ErrInvalid
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(s.document, &doc); err != nil || doc == nil {
		return "", fmt.Errorf("%w: invalid document JSON: %v", ErrInvalid, err)
	}
	var documentID string
	if err := json.Unmarshal(doc["id"], &documentID); err != nil || documentID != s.documentID {
		return "", fmt.Errorf("%w: document body identity mismatch", ErrInvalid)
	}
	var version int
	if err := json.Unmarshal(doc["version"], &version); err != nil || version != 1 {
		return "", fmt.Errorf("%w: unsupported document version", ErrInvalid)
	}

	var cards []map[string]json.RawMessage
	if err := json.Unmarshal(doc["cards"], &cards); err != nil || cards == nil {
		return "", fmt.Errorf("%w: invalid cards: %v", ErrInvalid, err)
	}
	found := false
	seen := make(map[string]bool, len(cards))
	for i := range cards {
		if cards[i] == nil {
			return "", fmt.Errorf("%w: null card", ErrInvalid)
		}
		var id string
		if err := json.Unmarshal(cards[i]["id"], &id); err != nil || id == "" || seen[id] {
			return "", fmt.Errorf("%w: invalid or duplicate card identity", ErrInvalid)
		}
		seen[id] = true
		if id != payload.CardID {
			continue
		}
		found = true
		var currentX, currentY float64
		if string(cards[i]["x"]) == "null" { return "", fmt.Errorf("%w: null card x coordinate", ErrInvalid) }
		if err := json.Unmarshal(cards[i]["x"], &currentX); err != nil {
			return "", fmt.Errorf("%w: invalid card x coordinate", ErrInvalid)
		}
		if string(cards[i]["y"]) == "null" { return "", fmt.Errorf("%w: null card y coordinate", ErrInvalid) }
		if err := json.Unmarshal(cards[i]["y"], &currentY); err != nil {
			return "", fmt.Errorf("%w: invalid card y coordinate", ErrInvalid)
		}
		if math.IsNaN(currentX) || math.IsInf(currentX, 0) ||
			math.IsNaN(currentY) || math.IsInf(currentY, 0) {
			return "", ErrInvalid
		}
		x, err := json.Marshal(payload.X)
		if err != nil {
			return "", ErrInvalid
		}
		y, err := json.Marshal(payload.Y)
		if err != nil {
			return "", ErrInvalid
		}
		// SUI DocumentV1 stores Card.x and Card.y directly, not Card.position.
		cards[i]["x"] = x
		cards[i]["y"] = y
	}
	if !found {
		return "", fmt.Errorf("%w: unknown card", ErrInvalid)
	}

	rawCards, err := json.Marshal(cards)
	if err != nil {
		return "", ErrInvalid
	}
	doc["cards"] = rawCards
	next, err := json.Marshal(doc)
	if err != nil {
		return "", ErrInvalid
	}
	s.document = next
	s.writes++
	s.revision = fmt.Sprintf("%s:%d", s.baseRevision, s.writes)
	return s.revision, nil
}
