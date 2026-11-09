package expectedstateupdate

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// SignedRecordSchema identifies the signed envelope around one run record.
//
// The record itself is unchanged; the envelope binds the canonical record bytes
// and a key identifier to an Ed25519 signature. Signing is optional: the
// existing unsigned record remains the reference default.
const SignedRecordSchema = "tei.reference.expected-state-signed-record/v0"

// SignatureAlgorithm is the only supported signature algorithm.
const SignatureAlgorithm = "ed25519"

// SignedRecord is one record plus the signature that covers it.
type SignedRecord struct {
	Schema    string `json:"schema"`
	KeyID     string `json:"keyId"`
	Algorithm string `json:"algorithm"`
	Signature string `json:"signature"`
	Record    Record `json:"record"`
}

func signedMessage(keyID string, record Record) ([]byte, error) {
	encoded, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	return append([]byte(keyID+"\n"), encoded...), nil
}

// SignRecord wraps a record in an Ed25519 signature. The key identifier is part
// of the signed bytes, so a signature cannot be re-labelled with another key id.
func SignRecord(record Record, keyID string, key ed25519.PrivateKey) (SignedRecord, error) {
	if keyID == "" {
		return SignedRecord{}, fmt.Errorf("expected-state signature: key id is required")
	}
	if len(key) != ed25519.PrivateKeySize {
		return SignedRecord{}, fmt.Errorf("expected-state signature: invalid Ed25519 private key")
	}
	message, err := signedMessage(keyID, record)
	if err != nil {
		return SignedRecord{}, err
	}
	return SignedRecord{
		Schema:    SignedRecordSchema,
		KeyID:     keyID,
		Algorithm: SignatureAlgorithm,
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(key, message)),
		Record:    record,
	}, nil
}

// VerifySignedRecord checks the envelope, the signature, and the record's own
// schema and digest. Any mismatch is an error: an invalid envelope is never
// reported as authentic.
func VerifySignedRecord(signed SignedRecord, keyID string, key ed25519.PublicKey) error {
	if signed.Schema != SignedRecordSchema {
		return fmt.Errorf("expected-state signature: unexpected envelope schema %q", signed.Schema)
	}
	if signed.Algorithm != SignatureAlgorithm {
		return fmt.Errorf("expected-state signature: unsupported algorithm %q", signed.Algorithm)
	}
	if keyID == "" || signed.KeyID != keyID {
		return fmt.Errorf("expected-state signature: key id does not match the verification key")
	}
	if signed.Signature == "" {
		return fmt.Errorf("expected-state signature: signature is missing")
	}
	if len(key) != ed25519.PublicKeySize {
		return fmt.Errorf("expected-state signature: invalid Ed25519 public key")
	}
	signature, err := base64.StdEncoding.DecodeString(signed.Signature)
	if err != nil {
		return fmt.Errorf("expected-state signature: signature is not base64: %w", err)
	}
	message, err := signedMessage(signed.KeyID, signed.Record)
	if err != nil {
		return err
	}
	if !ed25519.Verify(key, message, signature) {
		return fmt.Errorf("expected-state signature: signature does not match the record")
	}
	// The signed bytes are only meaningful if the embedded record is itself
	// well-formed and its digest matches its entries.
	encoded, err := json.Marshal(signed.Record)
	if err != nil {
		return err
	}
	if _, err := DecodeRecord(bytes.NewReader(encoded)); err != nil {
		return fmt.Errorf("expected-state signature: signed record is invalid: %w", err)
	}
	return nil
}
