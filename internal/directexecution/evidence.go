package directexecution

// Evidence is the output schema for a Direct execution baseline.
// It separates four independent pieces of evidence:
// 1. Execution request accepted
// 2. Worker executed
// 3. Result metadata obtained
// 4. Actual output verified
type Evidence struct {
	Schema string `json:"schema"`

	// ExecutionRequestAccepted records that the execution request was accepted.
	ExecutionRequestAccepted ExecutionRequestAccepted `json:"execution_request_accepted"`

	// WorkerExecuted records that the worker actually executed.
	WorkerExecuted WorkerExecuted `json:"worker_executed"`

	// ResultMetadataObtained records that result metadata was obtained.
	ResultMetadataObtained ResultMetadataObtained `json:"result_metadata_obtained"`

	// ActualOutputVerified records that the actual output was independently verified.
	ActualOutputVerified ActualOutputVerified `json:"actual_output_verified"`

	// ExecutionEligibility records whether the execution was eligible.
	ExecutionEligibility ExecutionEligibility `json:"execution_eligibility"`

	// PlacementAuthority records whether placement authority was granted.
	PlacementAuthority PlacementAuthority `json:"placement_authority"`
}

// ExecutionRequestAccepted records acceptance of the execution request.
type ExecutionRequestAccepted struct {
	Accepted         bool   `json:"accepted"`
	RequestID        string `json:"request_id,omitempty"`
	AcceptedAtMillis int64  `json:"accepted_at_millis,omitempty"`
	Authority        string `json:"authority,omitempty"`
}

// WorkerExecuted records that the worker actually ran.
type WorkerExecuted struct {
	Executed    bool   `json:"executed"`
	WorkerID    string `json:"worker_id,omitempty"`
	StartMillis int64  `json:"start_millis,omitempty"`
	EndMillis   int64  `json:"end_millis,omitempty"`
	ExitCode    int    `json:"exit_code,omitempty"`
	ProcessID   int    `json:"process_id,omitempty"`
}

// ResultMetadataObtained records that result metadata was obtained.
type ResultMetadataObtained struct {
	Obtained         bool   `json:"obtained"`
	ResultID         string `json:"result_id,omitempty"`
	OutputDigest     string `json:"output_digest,omitempty"`
	ObtainedAtMillis int64  `json:"obtained_at_millis,omitempty"`
	Authority        string `json:"authority,omitempty"`
}

// ActualOutputVerified records file bytes read and, when supplied, comparison
// to the caller-supplied digest. It does not establish output provenance.
type ActualOutputVerified struct {
	Verified         bool   `json:"verified"`
	OutputPath       string `json:"output_path,omitempty"`
	SizeBytes        int64  `json:"size_bytes,omitempty"`
	SHA256Hex        string `json:"sha256_hex,omitempty"`
	VerifiedAtMillis int64  `json:"verified_at_millis,omitempty"`
	Verifier         string `json:"verifier,omitempty"`
}

// ExecutionEligibility records whether execution was eligible.
type ExecutionEligibility struct {
	Eligible         bool `json:"eligible"`
	LeaseValid       bool `json:"lease_valid,omitempty"`
	WithinDeadline   bool `json:"within_deadline,omitempty"`
	RecoveryPossible bool `json:"recovery_possible,omitempty"`
}

// PlacementAuthority records whether placement authority was granted.
type PlacementAuthority struct {
	Granted         bool   `json:"granted"`
	PlacementID     string `json:"placement_id,omitempty"`
	Authority       string `json:"authority,omitempty"`
	GrantedAtMillis int64  `json:"granted_at_millis,omitempty"`
}

// NegativeTestCase represents a negative test case for execution.
type NegativeTestCase struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Expected    bool   `json:"expected"` // true means the negative condition should be detected
}

const EvidenceSchema = "tei.direct-execution-evidence/v0"
