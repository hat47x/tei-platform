package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestPluginPublicManualCandidateAgainstRealCLI connects the PDC-0 candidate
// manual to a built, real tei binary and the selected Plugin inventory.
// This deliberately does NOT read a Plugin Document from the Product: the
// accepted Plugin -> Document SourceRef / current Disclosure binding is absent.
func TestPluginPublicManualCandidateAgainstRealCLI(t *testing.T) {
	root := filepath.Join("..", "..")
	candidatePath := filepath.Join(root, "testdata", "plugin-doc-reference", "independent-expected.candidate.json")
	rawCandidate, err := os.ReadFile(candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	var candidate struct {
		Adoption string `json:"adoption"`
		Plugins  struct {
			A struct {
				ID string `json:"id"`
			} `json:"A"`
		} `json:"plugins"`
		Documents struct {
			Public struct {
				ID   string `json:"id"`
				Path string `json:"path"`
			} `json:"public"`
			Private struct {
				ID   string `json:"id"`
				Path string `json:"path"`
			} `json:"private"`
		} `json:"documents"`
	}
	if err := json.Unmarshal(rawCandidate, &candidate); err != nil {
		t.Fatal(err)
	}
	if candidate.Adoption != "proposal-only" || candidate.Plugins.A.ID == "" ||
		candidate.Documents.Public.ID == "" || candidate.Documents.Private.ID == "" {
		t.Fatal("candidate must remain an unaccepted, explicit A/Document reference")
	}
	public, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(candidate.Documents.Public.Path)))
	if err != nil {
		t.Fatal(err)
	}
	private, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(candidate.Documents.Private.Path)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(public, []byte("`tei commands`")) ||
		!bytes.Contains(public, []byte("`plugin-inventory`")) ||
		bytes.Equal(public, private) {
		t.Fatal("candidate guide lost its real discovery entry or document separation")
	}

	binary := buildCLIForLifecycle(t)
	commandsRaw, code := runCLIProcess(t, binary, "commands")
	if code != 0 {
		t.Fatalf("tei commands exit=%d; output=%s", code, commandsRaw)
	}
	var catalog cliCommandCatalog
	if err := json.Unmarshal(commandsRaw, &catalog); err != nil {
		t.Fatalf("tei commands is not the advertised JSON catalog: %v", err)
	}
	foundInventory := false
	for _, cmd := range catalog.Commands {
		switch cmd.Name {
		case "plugin-inventory":
			foundInventory = cmd.Effect == "read-only" &&
				cmd.SchemaVersion == "tei.cli.plugin-inventory/v0"
		case "plugin docs", "plugin-docs", "plugin document":
			t.Fatalf("candidate guide's unimplemented Product Doc command unexpectedly advertised: %q", cmd.Name)
		}
	}
	if !foundInventory {
		t.Fatal("candidate guide references a Plugin inventory entry absent from the actual CLI")
	}

	workspace := filepath.Join(root, "reference", "workspace")
	// A fixture definition is not an installed/selected Product Plugin.
	missingRaw, code := runCLIProcess(t, binary,
		"plugin-inventory", "-workspace", workspace,
		"-environment", "development", "-plugin", candidate.Plugins.A.ID)
	if code != 2 {
		t.Fatalf("candidate Plugin unexpectedly resolved: exit=%d body=%s", code, missingRaw)
	}
	var missing struct {
		OK    bool `json:"ok"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(missingRaw, &missing); err != nil {
		t.Fatal(err)
	}
	if missing.OK || missing.Error.Code != resultCodeInvalidArgument {
		t.Fatalf("fixture Plugin must not be treated as selected Product Plugin: %s", missingRaw)
	}

	checkDetail := func(lock string, expectedStatus string, expectedPinned *bool) {
		t.Helper()
		args := []string{
			"plugin-inventory", "-workspace", workspace, "-environment", "development",
			"-plugin", "tei.database.postgres",
		}
		if lock != "" {
			args = append(args, "-plugin-lock", lock)
		}
		data, exit := runCLIProcess(t, binary, args...)
		if exit != 0 {
			t.Fatalf("actual inventory detail exit=%d body=%s", exit, data)
		}
		var result struct {
			OK     bool   `json:"ok"`
			Mode   string `json:"mode"`
			Result struct {
				DocumentReadPreflight struct {
					PluginID        string `json:"pluginId"`
					ResolvedVersion string `json:"resolvedVersion"`
					SelectionStatus string `json:"selectionStatus"`
					LockPinned      *bool  `json:"lockPinned"`
					SourceBinding   string `json:"sourceBinding"`
					Disclosure      string `json:"disclosure"`
					ReadEligibility string `json:"readEligibility"`
				} `json:"documentReadPreflight"`
			} `json:"result"`
		}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}
		pre := result.Result.DocumentReadPreflight
		if !result.OK || result.Mode != pluginInventoryModeDetail ||
			pre.PluginID != "tei.database.postgres" ||
			pre.ResolvedVersion != "1.0.0" ||
			pre.SelectionStatus != expectedStatus ||
			pre.SourceBinding != "unbound" ||
			pre.Disclosure != "not-checked" ||
			pre.ReadEligibility != "not-assessed" {
			t.Fatalf("Product preflight promoted selection to Document read: %s", data)
		}
		if (expectedPinned == nil) != (pre.LockPinned == nil) {
			t.Fatalf("nil LockPinned is distinct from false: %s", data)
		}
		if expectedPinned != nil && pre.LockPinned != nil && *pre.LockPinned != *expectedPinned {
			t.Fatalf("unexpected LockPinned value: %s", data)
		}
		// The Product inventory is a selection projection, not a source
		// reader; it must not reveal candidate Document IDs, paths or bytes.
		for _, forbidden := range []string{
			candidate.Documents.Public.ID, candidate.Documents.Private.ID,
			candidate.Documents.Public.Path, candidate.Documents.Private.Path,
		} {
			if bytes.Contains(data, []byte(forbidden)) {
				t.Fatalf("Product inventory leaked candidate document reference %q", forbidden)
			}
		}
		if bytes.Contains(data, private) {
			t.Fatal("Product inventory disclosed the private candidate Manual body")
		}
	}
	checkDetail("", "lock-unknown", nil)

	lockRaw, code := runCLIProcess(t, binary, "plugin-lock", "-workspace", workspace)
	if code != 0 {
		t.Fatalf("actual plugin-lock exit=%d body=%s", code, lockRaw)
	}
	lockFile := filepath.Join(t.TempDir(), "plugin-lock.json")
	if err := os.WriteFile(lockFile, lockJSONBytes(t, lockRaw), 0o600); err != nil {
		t.Fatal(err)
	}
	pinned := true
	checkDetail(lockFile, "exact-version-pinned", &pinned)
}
