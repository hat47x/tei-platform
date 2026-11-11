package externalanalysis

import "fmt"

// Pins are the values recorded when an Evidence was first admitted. They exist
// so a second receipt of the "same" Evidence can be told apart from the first
// one: the analyzed source is revised, or the saved bytes are replaced, and
// neither is the same observation read twice.
type Pins struct {
	EvidenceID     string
	Fingerprint    string
	SourceRevision string
}

// VerifyPins reports the first way a received receipt fails to match the pins
// recorded when this Evidence was first admitted, and nil when it matches.
//
// A caller that recorded no pins is not asking a question, so an empty Pins is
// not a failure: a receipt that carries no Evidence at all already says so in
// its state, and answering a pin error there would send the reader looking for
// a mismatch that was never checked.
//
// An unpinned field is not checked: a caller that recorded only the Evidence ID
// is asking only whether it is the same Evidence. The revision is checked
// before the digest, because the two failures look alike from the bytes alone
// and mean different things: a changed revision says the analyzed source moved,
// while an unchanged revision with a changed digest says these are not the bytes
// that were admitted. Neither is the same observation read twice, and neither
// is a statement about the analysis result.
func (r Receipt) VerifyPins(pins Pins) error {
	if pins.EvidenceID == "" && pins.Fingerprint == "" && pins.SourceRevision == "" {
		return nil
	}
	if r.Evidence == nil {
		return fmt.Errorf("受領したbytesにEvidenceがありません: pinと照合できません")
	}
	if pins.EvidenceID != "" && r.Evidence.ID != pins.EvidenceID {
		return fmt.Errorf("Evidence IDが記録と異なります: have %s want %s", r.Evidence.ID, pins.EvidenceID)
	}
	if pins.SourceRevision != "" {
		if r.Source == nil || r.Source.Revision == "" {
			return fmt.Errorf("受領したbytesに対象のrevision宣言がありません: 記録されたrevision %sと照合できません", pins.SourceRevision)
		}
		if r.Source.Revision != pins.SourceRevision {
			return fmt.Errorf("対象Sourceのrevisionが記録と異なります: have %s want %s（Sourceが改版されたか、別の対象です）", r.Source.Revision, pins.SourceRevision)
		}
	}
	if pins.Fingerprint != "" && r.Raw.Fingerprint != pins.Fingerprint {
		return fmt.Errorf("受領bytesのdigestが記録と異なります: have %s want %s", r.Raw.Fingerprint, pins.Fingerprint)
	}
	return nil
}
