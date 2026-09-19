package workspace_test

import (
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestReferenceAuthenticationUsesNamedOutcomesWithoutSecretMaterial(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	result, err := workspace.Load(ws)
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}

	capability := semanticCapabilityByID(t, result.World.Semantics, "security.authenticate")
	if capability.Output != "" {
		t.Fatalf("security.authenticateに単一outputが残っています: %q", capability.Output)
	}

	inputs := map[string]string{}
	for _, input := range capability.Inputs {
		inputs[string(input.Name)] = string(input.Type)
	}
	if inputs["login_id"] != "std.identifier" {
		t.Fatalf("login_id input=%q want=%q", inputs["login_id"], "std.identifier")
	}
	if inputs["credential"] != "security.credential" {
		t.Fatalf("credential input=%q want=%q", inputs["credential"], "security.credential")
	}
	if len(inputs) != 2 {
		t.Fatalf("security.authenticate inputs=%#v", inputs)
	}

	if len(capability.Outcomes) != 2 {
		t.Fatalf("security.authenticate outcomes=%#v", capability.Outcomes)
	}
	if capability.Outcomes[0].Name != "authenticated" || capability.Outcomes[0].Payload != "security.authenticated_subject" {
		t.Fatalf("first normalized outcome=%#v", capability.Outcomes[0])
	}
	if capability.Outcomes[1].Name != "denied" || capability.Outcomes[1].Payload != "security.authentication_denial" {
		t.Fatalf("second normalized outcome=%#v", capability.Outcomes[1])
	}

	submit := semanticCapabilityByID(t, result.World.Semantics, "sales.order.submit")
	if submit.Output != "sales.order" || len(submit.Outcomes) != 0 {
		t.Fatalf("既存sales.order.submitの結果契約が変わっています: %#v", submit)
	}

	idx, err := workspace.BuildSourceIndex(ws, result)
	if err != nil {
		t.Fatalf("Source Indexの構築に失敗しました: %v", err)
	}
	refs := idx.ReferencesFor(sourceindex.Subject{Kind: "semantic-capability", ID: "security.authenticate"})
	if len(refs) != 1 {
		t.Fatalf("security.authenticateのsource references=%#v", refs)
	}
	if refs[0].ArtifactPath != "semantic/security/authentication.tei" {
		t.Fatalf("security.authenticateのsource path=%q", refs[0].ArtifactPath)
	}
}
