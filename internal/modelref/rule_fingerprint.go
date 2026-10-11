package modelref

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

// RuleFingerprint hashes the exact extension-provided edge policy independent
// of registration order. A digest is not a signature or Acceptance evidence;
// producers and consumers must additionally pin the plugin/version/source.
// Rule validity and conflicting definitions are checked by Validate.
func RuleFingerprint(rules []Rule) string {
	records := make([]string, 0, len(rules))
	for _, rule := range rules {
		encoded, _ := json.Marshal(rule) // Rule contains only fixed scalar fields.
		records = append(records, string(encoded))
	}
	sort.Strings(records)
	unique := records[:0]
	for _, record := range records {
		if len(unique) == 0 || unique[len(unique)-1] != record {
			unique = append(unique, record)
		}
	}
	sum := sha256.Sum256([]byte("tei.modelref.rules.v0\n" + strings.Join(unique, "\n")))
	return "sha256:" + hex.EncodeToString(sum[:])
}
