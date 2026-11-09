package languagereference

import (
	"errors"
	"fmt"
	"testing"
)

func testPin() Selection {
	return Selection{
		LanguageID: "std.dsl.tei", ContractVersion: "1.0.0",
		ContractFingerprint: "meaning-v1", ImplementationID: "reference.tei.parser",
		ImplementationFingerprint: "parser-v1",
	}
}

func TestExactSelectionIndependentOfRegistrationOrder(t *testing.T) {
	pin := testPin()
	original := []Provider[string]{
		{Selection: Selection{LanguageID: "std.dsl.other", ContractVersion: "1.0.0", ContractFingerprint: "other", ImplementationID: "other", ImplementationFingerprint: "other"}, Parse: func(string) (string, error) { return "wrong", nil }},
		{Selection: pin, Parse: func(src string) (string, error) { return "tei:" + src, nil }},
		{Selection: Selection{LanguageID: pin.LanguageID, ContractVersion: "2.0.0", ContractFingerprint: "v2", ImplementationID: pin.ImplementationID, ImplementationFingerprint: "parser-v2"}, Parse: func(string) (string, error) { return "newer", nil }},
	}
	for _, providers := range [][]Provider[string]{original, {original[2], original[1], original[0]}} {
		got, err := ParseSelected(pin, providers, "namespace sales")
		if err != nil || got != "tei:namespace sales" {
			t.Fatalf("wrong selection result %q, %v", got, err)
		}
	}
}

func TestSelectionFailuresAreExplicit(t *testing.T) {
	pin := testPin()
	provider := Provider[string]{Selection: pin, Parse: func(src string) (string, error) { return src, nil }}
	tests := []struct {
		name      string
		pin       Selection
		providers []Provider[string]
		want      error
	}{
		{"missing pin", Selection{}, []Provider[string]{provider}, ErrInvalidSelection},
		{"missing provider", pin, nil, ErrProviderMissing},
		{"mismatched fingerprint", Selection{pin.LanguageID, pin.ContractVersion, pin.ContractFingerprint, pin.ImplementationID, "other"}, []Provider[string]{provider}, ErrProviderMissing},
		{"duplicate", pin, []Provider[string]{provider, provider}, ErrProviderAmbiguous},
		{"parser absent", pin, []Provider[string]{{Selection: pin}}, ErrProviderMissing},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseSelected(tc.pin, tc.providers, "x")
			if !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}

func TestParserErrorRetainsCause(t *testing.T) {
	pin := testPin()
	sentinel := fmt.Errorf("invalid source at 2:3")
	provider := Provider[string]{Selection: pin, Parse: func(string) (string, error) { return "", sentinel }}
	_, err := ParseSelected(pin, []Provider[string]{provider}, "invalid")
	if !errors.Is(err, sentinel) {
		t.Fatalf("parser diagnostics lost: %v", err)
	}
}
