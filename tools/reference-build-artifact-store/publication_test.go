package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestPublishExactMembersRoundTripAndIdempotence(t *testing.T) {
	store := openTestStore(t, 1<<20)
	manifest := []byte(`{"kind":"manifest"}`)
	executable := []byte("exact executable bytes")
	expected := []artifactMemberRequest{
		{Role: "manifest", Digest: digestBytes(manifest)},
		{Role: "executable", Digest: digestBytes(executable)},
	}
	built := map[string][]byte{"manifest": manifest, "executable": executable}
	first, err := store.publishExactMembers(expected, built)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Members) != 2 || first.Members[0].Role != "executable" || first.Members[1].Role != "manifest" {
		t.Fatalf("published member roles are not deterministic: %#v", first.Members)
	}
	for _, member := range first.Members {
		if member.Published.Operation != "put" || member.Reverified.Operation != "get" {
			t.Fatalf("publication and subsequent read were conflated: %#v", member)
		}
		if member.Published.Digest != member.Digest || member.Reverified.Digest != member.Digest {
			t.Fatalf("member publication used the wrong digest: %#v", member)
		}
		got, _, err := store.GetBytes(member.Digest)
		if err != nil || !bytes.Equal(got, built[member.Role]) {
			t.Fatalf("published role %s is not retrievable by digest: %v", member.Role, err)
		}
	}
	again, err := store.publishExactMembers([]artifactMemberRequest{expected[1], expected[0]}, built)
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range again.Members {
		if member.Published.Operation != "put-existing" || member.Reverified.Operation != "get" {
			t.Fatalf("repeat publication was not idempotent: %#v", member)
		}
	}
}

func TestPublishExactMembersPreflightsEntireRoleSetBeforeWriting(t *testing.T) {
	value := []byte("valid artifact")
	other := []byte("other artifact")
	expected := []artifactMemberRequest{
		{Role: "a", Digest: digestBytes(value)},
		{Role: "b", Digest: digestBytes(other)},
	}
	cases := []struct {
		name     string
		requests []artifactMemberRequest
		outputs  map[string][]byte
	}{
		{"missing output role", expected, map[string][]byte{"a": value}},
		{"unexpected output role", expected, map[string][]byte{"a": value, "b": other, "c": []byte("extra")}},
		{"same size but different roles", expected, map[string][]byte{"a": value, "c": other}},
		{"wrong digest on later role", expected, map[string][]byte{"a": value, "b": []byte("mismatch")}},
		{"duplicate expected role", []artifactMemberRequest{{Role: "a", Digest: expected[0].Digest}, {Role: "a", Digest: expected[1].Digest}}, map[string][]byte{"a": value, "b": other}},
		{"blank expected role", []artifactMemberRequest{{Role: "a", Digest: expected[0].Digest}, {Role: " ", Digest: expected[1].Digest}}, map[string][]byte{"a": value, " ": other}},
		{"invalid expected digest", []artifactMemberRequest{{Role: "a", Digest: expected[0].Digest}, {Role: "b", Digest: "latest"}}, map[string][]byte{"a": value, "b": other}},
		{"empty request", nil, map[string][]byte{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := openTestStore(t, 1<<20)
			result, err := store.publishExactMembers(tc.requests, tc.outputs)
			if err == nil || len(result.Members) != 0 {
				t.Fatalf("invalid output set yielded publication evidence: %#v err=%v", result, err)
			}
			for _, possible := range [][]byte{value, other, []byte("mismatch"), []byte("extra")} {
				if _, _, err := store.GetBytes(digestBytes(possible)); !errors.Is(err, errArtifactNotFound) {
					t.Fatalf("preflight failure wrote bytes to CAS: %v", err)
				}
			}
		})
	}
}

func TestPublishExactMembersDoesNotClaimSuccessAfterStorageFailure(t *testing.T) {
	store := openTestStore(t, 1<<20)
	first := []byte("a real first artifact")
	second := []byte("a real second artifact")
	// The second digest is already occupied by corrupted data. Publication
	// may persist the first object, but must never claim that the set is ready.
	receipt, err := store.PutBytes(second)
	if err != nil {
		t.Fatal(err)
	}
	path := receiptPath(store, receipt)
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := store.publishExactMembers([]artifactMemberRequest{
		{Role: "a", Digest: digestBytes(first)},
		{Role: "z", Digest: digestBytes(second)},
	}, map[string][]byte{"a": first, "z": second})
	if !errors.Is(err, errArtifactCorrupt) || len(result.Members) != 0 {
		t.Fatalf("corrupted member produced successful set publication: %#v err=%v", result, err)
	}
	// There may be a valid first object from this partial publication. It
	// remains usable on its own, never as proof that the entire set exists.
	got, _, err := store.GetBytes(digestBytes(first))
	if err != nil || !bytes.Equal(got, first) {
		t.Fatalf("valid partial object was lost: %v", err)
	}
	members, err := store.getMembers([]artifactMemberRequest{
		{Role: "a", Digest: digestBytes(first)},
		{Role: "z", Digest: digestBytes(second)},
	})
	if err != nil || members.Available() || memberByRole(t, members, "z").Status != artifactMemberCorrupt {
		t.Fatalf("partial publication was incorrectly reported complete: %+v err=%v", members, err)
	}
}

func TestPublishExactMembersSupportsEmptyExactArtifact(t *testing.T) {
	store := openTestStore(t, 1<<20)
	expected := digestBytes(nil)
	receipt, err := store.publishExactMembers([]artifactMemberRequest{{Role: "empty", Digest: expected}}, map[string][]byte{"empty": nil})
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Members) != 1 || receipt.Members[0].Digest != expected {
		t.Fatalf("empty artifact content identity was lost: %+v", receipt)
	}
	got, _, err := store.GetBytes(expected)
	if err != nil || len(got) != 0 {
		t.Fatalf("empty artifact not stored exactly: %v", err)
	}
}

// Role order is observable only as a stable, sorted receipt order. It must not
// change content identity or let input map iteration affect publication.
func TestPublishExactMembersReceiptOrderIndependentOfInput(t *testing.T) {
	store := openTestStore(t, 1<<20)
	values := map[string][]byte{
		"z": []byte("z"),
		"a": []byte("a"),
		"m": []byte("m"),
	}
	req := []artifactMemberRequest{
		{Role: "z", Digest: digestBytes(values["z"])},
		{Role: "a", Digest: digestBytes(values["a"])},
		{Role: "m", Digest: digestBytes(values["m"])},
	}
	result, err := store.publishExactMembers(req, values)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, m := range result.Members {
		names = append(names, m.Role)
	}
	if !sort.StringsAreSorted(names) || strings.Join(names, ",") != "a,m,z" {
		t.Fatalf("unstable member publication order: %v", names)
	}
}

func TestPublishExactMembersRejectsLateOversizeBeforeAnyStoreWrite(t *testing.T) {
	store := openTestStore(t, 8)
	small := []byte("small")
	large := []byte("oversized")
	result, err := store.publishExactMembers([]artifactMemberRequest{
		{Role: "a", Digest: digestBytes(small)},
		{Role: "z", Digest: digestBytes(large)},
	}, map[string][]byte{"a": small, "z": large})
	if err == nil || len(result.Members) != 0 {
		t.Fatalf("oversized later member yielded publication evidence: %+v err=%v", result, err)
	}
	if _, _, err := store.GetBytes(digestBytes(small)); !errors.Is(err, errArtifactNotFound) {
		t.Fatalf("an earlier valid member was published before the late size rejection: %v", err)
	}
}

// The first Put's successful receipt is not enough to claim that the complete
// set is available. A prior member can become corrupt while a later Put is
// publishing; the final getMembers pass must then reject the entire set.
func TestPublishExactMembersRechecksEarlierMemberAfterLaterPut(t *testing.T) {
	store := openTestStore(t, 1<<20)
	first := []byte("first artifact before late corruption")
	second := []byte("second artifact remains valid")
	path, err := store.pathForDigest(digestBytes(first))
	if err != nil {
		t.Fatal(err)
	}

	publishLinks := 0
	setTestHook(t, func(stage string) {
		if stage != hookBeforeLink {
			return
		}
		publishLinks++
		if publishLinks != 2 {
			return
		}
		// First member has already been published; now change its bytes
		// just before the second member becomes visible in the CAS.
		if err := os.Chmod(path, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("tampered after first Put"), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	requests := []artifactMemberRequest{
		{Role: "a", Digest: digestBytes(first)},
		{Role: "z", Digest: digestBytes(second)},
	}
	published, err := store.publishExactMembers(requests, map[string][]byte{"a": first, "z": second})
	if err == nil || !strings.Contains(err.Error(), "incomplete after reverification") || len(published.Members) != 0 {
		t.Fatalf("late corruption yielded successful set publication: %+v err=%v", published, err)
	}
	if publishLinks != 2 {
		t.Fatalf("late corruption fixture did not pass both Put boundaries: %d", publishLinks)
	}
	members, err := store.getMembers(requests)
	if err != nil {
		t.Fatal(err)
	}
	if members.Available() ||
		memberByRole(t, members, "a").Status != artifactMemberCorrupt ||
		memberByRole(t, members, "z").Status != artifactMemberAvailable {
		t.Fatalf("late corruption lost member-specific status: %+v", members)
	}
}

// A real compiler output is distinct from the synthetic byte fixtures above.
// This tests the Build -> exact output bytes -> CAS set publication and
// reopened retrieval path, but does not claim TEI Builder integration or
// independent provenance/Host authorization of the expected digest.
func TestPublishExactMembersFromActualGoBuild(t *testing.T) {
	source := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(source, []byte("package main\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	binaryPath := filepath.Join(t.TempDir(), "reference-executable")
	cmd := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-o", binaryPath, source)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("actual Go build failed: %v\n%s", err, output)
	}
	executable, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(executable) == 0 {
		t.Fatal("actual build produced empty executable")
	}

	manifest := []byte("executable=" + digestBytes(executable) + "\n")
	expected := []artifactMemberRequest{
		{Role: "executable", Digest: digestBytes(executable)},
		{Role: "manifest", Digest: digestBytes(manifest)},
	}
	store := openTestStore(t, 16<<20)

	// A modified compiled output must not be published under the expected
	// Build digest. No member may be left available after this preflight error.
	modified := bytes.Clone(executable)
	modified[0] ^= 1
	bad, err := store.publishExactMembers(expected, map[string][]byte{
		"executable": modified,
		"manifest":   manifest,
	})
	if err == nil || len(bad.Members) != 0 {
		t.Fatalf("modified compiler output was published: %+v err=%v", bad, err)
	}
	for _, member := range expected {
		if _, _, err := store.GetBytes(member.Digest); !errors.Is(err, errArtifactNotFound) {
			t.Fatalf("preflight published a member after output mismatch: role=%s err=%v", member.Role, err)
		}
	}

	published, err := store.publishExactMembers(expected, map[string][]byte{
		"executable": executable,
		"manifest":   manifest,
	})
	if err != nil || len(published.Members) != len(expected) {
		t.Fatalf("actual Build output set publication failed: %+v err=%v", published, err)
	}
	reopened, err := openLocalArtifactStore(store.root, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := reopened.getMembers(expected)
	if err != nil || !verified.Available() {
		t.Fatalf("reopened CAS cannot reverify compiler output set: %+v err=%v", verified, err)
	}
	for _, member := range verified.Members {
		var want []byte
		switch member.Role {
		case "executable":
			want = executable
		case "manifest":
			want = manifest
		default:
			t.Fatalf("unexpected artifact role: %q", member.Role)
		}
		if !bytes.Equal(member.Bytes, want) || member.Receipt.Digest != member.Digest {
			t.Fatalf("actual compiler output changed during CAS round trip: role=%s", member.Role)
		}
	}
}

// A member disappearing after publication does not change the set's expected
// content identities. Adoption of an explicitly rebuilt member is separate
// from authorizing or running a Builder, and cannot silently update the lock.
func TestPublishedSetMissRebuildKeepsPinnedMemberDigest(t *testing.T) {
	store := openTestStore(t, 1<<20)
	executable := []byte("deterministically rebuilt executable")
	lockedDigest := digestBytes(executable)
	manifest := []byte("executable=" + lockedDigest + "\n")
	expected := []artifactMemberRequest{
		{Role: "executable", Digest: lockedDigest},
		{Role: "manifest", Digest: digestBytes(manifest)},
	}
	if _, err := store.publishExactMembers(expected, map[string][]byte{
		"executable": executable,
		"manifest":   manifest,
	}); err != nil {
		t.Fatal(err)
	}

	// Simulate eviction of only the executable from an already published set.
	// The manifest and its original reference remain available.
	path, err := store.pathForDigest(lockedDigest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	missing, err := store.getMembers(expected)
	if err != nil {
		t.Fatal(err)
	}
	if missing.Available() ||
		memberByRole(t, missing, "executable").Status != artifactMemberMissing ||
		memberByRole(t, missing, "manifest").Status != artifactMemberAvailable {
		t.Fatalf("set was available despite a missing locked member: %+v", missing)
	}

	// A different rebuild result must never replace the locked digest or
	// become a separately available fallback in the Artifact Store.
	different := bytes.Clone(executable)
	different[0] ^= 1
	if _, err := store.adoptRebuilt(lockedDigest, different); !errors.Is(err, errRebuildDigestMismatch) {
		t.Fatalf("different rebuild was accepted under the pinned digest: %v", err)
	}
	if _, _, err := store.GetBytes(digestBytes(different)); !errors.Is(err, errArtifactNotFound) {
		t.Fatalf("rejected rebuild was published as a new identity: %v", err)
	}
	stillMissing, err := store.getMembers(expected)
	if err != nil || stillMissing.Available() ||
		memberByRole(t, stillMissing, "executable").Status != artifactMemberMissing {
		t.Fatalf("rejected rebuild silently restored set availability: %+v err=%v", stillMissing, err)
	}

	// Only the exact original bytes can be adopted. The consumer must still
	// re-fetch every pinned member to establish the set's current availability.
	receipt, err := store.adoptRebuilt(lockedDigest, bytes.Clone(executable))
	if err != nil || receipt.Operation != "put" || receipt.Digest != lockedDigest {
		t.Fatalf("exact rebuild adoption lost locked identity: %+v err=%v", receipt, err)
	}
	reopened, err := openLocalArtifactStore(store.root, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := reopened.getMembers(expected)
	if err != nil || !verified.Available() {
		t.Fatalf("exact rebuild did not restore complete set availability: %+v err=%v", verified, err)
	}
	if !bytes.Equal(memberByRole(t, verified, "executable").Bytes, executable) ||
		!bytes.Equal(memberByRole(t, verified, "manifest").Bytes, manifest) ||
		expected[0].Digest != lockedDigest {
		t.Fatalf("rebuild changed the locked members: %+v", verified)
	}
}

// A successful Put receipt is not evidence that a member remains available
// while later members are being published. The final getMembers pass must
// reject an object disappearing after its Put, and a subsequent explicit
// retry may only restore the exact expected content identities.
func TestPublishExactMembersRejectsLateMissingMemberAndAllowsRetry(t *testing.T) {
	store := openTestStore(t, 1<<20)
	first := []byte("member a can disappear after its Put")
	second := []byte("member z survives the failed set")
	expected := []artifactMemberRequest{
		{Role: "a", Digest: digestBytes(first)},
		{Role: "z", Digest: digestBytes(second)},
	}
	built := map[string][]byte{"a": first, "z": second}
	firstPath, err := store.pathForDigest(digestBytes(first))
	if err != nil {
		t.Fatal(err)
	}

	linkCount := 0
	removed := false
	setTestHook(t, func(stage string) {
		if stage != hookBeforeLink {
			return
		}
		linkCount++
		if linkCount == 2 {
			// The a object exists after its Put; remove it just before z
			// is linked, leaving z valid but the expected set incomplete.
			if err := os.Remove(firstPath); err != nil {
				t.Fatal(err)
			}
			removed = true
		}
	})
	failed, err := store.publishExactMembers(expected, built)
	if !removed || linkCount != 2 {
		t.Fatalf("late disappearance did not occur after first Put: removed=%v links=%d", removed, linkCount)
	}
	if err == nil || !strings.Contains(err.Error(), "incomplete after reverification") || len(failed.Members) != 0 {
		t.Fatalf("missing earlier member yielded publication evidence: %+v err=%v", failed, err)
	}
	members, err := store.getMembers(expected)
	if err != nil || members.Available() ||
		memberByRole(t, members, "a").Status != artifactMemberMissing ||
		memberByRole(t, members, "z").Status != artifactMemberAvailable {
		t.Fatalf("partial CAS publication lost missing/available distinction: %+v err=%v", members, err)
	}

	// Retry only the original expected digest set. The existing z object
	// must be reused without clobbering, and a must be restored exactly.
	again, err := store.publishExactMembers(expected, built)
	if err != nil || len(again.Members) != 2 {
		t.Fatalf("retry after missing member failed: %+v err=%v", again, err)
	}
	if again.Members[0].Role != "a" || again.Members[0].Published.Operation != "put" ||
		again.Members[1].Role != "z" || again.Members[1].Published.Operation != "put-existing" {
		t.Fatalf("retry changed CAS publication identities: %+v", again)
	}
	for _, member := range again.Members {
		if member.Reverified.Operation != "get" || member.Published.Digest != member.Digest ||
			member.Reverified.Digest != member.Digest {
			t.Fatalf("retry lacked exact digest reverification: %+v", member)
		}
		got, _, err := store.GetBytes(member.Digest)
		if err != nil || !bytes.Equal(got, built[member.Role]) {
			t.Fatalf("retry failed to recover pinned role %q: %v", member.Role, err)
		}
	}
}
