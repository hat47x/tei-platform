package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestSemanticAnchorParserAcceptsCapabilityOutcomes(t *testing.T) {
	src := `namespace security

    data AuthenticatedSubject {}
    data AuthenticationDenial {}

    capability security.authenticate {
      outcome denied security.authentication_denial
      outcome authenticated security.authenticated_subject
    }`

	anchored, err := syntax.ParseSemanticFileAnchored(src)
	if err != nil {
		t.Fatal(err)
	}

	var capabilityAnchors int
	for _, anchor := range anchored.Anchors {
		if anchor.Kind == "semantic-capability" && anchor.ID == "security.authenticate" {
			capabilityAnchors++
		}
		if anchor.Kind == "semantic-outcome" {
			t.Fatalf("Outcome単位のSource Index identityを先回りして生成しています: %#v", anchor)
		}
	}
	if capabilityAnchors != 1 {
		t.Fatalf("security.authenticateのCapability anchor数=%d want=1; anchors=%#v", capabilityAnchors, anchored.Anchors)
	}
}
