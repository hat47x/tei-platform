package main

// InvocationReceipt is a read-only projection of an InvocationExecution.
// It carries the selected contract, provider, shape authority and byte
// digests without copying request or result payloads into audit metadata.
// A receipt records Reference observations, not execution or trust authority.
const invocationReceiptSchema = "tei.reference.capability-invocation-receipt/v0"

type InvocationReceipt struct {
	Schema                                    string     `json:"schema"`
	Resolution                                Resolution `json:"resolution"`
	RequestShape                              string     `json:"requestShape"`
	RequestShapeFingerprint                   string     `json:"requestShapeFingerprint"`
	RequestShapeAuthorityEvidenceFingerprint  string     `json:"requestShapeAuthorityEvidenceFingerprint"`
	RequestShapeTrustQualificationFingerprint string     `json:"requestShapeTrustQualificationFingerprint"`
	ResultShape                               string     `json:"resultShape"`
	ResultShapeFingerprint                    string     `json:"resultShapeFingerprint"`
	ResultShapeAuthorityEvidenceFingerprint   string     `json:"resultShapeAuthorityEvidenceFingerprint"`
	ResultShapeTrustQualificationFingerprint  string     `json:"resultShapeTrustQualificationFingerprint"`
	PayloadCodec                              string     `json:"payloadCodec"`
	RequestDigest                             string     `json:"requestDigest"`
	ResultDigest                              string     `json:"resultDigest"`
	AuthorityLimits                           []string   `json:"authorityLimits"`
}

// Receipt leaves the potentially private ResultPayload out of the projection.
// In particular, the digest fields are observations and not signatures.
func (execution InvocationExecution) Receipt() InvocationReceipt {
	return InvocationReceipt{
		Schema:                                    invocationReceiptSchema,
		Resolution:                                execution.Resolution,
		RequestShape:                              execution.RequestShape,
		RequestShapeFingerprint:                   execution.RequestShapeFingerprint,
		RequestShapeAuthorityEvidenceFingerprint:  execution.RequestShapeAuthorityEvidenceFingerprint,
		RequestShapeTrustQualificationFingerprint: execution.RequestShapeTrustQualificationFingerprint,
		ResultShape:                               execution.ResultShape,
		ResultShapeFingerprint:                    execution.ResultShapeFingerprint,
		ResultShapeAuthorityEvidenceFingerprint:   execution.ResultShapeAuthorityEvidenceFingerprint,
		ResultShapeTrustQualificationFingerprint:  execution.ResultShapeTrustQualificationFingerprint,
		PayloadCodec:                              execution.PayloadCodec,
		RequestDigest:                             execution.RequestDigest,
		ResultDigest:                              execution.ResultDigest,
		AuthorityLimits:                           append([]string(nil), execution.AuthorityLimits...),
	}
}
