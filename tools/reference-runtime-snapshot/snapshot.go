// Package runtimesnapshot is a read-only reference for consuming a precomputed
// Web Target manifest without importing the TEI Workspace or Driver Resolver.
// It has no business execution, authorization, or build-publishing authority.
package runtimesnapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Identity is the binding expected by the caller from an independently
// accepted handover and target build. It is NOT derived from the manifest.
type Identity struct {
	ApplicationID              string
	HandoverBaselineID         string
	BaselineRepositoryRevision string
	TargetBuildRevision        string
	ManifestSHA256             string
}

// Snapshot owns immutable bytes and can serve only the original manifest.
// The caller must bind the expected identity/digest independently; computing
// an expected digest from the same untrusted input would prove nothing.
type Snapshot struct {
	manifest []byte
}

func readTopLevelStrings(data []byte) (map[string]string, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	start, err := dec.Token()
	if err != nil || start != json.Delim('{') {
		return nil, fmt.Errorf("manifest must be a JSON object")
	}
	seen := make(map[string]bool)
	values := make(map[string]string)
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("read manifest property: %w", err)
		}
		key, ok := token.(string)
		if !ok || seen[key] {
			return nil, fmt.Errorf("duplicate or invalid manifest property %q", key)
		}
		seen[key] = true
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, fmt.Errorf("read manifest property %q: %w", key, err)
		}
		switch key {
		case "applicationId", "handoverBaselineId", "baselineRepositoryRevision", "targetBuildRevision":
			var value string
			if err := json.Unmarshal(raw, &value); err != nil || strings.TrimSpace(value) == "" {
				return nil, fmt.Errorf("manifest property %q must be a nonempty string", key)
			}
			values[key] = value
		}
	}
	if end, err := dec.Token(); err != nil || end != json.Delim('}') {
		return nil, fmt.Errorf("manifest has invalid ending")
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("manifest has trailing content")
	}
	return values, nil
}

// Accept checks a frozen manifest against independently supplied build/handover
// identity and the exact raw-byte digest. It never interprets Pattern meaning.
func Accept(raw []byte, expected Identity) (Snapshot, error) {
	if strings.TrimSpace(expected.ApplicationID) == "" ||
		strings.TrimSpace(expected.HandoverBaselineID) == "" ||
		strings.TrimSpace(expected.BaselineRepositoryRevision) == "" ||
		strings.TrimSpace(expected.TargetBuildRevision) == "" {
		return Snapshot{}, fmt.Errorf("expected application/handover/target identity is incomplete")
	}
	if len(expected.ManifestSHA256) != 64 {
		return Snapshot{}, fmt.Errorf("expected manifest SHA-256 must be 64 hex characters")
	}
	decodedDigest, err := hex.DecodeString(expected.ManifestSHA256)
	if err != nil || len(decodedDigest) != sha256.Size {
		return Snapshot{}, fmt.Errorf("expected manifest SHA-256 is invalid")
	}
	actual := sha256.Sum256(raw)
	if !bytes.Equal(actual[:], decodedDigest) {
		return Snapshot{}, fmt.Errorf("manifest bytes do not match pinned SHA-256")
	}
	values, err := readTopLevelStrings(raw)
	if err != nil {
		return Snapshot{}, err
	}
	for _, binding := range []struct{ property, expected string }{
		{"applicationId", expected.ApplicationID},
		{"handoverBaselineId", expected.HandoverBaselineID},
		{"baselineRepositoryRevision", expected.BaselineRepositoryRevision},
		{"targetBuildRevision", expected.TargetBuildRevision},
	} {
		if values[binding.property] != binding.expected {
			return Snapshot{}, fmt.Errorf("manifest %s does not match independent binding", binding.property)
		}
	}
	return Snapshot{manifest: append([]byte(nil), raw...)}, nil
}

// Handler provides only GET /manifest.json and GET /healthz. In particular,
// no business/command endpoint is registered: this reference cannot silently
// claim an authorization check or execution result it did not perform.
func (s Snapshot) Handler() http.Handler {
	if len(s.manifest) == 0 {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "unaccepted projection", http.StatusServiceUnavailable)
		})
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write(s.manifest)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}
