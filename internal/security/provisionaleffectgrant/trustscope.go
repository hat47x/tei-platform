package provisionaleffectgrant

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"tei/internal/sourceacceptance/hoststore"
)

// Trust scope (optional, per trust file entry).
//
// A trust entry may carry two extra members next to the ones hoststore knows:
//
//	"capabilities":     ["order_entry.submit_order", ...]  exact, case-sensitive
//	"resourcePatterns": ["order/42", "order/*", ...]       exact, or prefix wildcard
//
// A key with neither member is trusted for every Capability and Resource of its
// Applications (the behaviour before this field existed). A key with one of them
// is limited on that dimension only; the other dimension stays unrestricted.
//
// Resource pattern rules (no regular expressions):
//   - Without "*" the pattern is an exact match.
//   - With "*" it must be the LAST character and the character before it must
//     be "/" (for example "order/*"). A bare "*" (allow everything) is refused:
//     express "everything" by omitting the member. "order*", "a*b", "**" and
//     "*/x" are refused.
//   - "order/*" matches "order/" followed by at least one character, so it does
//     not match "order/" itself, "order" or "orders/1".
//   - A "." or ".." path segment is refused in a pattern, and a Grant Resource
//     containing such a segment never matches a wildcard pattern (it can only
//     match an exact pattern).
//
// Matching is on the Resource string exactly as signed: no normalisation,
// decoding or case folding. A Host that builds the Resource from external input
// (a URL, a path) must normalise it BEFORE it asks for a Grant; that is the
// Host's responsibility, not this package's.
//
// The scope is NOT read by hoststore: the extra members are removed here before
// the shared strict hoststore parser sees the document, so the Source
// Acceptance trust file keeps refusing them as unknown members.
const (
	maxScopeEntries = 64

	errTrustInvalid    = "HOST_STORE_TRUST_INVALID"
	errScopeInvalid    = "PROVISIONAL_GRANT_TRUST_SCOPE_INVALID"
	errOutOfTrustScope = "PROVISIONAL_GRANT_OUT_OF_TRUST_SCOPE"
)

// Scope limits what one trusted key may sign. A nil slice means unrestricted.
type Scope struct {
	Capabilities     []string `json:"capabilities,omitempty"`
	ResourcePatterns []string `json:"resourcePatterns,omitempty"`
}

// Restricted reports whether the scope limits anything.
func (s Scope) Restricted() bool { return s.Capabilities != nil || s.ResourcePatterns != nil }

// Allows reports whether a Grant for capability and resource is inside s.
func (s Scope) Allows(capability, resource string) bool {
	if s.Capabilities != nil && !containsString(s.Capabilities, capability) {
		return false
	}
	if s.ResourcePatterns != nil {
		for _, p := range s.ResourcePatterns {
			if resourceMatches(p, resource) {
				return true
			}
		}
		return false
	}
	return true
}

func containsString(values []string, v string) bool {
	for _, x := range values {
		if x == v {
			return true
		}
	}
	return false
}

func hasDotSegment(s string) bool {
	for _, seg := range strings.Split(s, "/") {
		if seg == "." || seg == ".." {
			return true
		}
	}
	return false
}

func resourceMatches(pattern, resource string) bool {
	if !strings.HasSuffix(pattern, "*") {
		return pattern == resource
	}
	prefix := strings.TrimSuffix(pattern, "*")
	return strings.HasPrefix(resource, prefix) && len(resource) > len(prefix) && !hasDotSegment(resource)
}

func validCapabilityName(s string) bool { return plainField(s) && !strings.Contains(s, "*") }

func validResourcePattern(p string) bool {
	if !plainField(p) || hasDotSegment(p) {
		return false
	}
	switch strings.Count(p, "*") {
	case 0:
		return true
	case 1:
		return strings.HasSuffix(p, "/*")
	}
	return false
}

func parseScopeList(raw json.RawMessage, valid func(string) bool) ([]string, error) {
	var list []string
	if err := json.Unmarshal(raw, &list); err != nil || len(list) == 0 || len(list) > maxScopeEntries {
		return nil, errors.New(errScopeInvalid)
	}
	seen := map[string]bool{}
	for _, v := range list {
		if !valid(v) || seen[v] {
			return nil, errors.New(errScopeInvalid)
		}
		seen[v] = true
	}
	return list, nil
}

// ScopedIssuer is a trust entry with its optional scope; it is the shape a
// writer (the CLI) marshals.
type ScopedIssuer struct {
	hoststore.TrustedIssuer
	Capabilities     []string `json:"capabilities,omitempty"`
	ResourcePatterns []string `json:"resourcePatterns,omitempty"`
}

// TrustFile is the document a writer marshals.
type TrustFile struct {
	Schema  string         `json:"schema"`
	Issuers []ScopedIssuer `json:"issuers"`
}

// Trust is a loaded effect-grant trust file: the shared hoststore policy plus
// the per-key scopes.
type Trust struct {
	Policy hoststore.TrustPolicy
	scopes map[string]Scope
}

func scopeKey(issuer, keyID string) string { return issuer + "\x00" + keyID }

// ScopeFor returns the scope of the entry (issuer, keyID).
func (t Trust) ScopeFor(issuer, keyID string) (Scope, bool) {
	s, ok := t.scopes[scopeKey(issuer, keyID)]
	return s, ok
}

// WithClock returns a copy whose key window is judged at now (nil: time.Now).
func (t Trust) WithClock(now func() time.Time) Trust {
	t.Policy = t.Policy.WithClock(now)
	return t
}

// LoadTrust reads and strictly validates an effect-grant trust file. Errors are
// fixed codes and never carry file content.
func LoadTrust(path string) (Trust, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Trust{}, errors.New("HOST_STORE_TRUST_UNREADABLE")
	}
	return ParseTrust(raw)
}

// readObject reads one JSON object, refusing duplicate keys.
func readObject(dec *json.Decoder) (map[string]json.RawMessage, error) {
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil, errors.New("not an object")
	}
	out := map[string]json.RawMessage{}
	for dec.More() {
		tok, err := dec.Token()
		key, ok := tok.(string)
		if err != nil || !ok {
			return nil, errors.New("bad key")
		}
		if _, dup := out[key]; dup {
			return nil, errors.New("duplicate key")
		}
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		out[key] = v
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return out, nil
}

// ParseTrust validates a trust document. Every hoststore rule applies; in
// addition duplicate members are refused and the optional scope is checked.
func ParseTrust(raw []byte) (Trust, error) {
	invalid := errors.New(errTrustInvalid)
	dec := json.NewDecoder(bytes.NewReader(raw))
	top, err := readObject(dec)
	if err != nil {
		return Trust{}, invalid
	}
	if _, err := dec.Token(); err != io.EOF {
		return Trust{}, invalid
	}
	var items []json.RawMessage
	if err := json.Unmarshal(top["issuers"], &items); err != nil {
		return Trust{}, invalid
	}
	scopes := make([]Scope, len(items))
	cleaned := make([]map[string]json.RawMessage, len(items))
	var scopeErr error
	for i, item := range items {
		d := json.NewDecoder(bytes.NewReader(item))
		obj, err := readObject(d)
		if err != nil {
			return Trust{}, invalid
		}
		if v, ok := obj["capabilities"]; ok {
			delete(obj, "capabilities")
			if scopes[i].Capabilities, err = parseScopeList(v, validCapabilityName); err != nil {
				scopeErr = err
			}
		}
		if v, ok := obj["resourcePatterns"]; ok {
			delete(obj, "resourcePatterns")
			if scopes[i].ResourcePatterns, err = parseScopeList(v, validResourcePattern); err != nil {
				scopeErr = err
			}
		}
		cleaned[i] = obj
	}
	rest, err := json.Marshal(cleaned)
	if err != nil {
		return Trust{}, invalid
	}
	top["issuers"] = rest
	doc, err := json.Marshal(top)
	if err != nil {
		return Trust{}, invalid
	}
	policy, err := hoststore.ParseTrustPolicyWithSchema(doc, TrustSchema)
	if err != nil {
		return Trust{}, err
	}
	if scopeErr != nil {
		return Trust{}, scopeErr
	}
	t := Trust{Policy: policy, scopes: map[string]Scope{}}
	for i, issuer := range policy.Issuers {
		t.scopes[scopeKey(issuer.Issuer, issuer.KeyID)] = scopes[i]
	}
	return t, nil
}
