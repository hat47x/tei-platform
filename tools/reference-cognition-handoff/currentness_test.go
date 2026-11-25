package cognitionhandoff

import (
	"path/filepath"
	"testing"
)

// The handoff must carry the proposal's binding into the change candidate, so a
// consumer can tell which revision and which judgement basis produced it.
// Without it, currentness cannot be established at all — reporting such a
// candidate as current would be the failure #5808 and #5809 both forbid.
func TestHandoffCarriesTheProposalBinding(t *testing.T) {
	binding := &ProposalBinding{Subject: "sales.order_line.quantity", Revision: "rev-7", Basis: "sha256:abc"}
	candidate, err := HandoffWithBinding(
		proposalPath(t), filepath.Join(repoRoot(t), "reference", "workspace"),
		"local-rules-a", "deterministic-rules", "1", "", "", binding,
	)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Binding == nil || candidate.Binding.Revision != "rev-7" || candidate.Binding.Basis != "sha256:abc" {
		t.Fatalf("binding was not carried through: %+v", candidate.Binding)
	}

	// The legacy call keeps working and states no binding: existing fixtures
	// keep producing their candidate, but never claim currentness.
	legacy, err := Handoff(proposalPath(t), filepath.Join(repoRoot(t), "reference", "workspace"), "local-rules-a", "deterministic-rules", "1", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Binding != nil {
		t.Fatalf("a handoff without a binding must not invent one: %+v", legacy.Binding)
	}
}

func TestRequireCurrentAtHandoffDistinguishesTheReasons(t *testing.T) {
	candidate := Candidate{
		ID:      "sales.candidate_obs_1",
		Binding: &ProposalBinding{Subject: "sales.order_line.quantity", Revision: "rev-1", Basis: "basis-a"},
	}

	t.Run("same binding is current but not acceptable", func(t *testing.T) {
		result := RequireCurrentAtHandoff(candidate, ProposalBinding{Subject: "sales.order_line.quantity", Revision: "rev-1", Basis: "basis-a"})
		if result.State != Current {
			t.Fatalf("state = %s, want %s", result.State, Current)
		}
		if result.Acceptable || result.GrantsAuthority {
			t.Fatal("currentness must not be reported as acceptance or authority")
		}
	})

	t.Run("moved revision is stale", func(t *testing.T) {
		result := RequireCurrentAtHandoff(candidate, ProposalBinding{Subject: "sales.order_line.quantity", Revision: "rev-2", Basis: "basis-a"})
		if result.State != StaleRevision {
			t.Fatalf("state = %s, want %s", result.State, StaleRevision)
		}
	})

	t.Run("revised basis at the same revision is stale", func(t *testing.T) {
		result := RequireCurrentAtHandoff(candidate, ProposalBinding{Subject: "sales.order_line.quantity", Revision: "rev-1", Basis: "basis-b"})
		if result.State != StaleBasis {
			t.Fatalf("state = %s, want %s", result.State, StaleBasis)
		}
	})

	t.Run("another subject is refused before it is called stale", func(t *testing.T) {
		result := RequireCurrentAtHandoff(candidate, ProposalBinding{Subject: "sales.order_line.price", Revision: "rev-1", Basis: "basis-a"})
		if result.State != ForeignSubject {
			t.Fatalf("state = %s, want %s", result.State, ForeignSubject)
		}
	})

	t.Run("an unbound candidate is never current", func(t *testing.T) {
		unbound := candidate
		unbound.Binding = nil
		result := RequireCurrentAtHandoff(unbound, ProposalBinding{Subject: "sales.order_line.quantity", Revision: "rev-1", Basis: "basis-a"})
		if result.State != Unbound {
			t.Fatalf("state = %s, want %s", result.State, Unbound)
		}
		if result.Acceptable {
			t.Fatal("an unbound candidate must not be marked acceptable")
		}
	})
}

// A candidate is only current for the subject the caller asks about, even when
// the binding's other fields happen to coincide.
func TestRequireCurrentAtHandoffRequiresASubjectOnBothSides(t *testing.T) {
	bound := Candidate{ID: "c", Binding: &ProposalBinding{Subject: "a", Revision: "r", Basis: "b"}}
	if result := RequireCurrentAtHandoff(bound, ProposalBinding{Subject: "", Revision: "r", Basis: "b"}); result.State != ForeignSubject {
		t.Fatalf("an empty current subject must not match: %s", result.State)
	}
}
