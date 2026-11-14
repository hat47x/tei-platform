package studiohost

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"tei/internal/externalanalysis"
	"tei/internal/operations"
)

const (
	localREAEvidenceID = "ev_a88c70bcd066b2f04fdf97cb349369f0779b9c68cc4ac6c0e5f38f3539477a93"
	localREANodeID     = "jsrg_node_237478f2a27c9614a522cbfc7ab60fa96943a458d164945d25af634f1240b652"
	localREARelationID = "jsrg_relation_0f6757a241c56fd84ecaec91f538d8297d222eb5625e4cdbcb2de7603f330718"
	localREADigest     = "sha256:7ea3780d8dc23d00c380145af62d63bf194e39decbaefab3d82c5a3b4134bba9"
)

var localREAReadAt = time.Date(2026, 11, 11, 5, 5, 0, 0, time.UTC)

func newLocalREAHostFixture(t *testing.T) (*ReadOnly, LocalREABinding, []byte) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "externalanalysis", "testdata", "rea-6.8.0", "analysis.json"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "analysis.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	host, err := NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	binding := LocalREABinding{
		WorkspaceID: "workspace-1",
		Selector:    "review-one",
		File:        "analysis.json",
		Version:     "6.8.0",
		EvidenceID:  localREAEvidenceID,
		Digest:      localREADigest,
		Tool:        externalanalysis.Tool{Name: "rea", Present: true, ResolvedPath: "/private/rea-executable"},
	}
	return host, binding, raw
}

func TestLocalREAHostBoundReaderProjectsTheExactCapturedEvidence(t *testing.T) {
	host, binding, raw := newLocalREAHostFixture(t)
	reader, err := NewLocalREAReader(host, []LocalREABinding{binding})
	if err != nil {
		t.Fatal(err)
	}
	// Changing a caller's configuration after construction must not retarget
	// a Host-pinned selector.
	binding.File = "other.json"
	result := reader.Read("workspace-1", "review-one", localREANodeID, localREARelationID, localREAReadAt)
	if !result.OK || result.Failure != nil || result.Value == nil {
		t.Fatalf("Host Evidence read failed: %+v", result.Failure)
	}
	want, err := operations.ExternalCandidatesFromBytes(operations.ExternalCandidateRequest{
		ToolVersion:            "6.8.0",
		ObservationID:          localREANodeID,
		RelationID:             localREARelationID,
		ExpectedEvidenceID:     localREAEvidenceID,
		ExpectedReceivedDigest: localREADigest,
		Tool:                   externalanalysis.Tool{Name: "rea", Present: true},
	}, raw, localREAReadAt)
	if err != nil || !reflect.DeepEqual(*result.Value, want) {
		t.Fatalf("Host fixed read differs from R2 receipt/selection: err=%v", err)
	}
	if result.Value.Attempt.ReceivedDigest != localREADigest || result.Value.Coverage == nil ||
		result.Value.Observation == nil || result.Value.Relation == nil ||
		result.Value.Unselected.Observations == 0 || len(result.Value.Unknowns) == 0 {
		t.Fatalf("loss of provenance or partial evidence: %+v", result.Value)
	}
	serial, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(serial), "/private/rea-executable") {
		t.Fatalf("local Host returned an executable path: %s", serial)
	}
}

func TestLocalREAHostBoundReaderDoesNotAcceptUnboundResources(t *testing.T) {
	host, binding, _ := newLocalREAHostFixture(t)
	reader, err := NewLocalREAReader(host, []LocalREABinding{binding})
	if err != nil {
		t.Fatal(err)
	}
	unknown := reader.Read("workspace-1", "../analysis.json", localREANodeID, localREARelationID, localREAReadAt)
	if unknown.OK || unknown.Failure == nil || unknown.Failure.Code != "OPERATION_FAILED" {
		t.Fatalf("caller-selected path reached Host: %+v", unknown)
	}
	otherWorkspace := reader.Read("workspace-2", "review-one", localREANodeID, localREARelationID, localREAReadAt)
	if otherWorkspace.OK || otherWorkspace.Failure == nil || otherWorkspace.Failure.Code != "WORKSPACE_NOT_ALLOWED" {
		t.Fatalf("cross-workspace selection succeeded: %+v", otherWorkspace)
	}
	wrongRelation := reader.Read("workspace-1", "review-one", localREANodeID, "private_relation_secret", localREAReadAt)
	if wrongRelation.OK || wrongRelation.Failure == nil || wrongRelation.Failure.Code != "OPERATION_FAILED" ||
		strings.Contains(wrongRelation.Failure.Message, "private_relation_secret") {
		t.Fatalf("R2 error leaked external identities: %+v", wrongRelation)
	}
}

func TestLocalREAHostBoundReaderRejectsMutationAndOversize(t *testing.T) {
	host, binding, raw := newLocalREAHostFixture(t)
	reader, err := NewLocalREAReader(host, []LocalREABinding{binding})
	if err != nil {
		t.Fatal(err)
	}
	root := host.roots["workspace-1"]
	for _, changed := range [][]byte{[]byte("private-not-json"), append(append([]byte{}, raw...), '\n'), make([]byte, localREAReadLimitBytes+1)} {
		if err := os.WriteFile(filepath.Join(root, "analysis.json"), changed, 0600); err != nil {
			t.Fatal(err)
		}
		got := reader.Read("workspace-1", "review-one", localREANodeID, localREARelationID, localREAReadAt)
		if got.OK || got.Value != nil || got.Failure == nil || got.Failure.Code != "OPERATION_FAILED" ||
			strings.Contains(got.Failure.Message, "private") {
			t.Fatalf("modified/oversized evidence projected or leaked: %+v", got)
		}
	}
}

func TestLocalREAHostBoundReaderRejectsRootReplacementAndSymlinkEscape(t *testing.T) {
	root, replacement := rootReplacementPair(t)
	host, err := NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	binding := LocalREABinding{
		WorkspaceID: "workspace-1", Selector: "fixed", File: "analysis.json",
		Version: "6.8.0", EvidenceID: localREAEvidenceID, Digest: localREADigest,
	}
	reader, err := NewLocalREAReader(host, []LocalREABinding{binding})
	if err != nil {
		t.Fatal(err)
	}
	replaceWorkspaceRoot(t, root, replacement)
	changed := reader.Read("workspace-1", "fixed", localREANodeID, localREARelationID, localREAReadAt)
	if changed.OK || changed.Failure == nil || changed.Failure.Code != "WORKSPACE_ROOT_CHANGED" {
		t.Fatalf("Host followed changed root: %+v", changed)
	}

	host2, binding2, raw := newLocalREAHostFixture(t)
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, raw, 0600); err != nil {
		t.Fatal(err)
	}
	binding2.File = "escape.json"
	if err := os.Symlink(outside, filepath.Join(host2.roots["workspace-1"], binding2.File)); err != nil {
		t.Skipf("cannot test symlink on this platform: %v", err)
	}
	reader2, err := NewLocalREAReader(host2, []LocalREABinding{binding2})
	if err != nil {
		t.Fatal(err)
	}
	escaped := reader2.Read("workspace-1", "review-one", localREANodeID, localREARelationID, localREAReadAt)
	if escaped.OK || escaped.Value != nil || escaped.Failure == nil || escaped.Failure.Code != "OPERATION_FAILED" {
		t.Fatalf("Host followed escaping symlink: %+v", escaped)
	}
}

func TestLocalREAHostBoundReaderRejectsInvalidStartupBindings(t *testing.T) {
	host, valid, _ := newLocalREAHostFixture(t)
	if _, err := NewLocalREAReader(nil, []LocalREABinding{valid}); err == nil {
		t.Fatal("nil Host accepted")
	}
	if _, err := NewLocalREAReader(host, nil); err == nil {
		t.Fatal("empty binding accepted")
	}
	for _, path := range []string{"../analysis.json", "/private/analysis.json", "./analysis.json", "model.xead"} {
		changed := valid
		changed.File = path
		if _, err := NewLocalREAReader(host, []LocalREABinding{changed}); err == nil {
			t.Fatalf("unsafe binding file %q accepted", path)
		}
	}
	for _, mutate := range []func(*LocalREABinding){
		func(b *LocalREABinding) { b.WorkspaceID = "workspace-2" },
		func(b *LocalREABinding) { b.Digest = "sha256:not-a-real-digest" },
		func(b *LocalREABinding) { b.Selector = "" },
		func(b *LocalREABinding) { b.EvidenceID = "" },
	} {
		changed := valid
		mutate(&changed)
		if _, err := NewLocalREAReader(host, []LocalREABinding{changed}); err == nil {
			t.Fatalf("invalid Host-configured binding %+v was accepted", changed)
		}
	}
	if _, err := NewLocalREAReader(host, []LocalREABinding{valid, valid}); err == nil {
		t.Fatal("duplicate binding accepted")
	}
}
