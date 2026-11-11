package verification

import (
	"regexp"
	"strings"
)

// timestampPattern matches common timestamp formats in artifacts.
var timestampPattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?`)

// uuidPattern matches UUIDs (with or without hyphens).
var uuidPattern = regexp.MustCompile(`(?i)[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}`)

// pathPrefixPattern matches a path's directory portion: everything from the
// delimiter before the path up to its last separator. The greedy match is what
// collapses the whole prefix rather than only its first segment, so the same
// file reached through different roots normalizes to the same value while the
// file name itself is preserved.
var pathPrefixPattern = regexp.MustCompile(`(^|"|\s)(/[^ "]*)/`)

// buildMetadataPattern matches the JSON members of a build identity whose value
// changes between two builds of the same meaning. The evidence bundle records
// these under their JSON names (internal/artifact.CLIBuildIdentity), so the
// pattern anchors on the member name and replaces only the value, keeping the
// surrounding JSON valid.
var buildMetadataPattern = regexp.MustCompile(`(?i)("(?:vcsRevision|declaredRevision|goVersion)"\s*:\s*")[^"]*(")`)

// SemanticNormalize applies domain-specific normalization rules to artifact
// content so that semantically equivalent artifacts can be compared despite
// differences in timestamps, UUIDs, path prefixes, or build metadata.
func SemanticNormalize(content string) string {
	// Strip timestamps.
	content = timestampPattern.ReplaceAllString(content, "<TIMESTAMP>")
	// Anonymize UUIDs.
	content = uuidPattern.ReplaceAllString(content, "<UUID>")
	// Normalize path prefixes: collapse leading directory segments.
	content = pathPrefixPattern.ReplaceAllString(content, "${1}<PATH>/")
	// Normalize build metadata so a different commit hash or Go toolchain
	// version does not change the meaning of an otherwise identical artifact.
	content = buildMetadataPattern.ReplaceAllString(content, "${1}<BUILD_METADATA>${2}")
	return strings.TrimSpace(content)
}
