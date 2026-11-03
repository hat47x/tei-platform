package provisionaleffectgrant

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	"tei/internal/sourceacceptance/hoststore"
)

// FileSource reads one signed provisional Grant file and a trust file. Both
// are re-read and re-verified on EVERY Read, so a revoked key, an edited
// window or an expired Grant is refused at the next effect without a restart.
// Nothing is cached and no file content appears in errors.
type FileSource struct {
	GrantPath string
	TrustPath string
	// Now is a test/Host clock (nil means time.Now). It is shared by the Grant
	// period and the trust file's key window.
	Now func() time.Time
}

// Read verifies the files against q.
func (s FileSource) Read(ctx context.Context, q Query) (Verified, error) {
	if ctx == nil || ctx.Err() != nil || s.GrantPath == "" || s.TrustPath == "" {
		return Verified{}, errors.New("PROVISIONAL_GRANT_UNAVAILABLE")
	}
	f, err := os.Open(s.GrantPath)
	if err != nil {
		return Verified{}, errors.New("PROVISIONAL_GRANT_UNREADABLE")
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxDocumentBytes+1))
	_ = f.Close()
	if err != nil {
		return Verified{}, errors.New("PROVISIONAL_GRANT_UNREADABLE")
	}
	g, err := Parse(raw)
	if err != nil {
		return Verified{}, err
	}
	policy, err := hoststore.LoadTrustPolicyWithSchema(s.TrustPath, TrustSchema)
	if err != nil {
		return Verified{}, err
	}
	v, err := Verify(g, q, policy.WithClock(s.Now), s.Now)
	if err != nil {
		return Verified{}, err
	}
	// A Reader may be slow; never return a result after cancellation.
	if ctx.Err() != nil {
		return Verified{}, errors.New("PROVISIONAL_GRANT_UNAVAILABLE")
	}
	return v, nil
}
