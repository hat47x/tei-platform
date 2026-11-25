// Provenance for one comparison run.
//
// #5807 and #5809 both require that a comparison be reproducible: the actual
// configuration, the model digest, the template, and the request/response
// fingerprints have to be recorded with the numbers. A rate without them is a
// number about an unknown system, and a later run cannot tell whether a
// difference is the model or the harness.
//
// This file records what a run actually used, and reports what it could not
// establish rather than filling a field with a placeholder. A digest is only
// written when the bytes were read; a template is only written when the server
// disclosed it. Nothing here validates the model, the endpoint, or the answer.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ProvenanceSchema identifies this document.
const ProvenanceSchema = "tei.reference.cognition-provenance/v0"

// Provenance is the record attached to one comparison run.
type Provenance struct {
	Schema string `json:"schema"`

	// RecordedAt is when the run finished, in RFC 3339 UTC.
	RecordedAt string `json:"recordedAt"`

	// CaseSetDigest fingerprints the case set the run used, so a later run can
	// tell whether it compared the same inputs.
	CaseSetDigest string `json:"caseSetDigest"`

	// Harness identifies the tool that produced the numbers.
	Harness string `json:"harness"`

	// Providers holds one entry per provider that ran.
	Providers []ProviderProvenance `json:"providers"`

	// Unrecorded lists the facts this run could not establish. It is written
	// instead of leaving a field empty, because an empty field reads as "not
	// applicable" while the truth is "not observed".
	Unrecorded []string `json:"unrecorded,omitempty"`
}

// ProviderProvenance is what one provider run used.
type ProviderProvenance struct {
	// Provider and Model are as declared, not as observed.
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Version  string `json:"version"`

	// ModelDigest is the SHA-256 of the model artifact, when the operator
	// supplied a readable artifact path with -model-artifact. Absent when no
	// path was given or the file could not be read.
	ModelDigest string `json:"modelDigest,omitempty"`
	ModelBytes  int64  `json:"modelBytes,omitempty"`

	// Endpoint is the declared endpoint, recorded so a run can be traced to a
	// process without recording the machine's address book.
	Endpoint string `json:"endpoint,omitempty"`

	// TemplateDigest and TemplateSource record the chat template the server
	// disclosed, when it disclosed one. TemplateSource is the endpoint the
	// template came from, so a digest can be traced to its origin.
	TemplateDigest string `json:"templateDigest,omitempty"`
	TemplateSource string `json:"templateSource,omitempty"`

	// ContextWindow and MaxTokens are the server's declared limits.
	ContextWindow int `json:"contextWindow,omitempty"`
	MaxTokens     int `json:"maxTokens,omitempty"`

	// Cases records one fingerprint per case, in the order they ran.
	Cases []CaseProvenance `json:"cases"`
}

// CaseProvenance fingerprints one request and the reply it produced.
type CaseProvenance struct {
	// Case is the case id from the case set.
	Case string `json:"case"`

	// RequestDigest fingerprints the context the port sent to the provider.
	// It is computed here from the same bytes the provider received, so a
	// mismatch between two runs means the inputs differed.
	RequestDigest string `json:"requestDigest"`

	// ReplyDigest fingerprints the provider's raw proposal bytes. Two runs of
	// the same case with different reply digests produced different proposals,
	// which is what makes a rate comparison meaningful.
	ReplyDigest string `json:"replyDigest,omitempty"`

	// Millis is the wall time of this case.
	Millis int64 `json:"millis"`
}

// digest returns the prefixed SHA-256 the rest of the reference uses, so a
// digest here can be compared with one in a receipt.
func digestBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// newProvenance starts a record for a finished run.
func newProvenance(caseSetDigest, casesPath string) Provenance {
	return Provenance{
		Schema:        ProvenanceSchema,
		RecordedAt:    time.Now().UTC().Format(time.RFC3339),
		CaseSetDigest: caseSetDigest,
		Harness:       "tei-cognition-compare",
	}
}

// artifactDigest reads a model artifact and reports its digest and size. A file
// that cannot be read is reported as unrecorded rather than as a zero digest:
// a wrong digest is worse than an absent one, because it would make two
// different models look identical.
func artifactDigest(path string) (digest string, size int64, err error) {
	if path == "" {
		return "", 0, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", 0, err
	}
	hasher := sha256.New()
	if _, err := copyBuffer(hasher, file); err != nil {
		return "", 0, err
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), info.Size(), nil
}

// templateDigest fingerprints a disclosed chat template.
func templateDigest(template string) string {
	if template == "" {
		return ""
	}
	return digestBytes([]byte(template))
}

// writeProvenance emits the record as indented JSON, creating the file with
// owner-only permissions because a model digest and an endpoint together
// describe a deployment.
func writeProvenance(path string, record Provenance) error {
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}

func copyBuffer(dst interface{ Write([]byte) (int, error) }, src *os.File) (int64, error) {
	buffer := make([]byte, 1<<20)
	var written int64
	for {
		read, err := src.Read(buffer)
		if read > 0 {
			n, writeErr := dst.Write(buffer[:read])
			written += int64(n)
			if writeErr != nil {
				return written, writeErr
			}
		}
		if err != nil {
			if err.Error() == "EOF" {
				return written, nil
			}
			return written, fmt.Errorf("read artifact: %w", err)
		}
	}
}
