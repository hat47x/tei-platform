package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/plugin"
	"tei/internal/syntax"
)

func TestVerifyDerivedReceiptComparesStoredBytes(t *testing.T) {
	root := copySourceTree(t)
	catalog := loadCatalog(t)
	intent := loadIntent(t)
	ctx := NewFixtureContext(root, catalog)
	original := string(mustMarshal(t, mustResolve(t, intent, root, catalog)))

	// 改行（CRLFと末尾の改行）の違いだけは受け入れる。
	for name, ok := range map[string]string{
		"same":          original,
		"trailing LF":   original + "\n",
		"CRLF":          strings.ReplaceAll(original, "\n", "\r\n"),
		"CRLF trailing": strings.ReplaceAll(original, "\n", "\r\n") + "\r\n",
	} {
		if err := VerifyDerivedReceipt(intent, ctx, []byte(ok)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}

	const planTarget = `"target": "host/linux/amd64",`
	if strings.Count(original, planTarget) != 1 {
		t.Fatal("fixture receipt layout changed")
	}
	// 復号した値だけを見ると元と同じになり得る編集を、保存bytesの比較で拒否する。
	edits := map[string]string{
		"duplicate key, original last":  strings.Replace(original, planTarget, `"target": "host/windows/amd64",`+"\n    "+planTarget, 1),
		"duplicate key, original first": strings.Replace(original, planTarget, planTarget+"\n    "+`"target": "host/windows/amd64",`, 1),
		"case variant key":              strings.Replace(original, `"planFingerprint"`, `"PLANFINGERPRINT"`, 1),
		"trailing JSON":                 original + "\n{}",
		"trailing token":                original + "\n0",
		"extra space":                   strings.Replace(original, `"authority": "derived-cache"`, `"authority":  "derived-cache"`, 1),
	}
	for name, edited := range edits {
		if edited == original {
			t.Fatalf("%s: edit did not change the receipt", name)
		}
		if err := VerifyDerivedReceipt(intent, ctx, []byte(edited)); err == nil || !strings.Contains(err.Error(), "diverges") {
			t.Fatalf("%s: edited derived receipt was accepted: %v", name, err)
		}
	}
}

func TestSourceDigestCoversSameLengthContentChange(t *testing.T) {
	root := copySourceTree(t)
	catalog := loadCatalog(t)
	intent := loadIntent(t)
	before := mustResolve(t, intent, root, catalog)

	path := filepath.Join(root, "impl", "main.go.txt")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	edited := bytes.Replace(raw, []byte("main() {}"), []byte("mian() {}"), 1)
	if len(edited) != len(raw) || bytes.Equal(edited, raw) {
		t.Fatal("edit must keep the length and change the content")
	}
	if err := os.WriteFile(path, edited, 0o644); err != nil {
		t.Fatal(err)
	}
	after := mustResolve(t, intent, root, catalog)
	if sourceByID(t, before.Plan, "implementation").Digest == sourceByID(t, after.Plan, "implementation").Digest {
		t.Fatal("same-length content change did not change the source digest")
	}
	if before.PlanFingerprint == after.PlanFingerprint {
		t.Fatal("same-length content change did not change the plan fingerprint")
	}
}

func TestDefinitionFingerprintFollowsParsedDefinition(t *testing.T) {
	root := copySourceTree(t)
	catalog := loadCatalog(t)
	intent := loadIntent(t)

	definitionPath := filepath.Join(root, "plugin.tei")
	raw, err := os.ReadFile(definitionPath)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := syntax.NewParser(string(raw)).ParsePlugin()
	if err != nil {
		t.Fatal(err)
	}
	before := mustResolve(t, intent, root, catalog)
	if want := plugin.BuildDefinitionFingerprint(parsed); before.Plan.PluginDefinitionFingerprint != want {
		t.Fatalf("definition fingerprint %s != plugin.BuildDefinitionFingerprint %s", before.Plan.PluginDefinitionFingerprint, want)
	}

	// idとversionを変えない意味的な編集で、照合値が変わる。
	edited := strings.Replace(string(raw), "application-start", "traffic-shift", 1)
	if edited == string(raw) {
		t.Fatal("fixture was not edited")
	}
	if err := os.WriteFile(definitionPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	after := mustResolve(t, intent, root, catalog)
	if after.Plan.Plugin != before.Plan.Plugin {
		t.Fatal("semantic edit must keep the plugin identity")
	}
	if after.Plan.PluginDefinitionFingerprint == before.Plan.PluginDefinitionFingerprint {
		t.Fatal("semantic definition edit did not change the definition fingerprint")
	}
	reparsed, err := syntax.NewParser(edited).ParsePlugin()
	if err != nil {
		t.Fatal(err)
	}
	if after.Plan.PluginDefinitionFingerprint != plugin.BuildDefinitionFingerprint(reparsed) {
		t.Fatal("definition fingerprint does not follow the parsed definition")
	}
}

func TestDefinitionSourceMustBeASingleFile(t *testing.T) {
	intent := loadIntent(t)
	intent.Sources[0].Ref = "impl"
	if _, err := Resolve(intent, NewFixtureContext(copySourceTree(t), loadCatalog(t))); err == nil {
		t.Fatal("a directory was accepted as the plugin definition source")
	}
}

func TestCatalogRejectsDuplicatesAndIgnoresOrder(t *testing.T) {
	for _, key := range []string{"providers", "dependencies", "targets"} {
		var generic map[string]any
		if err := json.Unmarshal(readTestdata(t, "catalog.json"), &generic); err != nil {
			t.Fatal(err)
		}
		list := generic[key].([]any)
		generic[key] = append(list, list[0])
		edited, _ := json.Marshal(generic)
		if _, err := ParseFixtureCatalog(edited); err == nil || !strings.Contains(err.Error(), "more than once") {
			t.Fatalf("%s: duplicate declaration was accepted: %v", key, err)
		}
	}

	// 同じ対象で記録だけが違う要素があっても、列挙順は照合値を変えない。
	catalog := loadCatalog(t)
	catalog.Dependencies = []FixtureDependency{
		{Kind: "build-artifact", ID: "tei.support.codec", Version: "1.4.0", ArtifactDigest: digestA},
		{Kind: "build-artifact", ID: "tei.support.codec", Version: "1.4.0", ArtifactDigest: digestB},
	}
	shuffled := catalog
	shuffled.Dependencies = []FixtureDependency{catalog.Dependencies[1], catalog.Dependencies[0]}
	shuffled.Providers = []FixtureProvider{catalog.Providers[1], catalog.Providers[0]}
	shuffled.Targets = []FixtureTargetEntry{catalog.Targets[1], {Target: catalog.Targets[0].Target, Profiles: []string{"debug", "release"}}}
	a, b := NewFixtureContext("unused", catalog), NewFixtureContext("unused", shuffled)
	for name, pair := range map[string][2]func() (string, error){
		"builder":    {a.Builders.CatalogFingerprint, b.Builders.CatalogFingerprint},
		"dependency": {a.Dependencies.CatalogFingerprint, b.Dependencies.CatalogFingerprint},
		"target":     {a.Targets.CatalogFingerprint, b.Targets.CatalogFingerprint},
	} {
		x, err1 := pair[0]()
		y, err2 := pair[1]()
		if err1 != nil || err2 != nil || x != y {
			t.Fatalf("%s: catalog order changed the fingerprint (%v %v)", name, err1, err2)
		}
	}
}

func TestSourceRefsRejectSymlinksCaseVariantsAndEmptyDirectories(t *testing.T) {
	catalog := loadCatalog(t)
	resolveRef := func(root, ref string) error {
		intent := loadIntent(t)
		intent.Sources[1].Ref = ref
		_, err := Resolve(intent, NewFixtureContext(root, catalog))
		return err
	}

	root := copySourceTree(t)
	if err := os.Mkdir(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := resolveRef(root, "empty"); err == nil || !strings.Contains(err.Error(), "contains no files") {
		t.Fatalf("empty directory resolved: %v", err)
	}
	if err := resolveRef(root, "IMPL"); err == nil {
		t.Fatal("case variant of an existing directory resolved")
	}
	if err := resolveRef(root, "impl/MAIN.GO.TXT"); err == nil {
		t.Fatal("case variant of an existing file resolved")
	}

	if err := os.Symlink("impl", filepath.Join(root, "inlink")); err != nil {
		t.Skipf("symlink is unavailable: %v", err)
	}
	if err := os.Symlink("main.go.txt", filepath.Join(root, "impl", "filelink.go.txt")); err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"inlink", "inlink/main.go.txt", "impl/filelink.go.txt"} {
		if err := resolveRef(root, ref); err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("ref %q through or to a symlink resolved: %v", ref, err)
		}
	}
}

func TestIntentRejectsDuplicateAndCaseVariantKeys(t *testing.T) {
	base := string(readTestdata(t, "intent.json"))
	const target = `"target": "host/linux/amd64",`
	if strings.Count(base, target) != 1 {
		t.Fatal("fixture layout changed")
	}
	cases := map[string]string{
		"duplicate, last wins":    strings.Replace(base, target, `"target": "host/windows/amd64", `+target, 1),
		"duplicate, first wins":   strings.Replace(base, target, target+` "target": "host/windows/amd64",`, 1),
		"duplicate nested":        strings.Replace(base, `"kind": "go-package", "id": "implementation"`, `"kind": "go-package", "kind": "x", "id": "implementation"`, 1),
		"case variant key":        strings.Replace(base, `"target"`, `"Target"`, 1),
		"case variant nested key": strings.Replace(base, `"recipe": {"kind"`, `"recipe": {"KIND"`, 1),
		"upper case top-level":    strings.Replace(base, `"sources"`, `"SOURCES"`, 1),
	}
	for name, edited := range cases {
		if edited == base {
			t.Fatalf("%s: edit did not change the intent", name)
		}
		if _, err := ParseIntent([]byte(edited)); err == nil {
			t.Fatalf("%s: accepted", name)
		}
	}
	// 導出値の名前は、別綴りでも導出値の層が先に拒否する（多層防御）。
	derived := strings.Replace(base, `"target"`, `"Digest": "x", "target"`, 1)
	if _, err := ParseIntent([]byte(derived)); err == nil || !strings.Contains(err.Error(), "derived field") {
		t.Fatalf("case variant of a derived key was not caught by the derived-key layer: %v", err)
	}
}
