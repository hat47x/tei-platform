package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strings"

	"tei/internal/core"
)

const referenceStage2HostSignatureDomain = "tei.reference.stage2-host-approval/ed25519/v1\x00"

// The Host signs an exact, typed registration. Its public verification key is
// a caller-provisioned trust anchor, not derived from the build evidence.
func referenceStage2HostSignedBytes(registration referenceStage2HostRegistration) ([]byte, error) {
	raw, err := json.Marshal(registration)
	if err != nil {
		return nil, err
	}
	message := make([]byte, 0, len(referenceStage2HostSignatureDomain)+len(raw))
	message = append(message, referenceStage2HostSignatureDomain...)
	message = append(message, raw...)
	return message, nil
}

func readReferenceHostPinnedFile(path string, limit int64) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("Host signing trust input is required")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("Host signing trust input must be a regular non-symlink file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		return nil, fmt.Errorf("Host signing trust input exceeds byte limit")
	}
	return raw, nil
}

func verifyReferenceStage2HostSignature(options exactEngineOptions, registration referenceStage2HostRegistration) (string, error) {
	if options.HostSignature == "" || options.TrustedHostPublicKey == "" || options.TrustedHostID == "" {
		return "", fmt.Errorf("stage2 Host signature, pinned public key, and trusted Host ID are all required")
	}
	expectedID, err := core.ParseCanonicalID(options.TrustedHostID)
	if err != nil {
		return "", fmt.Errorf("trusted Host ID: %w", err)
	}
	if registration.HostID != expectedID {
		return "", fmt.Errorf("stage2 Host registration does not match pinned Host ID")
	}
	rawKey, err := readReferenceHostPinnedFile(options.TrustedHostPublicKey, 16<<10)
	if err != nil {
		return "", fmt.Errorf("stage2 trusted Host public key: %w", err)
	}
	block, remainder := pem.Decode(rawKey)
	if block == nil || block.Type != "PUBLIC KEY" || len(bytes.TrimSpace(remainder)) != 0 {
		return "", fmt.Errorf("stage2 trusted Host public key must contain one PKIX PUBLIC KEY")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse stage2 trusted Host public key: %w", err)
	}
	public, ok := parsed.(ed25519.PublicKey)
	if !ok || len(public) != ed25519.PublicKeySize {
		return "", fmt.Errorf("stage2 trusted Host key must be Ed25519")
	}
	rawSignature, err := readReferenceHostPinnedFile(options.HostSignature, 4<<10)
	if err != nil {
		return "", fmt.Errorf("stage2 Host signature: %w", err)
	}
	signature, err := base64.StdEncoding.Strict().DecodeString(string(bytes.TrimSpace(rawSignature)))
	if err != nil || len(signature) != ed25519.SignatureSize {
		return "", fmt.Errorf("stage2 Host signature is not valid Ed25519 base64")
	}
	message, err := referenceStage2HostSignedBytes(registration)
	if err != nil {
		return "", err
	}
	if !ed25519.Verify(public, message, signature) {
		return "", fmt.Errorf("stage2 Host registration Ed25519 signature verification failed")
	}
	// Include the independent trust anchor and detached signature: rotating
	// the trusted Host signing key invalidates the accepted runtime policy.
	sum := sha256.New()
	_, _ = sum.Write([]byte("tei.reference.stage2-host-approval-evidence/v1\x00"))
	_, _ = sum.Write(public)
	_, _ = sum.Write(signature)
	_, _ = sum.Write(message)
	return hex.EncodeToString(sum.Sum(nil)), nil
}
