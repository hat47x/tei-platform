package main

import (
	"errors"
	"os"
	"testing"
)

// setTestHook は試験用の差込み点を設定し、試験の終了時に必ず外す。
// 差込み点は包みの変数なので、これらの試験は並列実行しない。
func setTestHook(t *testing.T, hook func(stage string)) {
	t.Helper()
	testHook = hook
	t.Cleanup(func() { testHook = nil })
}

// replaceWithSameBytes は、path のobjectを同じbytesの別inodeへ入れ替える。
// bytesが同じなので、digest照合だけでは入替えを検出できない。
func replaceWithSameBytes(t *testing.T, path string, value []byte) {
	t.Helper()
	replacement := path + ".replacement"
	if err := os.WriteFile(replacement, value, 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
}

func TestGetRejectsObjectReplacedBetweenLstatAndOpen(t *testing.T) {
	store := openTestStore(t, 1<<20)
	value := []byte("replaced between lstat and open")
	receipt, err := store.PutBytes(value)
	if err != nil {
		t.Fatal(err)
	}
	path := receiptPath(store, receipt)

	called := false
	setTestHook(t, func(stage string) {
		if stage == hookAfterLstat {
			called = true
			replaceWithSameBytes(t, path, value)
		}
	})
	if _, _, err := store.GetBytes(receipt.Digest); !errors.Is(err, errArtifactCorrupt) {
		t.Fatalf("object replaced before open was not rejected: %v", err)
	}
	if !called {
		t.Fatal("差込み点が呼ばれていない")
	}
}

func TestGetRejectsPathReplacedAfterRead(t *testing.T) {
	store := openTestStore(t, 1<<20)
	value := []byte("replaced after read")
	receipt, err := store.PutBytes(value)
	if err != nil {
		t.Fatal(err)
	}
	path := receiptPath(store, receipt)

	called := false
	setTestHook(t, func(stage string) {
		if stage == hookBeforeFinalStat {
			called = true
			replaceWithSameBytes(t, path, value)
		}
	})
	if _, _, err := store.GetBytes(receipt.Digest); !errors.Is(err, errArtifactCorrupt) {
		t.Fatalf("path replaced after read was not rejected: %v", err)
	}
	if !called {
		t.Fatal("差込み点が呼ばれていない")
	}
}

func TestGetWithoutHookStillSucceeds(t *testing.T) {
	store := openTestStore(t, 1<<20)
	value := []byte("no replacement")
	receipt, err := store.PutBytes(value)
	if err != nil {
		t.Fatal(err)
	}
	stages := []string{}
	setTestHook(t, func(stage string) { stages = append(stages, stage) })
	if _, _, err := store.GetBytes(receipt.Digest); err != nil {
		t.Fatalf("入替えがなければ取得できる: %v", err)
	}
	if len(stages) != 2 || stages[0] != hookAfterLstat || stages[1] != hookBeforeFinalStat {
		t.Fatalf("差込み点の呼出し順: %v", stages)
	}
}

func TestPutNeverOverwritesObjectThatAppearsBeforePublish(t *testing.T) {
	value := []byte("no-clobber publication")

	t.Run("同一bytesが先に現れた場合は既存objectを保つ", func(t *testing.T) {
		store := openTestStore(t, 1<<20)
		path, err := store.pathForDigest(digestBytes(value))
		if err != nil {
			t.Fatal(err)
		}
		var planted os.FileInfo
		setTestHook(t, func(stage string) {
			if stage != hookBeforeLink {
				return
			}
			if err := os.WriteFile(path, value, 0o444); err != nil {
				t.Fatal(err)
			}
			planted, err = os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
		})
		receipt, err := store.PutBytes(value)
		if err != nil {
			t.Fatal(err)
		}
		if planted == nil {
			t.Fatal("差込み点が呼ばれていない")
		}
		if receipt.Operation != "put-existing" {
			t.Fatalf("operation=%s", receipt.Operation)
		}
		after, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !os.SameFile(planted, after) {
			t.Fatal("公開が先に現れたobjectを上書きした")
		}
	})

	t.Run("破損したobjectが先に現れた場合は修復せず拒否する", func(t *testing.T) {
		store := openTestStore(t, 1<<20)
		path, err := store.pathForDigest(digestBytes(value))
		if err != nil {
			t.Fatal(err)
		}
		setTestHook(t, func(stage string) {
			if stage == hookBeforeLink {
				if err := os.WriteFile(path, []byte("tampered"), 0o444); err != nil {
					t.Fatal(err)
				}
			}
		})
		if _, err := store.PutBytes(value); !errors.Is(err, errArtifactCorrupt) {
			t.Fatalf("破損したobjectを黙って修復した: %v", err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "tampered" {
			t.Fatalf("破損したobjectが上書きされた: %q", got)
		}
	})
}
