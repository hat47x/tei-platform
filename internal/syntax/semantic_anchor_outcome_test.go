package syntax

import "testing"

func TestParseSemanticFileAnchoredAcceptsCapabilityOutcomes(t *testing.T) {
	src := `namespace security

capability security.authenticate {
    input login_id std.identifier
    input credential security.credential
    outcome authenticated security.authenticated_subject
    outcome denied security.authentication_denial
}
`

	parsed, err := ParseSemanticFileAnchored(src)
	if err != nil {
		t.Fatalf("名前付きOutcomeを持つCapabilityのSource Anchorを構築できません: %v", err)
	}
	if len(parsed.Value.Capabilities) != 1 {
		t.Fatalf("capabilities=%d want=1", len(parsed.Value.Capabilities))
	}
	if got := len(parsed.Value.Capabilities[0].Outcomes); got != 2 {
		t.Fatalf("outcomes=%d want=2", got)
	}

	for _, anchor := range parsed.Anchors {
		if anchor.Kind == "semantic-capability" && anchor.ID == "security.authenticate" {
			return
		}
	}
	t.Fatal("security.authenticateのSource Anchorが見つかりません")
}
