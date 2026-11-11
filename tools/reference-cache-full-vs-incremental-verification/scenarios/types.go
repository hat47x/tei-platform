package scenarios

import "fmt"

// Scenario defines a test scenario for cache vs full verification.
type Scenario struct {
	ID                string
	Name              string
	Description       string
	ComparisonMode    string
	ChangeType        string
	CounterexampleIDs []string
	Run               func() (VerificationReport, error)
}

// VerificationReport is a simplified version of the parent package's report.
type VerificationReport struct {
	SchemaVersion     string
	OK                bool
	ScenarioID        string
	ScenarioName      string
	FullRun           RunReport
	IncrementalRun    RunReport
	ComparisonMode    string
	Diffs             []DiffEntry
	CounterexampleIDs []string
	Notes             []string
}

// DiffSummary returns a one-line summary suitable for CI output.
func (r VerificationReport) DiffSummary() string {
	if r.OK {
		return fmt.Sprintf("PASS %s: full vs %s produced identical results (%s comparison)",
			r.FullRun.Label, r.IncrementalRun.Label, r.ComparisonMode)
	}
	return fmt.Sprintf("FAIL %s: %d differences detected between full and %s runs",
		r.ScenarioID, len(r.Diffs), r.IncrementalRun.Label)
}

// DiffEntry records one discovered difference.
type DiffEntry struct {
	Kind         string
	ArtifactPath string
	Expected     string
	Actual       string
	Message      string
	Severity     string
}

// RunReport summarizes one execution path.
type RunReport struct {
	Label                  string
	OK                     bool
	Error                  string
	WorldFingerprint       string
	SourceIndexFingerprint string
	Inputs                 []string
	Dependencies           []DependencyRef
	CacheEntries           []CacheEntryRef
	Artifacts              []ArtifactRef
	Evidence               []EvidenceRef
	Receipts               []string
	DurationMs             int64
	SkippedSteps           []string
}

// DependencyRef identifies one dependency observed during a run.
type DependencyRef struct {
	ID          string
	Version     string
	Epoch       uint64
	Fingerprint string
}

// CacheEntryRef identifies one cache entry under test.
type CacheEntryRef struct {
	Key               string
	Hit               bool
	StaleReason       string
	SourceFingerprint string
	RuleFingerprint   string
}

// ArtifactRef identifies one generated artifact.
type ArtifactRef struct {
	Kind      string
	Generator string
	Path      string
	Digest    string
	ByteSize  int
}

// EvidenceRef identifies one piece of execution evidence.
type EvidenceRef struct {
	Operation      string
	Status         string
	ErrorClass     string
	ReceiptVersion uint64
}
