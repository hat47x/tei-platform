package artifact

import (
	"encoding/json"
	"testing"
)

func testRule() Rule {
	return Rule{ID: "tei.artifact.table-definition", Version: "v1", Fingerprint: "sha256:rule"}
}

func TestNewComputesAStableDigest(t *testing.T) {
	first, err := New(KindTableDefinition, testRule(), "tei-generator/v1", "sha256:input", "table orders\n", nil)
	if err != nil {
		t.Fatalf("build artifact: %v", err)
	}
	second, err := New(KindTableDefinition, testRule(), "tei-generator/v1", "sha256:input", "table orders\n", nil)
	if err != nil {
		t.Fatalf("build artifact: %v", err)
	}
	if first.Digest == "" || first.Digest != second.Digest {
		t.Fatalf("the same input and rule must produce the same digest: %q vs %q", first.Digest, second.Digest)
	}
	if err := first.Verify(); err != nil {
		t.Fatalf("fresh artifact must verify: %v", err)
	}
}

func TestDigestChangesWithContentAndRule(t *testing.T) {
	base, err := New(KindTableDefinition, testRule(), "tei-generator/v1", "sha256:input", "table orders\n", nil)
	if err != nil {
		t.Fatal(err)
	}

	changed, err := New(KindTableDefinition, testRule(), "tei-generator/v1", "sha256:input", "table customers\n", nil)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Digest == base.Digest {
		t.Fatal("different content must produce a different digest")
	}

	otherRule := testRule()
	otherRule.Version = "v2"
	reversioned, err := New(KindTableDefinition, otherRule, "tei-generator/v1", "sha256:input", "table orders\n", nil)
	if err != nil {
		t.Fatal(err)
	}
	if reversioned.Digest == base.Digest {
		t.Fatal("a different rule version must produce a different digest")
	}

	otherInput, err := New(KindTableDefinition, testRule(), "tei-generator/v1", "sha256:other", "table orders\n", nil)
	if err != nil {
		t.Fatal(err)
	}
	if otherInput.Digest == base.Digest {
		t.Fatal("a different input revision must produce a different digest")
	}
}

func TestUnresolvedOrderDoesNotChangeTheDigest(t *testing.T) {
	first, err := New(KindTableDefinition, testRule(), "tei-generator/v1", "sha256:input", "table orders\n", []Unresolved{
		{Subject: "sales.order.customer_id", Reason: "missing-semantics", Detail: "no type"},
		{Subject: "sales.order.status", Reason: "missing-input", Detail: "no column"},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := New(KindTableDefinition, testRule(), "tei-generator/v1", "sha256:input", "table orders\n", []Unresolved{
		{Subject: "sales.order.status", Reason: "missing-input", Detail: "no column"},
		{Subject: "sales.order.customer_id", Reason: "missing-semantics", Detail: "no type"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest {
		t.Fatal("the unresolved order must not change the digest")
	}
	if first.Unresolved[0].Subject != "sales.order.customer_id" {
		t.Fatalf("unresolved list must be sorted: %+v", first.Unresolved)
	}
}

func TestVerifyRejectsATamperedArtifact(t *testing.T) {
	built, err := New(KindTableDefinition, testRule(), "tei-generator/v1", "sha256:input", "table orders\n", nil)
	if err != nil {
		t.Fatal(err)
	}

	tampered := built
	tampered.Content = "table customers\n"
	if err := tampered.Verify(); err == nil {
		t.Fatal("a changed content must not verify")
	}

	replaced := built
	replaced.Unresolved = []Unresolved{{Subject: "sales.order", Reason: "missing-input", Detail: "hidden"}}
	if err := replaced.Verify(); err == nil {
		t.Fatal("a changed unresolved list must not verify")
	}
}

func TestNewRejectsIncompleteBindings(t *testing.T) {
	rule := testRule()
	cases := []struct {
		name      string
		kind      Kind
		rule      Rule
		generator string
		revision  string
	}{
		{"no kind", "", rule, "tei-generator/v1", "sha256:input"},
		{"no rule id", KindTableDefinition, Rule{Version: "v1", Fingerprint: "sha256:rule"}, "tei-generator/v1", "sha256:input"},
		{"no rule version", KindTableDefinition, Rule{ID: "tei.artifact.table-definition", Fingerprint: "sha256:rule"}, "tei-generator/v1", "sha256:input"},
		{"no rule fingerprint", KindTableDefinition, Rule{ID: "tei.artifact.table-definition", Version: "v1"}, "tei-generator/v1", "sha256:input"},
		{"no generator", KindTableDefinition, rule, "", "sha256:input"},
		{"no input revision", KindTableDefinition, rule, "tei-generator/v1", ""},
	}
	for _, tc := range cases {
		if _, err := New(tc.kind, tc.rule, tc.generator, tc.revision, "x", nil); err == nil {
			t.Fatalf("%s must be rejected", tc.name)
		}
	}
}

func TestArtifactSerializesWithoutLosingTheContract(t *testing.T) {
	built, err := New(KindTableDefinition, testRule(), "tei-generator/v1", "sha256:input", "table orders\n", []Unresolved{
		{Subject: "sales.order", Reason: "missing-input", Detail: "no lifecycle"},
	})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(built)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Artifact
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := decoded.Verify(); err != nil {
		t.Fatalf("round-tripped artifact must verify: %v", err)
	}
	if decoded.Kind != string(KindTableDefinition) || decoded.InputRevision != "sha256:input" {
		t.Fatalf("round trip lost identity fields: %+v", decoded)
	}
}
