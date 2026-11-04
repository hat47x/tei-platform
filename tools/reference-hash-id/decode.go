package hashid

import (
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// DecodeDigest reverses the selected canonical encoding into exactly
// OutputBits/8 bytes. It does not verify the bytes against an input or grant
// authority to the caller: that requires Derive and a trusted source.
func DecodeDigest(s Scheme, text string) ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if s.Mode != ModeEncodedDigest {
		return nil, errors.New("exactCharacters cannot be decoded as a digest")
	}
	// Any v1 digest has at most maxOutputBits radix-2 characters. Reject
	// oversized untrusted text before a decoder allocates memory for it.
	if len(text) > maxOutputBits {
		return nil, errors.New("digest text exceeds maximum canonical width")
	}
	var decoded []byte
	var err error
	switch s.Encoding {
	case EncodingHex:
		decoded, err = hex.DecodeString(text)
	case EncodingBase32:
		decoded, err = base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(text))
	case EncodingBase64URL:
		decoded, err = base64.RawURLEncoding.DecodeString(text)
	case EncodingRadix:
		decoded, err = decodeRadix(s.Alphabet, s.OutputBits, text)
	default:
		return nil, errors.New("unsupported encoding")
	}
	if err != nil {
		return nil, fmt.Errorf("invalid digest text: %w", err)
	}
	if len(decoded) != s.OutputBits/8 {
		return nil, fmt.Errorf("decoded digest has %d bytes, requires %d", len(decoded), s.OutputBits/8)
	}
	canonical, err := encodeDigest(s, decoded)
	if err != nil {
		return nil, err
	}
	if canonical != text {
		return nil, errors.New("noncanonical digest text")
	}
	return decoded, nil
}

func decodeRadix(alphabet string, outputBits int, text string) ([]byte, error) {
	width := len(encodeRadix(alphabet, make([]byte, outputBits/8)))
	if len(text) != width {
		return nil, errors.New("wrong radix width")
	}
	base := big.NewInt(int64(len(alphabet)))
	n := new(big.Int)
	for i := 0; i < len(text); i++ {
		pos := strings.IndexByte(alphabet, text[i])
		if pos < 0 {
			return nil, fmt.Errorf("invalid radix character at offset %d", i)
		}
		n.Mul(n, base)
		n.Add(n, big.NewInt(int64(pos)))
	}
	if n.BitLen() > outputBits {
		return nil, errors.New("radix text exceeds digest bit width")
	}
	return n.FillBytes(make([]byte, outputBits/8)), nil
}
