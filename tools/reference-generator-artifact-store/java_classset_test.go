package generatorartifactstore

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	generatorselection "tei/tools/reference-generator-selection"
)

// These tests are the #2512 J2/J3 evidence: the store keeps the whole Java
// class set, and a clean class directory is reconstructed from stored bytes
// only and launched with an ordinary JVM.

type javaFixtureFile struct {
	Cases []struct {
		ID    string         `json:"id"`
		Input map[string]any `json:"input"`
		Want  map[string]any `json:"want"`
	} `json:"cases"`
}

func javaClassSetFixture(t *testing.T) (string, Receipt, javaFixtureFile) {
	t.Helper()
	if _, err := exec.LookPath("javac"); err != nil {
		t.Skip("javac is not available")
	}
	if _, err := exec.LookPath("java"); err != nil {
		t.Skip("java is not available")
	}
	workspace, contract, fixturesPath, _ := selectionPaths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	if _, err := generatorselection.GenerateVersion(contract, "java-template", "1",
		filepath.Join(candidates, "java-template")); err != nil {
		t.Fatal(err)
	}
	selection, err := generatorselection.Select(context.Background(), workspace, contract, fixturesPath, candidates)
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
	storeDir := filepath.Join(t.TempDir(), "store")
	put, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: selection.Candidates[0].ID,
		CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !put.Passed || put.ObjectPath == "" {
		t.Fatalf("java class set was not stored: %+v", put)
	}
	rawFixtures, err := os.ReadFile(fixturesPath)
	if err != nil {
		t.Fatal(err)
	}
	var fixtures javaFixtureFile
	if err := json.Unmarshal(rawFixtures, &fixtures); err != nil {
		t.Fatal(err)
	}
	return storeDir, put, fixtures
}

func storedObject(t *testing.T, put Receipt) Object {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(put.ObjectPath, "object.json"))
	if err != nil {
		t.Fatal(err)
	}
	var object Object
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	return object
}

func runJavaMain(t *testing.T, classpath, stdin string, extraArgs ...string) (string, string, error) {
	t.Helper()
	args := append(extraArgs, "-cp", classpath, "Main")
	cmd := exec.Command("java", args...)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func TestJavaClassSetIsStoredWithEveryMember(t *testing.T) {
	_, put, _ := javaClassSetFixture(t)
	object := storedObject(t, put)
	if object.ClassSetDigest == "" {
		t.Fatal("object has no class set digest")
	}
	if len(object.Classes) < 2 {
		t.Fatalf("class set has %d members, want the entry class and its inner class: %+v", len(object.Classes), object.Classes)
	}
	main, inner := false, false
	for _, member := range object.Classes {
		switch member.Path {
		case "Main.class":
			main = member.Role == classRoleMain && strings.HasPrefix(member.Digest, "sha256:")
		case "Main$Input.class":
			inner = member.Role == classRoleInner
		}
		if member.Size <= 0 {
			t.Fatalf("class member %q has size %d", member.Path, member.Size)
		}
	}
	if !main || !inner {
		t.Fatalf("class set is missing entry/inner roles: %+v", object.Classes)
	}
	if put.ClassSetDigest != object.ClassSetDigest || put.ClassCount != len(object.Classes) {
		t.Fatalf("put receipt class set=%q count=%d, object=%q count=%d",
			put.ClassSetDigest, put.ClassCount, object.ClassSetDigest, len(object.Classes))
	}
}

// J3: reconstruct a clean class directory from store bytes only and run all
// published fixtures with an ordinary JVM.
func TestJavaClassSetRestoreRunsFixturesFromStoreBytesOnly(t *testing.T) {
	storeDir, put, fixtures := javaClassSetFixture(t)
	isolated := filepath.Join(t.TempDir(), "isolated")
	restored, err := RestoreClassSet(storeDir, put.SourceDigest, isolated)
	if err != nil {
		t.Fatal(err)
	}
	if !restored.Passed || restored.ClassSetDigest != put.ClassSetDigest || restored.ClassCount != put.ClassCount {
		t.Fatalf("class set was not restored: %+v", restored)
	}
	if _, err := os.Stat(filepath.Join(isolated, "Main$Input.class")); err != nil {
		t.Fatalf("inner class was not restored: %v", err)
	}
	if len(fixtures.Cases) == 0 {
		t.Fatal("no fixtures were loaded")
	}
	for _, testCase := range fixtures.Cases {
		input, err := json.Marshal(testCase.Input)
		if err != nil {
			t.Fatal(err)
		}
		stdout, stderr, err := runJavaMain(t, isolated, string(input))
		if err != nil {
			t.Fatalf("case %s failed to run: %v stderr=%s", testCase.ID, err, stderr)
		}
		var got map[string]any
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("case %s produced non-JSON %q: %v", testCase.ID, stdout, err)
		}
		if !reflect.DeepEqual(got, testCase.Want) {
			t.Fatalf("case %s got=%v want=%v", testCase.ID, got, testCase.Want)
		}
	}
}

// The JVM must resolve the inner class from the restored directory, not from
// the build directory or an ambient classpath.
func TestJavaClassSetRestoreResolvesInnerClassFromRestoredDirectory(t *testing.T) {
	storeDir, put, _ := javaClassSetFixture(t)
	isolated := filepath.Join(t.TempDir(), "isolated")
	if restored, err := RestoreClassSet(storeDir, put.SourceDigest, isolated); err != nil || !restored.Passed {
		t.Fatalf("restore failed: %+v err=%v", restored, err)
	}
	// -verbose:class writes to stdout on current JDKs and to stderr on older
	// ones; inspect both rather than assume one stream.
	stdout, stderr, err := runJavaMain(t, isolated, `{"quantity":"2","unitPrice":"300","currency":"JPY","unit":"piece","perUnit":"piece"}`, "-verbose:class")
	if err != nil {
		t.Fatalf("verbose run failed: %v", err)
	}
	normalized := strings.ReplaceAll(stdout+stderr, "\\", "/")
	if !strings.Contains(normalized, "Main$Input") {
		t.Fatalf("class loading log does not mention the inner class: %s", stderr)
	}
	// The JVM prints the path it opened, which is not always the spelling this
	// test handed it: on the Windows CI runner the temporary directory is a
	// short name (C:\Users\RUNNER~1\...) that resolves to another spelling
	// (#3783). Accept either spelling of the same directory.
	spellings := []string{isolated}
	if resolved, err := filepath.EvalSymlinks(isolated); err == nil && resolved != isolated {
		spellings = append(spellings, resolved)
	}
	loadedFromRestored := false
	for _, spelling := range spellings {
		if strings.Contains(normalized, strings.ReplaceAll(spelling, "\\", "/")) {
			loadedFromRestored = true
			break
		}
	}
	if !loadedFromRestored {
		t.Fatalf("inner class was not loaded from the restored directory %q: %s", isolated, stderr)
	}
}

// The negative that #2077 JR-01 reported: with only the entry class, the JVM
// cannot start. Removing the restored inner class reproduces it against the
// store-produced directory.
func TestJavaStoreOnlyEntryClassCannotStart(t *testing.T) {
	storeDir, put, _ := javaClassSetFixture(t)
	isolated := filepath.Join(t.TempDir(), "isolated")
	if restored, err := RestoreClassSet(storeDir, put.SourceDigest, isolated); err != nil || !restored.Passed {
		t.Fatalf("restore failed: %+v err=%v", restored, err)
	}
	if err := os.Remove(filepath.Join(isolated, "Main$Input.class")); err != nil {
		t.Fatal(err)
	}
	_, stderr, err := runJavaMain(t, isolated, `{"quantity":"2","unitPrice":"300","currency":"JPY","unit":"piece","perUnit":"piece"}`)
	if err == nil {
		t.Fatal("missing inner class still started, so the stored closure is not proven")
	}
	if !strings.Contains(stderr, "NoClassDefFoundError") {
		t.Fatalf("unexpected failure for missing inner class: %s", stderr)
	}
}

func TestJavaClassSetRestoreRejectsTamperedMember(t *testing.T) {
	storeDir, put, _ := javaClassSetFixture(t)
	memberPath := filepath.Join(put.ObjectPath, "classes", "Main$Input.class")
	raw, err := os.ReadFile(memberPath)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)-1] ^= 0xff
	if err := os.WriteFile(memberPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreClassSet(storeDir, put.SourceDigest, filepath.Join(t.TempDir(), "isolated"))
	if err != nil {
		t.Fatal(err)
	}
	if restored.Passed || !strings.Contains(restored.Reason, "digest does not match") {
		t.Fatalf("tampered class member was accepted: %+v", restored)
	}
}

func TestJavaClassSetRestoreRefusesObjectWithoutClassSet(t *testing.T) {
	storeDir, put, _ := javaClassSetFixture(t)
	raw, err := os.ReadFile(filepath.Join(put.ObjectPath, "object.json"))
	if err != nil {
		t.Fatal(err)
	}
	var object Object
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	object.ClassSetDigest = ""
	object.Classes = nil
	updated, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(put.ObjectPath, "object.json"), append(updated, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreClassSet(storeDir, put.SourceDigest, filepath.Join(t.TempDir(), "isolated"))
	if err != nil {
		t.Fatal(err)
	}
	if restored.Passed || restored.Reason != "object has no stored class set" {
		t.Fatalf("object without a class set was restored: %+v", restored)
	}
}
