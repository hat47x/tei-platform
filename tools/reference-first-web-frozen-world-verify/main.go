// Diagnostic read-only World reserialization using the same typed IR and
// encoding/json convention as the existing build-side World generator.
// No Workspace parser, Acceptance service, new Lock, or permission grant.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"

	"tei/internal/ir"
)

type frozenEnvelope struct {
	World json.RawMessage `json:"world"`
}

func derivedWorldDigest(frozen []byte) (string, error) {
	var envelope frozenEnvelope
	if err := json.Unmarshal(frozen, &envelope); err != nil {
		return "", fmt.Errorf("parse frozen World envelope: %w", err)
	}
	if len(envelope.World) == 0 || bytes.Equal(bytes.TrimSpace(envelope.World), []byte("null")) {
		return "", errors.New("missing frozen normalized World")
	}
	decoder := json.NewDecoder(bytes.NewReader(envelope.World))
	decoder.DisallowUnknownFields()
	var world ir.World
	if err := decoder.Decode(&world); err != nil {
		return "", fmt.Errorf("decode frozen World as typed IR: %w", err)
	}
	var leftover any
	if err := decoder.Decode(&leftover); err != io.EOF {
		return "", errors.New("frozen World contains trailing JSON")
	}
	derived, err := json.MarshalIndent(world, "", "  ")
	if err != nil {
		return "", fmt.Errorf("serialize typed World: %w", err)
	}
	derived = append(derived, '\n')
	// The typed IR must represent ALL JSON data in the frozen World. Reject
	// additions that a permissive struct parser might otherwise silently
	// discard, even if the surrounding file digest was re-stamped.
	var incoming, canonical any
	if err := json.Unmarshal(envelope.World, &incoming); err != nil {
		return "", err
	}
	if err := json.Unmarshal(derived, &canonical); err != nil {
		return "", err
	}
	if !reflect.DeepEqual(incoming, canonical) {
		return "", errors.New("frozen World data differs from typed IR serialization")
	}
	sum := sha256.Sum256(derived)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./tools/reference-first-web-frozen-world-verify /absolute/frozen-world.json")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	value, err := derivedWorldDigest(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(value)
}
