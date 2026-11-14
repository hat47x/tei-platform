package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"unicode/utf16"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The two existing read-only commands consume different graph projections from
// one REA delivery. This regression tests their input and identity boundaries;
// it is not a second product Consumer or an acceptance of business meaning.
func TestREADualEntryCapturedInputAndIdentityBoundaries(t *testing.T) {
	fixture := filepath.Join("..", "..", "internal", "externalanalysis", "testdata", "rea-6.8.0", "analysis.json")
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	// REA 6.8.0's captured output is UTF-16LE with a BOM. Preserve the
	// original raw bytes and derive a separate UTF-8 JSON input for intake.
	if !bytes.HasPrefix(raw, []byte{0xff, 0xfe}) || len(raw)%2 != 0 {
		t.Fatal("captured fixture is not the recorded UTF-16LE BOM delivery")
	}
	units := make([]uint16, 0, (len(raw)-2)/2)
	for i := 2; i < len(raw); i += 2 {
		units = append(units, uint16(raw[i])|uint16(raw[i+1])<<8)
	}
	utf8JSON := []byte(string(utf16.Decode(units)))
	if !json.Valid(utf8JSON) {
		t.Fatal("transcoded REA fixture is not a single valid JSON document")
	}

	dir := t.TempDir()
	originalPath := filepath.Join(dir, "original-utf16le.json")
	utf8Path := filepath.Join(dir, "derived-utf8.json")
	for path, data := range map[string][]byte{originalPath: raw, utf8Path: utf8JSON} {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	const (
		semanticNodeID     = "jsrg_node_237478f2a27c9614a522cbfc7ab60fa96943a458d164945d25af634f1240b652"
		semanticRelationID = "jsrg_relation_0f6757a241c56fd84ecaec91f538d8297d222eb5625e4cdbcb2de7603f330718"
	)
	external := func(path, expectedDigest string) (int, externalCandidatesCLIResponse) {
		t.Helper()
		args := []string{"-evidence-path", path, "-tool-version", "6.8.0",
			"-observation-id", semanticNodeID, "-relation-id", semanticRelationID}
		if expectedDigest != "" {
			args = append(args, "-expected-received-digest", expectedDigest)
		}
		var out strings.Builder
		exit, err := runExternalCandidatesCommand(args, &out)
		if err != nil {
			t.Fatalf("external-candidates: %v", err)
		}
		var response externalCandidatesCLIResponse
		if err := json.Unmarshal([]byte(out.String()), &response); err != nil {
			t.Fatalf("external-candidates non-JSON reply: %v: %s", err, out.String())
		}
		return exit, response
	}
	type intakeResponse struct {
		OK     bool                  `json:"ok"`
		Error  *cliInspectionFailure `json:"error"`
		Result struct {
			AcceptedMeaning bool   `json:"acceptedMeaning"`
			OwnerDecision   string `json:"ownerDecision"`
			Attempt         struct {
				EvidenceID string `json:"evidenceId"`
				Provider   struct {
					Version string `json:"version"`
				} `json:"provider"`
			} `json:"attempt"`
			Candidates []struct {
				Kind  string `json:"kind"`
				Facts struct {
					ExternalIdentity     string `json:"externalIdentity"`
					ExternalOccurrenceID string `json:"externalOccurrenceId"`
				} `json:"facts"`
			} `json:"candidates"`
		} `json:"result"`
	}
	intake := func(path string) (int, intakeResponse) {
		t.Helper()
		var out strings.Builder
		exit, err := runThreeElementCommand([]string{"rea-intake", "-evidence", path}, &out)
		if err != nil {
			t.Fatalf("three-element rea-intake: %v", err)
		}
		var response intakeResponse
		if err := json.Unmarshal([]byte(out.String()), &response); err != nil {
			t.Fatalf("three-element rea-intake non-JSON reply: %v: %s", err, out.String())
		}
		return exit, response
	}
	digest := func(data []byte) string {
		sum := sha256.Sum256(data)
		return fmt.Sprintf("sha256:%x", sum)
	}

	// The semantic-graph selector accepts the captured UTF-16LE delivery.
	exit, original := external(originalPath, "")
	if exit != 0 || !original.OK || original.Result == nil {
		t.Fatalf("R2 refused captured UTF-16LE bytes: exit=%d error=%+v", exit, original.Error)
	}
	if original.Result.Attempt.ReceivedDigest != digest(raw) ||
		original.Result.Relation == nil ||
		original.Result.Relation.RelationID != semanticRelationID ||
		original.Result.Observation == nil ||
		original.Result.Observation.NodeID != semanticNodeID {
		t.Fatalf("R2 lost its raw digest or semantic-graph selector: %+v", original.Result)
	}

	// The triad intake currently requires UTF-8 JSON, not the raw UTF-16LE.
	exit, rejected := intake(originalPath)
	if exit != 2 || rejected.OK || rejected.Error == nil ||
		rejected.Error.Code != "EVIDENCE_INVALID_JSON" {
		t.Fatalf("intake unexpectedly accepted UTF-16LE bytes: exit=%d reply=%+v", exit, rejected)
	}

	// Both Readers can inspect the explicit UTF-8 *copy*, but its raw digest
	// differs from the original. The two Readers inspect different graphs.
	exit, derived := external(utf8Path, "")
	if exit != 0 || !derived.OK || derived.Result == nil {
		t.Fatalf("R2 refused transcoded UTF-8 bytes: exit=%d error=%+v", exit, derived.Error)
	}
	if derived.Result.Attempt.ReceivedDigest != digest(utf8JSON) ||
		derived.Result.Attempt.ReceivedDigest == original.Result.Attempt.ReceivedDigest ||
		derived.Result.Attempt.EvidenceID != original.Result.Attempt.EvidenceID {
		t.Fatalf("transcoding conflated raw identities: original=%+v derived=%+v",
			original.Result.Attempt, derived.Result.Attempt)
	}
	exit, review := intake(utf8Path)
	if exit != 0 || !review.OK || review.Error != nil {
		t.Fatalf("intake refused derived UTF-8 JSON: exit=%d reply=%+v", exit, review)
	}
	if review.Result.Attempt.EvidenceID != original.Result.Attempt.EvidenceID ||
		review.Result.Attempt.Provider.Version != "1" ||
		original.Result.Attempt.Provider.Version != "1" {
		t.Fatalf("producer Evidence or provider version drift: R2=%+v intake=%+v",
			original.Result.Attempt, review.Result.Attempt)
	}
	if review.Result.AcceptedMeaning || review.Result.OwnerDecision != "pending" {
		t.Fatalf("a read-only intake is not Owner acceptance: %+v", review.Result)
	}
	seenApplicationEdge := false
	for _, candidate := range review.Result.Candidates {
		if candidate.Kind != "cross-reference" {
			continue
		}
		if !strings.HasPrefix(candidate.Facts.ExternalIdentity, "jag_edge_") ||
			candidate.Facts.ExternalIdentity != candidate.Facts.ExternalOccurrenceID {
			t.Fatalf("unexpected application-graph edge identity: %+v", candidate.Facts)
		}
		if candidate.Facts.ExternalIdentity == semanticRelationID {
			t.Fatal("semantic-graph relation ID was silently substituted for application-graph edge ID")
		}
		seenApplicationEdge = true
	}
	if !seenApplicationEdge {
		t.Fatal("the captured application graph produced no relation candidates")
	}

	// Pinning the original UTF-16LE raw digest cannot authorize UTF-8 bytes.
	exit, stale := external(utf8Path, original.Result.Attempt.ReceivedDigest)
	if exit != 1 || stale.OK || stale.Error == nil ||
		stale.Error.Code != "EXTERNAL_CANDIDATE_STALE" {
		t.Fatalf("changed raw digest was not refused: exit=%d reply=%+v", exit, stale)
	}
}
