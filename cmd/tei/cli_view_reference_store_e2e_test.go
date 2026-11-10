package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// referenceStoreWorkspace writes a workspace whose record pattern declares a
// before-create and an after-create hook over a create capability.
//
// A create is used because the reference store starts empty: an update would be
// refused as record-missing before the after phase had a point to run at.
func referenceStoreWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("tei.workspace.yaml", "workspace: cli_reference_store\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n")
	write("semantic/sales/order.tei", `namespace sales

data Order {
    attribute orderId type std.identifier
}

capability sales.create_order {
    input order sales.order
    output sales.order
    effect create sales.order
}
`)
	write("applications/order_entry/application.tei", `application order_entry {
    uses semantic sales
}
`)
	write("applications/order_entry/patterns/order-record.tei", `pattern order_entry.order_record record {
    data sales.order

    interaction {
        item sales.order.order_id

        command sales.create_order {
            input order sales.order.order_id
        }

        hook before-create {
            reads sales.order.order_id
            on-failure reject
        }

        hook after-create {
            reads sales.order.order_id
            on-failure reject
        }
    }
}
`)
	return root
}

// runViewExecute runs one `view -execute` invocation and returns stdout. An
// answer with a non-zero exit code is still an answer here: the run reports
// refusals and argument errors as JSON documents.
func runViewExecute(t *testing.T, binary, workspace string, extra ...string) []byte {
	t.Helper()
	args := append([]string{"view", "-workspace", workspace,
		"-application", "order_entry",
		"-pattern", "order_entry.order_record",
		"-command", "sales.create_order",
		"-input", "order=ORD-1",
		"-input", "sales.order.order_id=ORD-1",
		"-format", "json"}, extra...)
	command := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		if _, isExit := err.(*exec.ExitError); !isExit {
			t.Fatalf("%v (%s)", err, stderr.String())
		}
	}
	return stdout.Bytes()
}

// hooksSection reads result.hooks from the document, or nil when the run
// carried none.
func hooksSection(t *testing.T, doc map[string]any) map[string]any {
	t.Helper()
	result, ok := doc["result"].(map[string]any)
	if !ok {
		return nil
	}
	hooks, ok := result["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("the run carried no hooks section: %+v", doc)
	}
	return hooks
}

// phaseSet reads one of result.hooks.executed or result.hooks.skipped as a set.
func phaseSet(hooks map[string]any, field string) map[string]bool {
	set := map[string]bool{}
	raw, ok := hooks[field].([]any)
	if !ok {
		return set
	}
	for _, item := range raw {
		if name, isName := item.(string); isName {
			set[name] = true
		}
	}
	return set
}

// TestViewExecuteReferenceStoreReachesTheAfterHook checks, through the real
// process a caller runs, that -reference-store reaches the after-change hook.
//
// The package-level tests call runDeclaredHooksHalf directly. This one starts
// the built binary, because flag parsing, the executor swap and the two passes
// are only wired together in the command's own path; a unit test can pass while
// the flag never reaches any of it.
func TestViewExecuteReferenceStoreReachesTheAfterHook(t *testing.T) {
	binary := buildExecuteCLI(t)
	workspace := referenceStoreWorkspace(t)

	raw := runViewExecute(t, binary, workspace, "-execute", "-reference-store", "-allow-capability", "sales.create_order")
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("not one JSON document: %v (%s)", err, string(raw))
	}
	hooks := hooksSection(t, doc)
	executed := phaseSet(hooks, "executed")
	if !executed["before-create"] {
		t.Fatalf("before-create did not run: %+v", hooks)
	}
	if !executed["after-create"] {
		t.Fatalf("after-create did not run through -reference-store: %+v", hooks)
	}
}

// TestViewExecuteWithoutReferenceStoreSkipsTheAfterHook checks the default the
// flag is opt-in from: without it the run writes nothing, so there is no
// after-change point.
func TestViewExecuteWithoutReferenceStoreSkipsTheAfterHook(t *testing.T) {
	binary := buildExecuteCLI(t)
	workspace := referenceStoreWorkspace(t)

	raw := runViewExecute(t, binary, workspace, "-execute", "-allow-capability", "sales.create_order")
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("not one JSON document: %v (%s)", err, string(raw))
	}
	hooks := hooksSection(t, doc)
	if phaseSet(hooks, "executed")["after-create"] {
		t.Fatalf("after-create ran without an applied change: %+v", hooks)
	}
	if !phaseSet(hooks, "skipped")["after-create"] {
		t.Fatalf("after-create was not reported as skipped: %+v", hooks)
	}
}

// TestViewReferenceStoreWithoutExecuteIsAnArgumentError checks the guard the flag
// needs: it changes what an execution does, so it must not be silently ignored
// without one.
func TestViewReferenceStoreWithoutExecuteIsAnArgumentError(t *testing.T) {
	binary := buildExecuteCLI(t)
	workspace := referenceStoreWorkspace(t)

	command := exec.Command(binary, "view", "-workspace", workspace,
		"-application", "order_entry",
		"-pattern", "order_entry.order_record",
		"-reference-store", "-format", "json")
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		exit, isExit := err.(*exec.ExitError)
		if !isExit {
			t.Fatalf("%v (%s)", err, stderr.String())
		}
		if exit.ExitCode() != 2 {
			t.Fatalf("exit code=%d, want 2 (%s)", exit.ExitCode(), stdout.String())
		}
	}
	var doc struct {
		OK    bool `json:"ok"`
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("not one JSON document: %v (%s)", err, stdout.String())
	}
	if doc.OK {
		t.Fatalf("-reference-store without -execute reported ok: %s", stdout.String())
	}
	if doc.Error == nil || doc.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("error=%+v (%s)", doc.Error, stdout.String())
	}
}

func TestViewReferenceStoreRejectsEveryNonExecutingMode(t *testing.T) {
	for _, mode := range [][]string{nil, {"-package", "order_entry.orders"}, {"-pattern", "order_entry.order_record"}} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			args := append([]string{"-workspace", referenceWorkspacePath(), "-application", "order_entry", "-reference-store", "-format", "json"}, mode...)
			var out bytes.Buffer
			code, err := runViewCommand(args, &out)
			if err != nil || code != 2 || !bytes.Contains(out.Bytes(), []byte(`"INVALID_ARGUMENT"`)) {
				t.Fatalf("mode %v accepted or failed for another reason: code=%d err=%v %s", mode, code, err, &out)
			}
		})
	}
}
