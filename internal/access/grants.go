package access

import (
	"encoding/json"
	"fmt"

	"tei/internal/permission"
)

// grantEntry is one grant in the file an adapter reads. It mirrors the document
// `tei grants` writes, so the two stay one format.
type grantEntry struct {
	Kind   string `json:"kind"`
	Access string `json:"access"`
	Scope  string `json:"scope"`
}

type grantDocument struct {
	Grants []grantEntry `json:"grants"`
}

// ParseGrants reads the grants file. An unknown kind or access is an error: a
// grant that cannot be understood must not be treated as covering anything.
func ParseGrants(data []byte) ([]permission.Spec, error) {
	var document grantDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse grants: %w", err)
	}
	grants := make([]permission.Spec, 0, len(document.Grants))
	for _, entry := range document.Grants {
		spec := permission.Spec{
			Kind:   permission.Kind(entry.Kind),
			Access: permission.Access(entry.Access),
			Scope:  entry.Scope,
		}
		if err := spec.Validate(); err != nil {
			return nil, fmt.Errorf("grant %s %s %s: %w", entry.Kind, entry.Access, entry.Scope, err)
		}
		grants = append(grants, spec)
	}
	return grants, nil
}
