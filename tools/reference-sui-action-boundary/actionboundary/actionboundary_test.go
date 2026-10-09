package actionboundary

import (
	"context"
	"errors"
	"testing"
)

type permit bool

func (p permit) Authorize(context.Context, string, string, string) error {
	if !p {
		return errors.New("forbidden")
	}
	return nil
}

type memoryStore struct {
	revision string
	writes   int
	payload  any
}

func (s *memoryStore) Commit(_ context.Context, in Intent) (string, error) {
	if in.ExpectedRevision != s.revision {
		return "", ErrConflict
	}
	if in.Payload == nil {
		return "", ErrInvalid
	}
	s.payload = in.Payload
	s.writes++
	s.revision = "r2"
	return s.revision, nil
}
func TestCommitBoundary(t *testing.T) {
	ctx := context.Background()
	mk := func() Intent {
		return Intent{ApplicationID: "sui", DocumentID: "doc", ActionID: "move", ExpectedRevision: "r1", Payload: map[string]int{"x": 2}}
	}
	t.Run("commit once", func(t *testing.T) {
		s := &memoryStore{revision: "r1"}
		h := Host{Authorizer: permit(true), Bindings: []Binding{{"sui", "move", s}}}
		rev, err := h.Commit(ctx, mk())
		if err != nil || rev != "r2" || s.writes != 1 {
			t.Fatalf("rev=%s err=%v writes=%d", rev, err, s.writes)
		}
	})
	t.Run("stale revision rejected", func(t *testing.T) {
		s := &memoryStore{revision: "r3"}
		h := Host{Authorizer: permit(true), Bindings: []Binding{{"sui", "move", s}}}
		_, err := h.Commit(ctx, mk())
		if !errors.Is(err, ErrConflict) || s.writes != 0 {
			t.Fatalf("err=%v writes=%d", err, s.writes)
		}
	})
	t.Run("denied before write", func(t *testing.T) {
		s := &memoryStore{revision: "r1"}
		h := Host{Authorizer: permit(false), Bindings: []Binding{{"sui", "move", s}}}
		_, err := h.Commit(ctx, mk())
		if !errors.Is(err, ErrDenied) || s.writes != 0 {
			t.Fatalf("err=%v writes=%d", err, s.writes)
		}
	})
	t.Run("cross application binding rejected", func(t *testing.T) {
		s := &memoryStore{revision: "r1"}
		h := Host{Authorizer: permit(true), Bindings: []Binding{{"other", "move", s}}}
		_, err := h.Commit(ctx, mk())
		if !errors.Is(err, ErrDenied) || s.writes != 0 {
			t.Fatalf("err=%v writes=%d", err, s.writes)
		}
	})
	t.Run("missing authorization fails closed", func(t *testing.T) {
		s := &memoryStore{revision: "r1"}
		h := Host{Bindings: []Binding{{"sui", "move", s}}}
		_, err := h.Commit(ctx, mk())
		if !errors.Is(err, ErrDenied) || s.writes != 0 {
			t.Fatalf("err=%v writes=%d", err, s.writes)
		}
	})
	t.Run("second application uses same host", func(t *testing.T) {
		s := &memoryStore{revision: "r1"}
		h := Host{Authorizer: permit(true), Bindings: []Binding{{"other", "resize", s}}}
		in := mk()
		in.ApplicationID = "other"
		in.ActionID = "resize"
		_, err := h.Commit(ctx, in)
		if err != nil || s.writes != 1 {
			t.Fatalf("err=%v writes=%d", err, s.writes)
		}
	})
}
