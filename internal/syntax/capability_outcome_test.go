package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestSemanticCapabilityParsesNamedOutcomes(t *testing.T) {
	src := `namespace security

    data AuthenticatedSubject {}
    data AuthenticationDenial {}

    capability security.authenticate {
      input loginId std.identifier
      outcome denied security.authentication_denial
      outcome authenticated security.authenticated_subject
    }`

	ns, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(ns.Capabilities) != 1 {
		t.Fatalf("capabilities=%#v", ns.Capabilities)
	}
	capability := ns.Capabilities[0]
	if capability.Output != "" {
		t.Fatalf("named Outcomeを持つCapabilityにoutputが設定されています: %s", capability.Output)
	}
	if len(capability.Outcomes) != 2 {
		t.Fatalf("outcomes=%#v", capability.Outcomes)
	}
	if capability.Outcomes[0].Name != "denied" || capability.Outcomes[0].Payload != "security.authentication_denial" {
		t.Fatalf("first outcome=%#v", capability.Outcomes[0])
	}
	if capability.Outcomes[1].Name != "authenticated" || capability.Outcomes[1].Payload != "security.authenticated_subject" {
		t.Fatalf("second outcome=%#v", capability.Outcomes[1])
	}
}
