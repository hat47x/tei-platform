package verification

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestSemanticNormalizeStripsTimestamps verifies that timestamps in the common
// RFC 3339 shapes an evidence bundle records are replaced, so two runs of the
// same meaning at different times normalize identically.
func TestSemanticNormalizeStripsTimestamps(t *testing.T) {
	cases := []string{
		`"generatedAt":"2026-11-11T15:04:05Z"`,
		`"generatedAt":"2026-11-11T15:04:05.123456789+09:00"`,
		`"generatedAt":"2026-11-11 15:04:05"`,
	}
	for _, body := range cases {
		got := SemanticNormalize(body)
		if !strings.Contains(got, "<TIMESTAMP>") {
			t.Errorf("%s: timestamp was not normalized: %s", body, got)
		}
		if strings.Contains(got, "2026-11-11") {
			t.Errorf("%s: raw timestamp survived normalization: %s", body, got)
		}
	}
}

// TestSemanticNormalizeAnonymizesUUIDs verifies UUIDs collapse to one token.
func TestSemanticNormalizeAnonymizesUUIDs(t *testing.T) {
	body := `"id":"3f2504e0-4f89-11d3-9a0c-0305e82c3301"`
	got := SemanticNormalize(body)
	if !strings.Contains(got, "<UUID>") {
		t.Fatalf("UUID was not normalized: %s", got)
	}
	if strings.Contains(got, "3f2504e0") {
		t.Fatalf("raw UUID survived normalization: %s", got)
	}
}

// TestSemanticNormalizeCollapsesPathPrefixes verifies a leading directory
// prefix collapses so the same file reached through different roots matches.
func TestSemanticNormalizeCollapsesPathPrefixes(t *testing.T) {
	a := SemanticNormalize(`"path":"/home/one/build/out/order.sql"`)
	b := SemanticNormalize(`"path":"/home/two/build/out/order.sql"`)
	if a != b {
		t.Fatalf("path prefixes were not collapsed:\n a=%s\n b=%s", a, b)
	}
}

// TestSemanticNormalizeBuildMetadata verifies the build-identity members an
// evidence bundle records under internal/artifact.CLIBuildIdentity are
// normalized. These are the values that legitimately differ between a full run
// and an incremental run of the same meaning.
func TestSemanticNormalizeBuildMetadata(t *testing.T) {
	full := `{"cli":{"goVersion":"go1.26.10","vcsRevision":"0ba87a4e0a96f70a198b03ff7652fb00e3826abe","declaredRevision":"0ba87a4e0a96f70a198b03ff7652fb00e3826abe"}}`
	incremental := `{"cli":{"goVersion":"go1.27.1","vcsRevision":"2a9a14c071b7e45365541425554f78a5d17583dc","declaredRevision":"2a9a14c071b7e45365541425554f78a5d17583dc"}}`

	gotFull := SemanticNormalize(full)
	gotIncremental := SemanticNormalize(incremental)

	if gotFull != gotIncremental {
		t.Fatalf("build metadata was not normalized:\n full=%s\n inc =%s", gotFull, gotIncremental)
	}
	for _, raw := range []string{"go1.26.10", "go1.27.1", "0ba87a4e0a96", "2a9a14c071b7"} {
		if strings.Contains(gotFull, raw) {
			t.Errorf("raw build value %q survived normalization: %s", raw, gotFull)
		}
	}
}

// TestSemanticNormalizeKeepsJSONValid verifies normalization replaces only the
// member value, so the result still parses as JSON. A rule that ate the closing
// quote would silently corrupt every semantic comparison.
func TestSemanticNormalizeKeepsJSONValid(t *testing.T) {
	body := `{"schemaVersion":"tei.cli.artifacts.verification/v0","worldFingerprint":"abc123","cli":{"vcsRevision":"0ba87a4e0a96f70a198b03ff7652fb00e3826abe","goVersion":"go1.26.10"}}`
	got := SemanticNormalize(body)

	var decoded map[string]any
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("normalized output is not valid JSON: %v\noutput: %s", err, got)
	}
	cli, ok := decoded["cli"].(map[string]any)
	if !ok {
		t.Fatalf("normalized output lost the cli object: %s", got)
	}
	if cli["vcsRevision"] != "<BUILD_METADATA>" {
		t.Errorf("vcsRevision was not normalized to the placeholder: %v", cli["vcsRevision"])
	}
	if cli["goVersion"] != "<BUILD_METADATA>" {
		t.Errorf("goVersion was not normalized to the placeholder: %v", cli["goVersion"])
	}
	// A member that is not build metadata must keep its value.
	if decoded["worldFingerprint"] != "abc123" {
		t.Errorf("unrelated member was altered: %v", decoded["worldFingerprint"])
	}
}

// TestSemanticNormalizeIsIdempotent verifies applying the rules twice does not
// change the result further, so a pipeline that normalizes at several stages
// still converges.
func TestSemanticNormalizeIsIdempotent(t *testing.T) {
	body := `{"generatedAt":"2026-11-11T15:04:05Z","id":"3f2504e0-4f89-11d3-9a0c-0305e82c3301","path":"/home/one/out/order.sql","cli":{"vcsRevision":"0ba87a4e0a96f70a198b03ff7652fb00e3826abe"}}`
	once := SemanticNormalize(body)
	twice := SemanticNormalize(once)
	if once != twice {
		t.Fatalf("normalization is not idempotent:\n once=%s\n twice=%s", once, twice)
	}
}
