package partition

// ProvenanceKind labels how an access occurrence was obtained.
type ProvenanceKind string

const (
	ProvDeclaredSource   ProvenanceKind = "declared-source"
	ProvParsedExtracted  ProvenanceKind = "parsed-extracted"
	ProvDerivedPotential ProvenanceKind = "derived-potential"
	ProvInferredIndirect ProvenanceKind = "inferred-indirect"
	ProvObservedEffect   ProvenanceKind = "observed-effect"
	ProvUnresolved       ProvenanceKind = "unresolved"
	ProvNotExamined      ProvenanceKind = "not-examined"
)
