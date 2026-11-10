package hashid

import (
	"errors"
	"fmt"
	"strings"
)

// TargetProfile is a caller-declared, ASCII-only model of one physical ID
// namespace. It does not discover a real DB's collation, filesystem rules,
// uniqueness constraints, or any caller's authority to allocate an ID.
type TargetProfile struct {
	Namespace       string
	ProfileVersion  string
	AllowedAlphabet string
	MinLength       int
	MaxLength       int
	ExactLength     int
	CaseFoldASCII   bool
	Reserved        []string
}

// TargetBinding is an independently obtained allocation snapshot.
// Subject is a qualified source identity, NOT a display name or a view ID.
type TargetBinding struct {
	Namespace      string
	ProfileVersion string
	Subject        string
	Scheme         Scheme
	Text           string
}

// CheckTargetBinding validates a *proposed* physical ID against a declared
// target policy and the complete current set of bindings in that namespace.
// It returns a normalized lookup key, but does not persist or authorize it.
// Callers must recheck uniqueness atomically at the actual storage boundary.
func CheckTargetBinding(r Result, subject string, profile TargetProfile, bindings []TargetBinding) (string, error) {
	if err := r.Scheme.Validate(); err != nil {
		return "", fmt.Errorf("invalid ID scheme: %w", err)
	}
	if subject == "" || strings.TrimSpace(subject) != subject {
		return "", errors.New("subject must be a nonblank qualified identity")
	}
	if err := profile.validate(); err != nil {
		return "", err
	}
	if err := validateResultText(r); err != nil {
		return "", err
	}
	// The output alphabet, not just this sample result, must be legal in
	// this target. Otherwise a later source could yield an unusable ID.
	for _, c := range effectiveAlphabet(r.Scheme) {
		if !strings.ContainsRune(profile.AllowedAlphabet, c) {
			return "", fmt.Errorf("generator character %q is unavailable in target alphabet", c)
		}
	}
	// A case-insensitive physical target collapses "A" and "a".
	// For such targets, any case-paired values in the generator's effective
	// alphabet make the advertised address space invalid; fail closed.
	if profile.CaseFoldASCII && containsCasePair(effectiveAlphabet(r.Scheme)) {
		return "", errors.New("generator alphabet is ambiguous under target ASCII case folding")
	}
	key, err := profile.normalize(r.Text)
	if err != nil {
		return "", fmt.Errorf("proposed ID: %w", err)
	}
	// Snapshot integrity is an independent precondition: an unrelated
	// duplicate allocation must not pass merely because it does not collide
	// with this new candidate. This is still not proof of snapshot freshness.
	seenSubjects := make(map[string]struct{}, len(bindings))
	seenKeys := make(map[string]string, len(bindings))
	for _, binding := range bindings {
		if binding.Namespace != profile.Namespace {
			continue
		}
		if binding.ProfileVersion != profile.ProfileVersion {
			return "", fmt.Errorf("target profile version mismatch for subject %q; explicit migration required", binding.Subject)
		}
		if binding.Subject == "" {
			return "", errors.New("allocation snapshot has an empty subject")
		}
		if _, exists := seenSubjects[binding.Subject]; exists {
			return "", fmt.Errorf("allocation snapshot duplicates subject %q", binding.Subject)
		}
		seenSubjects[binding.Subject] = struct{}{}
		if err := binding.Scheme.Validate(); err != nil {
			return "", fmt.Errorf("invalid allocation snapshot scheme for %q: %w", binding.Subject, err)
		}
		existing, err := profile.normalize(binding.Text)
		if err != nil {
			return "", fmt.Errorf("invalid allocation snapshot for %q: %w", binding.Subject, err)
		}
		if prior, exists := seenKeys[existing]; exists && prior != binding.Subject {
			return "", fmt.Errorf("allocation snapshot collides between %q and %q", prior, binding.Subject)
		}
		seenKeys[existing] = binding.Subject
		if key == existing && binding.Subject != subject {
			return "", fmt.Errorf("physical ID collision between %q and %q", subject, binding.Subject)
		}
		if binding.Subject == subject && binding.Scheme != r.Scheme {
			return "", fmt.Errorf("subject %q has an allocation under a different hash scheme; explicit migration required", subject)
		}
		if binding.Subject == subject && key != existing {
			return "", fmt.Errorf("subject %q has a different existing physical ID; explicit migration required", subject)
		}
	}
	return key, nil
}

func (p TargetProfile) validate() error {
	if !validToken(p.Namespace) {
		return errors.New("target namespace must be an explicit ASCII token")
	}
	if !validToken(p.ProfileVersion) {
		return errors.New("target profile version must be an explicit ASCII token")
	}
	if err := validAlphabet(p.AllowedAlphabet); err != nil {
		return fmt.Errorf("invalid target alphabet: %w", err)
	}
	if p.MinLength < 0 || p.MaxLength < 0 || p.ExactLength < 0 ||
		p.MinLength > 8192 || p.MaxLength > 8192 || p.ExactLength > 8192 {
		return errors.New("target lengths must be within [0,8192]")
	}
	if p.ExactLength > 0 && (p.MinLength > 0 || p.MaxLength > 0) {
		return errors.New("exact target length conflicts with min/max limits")
	}
	if p.MaxLength > 0 && p.MinLength > p.MaxLength {
		return errors.New("target minLength exceeds maxLength")
	}
	for _, reserved := range p.Reserved {
		if reserved == "" {
			return errors.New("reserved target name must not be empty")
		}
		// Reserved names must be valid under the same alphabet/length rules.
		if err := p.checkText(reserved, false); err != nil {
			return fmt.Errorf("invalid reserved name %q: %w", reserved, err)
		}
	}
	return nil
}

func (p TargetProfile) checkText(value string, checkLength bool) error {
	if value == "" || len(value) > 8192 {
		return errors.New("target text is empty or exceeds 8192 bytes")
	}
	if checkLength {
		if p.ExactLength > 0 && len(value) != p.ExactLength {
			return fmt.Errorf("target length %d does not equal %d", len(value), p.ExactLength)
		}
		if p.MinLength > 0 && len(value) < p.MinLength {
			return errors.New("target text is too short")
		}
		if p.MaxLength > 0 && len(value) > p.MaxLength {
			return errors.New("target text is too long")
		}
	}
	var allowed [128]bool
	for i := 0; i < len(p.AllowedAlphabet); i++ {
		allowed[p.AllowedAlphabet[i]] = true
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c >= 128 || !allowed[c] {
			return fmt.Errorf("character at byte %d is not permitted by target", i)
		}
	}
	return nil
}

func (p TargetProfile) normalize(value string) (string, error) {
	if err := p.checkText(value, true); err != nil {
		return "", err
	}
	key := value
	if p.CaseFoldASCII {
		key = strings.ToLower(value) // ASCII-only after checkText.
	}
	for _, reserved := range p.Reserved {
		candidate := reserved
		if p.CaseFoldASCII {
			candidate = strings.ToLower(reserved)
		}
		if key == candidate {
			return "", fmt.Errorf("target ID %q is reserved", value)
		}
	}
	return key, nil
}

func validateResultText(r Result) error {
	if r.Text == "" {
		return errors.New("result text is empty")
	}
	s := r.Scheme
	if s.Mode == ModeEncodedDigest {
		if len(r.Digest) != s.OutputBits/8 {
			return errors.New("result digest length differs from scheme")
		}
		expected, err := encodeDigest(s, r.Digest)
		if err != nil || r.Text != expected {
			return errors.New("result text is not the canonical encoding of its digest")
		}
		return nil
	}
	if len(r.Digest) != 0 || len(r.Text) != s.Length {
		return errors.New("exactCharacters has a digest or invalid text length")
	}
	for i := 0; i < len(r.Text); i++ {
		if !strings.ContainsRune(s.Alphabet, rune(r.Text[i])) {
			return errors.New("exactCharacters text is outside generator alphabet")
		}
	}
	return nil
}

func effectiveAlphabet(s Scheme) string {
	if s.Mode == ModeExactCharacters || s.Encoding == EncodingRadix {
		return s.Alphabet
	}
	switch s.Encoding {
	case EncodingHex:
		return "0123456789abcdef"
	case EncodingBase32:
		return "abcdefghijklmnopqrstuvwxyz234567"
	case EncodingBase64URL:
		return "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	}
	return ""
}

func containsCasePair(alphabet string) bool {
	var found [128]bool
	for i := 0; i < len(alphabet); i++ {
		c := alphabet[i]
		if c >= 'A' && c <= 'Z' {
			if found[c+('a'-'A')] {
				return true
			}
		}
		if c >= 'a' && c <= 'z' {
			if found[c-('a'-'A')] {
				return true
			}
		}
		found[c] = true
	}
	return false
}
