// Package semanticdistribution is an isolated, read-only E4 comparison reference.
// It is not a TEI Plugin loader, an artifact store, or an accepted schema.
package semanticdistribution

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

const ReferenceSchema = "tei.reference.semantic-export.v0"

// Identity describes a distribution candidate, independently of the business
// namespace identity. It must not be mistaken for a Runtime Plugin binding.
type Identity struct {
	PackageID string `json:"packageId"`
	Version   string `json:"version"`
}

// Envelope is the proof fixture's data-only content: it cannot supply executable
// plugin code, authorization, a Source Index, or an accepted business model.
type Envelope struct {
	Schema         string             `json:"schema"`
	Package        Identity           `json:"package"`
	Namespace      semantic.Namespace `json:"namespace"`
	SemanticDigest string             `json:"semanticDigest"`
}

// Expectation is chosen by the caller, not derived from untrusted payload bytes.
// The real provider/artifact selector belongs to the Composition Kernel.
type Expectation struct {
	Package        Identity
	NamespaceID    string
	ArtifactDigest string
	SemanticDigest string
}

func validateIdentity(identity Identity) error {
	if _, err := core.ParseCanonicalID(identity.PackageID); err != nil {
		return fmt.Errorf("package id: %w", err)
	}
	if !core.ValidNumericVersion(identity.Version) {
		return fmt.Errorf("package version %q must be major.minor.patch", identity.Version)
	}
	return nil
}

func digestNamespace(ns semantic.Namespace) (string, error) {
	content, err := json.Marshal(ns)
	if err != nil {
		return "", err
	}
	return sourceindex.Fingerprint(content), nil
}

// MaterializeReference creates a deterministic data-only comparison payload.
// A Plugin package builder must not assume this is its production format.
func MaterializeReference(packageID Identity, ns semantic.Namespace) ([]byte, error) {
	if err := validateIdentity(packageID); err != nil {
		return nil, err
	}
	if _, err := core.ParseCanonicalID(string(ns.ID)); err != nil {
		return nil, fmt.Errorf("namespace: %w", err)
	}
	if ns.ID == "std" {
		return nil, errors.New("standard library namespace is not a project-owned export")
	}
	digest, err := digestNamespace(ns)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(Envelope{
		Schema: ReferenceSchema, Package: packageID, Namespace: ns, SemanticDigest: digest,
	})
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}

// InspectSelectedReference validates one explicitly selected package artifact.
// Selection, signature trust, dependency closure and authorization must be
// established outside this function. No runtime code is started here.
func InspectSelectedReference(payload []byte, expected Expectation) (semantic.Namespace, error) {
	if err := validateIdentity(expected.Package); err != nil {
		return semantic.Namespace{}, err
	}
	if expected.NamespaceID == "" || expected.ArtifactDigest == "" || expected.SemanticDigest == "" {
		return semantic.Namespace{}, errors.New("explicit package identity, namespace and both digests are required")
	}
	if sourceindex.Fingerprint(payload) != expected.ArtifactDigest {
		return semantic.Namespace{}, errors.New("distribution artifact digest mismatch")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var value Envelope
	if err := decoder.Decode(&value); err != nil {
		return semantic.Namespace{}, fmt.Errorf("decode data-only export: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return semantic.Namespace{}, errors.New("unexpected trailing content after distribution envelope")
	}
	if value.Schema != ReferenceSchema {
		return semantic.Namespace{}, fmt.Errorf("unsupported reference schema %q", value.Schema)
	}
	if value.Package != expected.Package {
		return semantic.Namespace{}, fmt.Errorf("distribution identity mismatch: expected %s@%s, got %s@%s",
			expected.Package.PackageID, expected.Package.Version, value.Package.PackageID, value.Package.Version)
	}
	if string(value.Namespace.ID) != expected.NamespaceID {
		return semantic.Namespace{}, fmt.Errorf("exported namespace mismatch: expected %s, got %s", expected.NamespaceID, value.Namespace.ID)
	}
	digest, err := digestNamespace(value.Namespace)
	if err != nil {
		return semantic.Namespace{}, err
	}
	if digest != value.SemanticDigest || digest != expected.SemanticDigest {
		return semantic.Namespace{}, fmt.Errorf("semantic digest mismatch for %s", expected.NamespaceID)
	}
	if !strings.HasPrefix(digest, "sha256:") {
		return semantic.Namespace{}, errors.New("unsupported semantic digest")
	}
	return value.Namespace, nil
}
