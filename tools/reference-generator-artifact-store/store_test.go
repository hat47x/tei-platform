package generatorartifactstore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"reflect"
	"strings"
	"sync"
	"testing"

	generatorselection "tei/tools/reference-generator-selection"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func selectionPaths(t *testing.T) (workspace, contract, fixtures, candidates string) {
	t.Helper()
	base := filepath.Join(repoRoot(t), "tools", "reference-generator-selection", "testdata")
	return filepath.Join(repoRoot(t), "reference", "workspace"),
		filepath.Join(base, "line-amount", "contract.json"),
		filepath.Join(base, "line-amount", "fixtures.json"),
		filepath.Join(base, "candidates")
}

var (
	selectionReceiptOnce sync.Once
	selectionReceiptPath string
	selectionReceiptRaw  []byte
	selectionReceiptErr  error
)

func writeSelectionReceipt(t *testing.T) (string, generatorselection.Receipt) {
	t.Helper()
	selectionReceiptOnce.Do(func() {
		workspace, contract, fixtures, candidates := selectionPaths(t)
		selection, err := generatorselection.Select(context.Background(), workspace, contract, fixtures, candidates)
		if err != nil {
			selectionReceiptErr = err
			return
		}
		raw, err := json.MarshalIndent(selection, "", "  ")
		if err != nil {
			selectionReceiptErr = err
			return
		}
		selectionReceiptRaw = append(raw, '\n')
		dir, err := os.MkdirTemp("", "tei-artifact-selection-*")
		if err != nil {
			selectionReceiptErr = err
			return
		}
		selectionReceiptPath = filepath.Join(dir, "selection.json")
		selectionReceiptErr = os.WriteFile(selectionReceiptPath, selectionReceiptRaw, 0o600)
	})
	if selectionReceiptErr != nil {
		t.Fatal(selectionReceiptErr)
	}
	var selection generatorselection.Receipt
	if err := json.Unmarshal(selectionReceiptRaw, &selection); err != nil {
		t.Fatal(err)
	}
	return selectionReceiptPath, selection
}

func TestPutAndGetArtifact(t *testing.T) {
	receiptPath, selection := writeSelectionReceipt(t)
	_, _, _, candidates := selectionPaths(t)
	storeDir := t.TempDir()
	put, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: "template", CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !put.Passed || put.Reused || put.Language != generatorselection.LanguageGo || put.SourceDigest == "" || put.BinaryDigest == "" || put.BuildEvidenceDigest == "" || put.ObjectPath == "" {
		t.Fatalf("artifact put failed: %+v", put)
	}
	if _, err := os.Stat(filepath.Join(put.ObjectPath, "source")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(put.ObjectPath, "artifact")); err != nil {
		t.Fatal(err)
	}
	get, err := Get(storeDir, put.SourceDigest)
	if err != nil {
		t.Fatal(err)
	}
	if !get.Passed || get.Language != generatorselection.LanguageGo || get.SourceDigest != put.SourceDigest || get.BinaryDigest != put.BinaryDigest || get.BuildEvidenceDigest != put.BuildEvidenceDigest {
		t.Fatalf("artifact get failed: %+v", get)
	}
	again, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: "template", CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !again.Passed || !again.Reused {
		t.Fatalf("existing artifact was not reused: %+v", again)
	}
	var found bool
	for _, candidate := range selection.Candidates {
		if candidate.ID == "template" {
			found = candidate.SourceDigest == put.SourceDigest && candidate.BinaryDigest == put.BinaryDigest
		}
	}
	if !found {
		t.Fatal("stored digests do not match the selection receipt")
	}
}

func TestPutRejectsReceiptDigestMismatch(t *testing.T) {
	_, selection := writeSelectionReceipt(t)
	_, _, _, candidates := selectionPaths(t)
	for i := range selection.Candidates {
		if selection.Candidates[i].ID == "template" {
			selection.Candidates[i].BinaryDigest = "sha256:" + strings.Repeat("0", 64)
		}
	}
	raw, err := json.MarshalIndent(selection, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(t.TempDir(), "tampered.json")
	if err := os.WriteFile(receiptPath, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: "template", CandidatesDir: candidates, StoreDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || !strings.Contains(receipt.Reason, "binary digest") {
		t.Fatalf("tampered receipt was accepted: %+v", receipt)
	}
}

func TestGetRejectsTamperedArtifact(t *testing.T) {
	receiptPath, _ := writeSelectionReceipt(t)
	_, _, _, candidates := selectionPaths(t)
	storeDir := t.TempDir()
	put, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: "template", CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !put.Passed || put.ObjectPath == "" {
		t.Fatalf("artifact put failed before tamper test: %+v", put)
	}
	if err := os.WriteFile(filepath.Join(put.ObjectPath, "artifact"), []byte("tampered"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Get(storeDir, put.SourceDigest)
	if err != nil {
		t.Fatal(err)
	}
	if got.Passed || !strings.Contains(got.Reason, "artifact digest") {
		t.Fatalf("tampered artifact was accepted: %+v", got)
	}
}

func TestPutAndGetWasiArtifact(t *testing.T) {
	workspace, contract, fixtures, _ := selectionPaths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	if _, err := generatorselection.GenerateVersion(contract, "wasi-template", "1",
		filepath.Join(candidates, "wasi-template")); err != nil {
		t.Fatal(err)
	}
	selection, err := generatorselection.Select(context.Background(), workspace, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !selection.Passed || len(selection.Candidates) != 1 || !selection.Candidates[0].Passed {
		t.Fatalf("wasi candidate did not pass selection: %+v", selection.Candidates)
	}
	raw, err := json.MarshalIndent(selection, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(t.TempDir(), "selection.json")
	if err := os.WriteFile(receiptPath, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	storeDir := t.TempDir()
	put, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: selection.Candidates[0].ID,
		CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !put.Passed || put.ObjectPath == "" {
		t.Fatalf("wasi artifact was not stored: %+v", put)
	}
	rawObject, err := os.ReadFile(filepath.Join(put.ObjectPath, "object.json"))
	if err != nil {
		t.Fatal(err)
	}
	var object Object
	if err := json.Unmarshal(rawObject, &object); err != nil {
		t.Fatal(err)
	}
	if object.Language != generatorselection.LanguageWasi {
		t.Fatalf("stored language=%s want wasi", object.Language)
	}
	got, err := Get(storeDir, put.SourceDigest)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Passed || got.BinaryDigest != put.BinaryDigest {
		t.Fatalf("wasi artifact was not read back: %+v", got)
	}
}

func TestPutAndGetJavaArtifact(t *testing.T) {
	if _, err := exec.LookPath("javac"); err != nil {
		t.Skip("javac is not available")
	}
	if _, err := exec.LookPath("java"); err != nil {
		t.Skip("java is not available")
	}
	// Keep implicit classpaths and injected Java options out of this attempt.
	for _, key := range []string{"CLASSPATH", "JAVA_TOOL_OPTIONS", "JDK_JAVA_OPTIONS", "JDK_JAVAC_OPTIONS"} {
		t.Setenv(key, "")
	}
	workspace, contract, fixtures, _ := selectionPaths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	if _, err := generatorselection.GenerateVersion(contract, "java-template", "1",
		filepath.Join(candidates, "java-template")); err != nil {
		t.Fatal(err)
	}
	selection, err := generatorselection.Select(context.Background(), workspace, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !selection.Passed || len(selection.Candidates) != 1 || !selection.Candidates[0].Passed {
		t.Fatalf("java candidate did not pass selection: %+v", selection.Candidates)
	}
	raw, err := json.MarshalIndent(selection, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(t.TempDir(), "selection.json")
	if err := os.WriteFile(receiptPath, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	storeDir := t.TempDir()
	put, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: selection.Candidates[0].ID,
		CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !put.Passed || put.ObjectPath == "" {
		t.Fatalf("java artifact was not stored: %+v", put)
	}
	rawObject, err := os.ReadFile(filepath.Join(put.ObjectPath, "object.json"))
	if err != nil {
		t.Fatal(err)
	}
	var object Object
	if err := json.Unmarshal(rawObject, &object); err != nil {
		t.Fatal(err)
	}
	if object.Language != generatorselection.LanguageJava {
		t.Fatalf("stored language=%s want java", object.Language)
	}
	got, err := Get(storeDir, put.SourceDigest)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Passed || got.BinaryDigest != put.BinaryDigest {
		t.Fatalf("java artifact was not read back: %+v", got)
	}

	// JR-01 characterization: Put/Get verify the saved Main.class, not the
	// complete Java class closure. Do not treat the expected launch failure as
	// proof of restore-to-run acceptance or as a permanent product requirement.
	artifact, err := os.ReadFile(filepath.Join(got.ObjectPath, "artifact"))
	if err != nil {
		t.Fatal(err)
	}
	controlClasses := t.TempDir()
	compile := exec.Command("javac", "-d", controlClasses, "Main.java")
	compile.Dir = filepath.Join(candidates, "java-template")
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("control javac failed: %v: %s", err, output)
	}
	mainClass, err := os.ReadFile(filepath.Join(controlClasses, "Main.class"))
	if err != nil || !bytes.Equal(mainClass, artifact) {
		t.Fatalf("Get artifact differs from independently compiled Main.class: %v", err)
	}
	if _, err := os.Stat(filepath.Join(controlClasses, "Main$Input.class")); err != nil {
		t.Fatalf("full Java build lacks required nested class: %v", err)
	}

	completeCopy := t.TempDir()
	classCount := 0
	err = filepath.WalkDir(controlClasses, func(classPath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(controlClasses, classPath)
		if err != nil {
			return err
		}
		if !strings.HasSuffix(rel, ".class") {
			return fmt.Errorf("unexpected output in Java classes directory: %s", rel)
		}
		content, err := os.ReadFile(classPath)
		if err != nil {
			return err
		}
		classCount++
		t.Logf("JR-01 control class %s bytes=%d sha256=%x", filepath.ToSlash(rel), len(content), sha256.Sum256(content))
		target := filepath.Join(completeCopy, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o644)
	})
	if err != nil || classCount < 2 {
		t.Fatalf("full class closure could not be enumerated: classCount=%d err=%v", classCount, err)
	}

	fixtureRaw, err := os.ReadFile(fixtures)
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct {
			ID    string          `json:"id"`
			Input json.RawMessage `json:"input"`
			Want  json.RawMessage `json:"want"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(fixtureRaw, &fixture); err != nil || len(fixture.Cases) != 10 {
		t.Fatalf("invalid reference fixture: cases=%d err=%v", len(fixture.Cases), err)
	}
	runFixtures := func(classDir string) {
		t.Helper()
		for _, tc := range fixture.Cases {
			cmd := exec.Command("java", "-cp", classDir, "Main")
			cmd.Stdin = bytes.NewReader(tc.Input)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("java -cp %s Main, fixture %s: %v: %s", classDir, tc.ID, err, output)
			}
			var gotJSON, wantJSON any
			if json.Unmarshal(output, &gotJSON) != nil || json.Unmarshal(tc.Want, &wantJSON) != nil ||
				!reflect.DeepEqual(gotJSON, wantJSON) {
				t.Fatalf("fixture %s: got %s, want %s", tc.ID, output, tc.Want)
			}
		}
	}
	// A: complete direct build is runnable; C: the same full set copied into
	// another clean classpath is runnable, but is NOT a Store restoration.
	runFixtures(controlClasses)
	runFixtures(completeCopy)

	// B: only the artifact actually returned by Put/Get is restored.
	storeOnly := t.TempDir()
	if err := os.WriteFile(filepath.Join(storeOnly, "Main.class"), artifact, 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("java", "-cp", storeOnly, "Main")
	cmd.Stdin = bytes.NewReader(fixture.Cases[0].Input)
	output, launchErr := cmd.CombinedOutput()
	if launchErr == nil || !bytes.Contains(output, []byte("NoClassDefFoundError: Main$Input")) {
		t.Fatalf("current single-class Store unexpectedly runnable or failed differently: exit=%v output=%s", launchErr, output)
	}
	t.Logf("JR-01: Store.Put/Get passed (binary=%s), full classes=%d (10/10 twice), "+
		"Store-only launch fails because Main$Input.class is absent: %v", got.BinaryDigest, classCount, launchErr)
}

func TestPutAndGetSourceOnlyArtifacts(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is not available")
	}
	for _, test := range []struct {
		strategy string
		language string
	}{
		{strategy: "python-template", language: generatorselection.LanguagePython},
		{strategy: "javascript-template", language: generatorselection.LanguageJavaScript},
	} {
		t.Run(test.language, func(t *testing.T) {
			workspace, contract, fixtures, _ := selectionPaths(t)
			candidates := filepath.Join(t.TempDir(), "candidates")
			if _, err := generatorselection.GenerateVersion(contract, test.strategy, "1",
				filepath.Join(candidates, test.strategy)); err != nil {
				t.Fatal(err)
			}
			selection, err := generatorselection.Select(context.Background(), workspace, contract, fixtures, candidates)
			if err != nil {
				t.Fatal(err)
			}
			if !selection.Passed || len(selection.Candidates) != 1 || !selection.Candidates[0].Passed {
				t.Fatalf("%s candidate did not pass selection: %+v", test.language, selection.Candidates)
			}
			if selection.Candidates[0].BinaryDigest != "" {
				t.Fatalf("source-only candidate has a binary digest: %+v", selection.Candidates[0])
			}
			raw, err := json.MarshalIndent(selection, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			receiptPath := filepath.Join(t.TempDir(), "selection.json")
			if err := os.WriteFile(receiptPath, append(raw, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
			storeDir := t.TempDir()
			put, err := Put(context.Background(), StoreOptions{
				ReceiptPath: receiptPath, CandidateID: selection.Candidates[0].ID,
				CandidatesDir: candidates, StoreDir: storeDir,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !put.Passed || put.Language != test.language || put.ObjectPath == "" || put.BinaryDigest != "" || put.BuildEvidenceDigest != "" {
				t.Fatalf("%s artifact was not stored: %+v", test.language, put)
			}
			artifact, err := os.ReadFile(filepath.Join(put.ObjectPath, "artifact"))
			if err != nil {
				t.Fatal(err)
			}
			if len(artifact) != 0 {
				t.Fatalf("source-only artifact is not empty: %d bytes", len(artifact))
			}
			got, err := Get(storeDir, put.SourceDigest)
			if err != nil {
				t.Fatal(err)
			}
			if !got.Passed || got.Language != test.language || got.BinaryDigest != "" {
				t.Fatalf("%s artifact was not read back: %+v", test.language, got)
			}
			list, err := List(storeDir)
			if err != nil {
				t.Fatal(err)
			}
			if !list.Passed || len(list.Objects) != 1 || !list.Objects[0].Verified ||
				list.Objects[0].Language != test.language {
				t.Fatalf("%s source-only object failed list verification: %+v", test.language, list)
			}
			prune, err := Prune(PruneOptions{StoreDir: storeDir, Keep: []string{put.SourceDigest}})
			if err != nil {
				t.Fatal(err)
			}
			if !prune.Passed || len(prune.Kept) != 1 {
				t.Fatalf("%s source-only object failed prune verification: %+v", test.language, prune)
			}
		})
	}
}

func TestGetRejectsTamperedSelectionReceipt(t *testing.T) {
	receiptPath, _ := writeSelectionReceipt(t)
	_, _, _, candidates := selectionPaths(t)
	storeDir := t.TempDir()
	put, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: "template", CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !put.Passed {
		t.Fatalf("put failed: %+v", put)
	}
	if err := os.WriteFile(filepath.Join(put.ObjectPath, "receipt.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Get(storeDir, put.SourceDigest)
	if err != nil {
		t.Fatal(err)
	}
	if got.Passed || !strings.Contains(got.Reason, "selection receipt") {
		t.Fatalf("tampered selection receipt was accepted: %+v", got)
	}
}

func TestGetRejectsTamperedBuildEvidenceDigest(t *testing.T) {
	receiptPath, _ := writeSelectionReceipt(t)
	_, _, _, candidates := selectionPaths(t)
	storeDir := t.TempDir()
	put, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: "template", CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	rawObject, err := os.ReadFile(filepath.Join(put.ObjectPath, "object.json"))
	if err != nil {
		t.Fatal(err)
	}
	var object Object
	if err := json.Unmarshal(rawObject, &object); err != nil {
		t.Fatal(err)
	}
	object.BuildEvidenceDigest = "sha256:" + strings.Repeat("0", 64)
	changed, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(put.ObjectPath, "object.json"), append(changed, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Get(storeDir, put.SourceDigest)
	if err != nil {
		t.Fatal(err)
	}
	if got.Passed || !strings.Contains(got.Reason, "build evidence") {
		t.Fatalf("tampered build evidence digest was accepted: %+v", got)
	}
}

func TestListRejectsTamperedBuildEvidenceDigest(t *testing.T) {
	receiptPath, _ := writeSelectionReceipt(t)
	_, _, _, candidates := selectionPaths(t)
	storeDir := t.TempDir()
	put, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: "template", CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	rawObject, err := os.ReadFile(filepath.Join(put.ObjectPath, "object.json"))
	if err != nil {
		t.Fatal(err)
	}
	var object Object
	if err := json.Unmarshal(rawObject, &object); err != nil {
		t.Fatal(err)
	}
	object.BuildEvidenceDigest = "sha256:" + strings.Repeat("0", 64)
	changed, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(put.ObjectPath, "object.json"), append(changed, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	list, err := List(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	if list.Passed || len(list.Objects) != 1 || list.Objects[0].Verified {
		t.Fatalf("tampered build evidence digest passed list: %+v", list)
	}
}

func TestPutRejectsReuseWithDifferentProvenance(t *testing.T) {
	receiptPath, _ := writeSelectionReceipt(t)
	_, _, _, candidates := selectionPaths(t)
	storeDir := t.TempDir()
	first, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: "template", CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	rawObject, err := os.ReadFile(filepath.Join(first.ObjectPath, "object.json"))
	if err != nil {
		t.Fatal(err)
	}
	var object Object
	if err := json.Unmarshal(rawObject, &object); err != nil {
		t.Fatal(err)
	}
	object.ReceiptDigest = "sha256:" + strings.Repeat("0", 64)
	changed, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(first.ObjectPath, "object.json"), append(changed, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: "template", CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Passed || !strings.Contains(second.Reason, "provenance") {
		t.Fatalf("different provenance was reused: %+v", second)
	}
}

func TestPutRestoresMissingSelectionReceiptOnReuse(t *testing.T) {
	receiptPath, _ := writeSelectionReceipt(t)
	_, _, _, candidates := selectionPaths(t)
	storeDir := t.TempDir()
	first, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: "template", CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(first.ObjectPath, "receipt.json")); err != nil {
		t.Fatal(err)
	}
	second, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: "template", CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !second.Passed || !second.Reused {
		t.Fatalf("existing object was not reused: %+v", second)
	}
	got, err := Get(storeDir, first.SourceDigest)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Passed {
		t.Fatalf("restored receipt did not verify: %+v", got)
	}
}

func TestPutRejectsUnknownCandidate(t *testing.T) {
	receiptPath, _ := writeSelectionReceipt(t)
	_, _, _, candidates := selectionPaths(t)
	if _, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: "missing", CandidatesDir: candidates, StoreDir: t.TempDir(),
	}); err == nil {
		t.Fatal("unknown candidate was accepted")
	}
}
