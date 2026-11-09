package expectedstateupdate

import (
	"crypto/ed25519"
	"fmt"
)

// KeyEntry is one key in the ring. A verification-only entry may omit the
// private key; signing then fails.
type KeyEntry struct {
	Public  ed25519.PublicKey
	Private ed25519.PrivateKey
	Retired bool
}

// KeyRing selects signing and verification keys by keyId. Rotation retires the
// old key; the retired key is refused by Sign and Verify, and can only be used
// explicitly through VerifyArchived.
type KeyRing struct {
	keys map[string]KeyEntry
}

// NewKeyRing returns an empty key ring.
func NewKeyRing() *KeyRing {
	return &KeyRing{keys: map[string]KeyEntry{}}
}

// Add registers a key. Duplicate identifiers and empty identifiers are refused.
func (r *KeyRing) Add(id string, public ed25519.PublicKey, private ed25519.PrivateKey) error {
	if r.keys == nil {
		r.keys = map[string]KeyEntry{}
	}
	if id == "" {
		return fmt.Errorf("key ring: key id is required")
	}
	if len(public) != ed25519.PublicKeySize {
		return fmt.Errorf("key ring: invalid Ed25519 public key for %q", id)
	}
	if _, exists := r.keys[id]; exists {
		return fmt.Errorf("key ring: key %q already exists", id)
	}
	r.keys[id] = KeyEntry{Public: public, Private: private}
	return nil
}

// Rotate adds newID as an active key and retires oldID. The old key must exist
// and still be active.
func (r *KeyRing) Rotate(oldID, newID string, public ed25519.PublicKey, private ed25519.PrivateKey) error {
	old, ok := r.keys[oldID]
	if !ok {
		return fmt.Errorf("key ring: key %q is unknown", oldID)
	}
	if old.Retired {
		return fmt.Errorf("key ring: key %q is already retired", oldID)
	}
	if err := r.Add(newID, public, private); err != nil {
		return err
	}
	old.Retired = true
	r.keys[oldID] = old
	return nil
}

// Sign signs a record with the given active key. Unknown and retired keys are
// refused.
func (r *KeyRing) Sign(record Record, keyID string) (SignedRecord, error) {
	entry, ok := r.keys[keyID]
	if !ok {
		return SignedRecord{}, fmt.Errorf("key ring: key %q is unknown", keyID)
	}
	if entry.Retired {
		return SignedRecord{}, fmt.Errorf("key ring: key %q is retired", keyID)
	}
	return SignRecord(record, keyID, entry.Private)
}

// Verify checks a signed record with its key. Unknown and retired keys are
// refused; a record signed before rotation must be checked with VerifyArchived.
func (r *KeyRing) Verify(signed SignedRecord) error {
	entry, ok := r.keys[signed.KeyID]
	if !ok {
		return fmt.Errorf("key ring: key %q is unknown", signed.KeyID)
	}
	if entry.Retired {
		return fmt.Errorf("key ring: key %q is retired", signed.KeyID)
	}
	return VerifySignedRecord(signed, signed.KeyID, entry.Public)
}

// VerifyArchived checks a signed record with a retired key. It still refuses an
// unknown key id and an invalid signature.
func (r *KeyRing) VerifyArchived(signed SignedRecord) error {
	entry, ok := r.keys[signed.KeyID]
	if !ok {
		return fmt.Errorf("key ring: key %q is unknown", signed.KeyID)
	}
	return VerifySignedRecord(signed, signed.KeyID, entry.Public)
}

// Retire marks a key retired. Unknown keys are refused.
func (r *KeyRing) Retire(id string) error {
	entry, ok := r.keys[id]
	if !ok {
		return fmt.Errorf("key ring: key %q is unknown", id)
	}
	entry.Retired = true
	r.keys[id] = entry
	return nil
}
