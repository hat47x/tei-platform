package main

import (
	"fmt"

	"tei/internal/sourceindex"
	"tei/tools/internal/rmtevidence"
)

func sourceNodeToken(artifactSHA256 string, ordinal int) string {
	return digestString(fmt.Sprintf("%s|node-ordinal:%d", artifactSHA256, ordinal))
}

func BuildTranscriptionCandidateEvidence(
	format SourceFormat,
	doc ObservedDocument,
	contractBytes []byte,
	nodeOrdinal int,
) (rmtevidence.Evidence, error) {
	profile, err := validateSourceProfile(format, doc, contractBytes)
	if err != nil {
		return rmtevidence.Evidence{}, err
	}
	if nodeOrdinal < 1 || nodeOrdinal > len(doc.Nodes) {
		return rmtevidence.Evidence{}, fmt.Errorf(
			"source node ordinal %d is outside 1..%d",
			nodeOrdinal,
			len(doc.Nodes),
		)
	}
	mappings, err := parseConstructMappings(format, contractBytes)
	if err != nil {
		return rmtevidence.Evidence{}, err
	}

	node := doc.Nodes[nodeOrdinal-1]
	value := rmtevidence.Evidence{
		Schema:                     rmtevidence.Schema,
		SourceFormat:               string(format),
		SourceFormatVersion:        profile.FormatVersion,
		SourceArtifactDigest:       "sha256:" + doc.ArtifactSHA256,
		SourceToken:                sourceNodeToken(doc.ArtifactSHA256, nodeOrdinal),
		SourceConstructKnown:       node.Known,
		MappingContractFingerprint: sourceindex.Fingerprint(contractBytes),
		MappingStatus:              rmtevidence.MappingUnmapped,
		Classification:             []string{"unresolved"},
		AuthorityLimits: []string{
			"source-observation-is-not-canonical-authority",
			"mapping-is-candidate-not-semantic-acceptance",
			"rmt-evidence-does-not-grant-runtime-authority",
		},
	}
	if node.Known {
		value.SourceConstruct = node.Kind
	}
	if mapping, ok := mappings[node.Kind]; ok && node.Known {
		value.MappingStatus = rmtevidence.MappingMapped
		value.Classification = append([]string(nil), mapping.Classification...)
		value.Extension = mapping.Extension
		value.AuthorityLimits = append(value.AuthorityLimits, mapping.AuthorityLimits...)
	} else {
		value.AuthorityLimits = append(value.AuthorityLimits,
			"unmapped-source-construct-is-not-semantic-authority",
			"unknown-source-construct-must-not-be-silently-dropped",
		)
	}

	return rmtevidence.Prepare(value)
}


func VerifyTranscriptionCandidateEvidence(
	format SourceFormat,
	doc ObservedDocument,
	contractBytes []byte,
	nodeOrdinal int,
	recorded rmtevidence.Evidence,
) error {
	verified, err := rmtevidence.Verify(recorded)
	if err != nil {
		return err
	}
	current, err := BuildTranscriptionCandidateEvidence(format, doc, contractBytes, nodeOrdinal)
	if err != nil {
		return err
	}
	if verified.EvidenceFingerprint != current.EvidenceFingerprint {
		return fmt.Errorf(
			"RMT transcription Evidence is stale or belongs to a different source/mapping: recorded=%s current=%s",
			verified.EvidenceFingerprint,
			current.EvidenceFingerprint,
		)
	}
	return nil
}
