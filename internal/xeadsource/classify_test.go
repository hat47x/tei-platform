package xeadsource

import (
	"reflect"
	"testing"
)

func TestClassifyRequiresExplicitPolicyAndKeepsUnknownUnresolved(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table/><FutureThing/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := Classify(doc, ClassificationPolicy{
		"System": {Classifications: []string{"realization", "source-metadata"}, Extension: "reference.compat.xead.source"},
		"Table":  {Classifications: []string{"unresolved", "canonical-candidate"}, Extension: "reference.xead.relational-semantics"},
	})
	if len(got) != 3 {
		t.Fatalf("candidate count = %d", len(got))
	}
	if !reflect.DeepEqual(got[1].Classifications, []string{"canonical-candidate", "unresolved"}) {
		t.Fatalf("classifications must be deterministic: %#v", got[1])
	}
	if !got[2].Unresolved || len(got[2].Classifications) != 0 || got[2].Extension != "" {
		t.Fatalf("unknown source construct gained implicit meaning: %#v", got[2])
	}
}

func TestClassifyCarriesKeyAuthorityLimitsWithoutPromotingMeaning(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table><TableKey Type="PK"><TableKeyField FieldID="F1"/></TableKey></Table></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := Classify(doc, ClassificationPolicy{
		"TableKey": {
			Classifications: []string{"canonical-candidate", "source-key-assertion", "unresolved"},
			Extension:       "reference.xead.relational-semantics",
			AuthorityLimits: []string{
				"legacy-key-is-not-business-identity",
				"legacy-key-is-not-functional-dependency",
			},
		},
	})
	var key *Candidate
	for i := range got {
		if got[i].SourceName == "TableKey" {
			key = &got[i]
			break
		}
	}
	if key == nil {
		t.Fatal("TableKey candidate is missing")
	}
	if key.Unresolved {
		t.Fatalf("explicitly classified source construct remained unmapped: %#v", *key)
	}
	if !reflect.DeepEqual(key.Classifications, []string{"canonical-candidate", "source-key-assertion", "unresolved"}) {
		t.Fatalf("classifications drifted: %#v", *key)
	}
	if !reflect.DeepEqual(key.AuthorityLimits, []string{
		"legacy-key-is-not-business-identity",
		"legacy-key-is-not-functional-dependency",
	}) {
		t.Fatalf("authority limits drifted: %#v", *key)
	}
	if key.Extension != "reference.xead.relational-semantics" {
		t.Fatalf("extension drifted: %#v", *key)
	}
}

func TestClassificationPolicyFingerprintNormalizesRuleOrdering(t *testing.T) {
	a := ClassificationPolicy{
		"TableKey": {
			Classifications: []string{"unresolved", "canonical-candidate"},
			Extension:       "reference.xead.relational-semantics",
			AuthorityLimits: []string{"z-limit", "a-limit"},
		},
	}
	b := ClassificationPolicy{
		"TableKey": {
			Classifications: []string{"canonical-candidate", "unresolved"},
			Extension:       "reference.xead.relational-semantics",
			AuthorityLimits: []string{"a-limit", "z-limit"},
		},
	}
	if ClassificationPolicyFingerprint(a) == "" {
		t.Fatal("non-empty policy has no fingerprint")
	}
	if ClassificationPolicyFingerprint(a) != ClassificationPolicyFingerprint(b) {
		t.Fatalf("equivalent policy ordering changed fingerprint: a=%s b=%s",
			ClassificationPolicyFingerprint(a),
			ClassificationPolicyFingerprint(b),
		)
	}
	b["TableKey"] = CandidateRule{
		Classifications: []string{"canonical-candidate", "unresolved"},
		Extension:       "different.extension",
		AuthorityLimits: []string{"a-limit", "z-limit"},
	}
	if ClassificationPolicyFingerprint(a) == ClassificationPolicyFingerprint(b) {
		t.Fatal("different classification policy produced the same fingerprint")
	}
	if ClassificationPolicyFingerprint(nil) != "" {
		t.Fatal("nil policy must not claim classification provenance")
	}
}

func TestParseAloneDoesNotGrantCandidateMeaning(t *testing.T) {
	doc, err := Parse(FormatXeaf, []byte(`<System><Script/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := Classify(doc, nil)
	for _, candidate := range got {
		if !candidate.Unresolved || len(candidate.Classifications) != 0 || candidate.Extension != "" {
			t.Fatalf("source parsing granted semantic authority: %#v", candidate)
		}
	}
}

func TestClassifyRetainsDynamicScriptSourceMarkerWithoutExecutionAuthority(t *testing.T) {
	const marker = "source-script-text-mentions-executeScript-not-runtime-or-execution-authority"
	// Synthetic data only: both Table Script.Text and Function.Script are
	// source attributes, not invocation traces or accepted TEI meanings.
	doc, err := Parse(FormatXeaf, []byte(`<System xmlns:f="urn:foreign">
		<Table>
			<Script EventP="BU" Text="instance.executeScript(scriptText);" />
			<Script EventP="AR" Text="AT000_DERIVED.value = AT000_INPUT.value;" />
			<f:Script Text="instance.executeScript(secret);" />
		</Table>
		<Function Script="instance.executeScript(functionText);" />
	</System>`))
	if err != nil {
		t.Fatal(err)
	}

	withoutPolicy := Classify(doc, nil)
	if len(withoutPolicy) != 6 {
		t.Fatalf("unexpected candidate count: %d", len(withoutPolicy))
	}
	for _, c := range withoutPolicy {
		if !c.Unresolved || len(c.Classifications) != 0 || c.Extension != "" {
			t.Fatalf("lexical Script evidence granted implicit mapping: %#v", c)
		}
	}
	if !reflect.DeepEqual(withoutPolicy[2].AuthorityLimits, []string{marker}) {
		t.Fatalf("Table Script dynamic marker lost: %#v", withoutPolicy[2])
	}
	if len(withoutPolicy[3].AuthorityLimits) != 0 {
		t.Fatalf("Script without marker was inferred to execute dynamically: %#v", withoutPolicy[3])
	}
	if len(withoutPolicy[4].AuthorityLimits) != 0 || !withoutPolicy[4].Unresolved {
		t.Fatalf("foreign namespace gained Script evidence or authority: %#v", withoutPolicy[4])
	}
	if !reflect.DeepEqual(withoutPolicy[5].AuthorityLimits, []string{marker}) {
		t.Fatalf("Function.Script dynamic marker lost: %#v", withoutPolicy[5])
	}

	withPolicy := Classify(doc, ClassificationPolicy{
		"Script": {
			Classifications: []string{"unresolved"},
			AuthorityLimits: []string{marker, "owner-acceptance-required"},
		},
		"Function": {Classifications: []string{"unresolved"}},
	})
	if !reflect.DeepEqual(withPolicy[2].AuthorityLimits, []string{
		"owner-acceptance-required", marker,
	}) {
		t.Fatalf("source marker should not duplicate a policy limit: %#v", withPolicy[2])
	}
	if !reflect.DeepEqual(withPolicy[2].Classifications, []string{"unresolved"}) {
		t.Fatalf("source marker must not promote business meaning: %#v", withPolicy[2])
	}
	if len(withPolicy[3].AuthorityLimits) != 2 {
		t.Fatalf("explicit policy limits must remain even without a source marker: %#v", withPolicy[3])
	}
}

func TestClassifyDynamicScriptMarkerIsLexicalNotCallGraph(t *testing.T) {
	const marker = "source-script-text-mentions-executeScript-not-runtime-or-execution-authority"
	doc, err := Parse(FormatXeaf, []byte(`<System>
		<Script Text="// executeScript is mentioned in a comment only" />
		<Script Text="eval(otherSource);" />
	</System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := Classify(doc, nil)
	if !reflect.DeepEqual(got[1].AuthorityLimits, []string{marker}) {
		t.Fatalf("lexical mention must remain evidence, not a claimed call: %#v", got[1])
	}
	if len(got[2].AuthorityLimits) != 0 || !got[2].Unresolved {
		t.Fatalf("absence of executeScript must not claim Script purity: %#v", got[2])
	}
}

func TestClassifyIgnoresForeignNamespaceScriptAttribute(t *testing.T) {
	doc, err := Parse(FormatXeaf, []byte(`<System xmlns:f="urn:foreign">
		<Script f:Text="instance.executeScript(other);" />
		<Function f:Script="instance.executeScript(other);" />
	</System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := Classify(doc, nil)
	if len(got) != 3 {
		t.Fatalf("unexpected candidate count: %d", len(got))
	}
	for _, c := range got {
		if len(c.AuthorityLimits) != 0 || !c.Unresolved {
			t.Fatalf("foreign attribute was treated as X-TEA executable source: %#v", c)
		}
	}
}
