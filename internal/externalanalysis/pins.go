package externalanalysis

import (
	"fmt"
	"strings"
)

// Pins are caller-supplied expectations a receipt must match.
type Pins struct {
	EvidenceID     string
	Fingerprint    string
	SourceRevision string
}

// VerifyPins checks the receipt against caller-supplied expectations.
func (r Receipt) VerifyPins(pins Pins) error {
	var mismatches []string
	if strings.TrimSpace(pins.EvidenceID) != "" && (r.Evidence == nil || r.Evidence.ID != pins.EvidenceID) {
		mismatches = append(mismatches, fmt.Sprintf("evidence_id=%s != %s", r.safeEvidenceID(), pins.EvidenceID))
	}
	if strings.TrimSpace(pins.Fingerprint) != "" && r.Raw.Fingerprint != pins.Fingerprint {
		mismatches = append(mismatches, fmt.Sprintf("raw fingerprint=%s != %s", r.Raw.Fingerprint, pins.Fingerprint))
	}
	if strings.TrimSpace(pins.SourceRevision) != "" && (r.Source == nil || r.Source.Revision != pins.SourceRevision) {
		mismatches = append(mismatches, fmt.Sprintf("source revision=%v != %s", r.safeSourceRevision(), pins.SourceRevision))
	}
	if len(mismatches) > 0 {
		return fmt.Errorf("pin mismatch: %s", strings.Join(mismatches, "; "))
	}
	return nil
}

func (r Receipt) safeEvidenceID() string {
	if r.Evidence == nil {
		return ""
	}
	return r.Evidence.ID
}

func (r Receipt) safeSourceRevision() string {
	if r.Source == nil {
		return ""
	}
	return r.Source.Revision
}
