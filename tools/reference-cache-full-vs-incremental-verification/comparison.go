package verification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

// CompareArtifacts compares two artifact sets using the selected mode and
// returns the list of differences. It never mutates its inputs.
func CompareArtifacts(fullArtifacts, incrementalArtifacts []ArtifactRef, mode ComparisonMode) []DiffEntry {
	var diffs []DiffEntry

	fullIndex := indexArtifacts(fullArtifacts)
	incIndex := indexArtifacts(incrementalArtifacts)

	// Check for missing or extra artifacts.
	for path, full := range fullIndex {
		inc, ok := incIndex[path]
		if !ok {
			diffs = append(diffs, DiffEntry{
				Kind:         DiffArtifactMissing,
				ArtifactPath: path,
				Expected:     fmt.Sprintf("present (%s, %d bytes)", full.Digest, full.ByteSize),
				Actual:       "missing",
				Message:      "artifact present in full run but missing in incremental run",
				Severity:     "error",
			})
			continue
		}
		diffs = append(diffs, compareSingleArtifact(path, full, inc, mode)...)
	}
	for path, inc := range incIndex {
		if _, ok := fullIndex[path]; !ok {
			diffs = append(diffs, DiffEntry{
				Kind:         DiffArtifactMissing,
				ArtifactPath: path,
				Expected:     "missing",
				Actual:       fmt.Sprintf("present (%s, %d bytes)", inc.Digest, inc.ByteSize),
				Message:      "artifact present in incremental run but missing in full run",
				Severity:     "error",
			})
		}
	}

	sortDiffs(diffs)
	return diffs
}

// CompareDependencies compares two dependency sets and returns differences.
func CompareDependencies(fullDeps, incrementalDeps []DependencyRef) []DiffEntry {
	var diffs []DiffEntry

	fullIndex := indexDependencies(fullDeps)
	incIndex := indexDependencies(incrementalDeps)

	for id, full := range fullIndex {
		inc, ok := incIndex[id]
		if !ok {
			diffs = append(diffs, DiffEntry{
				Kind:    DiffDependencyMismatch,
				Message: fmt.Sprintf("dependency %s missing in incremental run", id),
				Expected: fmt.Sprintf("version=%s epoch=%d fingerprint=%s",
					full.Version, full.Epoch, full.Fingerprint),
				Actual:   "missing",
				Severity: "error",
			})
			continue
		}
		if full.Version != inc.Version || full.Epoch != inc.Epoch || full.Fingerprint != inc.Fingerprint {
			diffs = append(diffs, DiffEntry{
				Kind:    DiffDependencyMismatch,
				Message: fmt.Sprintf("dependency %s differs between runs", id),
				Expected: fmt.Sprintf("version=%s epoch=%d fingerprint=%s",
					full.Version, full.Epoch, full.Fingerprint),
				Actual: fmt.Sprintf("version=%s epoch=%d fingerprint=%s",
					inc.Version, inc.Epoch, inc.Fingerprint),
				Severity: "error",
			})
		}
	}
	for id, inc := range incIndex {
		if _, ok := fullIndex[id]; !ok {
			diffs = append(diffs, DiffEntry{
				Kind:     DiffDependencyMismatch,
				Message:  fmt.Sprintf("dependency %s present in incremental run but missing in full run", id),
				Expected: "missing",
				Actual: fmt.Sprintf("version=%s epoch=%d fingerprint=%s",
					inc.Version, inc.Epoch, inc.Fingerprint),
				Severity: "error",
			})
		}
	}

	sortDiffs(diffs)
	return diffs
}

// CompareInputs compares two input sets and returns differences.
func CompareInputs(fullInputs, incrementalInputs []string) []DiffEntry {
	var diffs []DiffEntry

	fullSet := make(map[string]bool, len(fullInputs))
	for _, inp := range fullInputs {
		fullSet[inp] = true
	}
	incSet := make(map[string]bool, len(incrementalInputs))
	for _, inp := range incrementalInputs {
		incSet[inp] = true
	}

	for inp := range fullSet {
		if !incSet[inp] {
			diffs = append(diffs, DiffEntry{
				Kind:     DiffInputMismatch,
				Message:  fmt.Sprintf("input %s present in full run but missing in incremental run", inp),
				Expected: "present",
				Actual:   "missing",
				Severity: "error",
			})
		}
	}
	for inp := range incSet {
		if !fullSet[inp] {
			diffs = append(diffs, DiffEntry{
				Kind:     DiffInputMismatch,
				Message:  fmt.Sprintf("input %s present in incremental run but missing in full run", inp),
				Expected: "missing",
				Actual:   "present",
				Severity: "error",
			})
		}
	}

	sortDiffs(diffs)
	return diffs
}

// CompareCacheBehavior compares cache behavior between runs.
func CompareCacheBehavior(fullCache, incrementalCache []CacheEntryRef) []DiffEntry {
	var diffs []DiffEntry

	fullHits := countHits(fullCache)
	incHits := countHits(incrementalCache)

	if fullHits != incHits {
		diffs = append(diffs, DiffEntry{
			Kind:     DiffCacheBehaviorMismatch,
			Message:  fmt.Sprintf("cache hit count differs: full=%d incremental=%d", fullHits, incHits),
			Expected: fmt.Sprintf("%d hits", fullHits),
			Actual:   fmt.Sprintf("%d hits", incHits),
			Severity: "warning",
		})
	}

	// Check for stale entries reused in incremental but not in full.
	for _, inc := range incrementalCache {
		if inc.Hit && inc.StaleReason != "" {
			foundStale := false
			for _, full := range fullCache {
				if full.Key == inc.Key && full.Hit && full.StaleReason == inc.StaleReason {
					foundStale = true
					break
				}
			}
			if !foundStale {
				diffs = append(diffs, DiffEntry{
					Kind:     DiffCacheBehaviorMismatch,
					Message:  fmt.Sprintf("cache entry %s reused with stale reason %q in incremental run only", inc.Key, inc.StaleReason),
					Expected: "not reused",
					Actual:   fmt.Sprintf("reused (stale reason=%s)", inc.StaleReason),
					Severity: "error",
				})
			}
		}
	}

	sortDiffs(diffs)
	return diffs
}

// CompareEvidence compares evidence sets between runs.
func CompareEvidence(fullEvidence, incrementalEvidence []EvidenceRef) []DiffEntry {
	var diffs []DiffEntry

	if len(fullEvidence) != len(incrementalEvidence) {
		diffs = append(diffs, DiffEntry{
			Kind:     DiffEvidenceMismatch,
			Message:  fmt.Sprintf("evidence count differs: full=%d incremental=%d", len(fullEvidence), len(incrementalEvidence)),
			Expected: fmt.Sprintf("%d entries", len(fullEvidence)),
			Actual:   fmt.Sprintf("%d entries", len(incrementalEvidence)),
			Severity: "warning",
		})
	}

	// Compare individual evidence entries by operation and status.
	fullIndex := indexEvidence(fullEvidence)
	incIndex := indexEvidence(incrementalEvidence)

	for op, full := range fullIndex {
		inc, ok := incIndex[op]
		if !ok {
			diffs = append(diffs, DiffEntry{
				Kind:     DiffEvidenceMismatch,
				Message:  fmt.Sprintf("evidence for operation %q missing in incremental run", op),
				Expected: fmt.Sprintf("status=%s errorClass=%s", full.Status, full.ErrorClass),
				Actual:   "missing",
				Severity: "error",
			})
			continue
		}
		if full.Status != inc.Status || full.ErrorClass != inc.ErrorClass || full.ReceiptVersion != inc.ReceiptVersion {
			diffs = append(diffs, DiffEntry{
				Kind:     DiffEvidenceMismatch,
				Message:  fmt.Sprintf("evidence for operation %q differs between runs", op),
				Expected: fmt.Sprintf("status=%s errorClass=%s receiptVersion=%d", full.Status, full.ErrorClass, full.ReceiptVersion),
				Actual:   fmt.Sprintf("status=%s errorClass=%s receiptVersion=%d", inc.Status, inc.ErrorClass, inc.ReceiptVersion),
				Severity: "error",
			})
		}
	}

	sortDiffs(diffs)
	return diffs
}

// CompareReceipts compares receipt sets between runs.
func CompareReceipts(fullReceipts, incrementalReceipts []string) []DiffEntry {
	var diffs []DiffEntry

	fullSet := make(map[string]bool, len(fullReceipts))
	for _, r := range fullReceipts {
		fullSet[r] = true
	}
	incSet := make(map[string]bool, len(incrementalReceipts))
	for _, r := range incrementalReceipts {
		incSet[r] = true
	}

	for r := range fullSet {
		if !incSet[r] {
			diffs = append(diffs, DiffEntry{
				Kind:     DiffReceiptMismatch,
				Message:  fmt.Sprintf("receipt %s present in full run but missing in incremental run", r),
				Expected: "present",
				Actual:   "missing",
				Severity: "error",
			})
		}
	}
	for r := range incSet {
		if !fullSet[r] {
			diffs = append(diffs, DiffEntry{
				Kind:     DiffReceiptMismatch,
				Message:  fmt.Sprintf("receipt %s present in incremental run but missing in full run", r),
				Expected: "missing",
				Actual:   "present",
				Severity: "error",
			})
		}
	}

	sortDiffs(diffs)
	return diffs
}

// MergeReports combines two run reports into a verification report.
func MergeReports(full, incremental RunReport, mode ComparisonMode, scenarioID, scenarioName string) VerificationReport {
	diffs := CompareArtifacts(full.Artifacts, incremental.Artifacts, mode)
	diffs = append(diffs, CompareDependencies(full.Dependencies, incremental.Dependencies)...)
	diffs = append(diffs, CompareInputs(full.Inputs, incremental.Inputs)...)
	diffs = append(diffs, CompareCacheBehavior(full.CacheEntries, incremental.CacheEntries)...)
	diffs = append(diffs, CompareEvidence(full.Evidence, incremental.Evidence)...)
	diffs = append(diffs, CompareReceipts(full.Receipts, incremental.Receipts)...)

	// If the full run failed but the incremental run succeeded, that is a
	// false pass: the incremental path silently dropped an error.
	if !full.OK && incremental.OK {
		diffs = append(diffs, DiffEntry{
			Kind:     DiffCacheBehaviorMismatch,
			Message:  "full run failed but incremental run succeeded; cache may have hidden an error",
			Expected: "failure",
			Actual:   "success",
			Severity: "error",
		})
	}

	return VerificationReport{
		SchemaVersion:     "tei.verification.cache-comparison/v0",
		OK:                len(diffs) == 0,
		ScenarioID:        scenarioID,
		ScenarioName:      scenarioName,
		FullRun:           full,
		IncrementalRun:    incremental,
		ComparisonMode:    mode,
		Diffs:             diffs,
		CounterexampleIDs: []string{},
		Notes:             []string{},
	}
}

// compareSingleArtifact compares two artifacts with the same path using the
// selected mode and returns any differences.
func compareSingleArtifact(path string, full, inc ArtifactRef, mode ComparisonMode) []DiffEntry {
	var diffs []DiffEntry

	// Digest mismatch always indicates a difference.
	if full.Digest != inc.Digest {
		switch mode {
		case ModeByteExact:
			diffs = append(diffs, DiffEntry{
				Kind:         DiffArtifactByteMismatch,
				ArtifactPath: path,
				Expected:     full.Digest,
				Actual:       inc.Digest,
				Message:      fmt.Sprintf("artifact digest differs (byte-exact): full=%s incremental=%s", full.Digest, inc.Digest),
				Severity:     "error",
			})
		case ModeStructural:
			// Structural comparison requires content; if we only have digests,
			// report a structural mismatch with the digest delta.
			diffs = append(diffs, DiffEntry{
				Kind:         DiffArtifactStructuralMismatch,
				ArtifactPath: path,
				Expected:     full.Digest,
				Actual:       inc.Digest,
				Message:      fmt.Sprintf("artifact digest differs (structural): full=%s incremental=%s", full.Digest, inc.Digest),
				Severity:     "error",
			})
		case ModeSemantic:
			diffs = append(diffs, DiffEntry{
				Kind:         DiffArtifactSemanticMismatch,
				ArtifactPath: path,
				Expected:     full.Digest,
				Actual:       inc.Digest,
				Message:      fmt.Sprintf("artifact digest differs (semantic): full=%s incremental=%s", full.Digest, inc.Digest),
				Severity:     "error",
			})
		}
	}

	// Size mismatch is a secondary signal.
	if full.ByteSize != inc.ByteSize {
		diffs = append(diffs, DiffEntry{
			Kind:         DiffArtifactByteMismatch,
			ArtifactPath: path,
			Expected:     fmt.Sprintf("%d bytes", full.ByteSize),
			Actual:       fmt.Sprintf("%d bytes", inc.ByteSize),
			Message:      "artifact size differs between runs",
			Severity:     "warning",
		})
	}

	return diffs
}

// indexArtifacts builds a path->ArtifactRef index.
func indexArtifacts(artifacts []ArtifactRef) map[string]ArtifactRef {
	idx := make(map[string]ArtifactRef, len(artifacts))
	for _, a := range artifacts {
		idx[a.Path] = a
	}
	return idx
}

// indexDependencies builds an ID->DependencyRef index.
func indexDependencies(deps []DependencyRef) map[string]DependencyRef {
	idx := make(map[string]DependencyRef, len(deps))
	for _, d := range deps {
		idx[d.ID] = d
	}
	return idx
}

// indexEvidence builds an operation->EvidenceRef index.
func indexEvidence(evidence []EvidenceRef) map[string]EvidenceRef {
	idx := make(map[string]EvidenceRef, len(evidence))
	for _, e := range evidence {
		key := e.Operation + ":" + e.Status + ":" + e.ErrorClass
		idx[key] = e
	}
	return idx
}

// countHits returns the number of cache hits in the slice.
func countHits(entries []CacheEntryRef) int {
	count := 0
	for _, e := range entries {
		if e.Hit {
			count++
		}
	}
	return count
}

// sortDiffs sorts diffs by Kind then ArtifactPath for deterministic output.
func sortDiffs(diffs []DiffEntry) {
	sort.Slice(diffs, func(i, j int) bool {
		if diffs[i].Kind != diffs[j].Kind {
			return diffs[i].Kind < diffs[j].Kind
		}
		return diffs[i].ArtifactPath < diffs[j].ArtifactPath
	})
}

// JSONCanonicalize returns a canonical JSON representation of v, or the string
// representation if JSON marshaling fails. It is used for structural comparison.
func JSONCanonicalize(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	var dst bytes.Buffer
	if err := json.Compact(&dst, b); err != nil {
		return string(b)
	}
	return dst.String()
}
