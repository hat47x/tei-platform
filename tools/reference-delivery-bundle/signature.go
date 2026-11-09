package deliverybundle

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

const (
	privateKeyType = "TEI ED25519 PRIVATE KEY"
	publicKeyType  = "TEI ED25519 PUBLIC KEY"
)

func GenerateKeypair(privatePath, publicPath string) error {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return err
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: privateKeyType, Bytes: privateKey.Seed()})
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: publicKeyType, Bytes: publicKey})
	if err := os.WriteFile(privatePath, privatePEM, 0o600); err != nil {
		return err
	}
	return os.WriteFile(publicPath, publicPEM, 0o644)
}

func LoadPrivateKey(path string) (ed25519.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil || block.Type != privateKeyType {
		return nil, fmt.Errorf("private key must be PEM type %q", privateKeyType)
	}
	if len(block.Bytes) != ed25519.SeedSize {
		return nil, fmt.Errorf("private key seed must be %d bytes", ed25519.SeedSize)
	}
	return ed25519.NewKeyFromSeed(block.Bytes), nil
}

func LoadPublicKey(path string) (ed25519.PublicKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil || block.Type != publicKeyType {
		return nil, fmt.Errorf("public key must be PEM type %q", publicKeyType)
	}
	if len(block.Bytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key must be %d bytes", ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(block.Bytes), nil
}

// SignFile returns a detached signature over the exact bundle bytes.
func SignFile(bundlePath, privateKeyPath string) ([]byte, error) {
	key, err := LoadPrivateKey(privateKeyPath)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(bundlePath)
	if err != nil {
		return nil, err
	}
	signature := ed25519.Sign(key, raw)
	encoded := base64.StdEncoding.EncodeToString(signature) + "\n"
	return []byte(encoded), nil
}

func WriteSignature(bundlePath, privateKeyPath, signaturePath string) error {
	signature, err := SignFile(bundlePath, privateKeyPath)
	if err != nil {
		return err
	}
	return os.WriteFile(signaturePath, signature, 0o644)
}

func VerifySignature(bundlePath, signaturePath, publicKeyPath string) error {
	publicKey, err := LoadPublicKey(publicKeyPath)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(bundlePath)
	if err != nil {
		return err
	}
	encoded, err := os.ReadFile(signaturePath)
	if err != nil {
		return err
	}
	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(encoded)))
	if err != nil {
		return fmt.Errorf("signature is not valid base64: %w", err)
	}
	if !ed25519.Verify(publicKey, raw, signature) {
		return fmt.Errorf("bundle signature does not match")
	}
	return nil
}
