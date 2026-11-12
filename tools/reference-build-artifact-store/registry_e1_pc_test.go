package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A pinned subject from the test-only P0-A Go/Node Registry fixture.
// Publisher and Consumer separately read that fixture: neither a store receipt
// nor a caller-supplied mutable latest pointer is an independent expected value.
const registryE1PinnedReleaseDigest = "sha256:b486c89cb4663762e1ac0d4d2d4391f9fd74b88d716526c8d10c760a5687e4c7"

type registryE1Fixture struct {
	ReleaseText string `json:"releaseText"`
	Attestation struct {
		Subject string `json:"subject"`
	} `json:"attestation"`
	Artifacts []struct {
		Role        string `json:"role"`
		BytesBase64 string `json:"bytesBase64"`
	} `json:"artifacts"`
}

func registryE1Expected(t *testing.T) ([]artifactMemberRequest, map[string][]byte) {
	t.Helper()
	// The signature, Profile and mock Policy are checked by the independent
	// tools/reference-plugin-registry E0 tests. Do NOT treat this fixture load
	// as a production Registry Trust/Publisher authorization check.
	raw, err := os.ReadFile(filepath.Join("..", "reference-plugin-registry", "fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture registryE1Fixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if digestBytes([]byte(fixture.ReleaseText)) != registryE1PinnedReleaseDigest ||
		fixture.Attestation.Subject != registryE1PinnedReleaseDigest {
		t.Fatal("not the pinned signed Release subject")
	}
	lines := strings.Split(fixture.ReleaseText, "\n")
	if len(lines) != 7 || lines[0] != "TEI-REF-RELEASE-V0" ||
		lines[1] != "plugin=tei:plugin:fixture-alpha" ||
		lines[2] != "version=1.0.0" || lines[3] != "epoch=3" || lines[6] != "" {
		t.Fatal("unexpected pinned Release frame")
	}
	requests := []artifactMemberRequest{
		{Role: "core", Digest: strings.TrimPrefix(lines[4], "artifact.core=")},
		{Role: "ui", Digest: strings.TrimPrefix(lines[5], "artifact.ui=")},
	}
	if !strings.HasPrefix(lines[4], "artifact.core=sha256:") ||
		!strings.HasPrefix(lines[5], "artifact.ui=sha256:") ||
		len(fixture.Artifacts) != len(requests) {
		t.Fatal("pinned Release required-member set is malformed")
	}
	built := make(map[string][]byte, len(requests))
	for _, artifact := range fixture.Artifacts {
		if _, exists := built[artifact.Role]; exists {
			t.Fatal("duplicate artifact role in test fixture")
		}
		data, err := base64.StdEncoding.Strict().DecodeString(artifact.BytesBase64)
		if err != nil || base64.StdEncoding.EncodeToString(data) != artifact.BytesBase64 {
			t.Fatal("test fixture contains noncanonical base64")
		}
		built[artifact.Role] = data
	}
	for _, req := range requests {
		if digestBytes(built[req.Role]) != req.Digest {
			t.Fatalf("pinned Release member mismatch for %s", req.Role)
		}
	}
	return requests, built
}

func registryE1PublishedStore(t *testing.T) (localArtifactStore, []artifactMemberRequest) {
	t.Helper()
	root := t.TempDir()
	store, err := openLocalArtifactStore(root, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	expected, built := registryE1Expected(t)
	published, err := store.publishExactMembers(expected, built)
	if err != nil {
		t.Fatal(err)
	}
	if len(published.Members) != len(expected) {
		t.Fatal("Publisher P did not publish the complete required set")
	}
	for _, member := range published.Members {
		if member.Published.Operation != "put" ||
			member.Reverified.Operation != "get" ||
			member.Digest != member.Published.Digest ||
			member.Digest != member.Reverified.Digest {
			t.Fatalf("Publisher P lacks exact publication/readback: %+v", member)
		}
	}
	return store, expected
}

// A Consumer C is a fresh OS process with a separately opened Store and
// separately loaded immutable fixture. It receives neither Publisher P's
// in-memory bytes nor P's receipts. It uses the same test-only Go Store reader,
// so it is NOT an independent implementation or an external Host.
func TestRegistryE1ConsumerSubprocess(t *testing.T) {
	if os.Getenv("TEI_REGISTRY_E1_CHILD") != "1" {
		return
	}
	root := os.Getenv("TEI_REGISTRY_E1_ROOT")
	if root == "" {
		t.Fatal("Consumer C requires an explicit Store root")
	}
	requests, expectedBytes := registryE1Expected(t)
	if os.Getenv("TEI_REGISTRY_E1_MUTATION") == "swap-roles" {
		requests[0].Digest, requests[1].Digest = requests[1].Digest, requests[0].Digest
	}
	store, err := openLocalArtifactStore(root, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	retrieved, err := store.getMembers(requests)
	if err != nil {
		t.Fatal(err)
	}
	if len(retrieved.Members) != len(requests) {
		t.Fatal("Consumer C observed an incomplete result")
	}
	for _, member := range retrieved.Members {
		if member.Status != artifactMemberAvailable {
			fmt.Printf("E1-C status=%s\n", member.Status)
			t.Fatalf("Consumer C cannot acquire member %s", member.Role)
		}
		if member.Receipt.Operation != "get" ||
			member.Receipt.Digest != member.Digest ||
			!bytes.Equal(member.Bytes, expectedBytes[member.Role]) {
			fmt.Println("E1-C status=role-mismatch")
			t.Fatalf("Consumer C cannot match pinned role %s", member.Role)
		}
	}
	fmt.Printf("E1-C status=complete pid=%d release=%s count=%d\n",
		os.Getpid(), registryE1PinnedReleaseDigest, len(retrieved.Members))
}

func registryE1RunConsumer(t *testing.T, root, mutation, expectedStatus string) {
	t.Helper()
	command := exec.Command(os.Args[0], "-test.run=^TestRegistryE1ConsumerSubprocess$", "-test.v")
	command.Env = append(os.Environ(),
		"TEI_REGISTRY_E1_CHILD=1", "TEI_REGISTRY_E1_ROOT="+root,
		"TEI_REGISTRY_E1_MUTATION="+mutation)
	output, err := command.CombinedOutput()
	if !bytes.Contains(output, []byte("E1-C status="+expectedStatus)) {
		t.Fatalf("Consumer C did not report %s: %s (err=%v)", expectedStatus, output, err)
	}
	if (err == nil) != (expectedStatus == "complete") {
		t.Fatalf("Consumer C exit result inconsistent with %s: %s (err=%v)", expectedStatus, output, err)
	}
	if expectedStatus == "complete" {
		if bytes.Contains(output, []byte(fmt.Sprintf("pid=%d ", os.Getpid()))) {
			t.Fatal("Publisher and Consumer did not use separate process identities")
		}
		if !bytes.Contains(output, []byte("count=2")) {
			t.Fatal("Consumer C did not retrieve both required member roles")
		}
	}
}

func TestRegistryE1PublisherAndConsumerDifferentProcesses(t *testing.T) {
	t.Run("complete publication and independent read attempt", func(t *testing.T) {
		publisher, _ := registryE1PublishedStore(t)
		registryE1RunConsumer(t, publisher.root, "", "complete")
	})
	t.Run("publication receipt does not survive missing member", func(t *testing.T) {
		publisher, expected := registryE1PublishedStore(t)
		memberPath, err := publisher.pathForDigest(expected[1].Digest)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(memberPath); err != nil {
			t.Fatal(err)
		}
		registryE1RunConsumer(t, publisher.root, "", "missing")
	})
	t.Run("publication receipt does not survive corrupted member", func(t *testing.T) {
		publisher, expected := registryE1PublishedStore(t)
		memberPath, err := publisher.pathForDigest(expected[1].Digest)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(memberPath, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(memberPath, []byte("tampered-after-P-readback"), 0o600); err != nil {
			t.Fatal(err)
		}
		registryE1RunConsumer(t, publisher.root, "", "corrupt")
	})
	t.Run("consumer cannot swap roles even when both bytes exist", func(t *testing.T) {
		publisher, _ := registryE1PublishedStore(t)
		registryE1RunConsumer(t, publisher.root, "swap-roles", "role-mismatch")
	})
}
