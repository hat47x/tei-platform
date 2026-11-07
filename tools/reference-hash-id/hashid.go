// Package hashid is an experimental reference for content-derived identifiers.
// It is not the CanonicalID, a signature, or an authorization mechanism.
package hashid

import (
	"crypto/sha256"
	"crypto/sha3"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"strconv"
	"strings"
)

const (
	ModeEncodedDigest   = "encodedDigest"
	ModeExactCharacters = "exactCharacters"
	AlgorithmSHA256     = "sha256"
	AlgorithmSHAKE256   = "shake256"
	AlgorithmCSHAKE256  = "cshake256"

	EncodingHex       = "base16lower"
	EncodingBase32    = "base32lower"
	EncodingBase64URL = "base64url"
	EncodingRadix     = "radix"
)

const (
	maxOutputBits = 4096
	maxCharacters = 1024
)

// Scheme describes all v1 parameters that affect the derived result.
// v1 accepts raw bytes only: other canonicalization profiles must not be
// silently interpreted as raw bytes.
type Scheme struct {
	SchemeID         string
	Purpose          string
	Scope            string
	InputKind        string
	Canonicalization string
	Algorithm        string
	Mode             string
	OutputBits       int
	Encoding         string
	Alphabet         string
	Length           int
}

// Result is scoped to the whole Scheme. Text alone is not a globally unique
// or authoritative identity. Digest is present only for encodedDigest mode.
type Result struct {
	Scheme Scheme
	Text   string
	Digest []byte
}

func (s Scheme) Validate() error {
	for name, value := range map[string]string{
		"schemeId": s.SchemeID,
		"purpose":  s.Purpose,
		"scope":    s.Scope,
	} {
		if !validToken(value) {
			return fmt.Errorf("%s: unsupported or missing ASCII token", name)
		}
	}
	if s.InputKind != "bytes" || s.Canonicalization != "raw-bytes/v1" {
		return errors.New("only bytes with raw-bytes/v1 canonicalization are supported")
	}
	switch s.Algorithm {
	case AlgorithmSHA256, AlgorithmSHAKE256, AlgorithmCSHAKE256:
	default:
		return fmt.Errorf("unsupported algorithm %q", s.Algorithm)
	}
	switch s.Mode {
	case ModeEncodedDigest:
		if s.Length != 0 {
			return errors.New("encodedDigest must not specify an exact character length")
		}
		if s.OutputBits < 8 || s.OutputBits > maxOutputBits || s.OutputBits%8 != 0 {
			return errors.New("digest outputBits must be a multiple of 8 within [8,4096]")
		}
		if s.Algorithm == AlgorithmSHA256 && s.OutputBits != 256 {
			return errors.New("SHA-256 is fixed at 256 bits (no implicit truncation)")
		}
		switch s.Encoding {
		case EncodingHex, EncodingBase32, EncodingBase64URL:
			if s.Alphabet != "" {
				return errors.New("named encoding must not specify an alphabet")
			}
		case EncodingRadix:
			if err := validAlphabet(s.Alphabet); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported digest encoding %q", s.Encoding)
		}
	case ModeExactCharacters:
		if s.OutputBits != 0 || s.Encoding != "" {
			return errors.New("exactCharacters must not specify digest outputBits or encoding")
		}
		if s.Algorithm == AlgorithmSHA256 {
			return errors.New("exactCharacters requires an XOF")
		}
		if s.Length < 1 || s.Length > maxCharacters {
			return errors.New("exact length must be within [1,1024]")
		}
		if err := validAlphabet(s.Alphabet); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported mode %q", s.Mode)
	}
	return nil
}

func validToken(v string) bool {
	if v == "" || len(v) > 256 {
		return false
	}
	for i := 0; i < len(v); i++ {
		c := v[i]
		if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '.' || c == '-' || c == '_' || c == ':' || c == '/') {
			return false
		}
	}
	return true
}

// v1 intentionally starts with printable, non-whitespace ASCII. Unicode
// scalar normalization and collation semantics require another profile.
func validAlphabet(a string) error {
	if len(a) < 2 || len(a) > 94 {
		return errors.New("alphabet must contain 2..94 distinct printable ASCII characters")
	}
	var seen [128]bool
	for i := 0; i < len(a); i++ {
		c := a[i]
		if c < '!' || c > '~' || seen[c] {
			return errors.New("alphabet must be unique printable non-whitespace ASCII")
		}
		seen[c] = true
	}
	return nil
}

// frame is unambiguous: each UTF-8 token and the input byte sequence are
// length-prefixed with an unsigned big-endian 64-bit size.
func frame(s Scheme, input []byte) []byte {
	fields := []string{
		"tei.hashid.frame.v1", s.SchemeID, s.Purpose, s.Scope,
		s.InputKind, s.Canonicalization, s.Algorithm, s.Mode,
		strconv.Itoa(s.OutputBits),
	}
	// In encodedDigest, presentation is not part of the hashed identity:
	// hex, base32 and radix must encode the same digest bytes.
	// In exactCharacters the alphabet order and length define the mapping,
	// and therefore must be bound to the XOF input.
	if s.Mode == ModeExactCharacters {
		fields = append(fields, s.Alphabet, strconv.Itoa(s.Length))
	}
	out := make([]byte, 0, len(input)+512)
	for _, f := range fields {
		out = appendField(out, []byte(f))
	}
	return appendField(out, input)
}

func appendField(dst, field []byte) []byte {
	var n [8]byte
	binary.BigEndian.PutUint64(n[:], uint64(len(field)))
	dst = append(dst, n[:]...)
	return append(dst, field...)
}

func newXOF(s Scheme, material []byte) (*sha3.SHAKE, error) {
	var xof *sha3.SHAKE
	switch s.Algorithm {
	case AlgorithmSHAKE256:
		xof = sha3.NewSHAKE256()
	case AlgorithmCSHAKE256:
		xof = sha3.NewCSHAKE256(nil, []byte("tei.hashid.cshake256.v1"))
	default:
		return nil, errors.New("algorithm does not support extensible output")
	}
	_, _ = xof.Write(material)
	return xof, nil
}

func Derive(s Scheme, input []byte) (Result, error) {
	if err := s.Validate(); err != nil {
		return Result{}, err
	}
	material := frame(s, input)
	r := Result{Scheme: s}
	switch s.Mode {
	case ModeEncodedDigest:
		digest := make([]byte, s.OutputBits/8)
		if s.Algorithm == AlgorithmSHA256 {
			sum := sha256.Sum256(material)
			copy(digest, sum[:])
		} else {
			xof, err := newXOF(s, material)
			if err != nil {
				return Result{}, err
			}
			if _, err := io.ReadFull(xof, digest); err != nil {
				return Result{}, err
			}
		}
		text, err := encodeDigest(s, digest)
		if err != nil {
			return Result{}, err
		}
		r.Digest, r.Text = digest, text
	case ModeExactCharacters:
		xof, err := newXOF(s, material)
		if err != nil {
			return Result{}, err
		}
		// For v1's ASCII alphabet, 2 <= A <= 94. A one-byte candidate is
		// sufficient; rejecting x >= floor(256/A)*A avoids modulo bias.
		a := len(s.Alphabet)
		threshold := (256 / a) * a
		out := make([]byte, 0, s.Length)
		var sample [1]byte
		// A bounded budget prevents untrusted schemes from consuming unbounded work.
		for draws := 0; len(out) < s.Length && draws < 16*s.Length+1024; draws++ {
			if _, err := io.ReadFull(xof, sample[:]); err != nil {
				return Result{}, err
			}
			if int(sample[0]) >= threshold {
				continue
			}
			out = append(out, s.Alphabet[int(sample[0])%a])
		}
		if len(out) != s.Length {
			return Result{}, errors.New("exactCharacters sampling budget exceeded")
		}
		r.Text = string(out)
	}
	return r, nil
}

func encodeDigest(s Scheme, digest []byte) (string, error) {
	switch s.Encoding {
	case EncodingHex:
		return hex.EncodeToString(digest), nil
	case EncodingBase32:
		return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(digest)), nil
	case EncodingBase64URL:
		return base64.RawURLEncoding.EncodeToString(digest), nil
	case EncodingRadix:
		return encodeRadix(s.Alphabet, digest), nil
	}
	return "", fmt.Errorf("unknown encoding %q", s.Encoding)
}

// encodeRadix treats digest bytes as one unsigned big-endian integer, including
// a canonical width large enough to represent all values of the digest size.
func encodeRadix(alphabet string, digest []byte) string {
	radix := big.NewInt(int64(len(alphabet)))
	max := new(big.Int).Lsh(big.NewInt(1), uint(8*len(digest)))
	max.Sub(max, big.NewInt(1))
	var width int
	for max.Sign() > 0 {
		max.Div(max, radix)
		width++
	}
	n := new(big.Int).SetBytes(digest)
	rem := new(big.Int)
	out := []byte(strings.Repeat(string(alphabet[0]), width))
	for i := width - 1; i >= 0 && n.Sign() > 0; i-- {
		n.DivMod(n, radix, rem)
		out[i] = alphabet[rem.Int64()]
	}
	return string(out)
}

// CollisionProbability estimates birthday collision risk for a population of
// independent, uniformly distributed derived IDs. It is not a safety proof.
func CollisionProbability(s Scheme, population uint64) (float64, error) {
	if err := s.Validate(); err != nil {
		return 0, err
	}
	if population < 2 {
		return 0, nil
	}
	if exceedsFiniteAddressSpace(s, population) {
		return 1, nil // pigeonhole certainty, not merely a birthday estimate
	}
	return -math.Expm1(-math.Exp(collisionLogLambda(s, population))), nil
}

// collisionLogLambda returns the logarithm of the birthday pair-collision
// union bound n(n-1)/(2*S). The assumed address space is 2^bits or A^length.
// It models independent uniform outputs, not adversarial inputs or real
// target collation, reservation, existing occupancy, or Owner acceptance.
func collisionLogLambda(s Scheme, population uint64) float64 {
	logSpace := float64(s.OutputBits) * math.Ln2
	if s.Mode == ModeExactCharacters {
		logSpace = float64(s.Length) * math.Log(float64(len(s.Alphabet)))
	}
	return math.Log(float64(population)) +
		math.Log(float64(population-1)) - math.Ln2 - logSpace
}

// exceedsFiniteAddressSpace detects pigeonhole certainty exactly, without
// floating-point rounding and without constructing arbitrarily large powers.
func exceedsFiniteAddressSpace(s Scheme, population uint64) bool {
	bound := new(big.Int).SetUint64(population)
	capacity := big.NewInt(1)
	if s.Mode == ModeEncodedDigest {
		capacity.Lsh(capacity, uint(s.OutputBits))
	} else {
		radix := big.NewInt(int64(len(s.Alphabet)))
		for i := 0; i < s.Length && capacity.Cmp(bound) < 0; i++ {
			capacity.Mul(capacity, radix)
		}
	}
	return capacity.Cmp(bound) < 0
}

// CollisionRiskUpperBound conservatively represents the union bound
// min(1, n(n-1)/(2*S)), rounding the float result upward (or to the
// smallest positive float after underflow). A birthday approximation can
// underestimate small finite address spaces, so the budget gate must use
// this bound rather than the birthday point estimate. Both assume that
// derivations are independent and uniformly distributed.
func CollisionRiskUpperBound(s Scheme, population uint64) (float64, error) {
	if err := s.Validate(); err != nil {
		return 0, err
	}
	if population < 2 {
		return 0, nil
	}
	if exceedsFiniteAddressSpace(s, population) {
		return 1, nil
	}
	logLambda := collisionLogLambda(s, population)
	if logLambda >= 0 {
		return 1, nil
	}
	bound := math.Exp(logLambda)
	if bound == 0 {
		// Underflow is a display/precision limit, not zero risk. Use the
		// smallest positive representable bound for a fail-closed estimate.
		return math.SmallestNonzeroFloat64, nil
	}
	// Round one representable step upward, never down, for budget comparisons.
	return math.Nextafter(bound, math.Inf(1)), nil
}

func CheckCollisionBudget(s Scheme, population uint64, maximum float64) error {
	if math.IsNaN(maximum) || math.IsInf(maximum, 0) || maximum < 0 || maximum > 1 {
		return errors.New("collision budget must be a finite probability within [0,1]")
	}
	upper, err := CollisionRiskUpperBound(s, population)
	if err != nil {
		return err
	}
	if population >= 2 && maximum == 0 {
		// A deterministic finite hash cannot certify zero collision risk.
		// In particular, for long IDs, floating-point underflow must not
		// transform a strictly positive risk into a passing zero value.
		return errors.New("zero collision risk budget is impossible for a population of two or more")
	}
	if population >= 2 && exceedsFiniteAddressSpace(s, population) {
		return errors.New("population exceeds finite hash ID address space; collision is certain")
	}
	if upper > maximum {
		return fmt.Errorf("collision risk upper bound %.6g exceeds budget %.6g", upper, maximum)
	}
	return nil
}
