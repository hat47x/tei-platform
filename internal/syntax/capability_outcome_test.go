package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseSemanticCapabilityAcceptsNamedOutcomes(t *testing.T) {
	src := `namespace security

type AuthenticatedSubject from std.identifier
type AuthenticationDenial from std.code

capability security.authenticate {
    input loginId std.identifier
    outcome authenticated security.authenticated_subject
    outcome denied security.authentication_denial
}`

	ns, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatalf("named Outcomeを持つSemanticのparseに失敗しました: %v", err)
	}
	if len(ns.Capabilities) != 1 {
		t.Fatalf("capabilities=%#v", ns.Capabilities)
	}
	capability := ns.Capabilities[0]
	if capability.Output != "" {
		t.Fatalf("named Outcome Capabilityへ通常outputが入り込んでいます: %q", capability.Output)
	}
	if len(capability.Outcomes) != 2 {
		t.Fatalf("outcomes=%#v", capability.Outcomes)
	}
	if capability.Outcomes[0].Name != "authenticated" || capability.Outcomes[0].Payload != "security.authenticated_subject" {
		t.Fatalf("authenticated Outcome=%#v", capability.Outcomes[0])
	}
	if capability.Outcomes[1].Name != "denied" || capability.Outcomes[1].Payload != "security.authentication_denial" {
		t.Fatalf("denied Outcome=%#v", capability.Outcomes[1])
	}
}

func TestParseSemanticCapabilityKeepsExistingOutputContract(t *testing.T) {
	src := `namespace sales

data Order {
    attribute id type std.identifier
}

capability sales.order.submit {
    input order sales.order
    output sales.order
}`

	ns, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatalf("既存output Capabilityのparseに失敗しました: %v", err)
	}
	capability := ns.Capabilities[0]
	if capability.Output != "sales.order" || len(capability.Outcomes) != 0 {
		t.Fatalf("既存output契約が変わっています: %#v", capability)
	}
}

func TestParseSemanticCapabilityRejectsOutputAndOutcomeTogether(t *testing.T) {
	for name, src := range map[string]string{
		"output-first": `namespace example

type Result from std.text

type Denial from std.code

capability run {
    output example.result
    outcome denied example.denial
}`,
		"outcome-first": `namespace example

type Result from std.text

type Denial from std.code

capability run {
    outcome denied example.denial
    output example.result
}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := syntax.NewParser(src).ParseSemanticFile(); err == nil {
				t.Fatal("outputとoutcomeを同じCapabilityで受理しました")
			}
		})
	}
}

func TestParseSemanticCapabilityRejectsDuplicateOutcomeName(t *testing.T) {
	src := `namespace example

type ResultA from std.text
type ResultB from std.text

capability run {
    outcome completed example.result_a
    outcome completed example.result_b
}`

	if _, err := syntax.NewParser(src).ParseSemanticFile(); err == nil {
		t.Fatal("同名Outcomeの重複を受理しました")
	}
}

func TestSemanticAnchoredParserAcceptsOutcomeSyntax(t *testing.T) {
	src := `namespace security

type AuthenticatedSubject from std.identifier

type AuthenticationDenial from std.code

capability security.authenticate {
    outcome authenticated security.authenticated_subject
    outcome denied security.authentication_denial
}`

	anchored, err := syntax.ParseSemanticFileAnchored(src)
	if err != nil {
		t.Fatalf("Source Anchor側がoutcome構文を受理できません: %v", err)
	}
	found := false
	for _, anchor := range anchored.Anchors {
		if anchor.Kind == "semantic-capability" && anchor.ID == "security.authenticate" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Capability source anchorが失われました: %#v", anchored.Anchors)
	}
}
