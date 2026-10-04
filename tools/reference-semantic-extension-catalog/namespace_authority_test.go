package main

import (
	"strings"
	"testing"
)

func TestSemanticExtensionCannotFirstClaimStandardNamespace(t *testing.T) {
	for _, namespace := range []string{"std", "std.validation", "std.data.custom"} {
		descriptor := ExtensionDescriptor{
			Schema:    extensionSchema,
			ID:        "third.party.extension",
			Namespace: namespace,
			Version:   "1.0.0",
		}
		if _, err := normalizeDescriptor(descriptor); err == nil || !strings.Contains(err.Error(), "reserved standard namespace") {
			t.Fatalf("namespace %q was not reserved: %v", namespace, err)
		}
	}
}

func TestSemanticExtensionReferenceNamespaceRemainsAvailable(t *testing.T) {
	descriptor := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.example",
		Namespace: "reference.example",
		Version:   "1.0.0",
	}
	if _, err := normalizeDescriptor(descriptor); err != nil {
		t.Fatalf("reference namespace was rejected: %v", err)
	}
}
