package actionboundary

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrRemoteOutcomeUnknown = errors.New("remote action outcome unknown")

// TrustedDelegation is produced by a server-side identity provider after
// validating the live session and the token's intended upstream audience.
// It is never populated from an Action body or an unverified identity header.
type TrustedDelegation struct {
	SubjectID   string
	TenantID    string
	Audience    string
	BearerToken string
}

// DelegationSource must establish the upstream user's own credential, not
// mint a user identity based on Host service privileges or browser claims.
type DelegationSource interface {
	Delegation(context.Context, Intent) (TrustedDelegation, error)
}

// TenantResolver obtains the tenant from a verified runtime session,
// independently of the application's Action payload and delegation response.
type TenantResolver interface {
	Tenant(context.Context) (string, error)
}

// validBearerToken accepts RFC 6750 b64token syntax, not arbitrary HTTP
// header bytes. Bound its size so a compromised provider cannot produce an
// unbounded secret-bearing request.
func validBearerToken(token string) bool {
	if len(token) == 0 || len(token) > 8192 {
		return false
	}
	padding := false
	for i := 0; i < len(token); i++ {
		b := token[i]
		if b == '=' {
			if i == 0 {
				return false
			}
			padding = true
			continue
		}
		if padding || !((b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
			(b >= '0' && b <= '9') || b == '-' || b == '.' || b == '_' ||
			b == '~' || b == '+' || b == '/') {
			return false
		}
	}
	return true
}

// HTTPSApplicationStore is a reference adapter for a separately hosted
// application-owned resource Action. It is not itself a session verifier:
// Subjects and Delegations must be bound to the SAME verified runtime identity
// and the upstream MUST independently authenticate/authorize the bearer token.
//
// This store is intentionally not registered automatically as a Host Binding.
type HTTPSApplicationStore struct {
	ApplicationID string
	ActionID      string
	// Origin is a fixed administrator-configured HTTPS origin, never from
	// browser JSON or a forwarded Host header. No path, query or userinfo.
	Origin string
	// Prefix and Suffix are fixed deployment paths surrounding an escaped
	// resource ID. Example: "/docs/" + {resourceID} + "/action-commit".
	Prefix string
	Suffix string
	Client *http.Client
	// Audience identifies this deployment's upstream application token
	// audience. It must be checked by the issuer and verified again by SUI.
	Audience string

	Subjects    SubjectResolver
	Tenants     TenantResolver
	Delegations DelegationSource
}

func (s HTTPSApplicationStore) Commit(ctx context.Context, in Intent) (string, error) {
	if in.ApplicationID == "" || in.ActionID == "" || in.ResourceID == "" ||
		in.ExpectedRevision == "" || s.ApplicationID != in.ApplicationID ||
		s.ActionID != in.ActionID || in.Payload == nil {
		return "", ErrInvalid
	}
	if s.Client == nil || s.Subjects == nil || s.Tenants == nil ||
		s.Delegations == nil || s.Audience == "" {
		return "", fmt.Errorf("%w: unconfigured trusted application transport", ErrDenied)
	}
	origin, err := url.Parse(s.Origin)
	if err != nil || origin.Scheme != "https" || origin.Host == "" ||
		origin.User != nil || origin.Path != "" || origin.RawQuery != "" ||
		origin.Fragment != "" || origin.Opaque != "" ||
		origin.ForceQuery || strings.ContainsAny(s.Origin, "?#") {
		return "", fmt.Errorf("%w: invalid upstream HTTPS origin", ErrDenied)
	}
	if !strings.HasPrefix(s.Prefix, "/") || strings.HasPrefix(s.Prefix, "//") ||
		strings.ContainsAny(s.Prefix+s.Suffix, "?#\\\r\n") ||
		strings.Contains(s.Prefix+s.Suffix, "..") {
		return "", fmt.Errorf("%w: invalid configured resource path", ErrDenied)
	}
	subject, err := s.Subjects.Subject(ctx)
	if err != nil || subject == "" {
		return "", fmt.Errorf("%w: authenticated runtime subject unavailable", ErrDenied)
	}
	tenant, err := s.Tenants.Tenant(ctx)
	if err != nil || tenant == "" {
		return "", fmt.Errorf("%w: authenticated runtime tenant unavailable", ErrDenied)
	}
	delegation, err := s.Delegations.Delegation(ctx, in)
	if err != nil || delegation.SubjectID != subject || delegation.TenantID != tenant ||
		delegation.Audience != s.Audience || !validBearerToken(delegation.BearerToken) {
		return "", fmt.Errorf("%w: verified user delegation unavailable or mismatched", ErrDenied)
	}
	// Distinguish an interface holding a typed nil or JSON null from a
	// real application payload. SUI and the generic v1 contract reject null.
	payload, err := json.Marshal(in.Payload)
	if err != nil || bytes.Equal(bytes.TrimSpace(payload), []byte("null")) {
		return "", ErrInvalid
	}
	body, err := json.Marshal(struct {
		ProtocolVersion  string          `json:"protocolVersion"`
		ApplicationID    string          `json:"applicationID"`
		ResourceID       string          `json:"resourceID"`
		ActionID         string          `json:"actionID"`
		ExpectedRevision string          `json:"expectedRevision"`
		Payload          json.RawMessage `json:"payload"`
	}{
		ProtocolVersion: "1", ApplicationID: in.ApplicationID,
		ResourceID: in.ResourceID, ActionID: in.ActionID,
		ExpectedRevision: in.ExpectedRevision, Payload: payload,
	})
	if err != nil || len(body) > 64<<10 {
		return "", ErrInvalid
	}
	target := s.Origin + s.Prefix + url.PathEscape(in.ResourceID) + s.Suffix
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("%w: invalid outbound request", ErrDenied)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-TEI-Action", "commit")
	req.Header.Set("Authorization", "Bearer "+delegation.BearerToken)
	// Force redirects off independently of the injected client's policy:
	// following a redirect could leak the user's bearer credential.
	client := *s.Client
	// Unbounded network waits keep an outcome indeterminate indefinitely.
	// A caller can set a stricter deadline through the client or context.
	if client.Timeout <= 0 {
		client.Timeout = 15 * time.Second
	}
	// Do not inject cookies from an HTTP client reused across identities.
	// The only credential on the wire is the verified delegated token.
	client.Jar = nil
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: upstream request failed", ErrRemoteOutcomeUnknown)
	}
	defer resp.Body.Close()
	limited, err := io.ReadAll(io.LimitReader(resp.Body, (8<<10)+1))
	if err != nil || len(limited) > 8<<10 {
		return "", fmt.Errorf("%w: malformed upstream response", ErrRemoteOutcomeUnknown)
	}
	version, revision, code, err := parseApplicationActionResult(limited)
	if err != nil || version != "1" {
		return "", fmt.Errorf("%w: invalid upstream result contract", ErrRemoteOutcomeUnknown)
	}
	switch {
	case resp.StatusCode == http.StatusOK && revision != "" && code == "":
		return revision, nil
	case resp.StatusCode == http.StatusConflict && code == "revision_conflict" && revision == "":
		// The selected Application Store contract guarantees no write on CAS conflict.
		return "", ErrConflict
	case resp.StatusCode == http.StatusForbidden && code == "action_denied" && revision == "":
		return "", ErrDenied
	case resp.StatusCode == http.StatusBadRequest &&
		(code == "invalid_action" || code == "invalid_payload") && revision == "":
		return "", ErrInvalid
	default:
		// A 5xx, unknown result, redirect, response mismatch or lost reply
		// cannot prove whether the upstream transaction committed.
		return "", fmt.Errorf("%w: unrecognized upstream outcome", ErrRemoteOutcomeUnknown)
	}
}

func parseApplicationActionResult(raw []byte) (version, revision, code string, err error) {
	invalid := errors.New("invalid Action result")
	d := json.NewDecoder(bytes.NewReader(raw))
	first, decErr := d.Token()
	if decErr != nil || first != json.Delim('{') {
		return "", "", "", invalid
	}
	values := map[string]string{}
	for d.More() {
		keyToken, keyErr := d.Token()
		if keyErr != nil {
			return "", "", "", invalid
		}
		key, ok := keyToken.(string)
		if !ok || (key != "protocolVersion" && key != "revision" && key != "error") {
			return "", "", "", invalid
		}
		if _, exists := values[key]; exists {
			return "", "", "", invalid
		}
		var value string
		if decErr := d.Decode(&value); decErr != nil || value == "" {
			return "", "", "", invalid
		}
		values[key] = value
	}
	end, decErr := d.Token()
	if decErr != nil || end != json.Delim('}') {
		return "", "", "", invalid
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return "", "", "", invalid
	}
	if len(values) != 2 {
		return "", "", "", invalid
	}
	version, revision, code = values["protocolVersion"], values["revision"], values["error"]
	if version != "1" || (revision == "") == (code == "") {
		return "", "", "", invalid
	}
	return version, revision, code, nil
}
