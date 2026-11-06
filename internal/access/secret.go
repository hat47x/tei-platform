package access

import (
	"context"
	"fmt"

	"tei/internal/core"
	"tei/internal/permission"
)

// SecretReader reads a secret after the check. A test can supply a fake one, so
// the check is exercised without a secret store.
type SecretReader interface {
	ReadSecret(ctx context.Context, name string) (string, error)
}

// Secrets reads secrets on behalf of one capability. The name is checked against
// the granted permissions and recorded before the value is read.
type Secrets struct {
	Capability core.CanonicalID
	Guard      Guard
	Reader     SecretReader
}

// Read returns the secret when the grants cover it.
func (s Secrets) Read(ctx context.Context, name string) (string, error) {
	requirement := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: name}
	decision, err := s.Guard.Check(s.Capability, []permission.Spec{requirement})
	if err != nil {
		return "", err
	}
	if !decision.Allowed {
		return "", fmt.Errorf("capability %s may not read the secret %s: the granted permissions do not cover it", s.Capability, name)
	}
	if s.Reader == nil {
		return "", fmt.Errorf("no secret reader is configured, so %s is not read", name)
	}
	return s.Reader.ReadSecret(ctx, name)
}
