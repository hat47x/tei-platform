package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
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
	bom := []byte{0xef, 0xbb, 0xbf}
	if !bytes.HasPrefix(raw, bom) {
		t.Fatal("captured fixture no longer has the UTF-8 BOM used by this input-contract regression")
	}
	plain := bytes.TrimPrefix(raw, bom)

	dir := t.TempDir()
	bomPath := filepath.Join(dir, "bom-analysis.json")
	plainPath := filepath.Join(dir, "plain-analysis.json")
	for path, data := range map[string][]byte{bomPath: raw, plainPath: plain} {
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
		OK    bool                  `json:"ok"`
		Error *cliInspectionFailure `json:"error"`
		Result struct {
			AcceptedMeaning bool   `json:"acceptedMeaning"`
			OwnerDecision   string `json:"ownerDecision"`
			Attempt struct {
				EvidenceID string `json:"evidenceId"`
				Provider struct {
					Version string `json:"version"`
				} `json:"provider"`
			} `json:"attempt"`
			Candidates []struct {
				Kind string `json:"kind"`
				Facts struct {
					ExternalIdentity   string `json:"externalIdentity"`
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

	// The semantic-graph selector accepts the original, BOM-prefixed bytes.
	exit, withBOM := external(bomPath, "")
	if exit != 0 || !withBOM.OK || withBOM.Result == nil {
		t.Fatalf("R2 refused captured bytes: exit=%d reply=%+v", exit, withBOM)
	}
	if withBOM.Result.Attempt.ReceivedDigest != digest(raw) ||
		withBOM.Result.Relation == nil ||
		withBOM.Result.Relation.RelationID != semanticRelationID ||
		withBOM.Result.Observation == nil ||
		withBOM.Result.Observation.NodeID != semanticNodeID {
		t.Fatalf("R2 lost its raw digest or semantic-graph selector: %+v", withBOM.Result)
	}

	// The intake currently requires a single BOM-free JSON envelope; it must
	// reject the raw bytes rather than silently claim a compatible input.
	exit, rejected := intake(bomPath)
	if exit != 2 || rejected.OK || rejected.Error == nil ||
		rejected.Error.Code != "EVIDENCE_INVALID_JSON" {
		t.Fatalf("intake unexpectedly accepted BOM bytes: exit=%d reply=%+v", exit, rejected)
	}

	// A BOM-free copy is a different raw delivery, despite the same producer
	// Evidence identity. The two commands still inspect *different graphs*.
	exit, withoutBOM := external(plainPath, "")
	if exit != 0 || !withoutBOM.OK || withoutBOM.Result == nil {
		t.Fatalf("R2 refused normalized bytes: exit=%d reply=%+v", exit, withoutBOM)
	}
	if withoutBOM.Result.Attempt.ReceivedDigest != digest(plain) ||
		withoutBOM.Result.Attempt.ReceivedDigest == withBOM.Result.Attempt.ReceivedDigest ||
		withoutBOM.Result.Attempt.EvidenceID != withBOM.Result.Attempt.EvidenceID {
		t.Fatalf("BOM normalization lost raw identity distinction: bom=%+v plain=%+v",
			withBOM.Result.Attempt, withoutBOM.Result.Attempt)
	}
	exit, review := intake(plainPath)
	if exit != 0 || !review.OK || review.Error != nil {
		t.Fatalf("intake refused BOM-free captured JSON: exit=%d reply=%+v", exit, review)
	}
	if review.Result.Attempt.EvidenceID != withBOM.Result.Attempt.EvidenceID ||
		review.Result.Attempt.Provider.Version != "1" ||
		withBOM.Result.Attempt.Provider.Version != "1" {
		t.Fatalf("producer Evidence or provider version drift: R2=%+v intake=%+v",
			withBOM.Result.Attempt, review.Result.Attempt)
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

	// Pinning the pre-normalization raw digest must not authorize the modified
	// bytes as if both CLI invocations consumed the same byte sequence.
	exit, stale := external(plainPath, withBOM.Result.Attempt.ReceivedDigest)
	if exit != 1 || stale.OK || stale.Error == nil ||
		stale.Error.Code != "EXTERNAL_CANDIDATE_STALE" {
		t.Fatalf("changed raw digest was not refused: exit=%d reply=%+v", exit, stale)
	}
}
