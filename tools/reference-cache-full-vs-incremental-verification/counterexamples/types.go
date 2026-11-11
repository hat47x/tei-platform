package counterexamples

// CounterexampleScenario defines a counterexample scenario for false positives.
type CounterexampleScenario struct {
	ID             string
	Name           string
	Description    string
	ComparisonMode string
	Run            func() (VerificationReport, error)
}

// VerificationReport is a simplified version of the parent package's report.
type VerificationReport struct {
	SchemaVersion  string
	OK             bool
	ScenarioID     string
	ScenarioName   string
	ComparisonMode string
	Diffs          []DiffEntry
	Notes          []string
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
