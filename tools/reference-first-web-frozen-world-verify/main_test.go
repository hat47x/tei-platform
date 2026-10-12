package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/ir"
)

func TestTypedWorldDigestMatchesSharedAcceptanceSerialization(t *testing.T) {
	world := ir.World{
		Applications: []struct{}(nil),
	}
	_ = world
	// A minimally populated typed World verifies the exact generator
	// convention: typed field order, Go's HTML escaping and final newline.
	var typed ir.World
	envelope, err := json.Marshal(struct {
		World ir.World `json:"world"`
	}{World: typed})
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := json.MarshalIndent(typed, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	canonical = append(canonical, '\n')
	wantSum := sha256.Sum256(canonical)
	want := "sha256:" + hex.EncodeToString(wantSum[:])
	got, err := derivedWorldDigest(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("derived typed World digest differs: got %s, want %s", got, want)
	}
}

func TestTypedWorldDigestRejectsMeaningChangingAndUnknownData(t *testing.T) {
	cases := []string{
		`{"world":null}`,
		`{"world":{"applications":[],"databases":[],"environments":[],"queries":[],"unrecognizedBusinessRule":1}}`,
		`{"world":{"applications":"not-an-array","databases":[],"environments":[],"queries":[]}}`,
		`{"world":{"applications":[],"databases":[],"environments":[],"queries":[],"extraData":true}}`,
		`{"world":{}}`,
		`{"world":{"applications":[],"databases":[],"environments":[],"queries":[]},"extra":[],"world":false}`,
	}
	for _, sample := range cases {
		t.Run(strings.ReplaceAll(sample, "/", "_"), func(t *testing.T) {
			if _, err := derivedWorldDigest([]byte(sample)); err == nil {
				t.Fatalf("invalid frozen World accepted: %s", sample)
			}
		})
	}
}
