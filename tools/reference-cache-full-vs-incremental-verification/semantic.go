package verification

import (
	"regexp"
	"strings"
)

// timestampPattern matches common timestamp formats in artifacts.
var timestampPattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?`)

// uuidPattern matches UUIDs (with or without hyphens).
var uuidPattern = regexp.MustCompile(`(?i)[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}`)

// pathPrefixPattern matches a leading path prefix up to a known directory.
var pathPrefixPattern = regexp.MustCompile(`(^|"|\s)(/[^ "]+?)/`)

// SemanticNormalize applies domain-specific normalization rules to artifact
// content so that semantically equivalent artifacts can be compared despite
// differences in timestamps, UUIDs, or path prefixes.
func SemanticNormalize(content string) string {
	// Strip timestamps.
	content = timestampPattern.ReplaceAllString(content, "<TIMESTAMP>")
	// Anonymize UUIDs.
	content = uuidPattern.ReplaceAllString(content, "<UUID>")
	// Normalize path prefixes: collapse leading directory segments.
	content = pathPrefixPattern.ReplaceAllString(content, "${1}<PATH>/")
	return strings.TrimSpace(content)
}
