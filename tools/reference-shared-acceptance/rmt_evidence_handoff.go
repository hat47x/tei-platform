package main

import (
	"tei/internal/sourceindex"
	"tei/tools/internal/rmtevidence"
)

type rmtTranscriptionArtifact = rmtevidence.Evidence

func attachRMTTranscriptionEvidence(candidateData, evidenceData []byte) (Candidate, error) {
	candidate, err := decodeCandidate(candidateData)
	if err != nil {
		return Candidate{}, err
	}
	evidence, err := rmtevidence.Decode(evidenceData)
	if err != nil {
		return Candidate{}, err
	}
	payload, err := rmtevidence.CanonicalBytes(evidence)
	if err != nil {
		return Candidate{}, err
	}
	artifact := EvidenceArtifact{
		Kind:                evidenceArtifactKindRMTTranscription,
		Schema:              rmtevidence.Schema,
		ArtifactFingerprint: sourceindex.Fingerprint(payload),
		Payload:             payload,
	}
	candidate.Evidence = append(candidate.Evidence, Evidence{Artifact: &artifact})
	normalized, _, err := normalizeCandidate(candidate)
	if err != nil {
		return Candidate{}, err
	}
	return normalized, nil
}

func rmtTranscriptionFromEvidence(e Evidence) (rmtTranscriptionArtifact, bool) {
	if e.Artifact == nil || e.Artifact.Kind != evidenceArtifactKindRMTTranscription {
		return rmtTranscriptionArtifact{}, false
	}
	value, err := rmtevidence.Decode(e.Artifact.Payload)
	if err != nil {
		return rmtTranscriptionArtifact{}, false
	}
	return value, true
}
