package targetprofile

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	generatorselection "tei/tools/reference-generator-selection"
)

func networkProfile(t *testing.T, generator string) (Profile, string) {
	t.Helper()
	profile, err := Load(filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", "server.json"))
	if err != nil {
		t.Fatal(err)
	}
	profile.Network, profile.GeneratorID = NetworkNone, generator
	raw, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "profile.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return profile, path
}

func TestNetworkUnavailableRunsNoCandidate(t *testing.T) {
	t.Setenv("PATH", "")
	_, profilePath := networkProfile(t, "must-not-execute")
	receipt, err := Run(context.Background(), RunOptions{ProfilePath: profilePath, EnforceNetwork: true})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || receipt.Network == nil || receipt.Network.Available || receipt.Network.Reason == "" || len(receipt.Selection.Candidates) != 0 || receipt.ProcessModel != nil || receipt.Enforcement["network"] != "unavailable" {
		t.Fatalf("unavailable isolation ran a candidate: %+v", receipt)
	}
}

func TestNetworkEnforcementRejectsUnsupportedPolicies(t *testing.T) {
	for _, policy := range []string{NetworkLoopback, NetworkAllowed} {
		evidence := prepareNetwork(context.Background(), Profile{Network: policy})
		if evidence.Available || !strings.Contains(evidence.Reason, "only network:none") {
			t.Fatalf("unsupported policy accepted: %+v", evidence)
		}
	}
}

func TestNetworkNamespaceBlocksHostListenerAndPreservesFixtures(t *testing.T) {
	profile, profilePath := networkProfile(t, "network-probe")
	evidence := prepareNetwork(context.Background(), profile)
	if !evidence.Available {
		t.Skip("real network namespace unavailable: " + evidence.Reason)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	workspace, contract, fixtures, _ := selectionPaths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	candidateDir := filepath.Join(candidates, profile.GeneratorID)
	generated, err := generatorselection.Generate(contract, "long-running-template", candidateDir)
	if err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(candidateDir, "main.go")
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(string(raw), "import (", "import (\n\t\"net\"\n\t\"time\"", 1)
	source = strings.Replace(source, "func main() {", "func businessMain() {", 1)
	source += fmt.Sprintf(`
func main() {
 connection,err := net.DialTimeout("tcp4",%q,time.Second)
 if err == nil { connection.Close(); fmt.Println("{\"state\":\"error\",\"code\":\"NETWORK_LEAK\"}"); return }
 businessMain()
}
`, listener.Addr().String())
	if err := os.WriteFile(sourcePath, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	manifest := generated.Manifest
	manifest.ID, manifest.Origin, manifest.Generator, manifest.SourceDigest = profile.GeneratorID, "handwritten", nil, digestBytes([]byte(source))
	changed, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(candidateDir, "candidate.json"), changed, 0600); err != nil {
		t.Fatal(err)
	}
	opts := RunOptions{ProfilePath: profilePath, WorkspaceDir: workspace, ContractPath: contract, FixturePath: fixtures, CandidatesDir: candidates}
	// The same listener and same fixed business fixture give a positive control:
	// without isolation the candidate really connects, then reports NETWORK_LEAK.
	unrestricted, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	selected := selectedCandidate(unrestricted)
	if unrestricted.Passed || selected == nil || len(selected.Results) != 10 || selected.Results[0].Got.Code != "NETWORK_LEAK" {
		t.Fatalf("host listener positive control failed: %+v", unrestricted)
	}
	opts.EnforceNetwork = true
	isolated, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if !isolated.Passed || !isolated.Network.Available || isolated.Network.ParentNamespace == isolated.Network.ProbeNamespace || len(isolated.Selection.Execution.CommandPrefix) == 0 || isolated.ProcessModel.Processes != 1 || isolated.ProcessModel.Responses != 10 || isolated.Enforcement["network"] != "linux:unshare-user-net" {
		t.Fatalf("isolated fixture/probe failed: %+v", isolated)
	}
	// Check the process-per-case path under the same isolation.
	profile.ProcessModel = ProcessOneShot
	raw, err = json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(profilePath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	oneshot, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if !oneshot.Passed || oneshot.ProcessModel.Processes != 10 {
		t.Fatalf("isolated oneshot failed: %+v", oneshot)
	}
}

func TestNetworkProbeRejectsWrapperWithoutIsolation(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux probe required")
	}
	// A fake wrapper is a negative fixture only: returning the host namespace
	// cannot be promoted into successful isolation evidence.
	dir := t.TempDir()
	for name, source := range map[string]string{"unshare": "#!/bin/sh\nshift 4\nexec \"$@\"\n", "readlink": "#!/bin/sh\n/bin/readlink /proc/self/ns/net\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	evidence := prepareNetwork(context.Background(), Profile{Network: NetworkNone})
	if evidence.Available || evidence.Reason != "probe did not enter a different network namespace" {
		t.Fatalf("no-op wrapper passed: %+v", evidence)
	}
}

func TestNetworkNamespaceScriptProfileComparison(t *testing.T) {
	profile, _ := networkProfile(t, "probe")
	evidence := prepareNetwork(context.Background(), profile)
	if !evidence.Available {
		t.Skip("real network namespace unavailable: " + evidence.Reason)
	}
	workspace, contract, fixtures, _ := selectionPaths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	var profiles []string
	for _, language := range []string{generatorselection.LanguagePython, generatorselection.LanguageJavaScript} {
		if _, runtime := generatorselection.ResolveScriptRuntime(language); !runtime.Available {
			t.Skip(language + " runtime unavailable")
		}
		strategy := language + "-long-running-template"
		if _, err := generatorselection.Generate(contract, strategy, filepath.Join(candidates, strategy)); err != nil {
			t.Fatal(err)
		}
		p, path := networkProfile(t, strategy)
		p.ID = strategy
		raw, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		profiles = append(profiles, path)
	}
	receipt, err := Compare(context.Background(), CompareOptions{WorkspaceDir: workspace, ContractPath: contract, FixturePath: fixtures, CandidatesDir: candidates, ProfilePaths: profiles, EnforceNetwork: true})
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || !receipt.SemanticsEqual || len(receipt.Cases) != 10 {
		t.Fatalf("isolated script comparison failed: %+v", receipt)
	}
	for _, profile := range receipt.Profiles {
		if profile.Enforcement["network"] != "linux:unshare-user-net" {
			t.Fatalf("comparison lost isolation: %+v", profile)
		}
	}
}
