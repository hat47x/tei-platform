package rmtevidence

import (
	"strings"
	"testing"
)

func fixture() Evidence {
	return Evidence{
		Schema:                     Schema,
		SourceFormat:               "xead",
		SourceFormatVersion:        "1.2",
		SourceArtifactDigest:       "sha256:" + strings.Repeat("1", 64),
		SourceToken:                "sha256:" + strings.Repeat("2", 64),
		SourceConstructKnown:       true,
		SourceConstruct:            "Table",
		MappingContractFingerprint: "sha256:" + strings.Repeat("3", 64),
		MappingStatus:              MappingMapped,
		Classification:             []string{"unresolved", "canonical-candidate", "canonical-candidate"},
		Extension:                  "reference.xead.relational-semantics",
		AuthorityLimits:            []string{"source-observation-is-not-canonical-authority", "mapping-is-candidate-not-semantic-acceptance"},
	}
}

func TestPrepareVerifyRoundTrip(t *testing.T) {
	prepared, err := Prepare(fixture())
	if err != nil {
		t.Fatal(err)
	}
	if !fingerprintPattern.MatchString(prepared.EvidenceFingerprint) {
		t.Fatalf("fingerprint=%q", prepared.EvidenceFingerprint)
	}
	if strings.Join(prepared.Classification, ",") != "canonical-candidate,unresolved" {
		t.Fatalf("classification=%v", prepared.Classification)
	}
	encoded, err := CanonicalBytes(prepared)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.EvidenceFingerprint != prepared.EvidenceFingerprint {
		t.Fatalf("fingerprint drift: %s != %s", decoded.EvidenceFingerprint, prepared.EvidenceFingerprint)
	}
}

func TestTamperFailsClosed(t *testing.T) {
	prepared, err := Prepare(fixture())
	if err != nil {
		t.Fatal(err)
	}
	prepared.Extension = "other.extension"
	if _, err := Verify(prepared); err == nil || !strings.Contains(err.Error(), "fingerprint") {
		t.Fatalf("tampered Evidence accepted: %v", err)
	}
}

func TestUnknownConstructNameMustBeRedacted(t *testing.T) {
	value := fixture()
	value.SourceConstructKnown = false
	value.SourceConstruct = "PrivateBusinessThing"
	value.MappingStatus = MappingUnmapped
	value.Extension = ""
	value.Classification = []string{"unresolved"}
	if _, err := Prepare(value); err == nil || !strings.Contains(err.Error(), "公開できません") {
		t.Fatalf("unknown source name was accepted: %v", err)
	}

	value.SourceConstruct = ""
	prepared, err := Prepare(value)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := CanonicalBytes(prepared)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "PrivateBusinessThing") {
		t.Fatalf("redacted source name leaked: %s", encoded)
	}
}

func TestMappedEvidenceRequiresKnownConstructAndExtension(t *testing.T) {
	value := fixture()
	value.SourceConstructKnown = false
	value.SourceConstruct = ""
	if _, err := Prepare(value); err == nil {
		t.Fatal("unknown construct was accepted as mapped")
	}

	value = fixture()
	value.Extension = ""
	if _, err := Prepare(value); err == nil {
		t.Fatal("mapped Evidence without Extension was accepted")
	}
}

func TestPrepareAddsMandatoryAuthorityLimits(t *testing.T) {
	value := fixture()
	value.AuthorityLimits = []string{"producer-specific-limit"}
	prepared, err := Prepare(value)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range RequiredAuthorityLimits {
		found := false
		for _, actual := range prepared.AuthorityLimits {
			if actual == required {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("mandatory authority limit %q is missing: %v", required, prepared.AuthorityLimits)
		}
	}
}

func TestFingerprintFieldsRequireExactSHA256(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Evidence)
	}{
		{
			name: "artifact digest trailing data",
			mutate: func(value *Evidence) {
				value.SourceArtifactDigest += "x"
			},
		},
		{
			name: "source token prefix",
			mutate: func(value *Evidence) {
				value.SourceToken = "x" + value.SourceToken
			},
		},
		{
			name: "mapping fingerprint uppercase",
			mutate: func(value *Evidence) {
				value.MappingContractFingerprint = strings.ToUpper(value.MappingContractFingerprint)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value := fixture()
			tc.mutate(&value)
			if _, err := Prepare(value); err == nil || !strings.Contains(err.Error(), "exact sha256") {
				t.Fatalf("invalid fingerprint field was accepted: %v", err)
			}
		})
	}
}

func TestKnownConstructMustUsePublicStandardVocabulary(t *testing.T) {
	value := fixture()
	value.SourceConstruct = "PrivateBusinessThing"
	if _, err := Prepare(value); err == nil || !strings.Contains(err.Error(), "標準construct") {
		t.Fatalf("private construct name was accepted as known: %v", err)
	}

	value = fixture()
	value.SourceFormat = "xead"
	value.SourceConstruct = "Button"
	if _, err := Prepare(value); err == nil || !strings.Contains(err.Error(), "標準construct") {
		t.Fatalf("Xeaf-only construct was accepted as Xead: %v", err)
	}
}

func TestSourceFormatVersionMustBePublicNumericVersion(t *testing.T) {
	value := fixture()
	value.SourceFormatVersion = "private-version"
	if _, err := Prepare(value); err == nil || !strings.Contains(err.Error(), "数値version") {
		t.Fatalf("free-text source version was accepted: %v", err)
	}
}

func TestClassificationUsesControlledVocabulary(t *testing.T) {
	value := fixture()
	value.Classification = []string{"private-classification"}
	if _, err := Prepare(value); err == nil || !strings.Contains(err.Error(), "classification") {
		t.Fatalf("unknown classification was accepted: %v", err)
	}
}

func TestUnmappedEvidenceCannotClaimMappedClassification(t *testing.T) {
	value := fixture()
	value.MappingStatus = MappingUnmapped
	value.Extension = ""
	value.Classification = []string{"canonical-candidate"}
	if _, err := Prepare(value); err == nil || !strings.Contains(err.Error(), "unresolved") {
		t.Fatalf("unmapped Evidence claimed mapped classification: %v", err)
	}
}


func TestPublicFreeTextFieldsUseSafeTokenSyntax(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Evidence)
	}{
		{
			name: "extension prose",
			mutate: func(value *Evidence) {
				value.Extension = "private customer name"
			},
		},
		{
			name: "extension unicode",
			mutate: func(value *Evidence) {
				value.Extension = "reference.顧客"
			},
		},
		{
			name: "authority limit prose",
			mutate: func(value *Evidence) {
				value.AuthorityLimits = []string{"private customer name"}
			},
		},
		{
			name: "authority limit too long",
			mutate: func(value *Evidence) {
				value.AuthorityLimits = []string{"a" + strings.Repeat("b", publicTokenMaxLength)}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value := fixture()
			tc.mutate(&value)
			if _, err := Prepare(value); err == nil || !strings.Contains(err.Error(), "token") {
				t.Fatalf("unsafe public string was accepted: %v", err)
			}
		})
	}
}
