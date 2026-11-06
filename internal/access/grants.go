package access

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/core"
	"tei/internal/permission"
)

// Entry is one grant in the file an adapter reads. IssuedBy names the party that
// decided the grant; the adapter refuses an entry without it, so a grant that
// merely exists is not the same as a grant that someone decided.
type Entry struct {
	IssuedBy       string           `json:"issuedBy"`
	CapabilityKind string           `json:"capabilityKind,omitempty"`
	CapabilityID   core.CanonicalID `json:"capabilityId,omitempty"`
	Kind           string           `json:"kind"`
	Access         string           `json:"access"`
	Scope          string           `json:"scope"`
}

// Document is the grants file. It is a plain list so a tool does not have to
// depend on the workspace model to read it.
type Document struct {
	Application string  `json:"application,omitempty"`
	Grants      []Entry `json:"grants"`
}

// EncodeDocument renders the file bytes. Every entry must name its issuer: a
// writer that cannot say who decided cannot produce a file an adapter trusts.
func EncodeDocument(document Document) ([]byte, error) {
	if document.Grants == nil {
		document.Grants = []Entry{}
	}
	for index, entry := range document.Grants {
		if entry.IssuedBy == "" {
			return nil, fmt.Errorf("grant %d does not name who issued it", index+1)
		}
	}
	raw, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode grants: %w", err)
	}
	return append(raw, '\n'), nil
}

// WriteDocument writes the grants file. A grant file decides what a run may
// touch, so it is written owner-only: a process that can rewrite it can widen
// its own access.
func WriteDocument(path string, document Document) error {
	raw, err := EncodeDocument(document)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("prepare %s: %w", path, err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// ParseDocument reads the grants file without discarding the issuer, so a caller
// can decide whether it accepts who issued the grant. An entry whose permission
// cannot be understood is an error.
func ParseDocument(data []byte) (Document, error) {
	var document Document
	if err := json.Unmarshal(data, &document); err != nil {
		return Document{}, fmt.Errorf("parse grants: %w", err)
	}
	for index, entry := range document.Grants {
		spec := permission.Spec{
			Kind:   permission.Kind(entry.Kind),
			Access: permission.Access(entry.Access),
			Scope:  entry.Scope,
		}
		if err := spec.Validate(); err != nil {
			return Document{}, fmt.Errorf("grant %d (%s %s %s): %w", index+1, entry.Kind, entry.Access, entry.Scope, err)
		}
	}
	return document, nil
}

// ParseGrants reads the grants file as permissions. It keeps the historical
// shape for callers that only need what is granted; a caller that must judge
// who issued the grant uses ParseDocument instead.
func ParseGrants(data []byte) ([]permission.Spec, error) {
	document, err := ParseDocument(data)
	if err != nil {
		return nil, err
	}
	grants := make([]permission.Spec, 0, len(document.Grants))
	for _, entry := range document.Grants {
		grants = append(grants, permission.Spec{
			Kind:   permission.Kind(entry.Kind),
			Access: permission.Access(entry.Access),
			Scope:  entry.Scope,
		})
	}
	return grants, nil
}

// AcceptIssuedBy reports whether the document may be used by capabilityID: every
// entry must name an issuer, and that issuer must not be capabilityID itself.
// This refuses a grant a capability wrote for itself; it cannot prove that a
// third party approved the grant, because the issuer name is not signed.
func AcceptIssuedBy(document Document, capabilityID core.CanonicalID) error {
	for index, entry := range document.Grants {
		if entry.IssuedBy == "" {
			return fmt.Errorf("grant %d does not name who issued it", index+1)
		}
		if entry.IssuedBy == string(capabilityID) {
			return fmt.Errorf("grant %d was issued by %s, which is the capability that wants to use it", index+1, capabilityID)
		}
	}
	return nil
}
