package actionboundary

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/semantic"
)

type fakePatternResolver struct {
	resolution driver.PatternResolution
	err        error
}

func (f fakePatternResolver) ResolvePattern(_, _ core.CanonicalID) (driver.PatternResolution, error) {
	return f.resolution, f.err
}
func TestCanonicalCommandGate(t *testing.T) {
	intent := Intent{ApplicationID: "sui", DocumentID: "doc", ActionID: "sui.move", ExpectedRevision: "r1", Payload: "move"}
	command := driver.PatternResolution{
		Application: application.Application{ID: core.CanonicalID("sui")},
		Pattern:     application.Pattern{ID: core.CanonicalID("sui.move"), Kind: application.PatternCommand},
		Capability:  &semantic.Capability{ID: core.CanonicalID("sui.move_capability")},
	}
	cases := []struct {
		name       string
		resolution driver.PatternResolution
		err        error
		denied     bool
	}{
		{"declared command", command, nil, false},
		{"unknown command", driver.PatternResolution{}, errors.New("not found"), true},
		{"browse is not command", driver.PatternResolution{Application: command.Application, Pattern: application.Pattern{ID: command.Pattern.ID, Kind: application.PatternBrowse}, Capability: command.Capability}, nil, true},
		{"missing capability", driver.PatternResolution{Application: command.Application, Pattern: command.Pattern}, nil, true},
		{"other application", driver.PatternResolution{Application: application.Application{ID: core.CanonicalID("other")}, Pattern: command.Pattern, Capability: command.Capability}, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &rejectingStore{}
			h := Host{Authorizer: permit(true), CommandGate: &CanonicalCommandGate{Resolver: fakePatternResolver{resolution: tc.resolution, err: tc.err}}, Bindings: []Binding{{ApplicationID: "sui", ActionID: "sui.move", Store: s}}}
			_, err := h.Commit(context.Background(), intent)
			if tc.denied {
				if !errors.Is(err, ErrDenied) || s.calls != 0 {
					t.Fatalf("err=%v calls=%d", err, s.calls)
				}
			} else if err != nil || s.calls != 1 {
				t.Fatalf("err=%v calls=%d", err, s.calls)
			}
		})
	}
}
