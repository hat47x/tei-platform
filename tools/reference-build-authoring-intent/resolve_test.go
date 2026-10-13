package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/plugin"
)

const (
	digestA = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	digestB = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
)

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func loadIntent(t *testing.T) BuildIntent {
	t.Helper()
	intent, err := ParseIntent(readTestdata(t, "intent.json"))
	if err != nil {
		t.Fatal(err)
	}
	return intent
}

func loadCatalog(t *testing.T) FixtureCatalog {
	t.Helper()
	catalog, err := ParseFixtureCatalog(readTestdata(t, "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

// copySourceTree はtestdata/srcを書き換え可能な一時領域へ複製する。
func copySourceTree(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	src := filepath.Join("testdata", "src")
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func mustResolve(t *testing.T, intent BuildIntent, root string, catalog FixtureCatalog) ResolvedBuildReceipt {
	t.Helper()
	receipt, err := Resolve(intent, NewFixtureContext(root, catalog))
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func mustMarshal(t *testing.T, r ResolvedBuildReceipt) []byte {
	t.Helper()
	raw, err := MarshalReceipt(r)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func sourceByID(t *testing.T, plan plugin.BuildPlan, id string) plugin.BuildSource {
	t.Helper()
	for _, s := range plan.Sources {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("source %q not found", id)
	return plugin.BuildSource{}
}

func TestResolveDerivesExactBuildPlanFromIntentWithoutAuthoredDigests(t *testing.T) {
	raw := readTestdata(t, "intent.json")
	for _, forbidden := range []string{"sha256:", "ingerprint", "igest"} {
		if bytes.Contains(raw, []byte(forbidden)) {
			t.Fatalf("authoring fixture must not contain derived value marker %q", forbidden)
		}
	}
	root := copySourceTree(t)
	catalog := loadCatalog(t)
	intent := loadIntent(t)

	first := mustResolve(t, intent, root, catalog)
	second := mustResolve(t, intent, root, catalog)
	if !bytes.Equal(mustMarshal(t, first), mustMarshal(t, second)) {
		t.Fatal("same intent and same resolution context produced different receipts")
	}
	if err := first.Plan.Validate(); err != nil {
		t.Fatalf("derived plan is not a valid plugin.BuildPlan: %v", err)
	}
	want, err := plugin.BuildPlanFingerprint(first.Plan)
	if err != nil {
		t.Fatal(err)
	}
	if first.PlanFingerprint != want {
		t.Fatalf("receipt plan fingerprint %s != plugin.BuildPlanFingerprint %s", first.PlanFingerprint, want)
	}
	if first.Authority != AuthorityDerivedCache {
		t.Fatalf("receipt must declare itself derived, got %q", first.Authority)
	}
	if first.Plan.Builder.ArtifactDigest != digestA {
		t.Fatalf("linux target resolved builder %s", first.Plan.Builder.ArtifactDigest)
	}
	if first.Plan.Builder.ContractVersion != "tei.plugin-build/v1" {
		t.Fatalf("builder contract version = %q", first.Plan.Builder.ContractVersion)
	}
	if len(first.Plan.Dependencies) != 1 || first.Plan.Dependencies[0].ArtifactDigest != "sha256:3333333333333333333333333333333333333333333333333333333333333333" {
		t.Fatalf("dependency digest was not derived from the catalog: %+v", first.Plan.Dependencies)
	}
	for _, s := range first.Plan.Sources {
		if !sha256Digest.MatchString(s.Digest) {
			t.Fatalf("source %s digest %q was not derived from bytes", s.ID, s.Digest)
		}
	}
	// 実BuildPlanがLockへ進める。
	if _, err := plugin.LockBuildOutput(first.Plan, digestB); err != nil {
		t.Fatal(err)
	}
}

func TestIntentFieldOrderDoesNotChangeResult(t *testing.T) {
	root := copySourceTree(t)
	catalog := loadCatalog(t)
	original := loadIntent(t)

	var generic map[string]any
	if err := json.Unmarshal(readTestdata(t, "intent.json"), &generic); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"sources", "parameters", "dependencies", "secretRequirements"} {
		list := generic[key].([]any)
		for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
			list[i], list[j] = list[j], list[i]
		}
	}
	reordered, err := json.Marshal(generic) // map経由でkey順も変わる
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(reordered, readTestdata(t, "intent.json")) {
		t.Fatal("fixture was not reordered")
	}
	parsed, err := ParseIntent(reordered)
	if err != nil {
		t.Fatal(err)
	}
	a, err := IntentFingerprint(original)
	if err != nil {
		t.Fatal(err)
	}
	b, err := IntentFingerprint(parsed)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatal("intent fingerprint depends on declaration order")
	}
	if !bytes.Equal(mustMarshal(t, mustResolve(t, original, root, catalog)), mustMarshal(t, mustResolve(t, parsed, root, catalog))) {
		t.Fatal("declaration order changed the resolved receipt")
	}
}

func TestSourceContentChangeChangesDerivedIdentityButNotIntent(t *testing.T) {
	root := copySourceTree(t)
	catalog := loadCatalog(t)
	intent := loadIntent(t)
	before := mustResolve(t, intent, root, catalog)

	if err := os.WriteFile(filepath.Join(root, "impl", "main.go.txt"), []byte("package main\n\nfunc main() { println(1) }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	after := mustResolve(t, intent, root, catalog)

	if before.IntentFingerprint != after.IntentFingerprint {
		t.Fatal("source content change must not change the intent fingerprint")
	}
	if before.SourceClosureFingerprint == after.SourceClosureFingerprint ||
		before.ResolutionContextFingerprint == after.ResolutionContextFingerprint ||
		before.PlanFingerprint == after.PlanFingerprint {
		t.Fatal("source content change did not change closure, context and plan fingerprints")
	}
	if sourceByID(t, before.Plan, "implementation").Digest == sourceByID(t, after.Plan, "implementation").Digest {
		t.Fatal("implementation source digest did not change")
	}
	if sourceByID(t, before.Plan, "definition").Digest != sourceByID(t, after.Plan, "definition").Digest {
		t.Fatal("untouched definition source digest changed")
	}
	if before.Plan.PluginDefinitionFingerprint != after.Plan.PluginDefinitionFingerprint {
		t.Fatal("definition fingerprint changed although the definition did not")
	}
}

func TestSourceClosureDigestIgnoresRootLocationAndRejectsUnsafeMembers(t *testing.T) {
	catalog := loadCatalog(t)
	intent := loadIntent(t)
	a := mustResolve(t, intent, copySourceTree(t), catalog)
	b := mustResolve(t, intent, copySourceTree(t), catalog)
	if a.SourceClosureFingerprint != b.SourceClosureFingerprint {
		t.Fatal("source closure depends on the Host path of the root")
	}

	root := copySourceTree(t)
	if err := os.Symlink("main.go.txt", filepath.Join(root, "impl", "link.go.txt")); err != nil {
		t.Skipf("symlink is unavailable: %v", err)
	}
	if _, err := Resolve(intent, NewFixtureContext(root, catalog)); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("symlink member was accepted: %v", err)
	}
}

func TestIntentRejectsEscapingAndMissingSourceRefs(t *testing.T) {
	for _, ref := range []string{"../outside", "/abs", `a\b`, "impl/../x", ".", ""} {
		intent := loadIntent(t)
		intent.Sources[1].Ref = ref
		if err := intent.Validate(); err == nil {
			t.Fatalf("source ref %q was accepted", ref)
		}
	}
	intent := loadIntent(t)
	intent.Sources[1].Ref = "does-not-exist"
	if _, err := Resolve(intent, NewFixtureContext(copySourceTree(t), loadCatalog(t))); err == nil {
		t.Fatal("missing source ref resolved")
	}
}

func TestBuilderProviderAmbiguityStopsRegardlessOfCatalogOrder(t *testing.T) {
	root := copySourceTree(t)
	intent := loadIntent(t)
	catalog := loadCatalog(t)
	alt := catalog.Providers[0]
	alt.Plugin = "tei.builder.go_alt"
	alt.ArtifactDigest = digestB
	catalog.Providers = append(catalog.Providers, alt)

	_, err := Resolve(intent, NewFixtureContext(root, catalog))
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity, got %v", err)
	}
	reversed := catalog
	reversed.Providers = append([]FixtureProvider(nil), catalog.Providers...)
	for i, j := 0, len(reversed.Providers)-1; i < j; i, j = i+1, j-1 {
		reversed.Providers[i], reversed.Providers[j] = reversed.Providers[j], reversed.Providers[i]
	}
	_, err2 := Resolve(intent, NewFixtureContext(root, reversed))
	if err2 == nil || err2.Error() != err.Error() {
		t.Fatalf("catalog order changed the ambiguity result: %v vs %v", err, err2)
	}
}

func TestExplicitWiringAndPinFixTheExactBuilder(t *testing.T) {
	root := copySourceTree(t)
	catalog := loadCatalog(t)
	baseline := mustResolve(t, loadIntent(t), root, catalog)

	alt := catalog.Providers[0]
	alt.Plugin = "tei.builder.go_alt"
	alt.ArtifactDigest = digestB
	ambiguous := catalog
	ambiguous.Providers = append(append([]FixtureProvider(nil), catalog.Providers...), alt)

	wired := loadIntent(t)
	wired.Builder.Provider = &ProviderWiring{Plugin: "tei.builder.go", Implementation: "go.builder"}
	got := mustResolve(t, wired, root, ambiguous)
	if got.Plan.Builder.ArtifactDigest != digestA || got.Plan.Builder.ID != "tei.builder.go" {
		t.Fatalf("wiring did not select the requested provider: %+v", got.Plan.Builder)
	}
	if got.PlanFingerprint != baseline.PlanFingerprint {
		t.Fatal("explicit wiring resolved a different exact plan than the unambiguous catalog")
	}
	if got.IntentFingerprint == baseline.IntentFingerprint {
		t.Fatal("wiring is part of the intent and must change the intent fingerprint")
	}
	if got.ResolutionContextFingerprint == baseline.ResolutionContextFingerprint {
		t.Fatal("adding a provider must change the resolution context fingerprint")
	}

	// 同じlogical Providerに複数のphysical artifactがある場合、wiringだけでは曖昧である。
	twin := catalog.Providers[0]
	twin.Version = "1.3.0"
	twin.ArtifactDigest = digestB
	twins := catalog
	twins.Providers = append(append([]FixtureProvider(nil), catalog.Providers...), twin)
	if _, err := Resolve(wired, NewFixtureContext(root, twins)); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("wiring without a pin must stay ambiguous: %v", err)
	}
	pinned := wired
	pinned.Builder.Pin = &ArtifactPin{ArtifactDigest: digestB}
	pinnedReceipt := mustResolve(t, pinned, root, twins)
	if pinnedReceipt.Plan.Builder.ArtifactDigest != digestB || pinnedReceipt.Plan.Builder.Version != "1.3.0" || !pinnedReceipt.BuilderResolution.Pinned {
		t.Fatalf("pin did not fix the exact artifact: %+v", pinnedReceipt.BuilderResolution)
	}
	missing := wired
	missing.Builder.Pin = &ArtifactPin{ArtifactDigest: "sha256:" + strings.Repeat("9", 64)}
	if _, err := Resolve(missing, NewFixtureContext(root, twins)); err == nil || !strings.Contains(err.Error(), "pinned") {
		t.Fatalf("pin to an unknown artifact resolved: %v", err)
	}
}

func TestBuilderRequiresBuildToolScopeContractAndRecipe(t *testing.T) {
	root := copySourceTree(t)
	mutations := map[string]func(*FixtureCatalog){
		"scope":    func(c *FixtureCatalog) { c.Providers[0].AuthorityScope = "runtime" },
		"major":    func(c *FixtureCatalog) { c.Providers[0].ContractMajor = 2 },
		"recipe":   func(c *FixtureCatalog) { c.Providers[0].RecipeKinds = []string{"cargo"} },
		"profile":  func(c *FixtureCatalog) { c.Providers[0].Profiles = []string{"debug"} },
		"contract": func(c *FixtureCatalog) { c.Providers[0].Contract = "other.build" },
	}
	for name, mutate := range mutations {
		catalog := loadCatalog(t)
		mutate(&catalog)
		if _, err := Resolve(loadIntent(t), NewFixtureContext(root, catalog)); err == nil || !strings.Contains(err.Error(), "no compatible builder provider") {
			t.Fatalf("%s: incompatible provider was accepted: %v", name, err)
		}
	}
}

func TestIntentRejectsAuthoredDerivedValues(t *testing.T) {
	base := readTestdata(t, "intent.json")
	var generic map[string]any
	if err := json.Unmarshal(base, &generic); err != nil {
		t.Fatal(err)
	}
	inject := map[string]func(map[string]any){
		"source digest": func(m map[string]any) {
			m["sources"].([]any)[1].(map[string]any)["digest"] = digestA
		},
		"definition fingerprint": func(m map[string]any) {
			m["pluginDefinitionFingerprint"] = strings.Repeat("a", 64)
		},
		"locator": func(m map[string]any) {
			m["sources"].([]any)[1].(map[string]any)["locator"] = "impl"
		},
		"builder artifact outside pin": func(m map[string]any) {
			m["builder"].(map[string]any)["artifactDigest"] = digestA
		},
		"resolved plan": func(m map[string]any) {
			m["plan"] = map[string]any{"formatVersion": 1}
		},
		"dependency digest": func(m map[string]any) {
			m["dependencies"].([]any)[0].(map[string]any)["artifactDigest"] = digestA
		},
		"unknown field": func(m map[string]any) {
			m["shellCommand"] = "make"
		},
	}
	for name, mutate := range inject {
		var copyMap map[string]any
		raw, _ := json.Marshal(generic)
		_ = json.Unmarshal(raw, &copyMap)
		mutate(copyMap)
		edited, _ := json.Marshal(copyMap)
		if _, err := ParseIntent(edited); err == nil {
			t.Fatalf("%s: authored derived value was accepted", name)
		}
	}
	// 意図としてのpinだけは書ける。
	var withPin map[string]any
	_ = json.Unmarshal(base, &withPin)
	withPin["builder"].(map[string]any)["pin"] = map[string]any{"artifactDigest": digestA}
	raw, _ := json.Marshal(withPin)
	if _, err := ParseIntent(raw); err != nil {
		t.Fatalf("intended pin was rejected: %v", err)
	}
	if _, err := ParseIntent(append(append([]byte(nil), base...), []byte(`{}`)...)); err == nil {
		t.Fatal("trailing data was accepted")
	}
}

func TestTargetVariationChangesOnlyTheResolvedPlanParts(t *testing.T) {
	root := copySourceTree(t)
	catalog := loadCatalog(t)
	linux := mustResolve(t, loadIntent(t), root, catalog)

	windowsIntent := loadIntent(t)
	windowsIntent.Target = "host/windows/amd64"
	windows := mustResolve(t, windowsIntent, root, catalog)

	if linux.PlanFingerprint == windows.PlanFingerprint || linux.Plan.Target == windows.Plan.Target {
		t.Fatal("target change did not produce a different resolved plan")
	}
	if linux.Plan.Builder.ArtifactDigest == windows.Plan.Builder.ArtifactDigest {
		t.Fatal("target change did not select the target-specific builder")
	}
	if windows.Plan.Builder.ArtifactDigest != digestB {
		t.Fatalf("windows builder = %s", windows.Plan.Builder.ArtifactDigest)
	}
	if linux.SourceClosureFingerprint != windows.SourceClosureFingerprint ||
		linux.Plan.PluginDefinitionFingerprint != windows.Plan.PluginDefinitionFingerprint ||
		linux.DependencyResolutionFingerprint != windows.DependencyResolutionFingerprint {
		t.Fatal("target change altered resolved parts that do not depend on the target")
	}

	unknown := loadIntent(t)
	unknown.Target = "host/plan9/amd64"
	if _, err := Resolve(unknown, NewFixtureContext(root, catalog)); err == nil {
		t.Fatal("unknown target resolved")
	}
	badProfile := windowsIntent
	badProfile.Profile = "debug"
	if _, err := Resolve(badProfile, NewFixtureContext(root, catalog)); err == nil {
		t.Fatal("profile undefined for the target resolved")
	}
}

func TestCatalogChangesAffectContextOnlyWhereRelevantToThePlan(t *testing.T) {
	root := copySourceTree(t)
	catalog := loadCatalog(t)
	intent := loadIntent(t)
	base := mustResolve(t, intent, root, catalog)

	// 無関係なProvider追加はContextを変えるが、同じexact planを導く。
	unrelated := catalog
	unrelated.Providers = append(append([]FixtureProvider(nil), catalog.Providers...), FixtureProvider{
		Plugin: "tei.builder.rust", Version: "0.1.0", Implementation: "rust.builder",
		Contract: "tei.plugin-build", ContractMajor: 1, AuthorityScope: "build-tool",
		ArtifactDigest: digestB, RecipeKinds: []string{"cargo"}, Targets: []string{"host/linux/amd64"}, Profiles: []string{"release"},
	})
	got := mustResolve(t, intent, root, unrelated)
	if got.PlanFingerprint != base.PlanFingerprint {
		t.Fatal("unrelated provider changed the resolved plan")
	}
	if got.ResolutionContextFingerprint == base.ResolutionContextFingerprint {
		t.Fatal("catalog change was not visible in the resolution context")
	}

	// Catalogの列挙順だけではContextは変わらない。
	shuffled := catalog
	shuffled.Providers = []FixtureProvider{catalog.Providers[1], catalog.Providers[0]}
	if mustResolve(t, intent, root, shuffled).ResolutionContextFingerprint != base.ResolutionContextFingerprint {
		t.Fatal("catalog listing order changed the resolution context")
	}

	// Build依存のproducer outputが変わればexact planが変わる。
	rebuilt := catalog
	rebuilt.Dependencies = []FixtureDependency{{
		Kind: "build-artifact", ID: "tei.support.codec", Version: "1.4.0", ArtifactDigest: digestB,
	}}
	changed := mustResolve(t, intent, root, rebuilt)
	if changed.PlanFingerprint == base.PlanFingerprint || changed.IntentFingerprint != base.IntentFingerprint ||
		changed.DependencyResolutionFingerprint == base.DependencyResolutionFingerprint {
		t.Fatal("dependency producer change was not reflected only in the derived plan")
	}

	// 依存が見つからない、または曖昧な場合は停止する。
	empty := catalog
	empty.Dependencies = nil
	if _, err := Resolve(intent, NewFixtureContext(root, empty)); err == nil {
		t.Fatal("missing build dependency resolved")
	}
	twice := catalog
	twice.Dependencies = append(append([]FixtureDependency(nil), catalog.Dependencies...), FixtureDependency{
		Kind: "build-artifact", ID: "tei.support.codec", Version: "1.5.0", ArtifactDigest: digestB,
	})
	if _, err := Resolve(intent, NewFixtureContext(root, twice)); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous dependency resolved: %v", err)
	}
}

func TestSecretsAreNeverAuthoredOrPlacedInDerivedOutput(t *testing.T) {
	const secretValue = "s3cr3t-value-must-not-leak"
	t.Setenv("SIGNING_KEY", secretValue)
	root := copySourceTree(t)
	catalog := loadCatalog(t)
	intent := loadIntent(t)

	withParam := intent
	withParam.Parameters = append(append([]plugin.BuildParameter(nil), intent.Parameters...), plugin.BuildParameter{Name: "api_token", Value: secretValue})
	if err := withParam.Validate(); err == nil {
		t.Fatal("secret-like parameter was accepted")
	}

	var generic map[string]any
	if err := json.Unmarshal(readTestdata(t, "intent.json"), &generic); err != nil {
		t.Fatal(err)
	}
	generic["secretRequirements"].([]any)[0].(map[string]any)["value"] = secretValue
	edited, _ := json.Marshal(generic)
	if _, err := ParseIntent(edited); err == nil {
		t.Fatal("secret value in secretRequirements was accepted")
	}

	receipt := mustResolve(t, intent, root, catalog)
	raw := mustMarshal(t, receipt)
	if bytes.Contains(raw, []byte(secretValue)) {
		t.Fatal("secret value leaked into the derived receipt")
	}
	planRaw, _ := json.Marshal(receipt.Plan)
	if bytes.Contains(planRaw, []byte("SIGNING_KEY")) {
		t.Fatal("secret requirement leaked into the BuildPlan")
	}
	if len(receipt.SecretRequirementNames) != 1 || receipt.SecretRequirementNames[0] != "SIGNING_KEY" {
		t.Fatalf("secret requirement names = %v", receipt.SecretRequirementNames)
	}
	without := intent
	without.SecretRequirements = nil
	if mustResolve(t, without, root, catalog).PlanFingerprint != receipt.PlanFingerprint {
		t.Fatal("secret requirement changed the BuildPlan fingerprint")
	}
}

func TestDerivedReceiptIsRegenerableAndEditsDoNotBecomeSource(t *testing.T) {
	root := copySourceTree(t)
	catalog := loadCatalog(t)
	intent := loadIntent(t)
	ctx := NewFixtureContext(root, catalog)

	receiptPath := filepath.Join(t.TempDir(), "resolved.json")
	original := mustMarshal(t, mustResolve(t, intent, root, catalog))
	if err := os.WriteFile(receiptPath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyDerivedReceipt(intent, ctx, original); err != nil {
		t.Fatal(err)
	}

	// 導出物を削除しても、Intentと同じContextから同じbytesを再生成できる。
	if err := os.Remove(receiptPath); err != nil {
		t.Fatal(err)
	}
	if regenerated := mustMarshal(t, mustResolve(t, intent, root, catalog)); !bytes.Equal(regenerated, original) {
		t.Fatal("deleted derived receipt was not regenerated identically")
	}

	// 人が導出物を編集しても、Sourceにはならず乖離として拒否される。
	for name, edit := range map[string]func(*ResolvedBuildReceipt){
		"builder digest": func(r *ResolvedBuildReceipt) { r.Plan.Builder.ArtifactDigest = digestB },
		"target":         func(r *ResolvedBuildReceipt) { r.Plan.Target = "host/windows/amd64" },
		"source digest":  func(r *ResolvedBuildReceipt) { r.Plan.Sources[0].Digest = digestB },
		"plan fp":        func(r *ResolvedBuildReceipt) { r.PlanFingerprint = strings.Repeat("0", 64) },
	} {
		var edited ResolvedBuildReceipt
		if err := json.Unmarshal(original, &edited); err != nil {
			t.Fatal(err)
		}
		edit(&edited)
		editedRaw, err := MarshalReceipt(edited)
		if err != nil {
			t.Fatal(err)
		}
		if err := VerifyDerivedReceipt(intent, ctx, editedRaw); err == nil || !strings.Contains(err.Error(), "diverges") {
			t.Fatalf("%s: edited derived receipt was accepted: %v", name, err)
		}
		// 編集した導出物はResolveの入力にならず、結果も変えない。
		if !bytes.Equal(mustMarshal(t, mustResolve(t, intent, root, catalog)), original) {
			t.Fatalf("%s: resolution depended on a stored derived file", name)
		}
	}

	// 導出物をauthoring sourceとして読ませることはできない。
	if _, err := ParseIntent(original); err == nil {
		t.Fatal("derived receipt was accepted as a build intent")
	}
}

func TestPluginDefinitionMustMatchSubject(t *testing.T) {
	root := copySourceTree(t)
	raw, err := os.ReadFile(filepath.Join(root, "plugin.tei"))
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(raw), "v1.0.0", "v1.0.1", 1)
	if edited == string(raw) {
		t.Fatal("fixture was not edited")
	}
	if err := os.WriteFile(filepath.Join(root, "plugin.tei"), []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(loadIntent(t), NewFixtureContext(root, loadCatalog(t))); err == nil || !strings.Contains(err.Error(), "does not match build intent subject") {
		t.Fatalf("definition/subject mismatch resolved: %v", err)
	}
}

func TestCLIResolvesAndVerifiesFixtures(t *testing.T) {
	root := copySourceTree(t)
	args := []string{
		"-intent", filepath.Join("testdata", "intent.json"),
		"-catalog", filepath.Join("testdata", "catalog.json"),
		"-source-root", root,
	}
	var out, errOut bytes.Buffer
	if code := run(args, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	var receipt ResolvedBuildReceipt
	if err := json.Unmarshal(out.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if err := receipt.Plan.Validate(); err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(t.TempDir(), "resolved.json")
	if err := os.WriteFile(receiptPath, out.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()
	if code := run(append(args, "-verify", receiptPath), &out, &errOut); code != 0 {
		t.Fatalf("verify exit %d: %s", code, errOut.String())
	}
	if code := run([]string{"-intent", "x"}, &out, &errOut); code != 2 {
		t.Fatalf("missing flags exit = %d", code)
	}
}
