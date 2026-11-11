package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// This is a bounded consumer test for #4191 / merged #3742. The fixture
// deliberately remains historical: the live X-TEA Source has subsequently
// gained Setting roles. It must not replace the pinned A/B expected input.
// Both independent processes use exactly one isolated Workspace per version.
// Equal transports are NOT independent proof of complete Business meaning.
func TestHistoricalABSourceRevisionsReachCLIAndMCPConsumers(t *testing.T) {
	sourceB, err := os.ReadFile(filepath.Join("..", "..", "cmd", "tei", "testdata", "xtea_standard_20261108_b.tei"))
	if err != nil {
		t.Fatal(err)
	}
	added := []byte("    attribute rule type std.identifier display \"規則\" role numbering-rule\n")
	if !bytes.HasPrefix(sourceB, []byte{0xef, 0xbb, 0xbf}) ||
		bytes.Count(sourceB, added) != 1 ||
		!bytes.HasSuffix(sourceB, []byte("}\n\r\n")) {
		t.Fatal("historical B fixture is no longer the known pinned form")
	}
	sourceA := bytes.TrimPrefix(sourceB, []byte{0xef, 0xbb, 0xbf})
	sourceA = bytes.Replace(sourceA, added, nil, 1)
	sourceA = bytes.TrimSuffix(sourceA, []byte("\r\n"))

	binDir := t.TempDir()
	cliBinary := filepath.Join(binDir, "tei")
	mcpBinary := filepath.Join(binDir, "tei-mcp")
	if runtime.GOOS == "windows" {
		cliBinary += ".exe"
		mcpBinary += ".exe"
	}
	for _, build := range []struct{ bin, pkg string }{
		{cliBinary, "../../cmd/tei"},
		{mcpBinary, "."},
	} {
		cmd := exec.Command("go", "build", "-o", build.bin, build.pkg)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v: %s", build.pkg, err, out)
		}
	}

	type witness struct {
		world, sourceIndex string
	}
	readVersion := func(version string, source []byte) witness {
		t.Helper()
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "semantic", "master"), 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := []byte("workspace: historical_ab\nformatVersion: 1\nsources:\n  - semantic\n")
		if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), manifest, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "semantic", "master", "standard.tei"), source, 0o644); err != nil {
			t.Fatal(err)
		}

		cliCmd := exec.Command(cliBinary, "validate", "-workspace", root, "-json")
		cliOut, err := cliCmd.Output()
		if err != nil {
			t.Fatalf("%s: actual CLI validate failed: %v", version, err)
		}
		var cliDoc map[string]any
		if err := json.Unmarshal(cliOut, &cliDoc); err != nil {
			t.Fatalf("%s: invalid CLI JSON: %v", version, err)
		}

		// The adapter is an actual stdio process, not an in-process call to
		// operations.ValidateWorkspace. Its single JSON-RPC frame must be
		// received and contain exactly one complete text document.
		mcpCmd := exec.Command(mcpBinary, "-workspace", root)
		mcpCmd.Stdin = bytes.NewReader([]byte("{\"jsonrpc\":\"2.0\",\"id\":41,\"method\":\"tools/call\",\"params\":{\"name\":\"tei_validate\",\"arguments\":{}}}\n"))
		var mcpErr bytes.Buffer
		mcpCmd.Stderr = &mcpErr
		mcpOut, err := mcpCmd.Output()
		if err != nil {
			t.Fatalf("%s: actual MCP process failed: %v stderr=%s", version, err, mcpErr.String())
		}
		var reply struct {
			JSONRPC string `json:"jsonrpc"`
			ID      int    `json:"id"`
			Result  struct {
				IsError bool `json:"isError"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"result"`
		}
		if err := json.Unmarshal(mcpOut, &reply); err != nil {
			t.Fatalf("%s: MCP reply was not one complete JSON-RPC frame: %v", version, err)
		}
		if reply.JSONRPC != "2.0" || reply.ID != 41 ||
			reply.Result.IsError || len(reply.Result.Content) != 1 ||
			reply.Result.Content[0].Type != "text" {
			t.Fatalf("%s: MCP reply incomplete/error: %+v", version, reply)
		}
		var mcpDoc map[string]any
		if err := json.Unmarshal([]byte(reply.Result.Content[0].Text), &mcpDoc); err != nil {
			t.Fatalf("%s: MCP text is not complete validation JSON: %v", version, err)
		}
		if !reflect.DeepEqual(cliDoc, mcpDoc) {
			t.Fatalf("%s: actual CLI and MCP consumer documents differ:
CLI=%v
MCP=%v", version, cliDoc, mcpDoc)
		}
		if cliDoc["schemaVersion"] != "tei.cli.validate/v0" || cliDoc["ok"] != true {
			t.Fatalf("%s: bounded valid Source was not validated: %v", version, cliDoc)
		}
		world, worldOK := cliDoc["worldFingerprint"].(string)
		index, indexOK := cliDoc["sourceIndexFingerprint"].(string)
		if !worldOK || !indexOK || world == "" || index == "" {
			t.Fatalf("%s: validation reply lacks revision witness: %v", version, cliDoc)
		}
		return witness{world: world, sourceIndex: index}
	}

	a := readVersion("A", sourceA)
	b := readVersion("B", sourceB)
	if a.world == b.world || a.sourceIndex == b.sourceIndex {
		t.Fatalf("Source A/B drift was lost in actual consumer replies: A=%+v B=%+v", a, b)
	}
	// This only proves bounded A/B validation delivery to two local processes.
	// It does not prove a common atomic read attempt, complete Required Rules,
	// a real H3 client, current Disclosure/Grant, Provider values, or Effect.
}
