package buildintent

import (
	"strings"
	"testing"
)

// These tests pin the declaration surface that moved out of the reference tool
// in SB-C0 §4.13 (I-1a). They exercise the reader directly, so the CLI's own
// dependency on it cannot silently regress.

const validIntent = `{
  "formatVersion": 1,
  "subject": {"kind": "plugin", "id": "tei.plugin.reader_probe", "version": "1.0.0", "definitionSource": "definition"},
  "sources": [{"kind": "plugin-definition", "id": "definition", "ref": "plugin.tei"}],
  "builder": {"contract": "tei.plugin-build", "compatibleMajor": 1},
  "recipe": {"kind": "go-package", "entrypoint": "plugin.tei"},
  "outputKind": "plugin-package",
  "target": "host/linux/amd64",
  "profile": "release"
}`

// A valid declaration parses, and its fingerprint is stable across field order.
func TestParseIntentReadsAValidDeclaration(t *testing.T) {
	intent, err := ParseIntent([]byte(validIntent))
	if err != nil {
		t.Fatalf("valid declaration was refused: %v", err)
	}
	if intent.Subject.ID != "tei.plugin.reader_probe" {
		t.Fatalf("subject id was not read: %q", intent.Subject.ID)
	}
	if string(intent.OutputKind) != "plugin-package" {
		t.Fatalf("output kind was not read: %q", intent.OutputKind)
	}

	first, err := IntentFingerprint(intent)
	if err != nil {
		t.Fatal(err)
	}
	reordered := `{
  "target": "host/linux/amd64",
  "profile": "release",
  "outputKind": "plugin-package",
  "recipe": {"entrypoint": "plugin.tei", "kind": "go-package"},
  "builder": {"compatibleMajor": 1, "contract": "tei.plugin-build"},
  "sources": [{"ref": "plugin.tei", "id": "definition", "kind": "plugin-definition"}],
  "subject": {"definitionSource": "definition", "version": "1.0.0", "id": "tei.plugin.reader_probe", "kind": "plugin"},
  "formatVersion": 1
}`
	again, err := ParseIntent([]byte(reordered))
	if err != nil {
		t.Fatalf("reordered declaration was refused: %v", err)
	}
	second, err := IntentFingerprint(again)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("field order changed the fingerprint: %s vs %s", first, second)
	}
	if len(first) != 64 {
		t.Fatalf("fingerprint is not a sha256 hex digest: %q", first)
	}
}

// A hand-written derived value is refused: the resolver derives those.
func TestParseIntentRefusesAuthoredDerivedValues(t *testing.T) {
	authored := strings.Replace(validIntent, `"outputKind": "plugin-package",`, `"outputKind": "plugin-package", "planFingerprint": "deadbeef",`, 1)
	if _, err := ParseIntent([]byte(authored)); err == nil {
		t.Fatal("an authored planFingerprint was accepted")
	} else if !strings.Contains(err.Error(), "derived field") {
		t.Fatalf("the refusal did not name the derived field: %v", err)
	}
}

// A case-variant key is refused rather than folded into the field it resembles.
func TestParseIntentRefusesCaseVariantKeys(t *testing.T) {
	variant := strings.Replace(validIntent, `"target"`, `"Target"`, 1)
	if _, err := ParseIntent([]byte(variant)); err == nil {
		t.Fatal("a case-variant key was accepted")
	}
}

// An unknown key is refused instead of silently ignored.
func TestParseIntentRefusesUnknownKeys(t *testing.T) {
	unknown := strings.Replace(validIntent, `"target": "host/linux/amd64",`, `"target": "host/linux/amd64", "extra": 1,`, 1)
	if _, err := ParseIntent([]byte(unknown)); err == nil {
		t.Fatal("an unknown key was accepted")
	}
}

// Trailing content after the object is refused.
func TestParseIntentRefusesTrailingContent(t *testing.T) {
	if _, err := ParseIntent([]byte(validIntent + " {}")); err == nil {
		t.Fatal("trailing content was accepted")
	}
}

// A source reference that is not a clean relative slash path is refused.
func TestValidateSourceRefRefusesEscapes(t *testing.T) {
	for _, ref := range []string{"", "../outside", "/absolute", "C:/volume", `back\slash`, "a/../b", "./a"} {
		if err := ValidateSourceRef(ref); err == nil {
			t.Fatalf("source ref %q was accepted", ref)
		}
	}
	for _, ref := range []string{"plugin.tei", "src/plugin.tei", "a/b/c.txt"} {
		if err := ValidateSourceRef(ref); err != nil {
			t.Fatalf("source ref %q was refused: %v", ref, err)
		}
	}
}

// An explicit pin must use the declared digest shape.
func TestIsSHA256DigestMatchesOnlyTheDeclaredShape(t *testing.T) {
	if !IsSHA256Digest("sha256:" + strings.Repeat("a", 64)) {
		t.Fatal("a well-formed digest was refused")
	}
	for _, bad := range []string{
		"",
		"sha256:",
		"sha256:" + strings.Repeat("A", 64),
		"sha256:" + strings.Repeat("a", 63),
		strings.Repeat("a", 64),
	} {
		if IsSHA256Digest(bad) {
			t.Fatalf("digest %q was accepted", bad)
		}
	}
}

// Validate refuses a declaration whose definition source is not declared.
func TestValidateRefusesAnUndeclaredDefinitionSource(t *testing.T) {
	intent, err := ParseIntent([]byte(validIntent))
	if err != nil {
		t.Fatal(err)
	}
	intent.Subject.DefinitionSource = "missing"
	if err := intent.Validate(); err == nil {
		t.Fatal("an undeclared definitionSource was accepted")
	}
}

// A secret-like parameter name is refused: secrets belong in
// secretRequirements, not in parameters.
func TestValidateRefusesSecretLikeParameterNames(t *testing.T) {
	intent, err := ParseIntent([]byte(validIntent))
	if err != nil {
		t.Fatal(err)
	}
	intent.Parameters = append(intent.Parameters, buildParameterForTest("API_KEY"))
	if err := intent.Validate(); err == nil {
		t.Fatal("a secret-like parameter name was accepted")
	}
}
