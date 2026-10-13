package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type memberFixture struct {
	role  string
	value []byte
}

func putMembers(t *testing.T, store localArtifactStore, fixtures []memberFixture) ([]artifactMemberRequest, map[string]artifactStoreReceipt) {
	t.Helper()
	requests := make([]artifactMemberRequest, 0, len(fixtures))
	receipts := map[string]artifactStoreReceipt{}
	for _, fixture := range fixtures {
		receipt, err := store.PutBytes(fixture.value)
		if err != nil {
			t.Fatal(err)
		}
		requests = append(requests, artifactMemberRequest{Role: fixture.role, Digest: receipt.Digest})
		receipts[fixture.role] = receipt
	}
	return requests, receipts
}

func memberByRole(t *testing.T, members artifactMembers, role string) artifactMember {
	t.Helper()
	for _, member := range members.Members {
		if member.Role == role {
			return member
		}
	}
	t.Fatalf("role %s is not in the result", role)
	return artifactMember{}
}

func TestGetMembersReturnsEveryRoleWhenAllAreStored(t *testing.T) {
	store := openTestStore(t, 1<<20)
	requests, _ := putMembers(t, store, []memberFixture{
		{"manifest", []byte(`{"members":2}`)},
		{"executable", []byte("executable bytes")},
	})

	members, err := store.getMembers(requests)
	if err != nil {
		t.Fatal(err)
	}
	if !members.Available() {
		t.Fatalf("all roles are stored but the set was reported unavailable: %#v", members)
	}
	if len(members.Members) != 2 || members.Members[0].Role != "executable" || members.Members[1].Role != "manifest" {
		t.Fatalf("members are not returned in role order: %#v", members.Members)
	}
	if got := memberByRole(t, members, "executable"); !bytes.Equal(got.Bytes, []byte("executable bytes")) || got.Receipt.Operation != "get" {
		t.Fatalf("executable member: %#v", got)
	}
}

// manifestがRoleを列挙していても、一件のbytesが無ければ集合を利用可能と報告しない。
func TestGetMembersDoesNotReportAvailableWhenOneMemberIsMissing(t *testing.T) {
	store := openTestStore(t, 1<<20)
	requests, _ := putMembers(t, store, []memberFixture{{"manifest", []byte("manifest bytes")}})
	// 保存していないbytesのdigestを、同じ集合のRoleとして要求する。
	requests = append(requests, artifactMemberRequest{Role: "symbols", Digest: digestBytes([]byte("never stored"))})

	members, err := store.getMembers(requests)
	if err != nil {
		t.Fatal(err)
	}
	if members.Available() {
		t.Fatal("a set with a missing member was reported available from its manifest alone")
	}
	if got := memberByRole(t, members, "symbols"); got.Status != artifactMemberMissing || got.Bytes != nil {
		t.Fatalf("missing member: %#v", got)
	}
	if got := memberByRole(t, members, "manifest"); got.Status != artifactMemberAvailable {
		t.Fatalf("an intact member was affected by the missing one: %#v", got)
	}
}

func TestGetMembersDistinguishesCorruptionFromMissing(t *testing.T) {
	store := openTestStore(t, 1<<20)
	requests, receipts := putMembers(t, store, []memberFixture{
		{"manifest", []byte("manifest bytes")},
		{"executable", []byte("executable bytes")},
	})
	path := filepath.Join(store.root, filepath.FromSlash(receipts["executable"].Locator))
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}

	members, err := store.getMembers(requests)
	if err != nil {
		t.Fatal(err)
	}
	if members.Available() {
		t.Fatal("a set with a corrupt member was reported available")
	}
	got := memberByRole(t, members, "executable")
	if got.Status != artifactMemberCorrupt {
		t.Fatalf("corruption was not distinguished from absence: %#v", got)
	}
	if got.Bytes != nil {
		t.Fatal("corrupt bytes were returned to the caller")
	}
}

func TestGetMembersRejectsMalformedRequests(t *testing.T) {
	store := openTestStore(t, 1<<20)
	valid := digestBytes([]byte("x"))
	cases := map[string][]artifactMemberRequest{
		"empty request":    nil,
		"blank role":       {{Role: " ", Digest: valid}},
		"duplicate role":   {{Role: "a", Digest: valid}, {Role: "a", Digest: valid}},
		"malformed digest": {{Role: "a", Digest: "sha256:zz"}},
		"missing prefix":   {{Role: "a", Digest: strings.Repeat("a", 64)}},
	}
	for name, requests := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := store.getMembers(requests); err == nil {
				t.Fatal("malformed member request was accepted")
			}
		})
	}
}

// 再buildしたbytesが期待digestと一致するときだけ、storeへ採用する。
func TestAdoptRebuiltAcceptsOnlyBytesWithTheExpectedDigest(t *testing.T) {
	store := openTestStore(t, 1<<20)
	original := []byte("deterministically rebuilt artifact")
	expected := digestBytes(original)

	// cache missの状態から、同じbytesを再buildして採用する。
	if _, _, err := store.GetBytes(expected); !errors.Is(err, errArtifactNotFound) {
		t.Fatalf("precondition: expected a cache miss, got %v", err)
	}
	receipt, err := store.adoptRebuilt(expected, original)
	if err != nil {
		t.Fatalf("rebuilt bytes with the expected digest were refused: %v", err)
	}
	if receipt.Digest != expected {
		t.Fatalf("adopted receipt digest %s != expected %s", receipt.Digest, expected)
	}
	got, _, err := store.GetBytes(expected)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatalf("adopted bytes are not retrievable by the expected digest: %v", err)
	}

	// 同じbytesを再度採用しても冪等である。16進部分の大文字は同じdigestとして扱う。
	upperHex := "sha256:" + strings.ToUpper(strings.TrimPrefix(expected, "sha256:"))
	again, err := store.adoptRebuilt(upperHex, original)
	if err != nil || again.Operation != "put-existing" {
		t.Fatalf("repeated adoption: %#v err=%v", again, err)
	}
	// アルゴリズム名まで大文字にした形式は、既存のdigest正規化と同じく受け付けない。
	if _, err := store.adoptRebuilt(strings.ToUpper(expected), original); err == nil {
		t.Fatal("a non-canonical algorithm prefix was accepted")
	}
}

func TestAdoptRebuiltRejectsDifferentBytesWithoutWritingThem(t *testing.T) {
	store := openTestStore(t, 1<<20)
	expected := digestBytes([]byte("the locked artifact"))
	different := []byte("a different build output")

	_, err := store.adoptRebuilt(expected, different)
	if !errors.Is(err, errRebuildDigestMismatch) {
		t.Fatalf("different bytes were not refused as a digest mismatch: %v", err)
	}
	// 期待digestで取得できず、別のbytesもstoreへ書き込まれていない。
	if _, _, err := store.GetBytes(expected); !errors.Is(err, errArtifactNotFound) {
		t.Fatalf("expected digest became retrievable after a refused adoption: %v", err)
	}
	if _, _, err := store.GetBytes(digestBytes(different)); !errors.Is(err, errArtifactNotFound) {
		t.Fatalf("refused bytes were written to the store: %v", err)
	}
}

func TestAdoptRebuiltDoesNotRepairCorruptStoredObject(t *testing.T) {
	store := openTestStore(t, 1<<20)
	value := []byte("locked bytes")
	receipt, err := store.PutBytes(value)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.root, filepath.FromSlash(receipt.Locator))
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := store.adoptRebuilt(receipt.Digest, value); !errors.Is(err, errArtifactCorrupt) {
		t.Fatalf("a corrupt object was silently replaced by rebuilt bytes: %v", err)
	}
}

func TestAdoptRebuiltRejectsMalformedExpectedDigest(t *testing.T) {
	store := openTestStore(t, 1<<20)
	if _, err := store.adoptRebuilt("latest", []byte("x")); err == nil {
		t.Fatal("a mutable tag was accepted as an expected digest")
	}
}
