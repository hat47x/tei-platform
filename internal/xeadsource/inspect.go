package xeadsource

type Inspection struct {
	Document         Document              `json:"document"`
	Candidates       []Candidate           `json:"candidates"`
	RelationalFacts  RelationalSourceFacts `json:"relationalFacts"`
	AuthorityLimits  []string              `json:"authorityLimits"`
}

func Inspect(format Format, source []byte, policy ClassificationPolicy) (Inspection, error) {
	doc, err := Parse(format, source)
	if err != nil {
		return Inspection{}, err
	}
	return Inspection{
		Document:        doc,
		Candidates:      Classify(doc, policy),
		RelationalFacts: ExtractRelationalSourceFacts(doc),
		AuthorityLimits: []string{
			"source-inspection-is-not-canonical-authority",
			"source-fact-is-not-accepted-semantic-meaning",
			"candidate-classification-is-not-shared-acceptance",
			"unresolved-source-meaning-must-not-be-auto-completed",
		},
	}, nil
}
