package xeadsource

type Inspection struct {
	Document                         Document               `json:"document"`
	Candidates                       []Candidate            `json:"candidates"`
	RelationalFacts                  RelationalSourceFacts  `json:"relationalFacts"`
	RelationalDiagnostics            []RelationalDiagnostic `json:"relationalDiagnostics,omitempty"`
	RelationalResolutionStatus       string                 `json:"relationalResolutionStatus"`
	ClassificationPolicyFingerprint  string                 `json:"classificationPolicyFingerprint,omitempty"`
	TranscriptionContractFingerprint string                 `json:"transcriptionContractFingerprint,omitempty"`
	AuthorityLimits                  []string               `json:"authorityLimits"`
}

func Inspect(format Format, source []byte, policy ClassificationPolicy) (Inspection, error) {
	doc, err := Parse(format, source)
	if err != nil {
		return Inspection{}, err
	}
	facts := ExtractRelationalSourceFacts(doc)
	diagnostics := DiagnoseRelationalSourceFacts(doc, facts)
	resolutionStatus := "resolved"
	if len(diagnostics) != 0 {
		resolutionStatus = "unresolved"
	}
	return Inspection{
		Document:                   doc,
		Candidates:                 Classify(doc, policy),
		RelationalFacts:            facts,
		RelationalDiagnostics:      diagnostics,
		RelationalResolutionStatus: resolutionStatus,
		ClassificationPolicyFingerprint: ClassificationPolicyFingerprint(policy),
		AuthorityLimits: []string{
			"source-inspection-is-not-canonical-authority",
			"source-fact-is-not-accepted-semantic-meaning",
			"candidate-classification-is-not-shared-acceptance",
			"unresolved-source-meaning-must-not-be-auto-completed",
			"relational-resolution-status-is-source-consistency-not-canonical-meaning",
			"unresolved-relational-resolution-blocks-deterministic-semantic-lowering",
		},
	}, nil
}
