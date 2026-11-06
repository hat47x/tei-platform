package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// この試験は、定義を読む側がSourceの置き場所や注記を固定対象にしていないことを
// 現在のreference/workspaceの複製で確かめる。実Providerの実行、他のEditor・Toolでの
// 同じ扱い、固定したGate B/C版での再現、実利用者の負担は示さない。
func TestReferenceSourceRelocationKeepsDefinitionIdentity(t *testing.T) {
	dir := scratchWorkspace(t)
	bin := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if output, err := exec.Command("go", "build", "-o", bin, "../../cmd/tei").CombinedOutput(); err != nil {
		t.Fatalf("teiをbuildできません: %v\n%s", err, output)
	}
	run := func(wantOK bool, args ...string) []byte {
		t.Helper()
		cmd := exec.Command(bin, args...)
		var out, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &stderr
		err := cmd.Run()
		ok := err == nil
		if err != nil {
			if _, isExit := err.(*exec.ExitError); !isExit {
				t.Fatal(err)
			}
		}
		if ok != wantOK {
			t.Fatalf("%v の成功=%v、期待=%v\nstdout=%s\nstderr=%s", args, ok, wantOK, out.String(), stderr.String())
		}
		return out.Bytes()
	}
	lock := func() []byte {
		t.Helper()
		var envelope struct {
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(run(true, "plugin-lock", "-workspace", dir), &envelope); err != nil || len(envelope.Result) == 0 {
			t.Fatalf("Lockを取得できません: %v", err)
		}
		return envelope.Result
	}
	baseline := lock()

	relocate := func(from, to string) {
		src, dst := filepath.Join(dir, filepath.FromSlash(from)), filepath.Join(dir, filepath.FromSlash(to))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(src, dst); err != nil {
			t.Fatal(err)
		}
	}
	relocate("semantic/sales/order.tei", "semantic/sales/types/order.tei")
	relocate("plugins/tei-health-reference/1.0.0/plugin.tei", "plugins/tei-health-reference/1.0.0/def/plugin.tei")

	run(true, "validate", "-workspace", dir)
	if movedLock := lock(); !bytes.Equal(movedLock, baseline) {
		t.Fatalf("Sourceの置き場所だけでLockが変わりました:\nbefore=%s\nafter=%s", baseline, movedLock)
	}
	lockPath := filepath.Join(t.TempDir(), "before-move-lock.json")
	writeFile(t, lockPath, string(baseline))
	resolved := run(true, "plugin-resolve", "-workspace", dir, "-environment", "development", "-plugin-lock", lockPath)
	if !bytes.Contains(resolved, []byte("tei.health.reference")) {
		t.Fatalf("移動したPlugin定義が選択されていません: %s", resolved)
	}

	// 注記の追加も定義の同一性を変えない。整形や説明の追記に固定のやり直しを要求しない。
	pluginPath := filepath.Join(dir, "plugins", "tei-health-reference", "1.0.0", "def", "plugin.tei")
	content, err := os.ReadFile(pluginPath)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, pluginPath, string(content)+"\n// 移動後に追加した注記\n")
	run(true, "validate", "-workspace", dir)
	if commented := lock(); !bytes.Equal(commented, baseline) {
		t.Fatalf("注記の追加でLockが変わりました:\nbefore=%s\nafter=%s", baseline, commented)
	}

	// 置き場所ではなく内容の存在が対象であることを負例で示す。
	if err := os.Remove(filepath.Join(dir, "semantic", "sales", "types", "order.tei")); err != nil {
		t.Fatal(err)
	}
	run(false, "validate", "-workspace", dir)
}
