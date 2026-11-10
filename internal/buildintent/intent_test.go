package buildintent

import (
	"strings"
	"testing"
)

// These tests pin the declaration reader now that it lives in a package both
// the reference resolver and cmd/tei can import (SB-C0 §4.13 I-1a). They check
// the rules that make the shared reader trustworthy, not the resolver's plan
// derivation: that stays in tools/reference-build-authoring-intent.

const validIntent = `{
  "formatVersion": 1,
  "subject": {
    "kind": "plugin",
    "id": "tei.plugin.buildintent_test",
    "version": "1.0.0",
    "definitionSource": "definition"
  },
  "sources": [
    {"kind": "plugin-definition", "id": "definition", "ref": "plugin.tei"}
  ],
  "builder": {"contract": "tei.plugin-build", "compatibleMajor": 1},
  "recipe": {"kind": "go-package", "entrypoint": "./impl"},
  "outputKind": "runtime-artifact",
  "target": "host/linux/amd64"
}
`

func TestParseIntentAcceptsAValidDeclaration(t *testing.T) {
	intent, err := ParseIntent([]byte(validIntent))
	if err != nil {
		t.Fatalf("a valid declaration was rejected: %v", err)
	}
	if intent.Subject.Kind != "plugin" || intent.Subject.Version != "1.0.0" {
		t.Fatalf("declaration was not read: %+v", intent.Subject)
	}
	if len(intent.Sources) != 1 || intent.Sources[0].Ref != "plugin.tei" {
		t.Fatalf("sources were not read: %+v", intent.Sources)
	}
}

// A declaration must not carry a value the resolver derives. Accepting one
// would let an author pin a digest no one observed.
func TestParseIntentRejectsAnAuthoredDerivedValue(t *testing.T) {
	withDigest := strings.Replace(validIntent,
		`"sources": [`, `"planFingerprint": "00", "sources": [`, 1)
	if _, err := ParseIntent([]byte(withDigest)); err == nil ||
		!strings.Contains(err.Error(), "derived field") {
		t.Fatalf("an authored derived value was accepted: %v", err)
	}
}

// builder.pin.artifactDigest is the one value an author may pin on purpose.
func TestParseIntentAllowsTheIntendedBuilderPin(t *testing.T) {
	pinned := strings.Replace(validIntent,
		`"builder": {"contract": "tei.plugin-build", "compatibleMajor": 1}`,
		`"builder": {"contract": "tei.plugin-build", "compatibleMajor": 1, "pin": {"artifactDigest": "sha256:`+strings.Repeat("a", 64)+`"}}`, 1)
	if _, err := ParseIntent([]byte(pinned)); err != nil {
		t.Fatalf("the intended builder pin was rejected: %v", err)
	}
}

// Unknown keys, case variants and trailing JSON are refused: the reader is
// strict so a typo cannot silently change the meaning of a declaration.
func TestParseIntentRejectsUnknownCaseVariantAndTrailingKeys(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "unknown key",
			body: strings.Replace(validIntent, `"recipe":`, `"extra": 1, "recipe":`, 1),
			want: "does not match a field name exactly",
		},
		{
			name: "case variant",
			body: strings.Replace(validIntent, `"formatVersion"`, `"FormatVersion"`, 1),
			want: "does not match a field name exactly",
		},
		{
			name: "trailing document",
			body: validIntent + `{"formatVersion": 1}`,
			// The strict value parser refuses a second document before the
			// trailing-token check runs; either rejection is correct.
			want: "after",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseIntent([]byte(tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want a rejection containing %q", err, tc.want)
			}
		})
	}
}

// Field order must not change the identity, otherwise two authors writing the
// same declaration would produce different fingerprints.
func TestIntentFingerprintIgnoresFieldOrder(t *testing.T) {
	first, err := ParseIntent([]byte(validIntent))
	if err != nil {
		t.Fatal(err)
	}
	// Reordering the top-level keys only.
	reordered := `{
  "target": "host/linux/amd64",
  "outputKind": "runtime-artifact",
  "recipe": {"kind": "go-package", "entrypoint": "./impl"},
  "builder": {"contract": "tei.plugin-build", "compatibleMajor": 1},
  "sources": [
    {"kind": "plugin-definition", "id": "definition", "ref": "plugin.tei"}
  ],
  "subject": {
    "kind": "plugin",
    "id": "tei.plugin.buildintent_test",
    "version": "1.0.0",
    "definitionSource": "definition"
  },
  "formatVersion": 1
}
`
	second, err := ParseIntent([]byte(reordered))
	if err != nil {
		t.Fatalf("a reordered declaration was rejected: %v", err)
	}
	a, err := IntentFingerprint(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := IntentFingerprint(second)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("field order changed the fingerprint: %q != %q", a, b)
	}
}

// A declaration whose subject version is not numeric must not fingerprint: the
// identity would then name a subject that cannot exist.
func TestIntentFingerprintRejectsAnInvalidDeclaration(t *testing.T) {
	bad, err := ParseIntent([]byte(strings.Replace(validIntent, `"version": "1.0.0"`, `"version": "latest"`, 1)))
	if err == nil {
		t.Fatal("a non-numeric subject version was accepted")
	}
	_ = bad
}

func TestValidateSourceRefRejectsUnsafeReferences(t *testing.T) {
	for _, ref := range []string{
		"",
		"/abs/path",
		`Windows\path`,
		"C:/volume",
		"a/../b",
		"../escape",
		".",
		"..",
		"a//b",
	} {
		if err := ValidateSourceRef(ref); err == nil {
			t.Fatalf("source ref %q was accepted", ref)
		}
	}
	for _, ref := range []string{"plugin.tei", "impl/main.go", "a/b/c.txt"} {
		if err := ValidateSourceRef(ref); err != nil {
			t.Fatalf("source ref %q was rejected: %v", ref, err)
		}
	}
}

// The declaration states a requirement as a contract name and a major; the plan
// records the resolved identity as one string. This function is the one stated
// correspondence between the two, so its form is pinned here: a caller that
// compares must use it rather than spell the format out again.
func TestBuilderContractVersionCorrespondsToTheResolvedIdentityForm(t *testing.T) {
	cases := []struct {
		contract string
		major    int
		want     string
	}{
		{"tei.plugin-build", 1, "tei.plugin-build/v1"},
		{"tei.plugin-build", 2, "tei.plugin-build/v2"},
		{"tei.other-build", 10, "tei.other-build/v10"},
	}
	for _, tc := range cases {
		if got := BuilderContractVersion(tc.contract, tc.major); got != tc.want {
			t.Fatalf("BuilderContractVersion(%q, %d) = %q, want %q", tc.contract, tc.major, got, tc.want)
		}
	}
}
