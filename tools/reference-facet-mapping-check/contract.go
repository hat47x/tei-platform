package facetmappingcheck

// contractColumns are the twelve logical columns of the mapping contract, in the
// order the issue lists them. They are exported as data so a consumer (including
// a generating model) can read the contract from the report itself instead of
// learning it from prose.
var contractColumns = []string{
	"source",
	"target",
	"identity",
	"order",
	"multiset",
	"grain",
	"presence",
	"coverage",
	"quantity",
	"time",
	"loss",
	"operation",
}

var lossVocabulary = []string{
	LossOrder, LossOccurrenceIdentity, LossEdgeMultiplicity, LossParentArity,
	LossHiddenEdge, LossGrain, LossFactIdentity, LossAggregationInverse,
	LossQuantityScale, LossTimeAxisRole, LossSelectionIdentity, LossPresenceStates,
	LossCoverage, LossSourceRevision, LossUnitConversion, LossUnitOriginal,
	LossCRS, LossPrecision, LossFanout, LossRelationArity, LossInjectivity,
	LossContextIdentity,
}

var verdictVocabulary = []string{
	VerdictPreserved, VerdictLossy, VerdictRejected, VerdictUnqualified, VerdictUnknown,
}

var inverseVocabulary = []string{InverseEligible, InverseIneligible, InverseNotClaimed}

// Contract self-description. Reading it is not an adoption of a TEI Core
// schema: it describes this reference tool only.
type Contract struct {
	SchemaVersion string   `json:"schemaVersion"`
	Columns       []string `json:"columns"`
	Verdicts      []string `json:"verdicts"`
	Losses        []string `json:"losses"`
	Inverses      []string `json:"inverses"`
	// VerdictRules states, in one line each, when a verdict applies. The rules
	// are ordered by precedence.
	VerdictRules []string `json:"verdictRules"`
	// InverseRules states when a declared inverse stays eligible.
	InverseRules []string `json:"inverseRules"`
	// NotClaims lists what this reference does not decide.
	NotClaims []string `json:"notClaims"`
}

// ContractDescription returns the self-description a consumer can read without
// parsing prose. It is the same content the report embeds.
func ContractDescription() Contract {
	return Contract{
		SchemaVersion: ReportSchema,
		Columns:       contractColumns,
		Verdicts:      verdictVocabulary,
		Losses:        lossVocabulary,
		Inverses:      inverseVocabulary,
		VerdictRules: []string{
			"rejected: a claim and an observation cannot both hold, or a required declaration is contradicted by the observed facet",
			"unqualified: the transformation needs a declaration (conversion policy, instant, CRS transform, clock) that is absent; no default fills it",
			"unknown: the comparison cannot be made, or a preservation claim carries no evidence",
			"lossy: the transformation holds but drops meaning that is listed in losses",
			"preserved: every declared preservation claim carries evidence and no loss was observed",
		},
		InverseRules: []string{
			"not-claimed: the contract does not declare an edit inverse",
			"ineligible: the verdict is rejected, unqualified or unknown",
			"ineligible: a non-invertible loss is present (order, occurrence identity, grain, fact identity, presence, coverage, source revision, injectivity, time axis role, quantity scale, fanout)",
			"eligible: otherwise, and only for the declared inverse of this mapping",
		},
		NotClaims: []string{
			"no Core type, type syntax, or public mapping schema is adopted",
			"no renderer, provider, database, host, or media playback is exercised",
			"no business meaning is accepted; preserved is a comparison result on synthetic input",
			"a successful read-only projection never implies a usable edit inverse",
		},
	}
}
