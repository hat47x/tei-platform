package actionboundary

import (
	"context"
	"errors"
	"testing"
)

type rejectingStore struct {
	calls int
	err   error
}

func (s *rejectingStore) Commit(context.Context, Intent) (string, error) {
	s.calls++
	return "", s.err
}

func TestCommitNegativeMatrix(t *testing.T) {
	ctx := context.Background()
	valid := Intent{ApplicationID: "sui", DocumentID: "doc", ActionID: "move", ExpectedRevision: "r1", Payload: "opaque"}
	cases := []struct {
		name   string
		mutate func(*Intent)
	}{
		{"missing application", func(i *Intent) { i.ApplicationID = "" }},
		{"missing document", func(i *Intent) { i.DocumentID = "" }},
		{"missing action", func(i *Intent) { i.ActionID = "" }},
		{"missing expected revision", func(i *Intent) { i.ExpectedRevision = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &rejectingStore{}
			h := Host{Authorizer: permit(true), Bindings: []Binding{{ApplicationID: "sui", ActionID: "move", Store: s}}}
			in := valid
			tc.mutate(&in)
			_, err := h.Commit(ctx, in)
			if !errors.Is(err, ErrInvalid) || s.calls != 0 {
				t.Fatalf("err=%v calls=%d", err, s.calls)
			}
		})
	}
	t.Run("duplicate bindings fail closed", func(t *testing.T) {
		s := &rejectingStore{}
		h := Host{Authorizer: permit(true), Bindings: []Binding{
			{ApplicationID: "sui", ActionID: "move", Store: s},
			{ApplicationID: "sui", ActionID: "move", Store: s},
		}}
		_, err := h.Commit(ctx, valid)
		if !errors.Is(err, ErrInvalid) || s.calls != 0 {
			t.Fatalf("err=%v calls=%d", err, s.calls)
		}
	})
	t.Run("nil store fails closed", func(t *testing.T) {
		h := Host{Authorizer: permit(true), Bindings: []Binding{{ApplicationID: "sui", ActionID: "move"}}}
		_, err := h.Commit(ctx, valid)
		if !errors.Is(err, ErrDenied) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("application owned validation error propagates", func(t *testing.T) {
		s := &rejectingStore{err: ErrInvalid}
		h := Host{Authorizer: permit(true), Bindings: []Binding{{ApplicationID: "sui", ActionID: "move", Store: s}}}
		_, err := h.Commit(ctx, valid)
		if !errors.Is(err, ErrInvalid) || s.calls != 1 {
			t.Fatalf("err=%v calls=%d", err, s.calls)
		}
	})
	t.Run("application owned conflict propagates", func(t *testing.T) {
		s := &rejectingStore{err: ErrConflict}
		h := Host{Authorizer: permit(true), Bindings: []Binding{{ApplicationID: "sui", ActionID: "move", Store: s}}}
		_, err := h.Commit(ctx, valid)
		if !errors.Is(err, ErrConflict) || s.calls != 1 {
			t.Fatalf("err=%v calls=%d", err, s.calls)
		}
	})
}
