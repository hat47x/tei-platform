package pluginbuildsupply

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"tei/internal/plugin"
)

type cliCrossCheck struct {
	ExitCode             int                `json:"exitCode"`
	IntentFingerprint    string             `json:"intentFingerprint"`
	PlanFingerprint      string             `json:"planFingerprint"`
	NonSourceFieldsEqual bool               `json:"nonSourceFieldsEqual"`
	SourceDifferences    []sourceDifference `json:"sourceDifferences"`
}

type sourceDifference struct {
	ID            string `json:"id"`
	CLILocator    string `json:"cliLocator"`
	CLIDigest     string `json:"cliDigest"`
	SampleLocator string `json:"sampleLocator"`
	SampleDigest  string `json:"sampleDigest"`
}

type throughRun struct {
	intentPath string
	intent     Intent
	toolchain  GoToolchain
	catalog    []Provider
	s1         Stage1
	cli        cliCrossCheck

	host     *Host
	attempts map[string]AttemptRecord
	killErr  error
	history  History
	rep      Reproducibility
	store    Store
	put      StoreReceipt

	runner      RunnerResult
	poisonEmpty map[string]bool
	s3          Stage3
	s4          Stage4

	inputs []stageInput
	chain  Chain
	result ThroughResult
}

func throughFixture(t *testing.T) *throughRun {
	t.Helper()
	env := requireEnvironment(t)
	shared.throughOnce.Do(func() {
		shared.throughErr = errors.New("the through run failed in " + t.Name())
		shared.through = runThrough(t, env)
		shared.throughErr = nil
	})
	if shared.throughErr != nil {
		t.Fatal(shared.throughErr)
	}
	return shared.through
}

// fixtureCatalog renders providers in the authoring-intent catalog format.
func fixtureCatalog(providers []Provider, target, profile string) map[string]any {
	list := make([]map[string]any, 0, len(providers))
	for _, p := range providers {
		list = append(list, map[string]any{
			"plugin": p.ID, "version": p.Version, "implementation": p.Implementation, "contract": p.Contract,
			"contractMajor": p.ContractMajor, "authorityScope": p.AuthorityScope, "artifactDigest": p.ArtifactDigest,
			"recipeKinds": p.RecipeKinds, "targets": p.Targets, "profiles": p.Profiles,
		})
	}
	return map[string]any{
		"formatVersion": 1, "providers": list, "dependencies": []any{},
		"targets": []any{map[string]any{"target": target, "profiles": []string{profile}}},
	}
}

func runAuthoringIntent(t *testing.T, env environment, intentPath string, providers []Provider, intent Intent, name string) (int, string, string) {
	t.Helper()
	catalogPath := filepath.Join(env.work, "catalog-"+name+".json")
	writeJSON(t, catalogPath, fixtureCatalog(providers, intent.Target, intent.Profile))
	return runTool(t, env.tools["authoring-intent"], "-intent", intentPath, "-catalog", catalogPath, "-source-root", env.repo)
}

func crossCheck(t *testing.T, env environment, r *throughRun) cliCrossCheck {
	t.Helper()
	code, stdout, stderr := runAuthoringIntent(t, env, r.intentPath, r.catalog, r.intent, "single")
	out := cliCrossCheck{ExitCode: code}
	if code != 0 {
		t.Fatalf("authoring-intent CLI: exit %d: %s", code, stderr)
	}
	var receipt struct {
		IntentFingerprint string           `json:"intentFingerprint"`
		PlanFingerprint   string           `json:"planFingerprint"`
		Plan              plugin.BuildPlan `json:"plan"`
	}
	if err := json.Unmarshal([]byte(stdout), &receipt); err != nil {
		t.Fatal(err)
	}
	out.IntentFingerprint, out.PlanFingerprint = receipt.IntentFingerprint, receipt.PlanFingerprint
	cli, sample := plugin.NormalizeBuildPlan(receipt.Plan), plugin.NormalizeBuildPlan(r.s1.Plan)
	for _, c := range cli.Sources {
		for _, s := range sample.Sources {
			if c.ID == s.ID && (c.Digest != s.Digest || c.Locator != s.Locator) {
				out.SourceDifferences = append(out.SourceDifferences, sourceDifference{c.ID, c.Locator, c.Digest, s.Locator, s.Digest})
			}
		}
	}
	cli.Sources, sample.Sources = nil, nil
	out.NonSourceFieldsEqual = reflect.DeepEqual(cli, sample)
	return out
}

func runThrough(t *testing.T, env environment) *throughRun {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	r := &throughRun{attempts: map[string]AttemptRecord{}}
	var err error

	// Stage 1: the declaration, real files and the observed toolchain give
	// the confirmed plan and the resolution lock.
	if r.intentPath, err = filepath.Abs(filepath.Join("testdata", "intent.json")); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(r.intentPath)
	if err != nil {
		t.Fatal(err)
	}
	if r.intent, err = ParseIntent(raw); err != nil {
		t.Fatal(err)
	}
	if r.toolchain, err = ObserveGoToolchain(env.goPath); err != nil {
		t.Fatal(err)
	}
	r.catalog = []Provider{GoSeedProvider(r.toolchain, r.intent.Target, r.intent.Profile)}
	if r.s1, err = Derive(env.repo, r.intent, DeriveOptions{Toolchain: r.toolchain, Catalog: r.catalog, StateDir: filepath.Join(env.work, "stage1-state")}); err != nil {
		t.Fatal(err)
	}
	r.cli = crossCheck(t, env, r)

	// Stage 2: two attempts of the same plan start together and one is
	// killed; a third attempt follows. Only host-issued occurrences count.
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	r.host = NewHost(exe, filepath.Join(env.work, "attempts"))
	spec := AttemptSpec{Plan: r.s1.Plan, Lock: r.s1.Lock, Repo: env.repo, GoPath: r.toolchain.Executable, Mode: ModeBuild}
	killed := spec
	killed.OnStarted = func(_ string, kill func() error) { r.killErr = kill() }
	var wg sync.WaitGroup
	var recA, recB AttemptRecord
	var errA, errB error
	wg.Add(2)
	go func() { defer wg.Done(); recA, errA = r.host.Run(ctx, spec) }()
	go func() { defer wg.Done(); recB, errB = r.host.Run(ctx, killed) }()
	wg.Wait()
	if errA != nil || errB != nil {
		t.Fatalf("concurrent attempts: %v / %v", errA, errB)
	}
	recC, err := r.host.Run(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	r.attempts["A"], r.attempts["B-killed"], r.attempts["C"] = recA, recB, recC
	r.history = History{PlanFingerprint: r.s1.PlanFingerprint}
	for _, rec := range []AttemptRecord{recA, recB, recC} {
		if err := r.history.Append(r.host, rec); err != nil {
			t.Fatal(err)
		}
	}
	if r.rep, err = DeriveReproducibility(r.host, r.history); err != nil {
		t.Fatalf("reproducibility: %v (A=%s/%s C=%s/%s)", err, recA.Outcome, recA.ArtifactDigest, recC.Outcome, recC.ArtifactDigest)
	}
	output, err := recA.OutputOf()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	r.store = Store{Root: filepath.Join(env.work, "store")}
	if r.put, err = r.store.PutReproducible(r.rep, data); err != nil {
		t.Fatal(err)
	}

	// Stage 3: the existing runner, unchanged, with host Go settings that
	// would leak a cache or module mode into a build that inherited them.
	poison := map[string]string{}
	runnerEnv := withoutKeys(os.Environ(), "GOFLAGS", "GOCACHE", "GOMODCACHE", "GOPATH")
	for _, key := range []string{"GOCACHE", "GOMODCACHE", "GOPATH"} {
		dir := filepath.Join(env.work, "poison-"+strings.ToLower(key))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		poison[key] = dir
		runnerEnv = append(runnerEnv, key+"="+dir)
	}
	runnerEnv = append(runnerEnv, "GOFLAGS=-mod=mod")
	if r.runner, err = RunSelfBuild(ctx, env.tools["self-build"], env.repo, r.toolchain.Executable, runnerEnv); err != nil {
		t.Fatal(err)
	}
	if r.runner.ExitCode != 0 || r.runner.Evidence == nil {
		t.Fatalf("self-build runner: exit %d: %s", r.runner.ExitCode, r.runner.Stderr)
	}
	r.poisonEmpty = map[string]bool{}
	for key, dir := range poison {
		entries, err := os.ReadDir(dir)
		r.poisonEmpty[key] = err == nil && len(entries) == 0
	}
	if r.s3, err = BindStage3(r.s1, r.rep, *r.runner.Evidence, TrustStore{}); err != nil {
		t.Fatal(err)
	}

	// Stage 4: call the stored bytes, not a rebuild.
	if r.s4, err = InvokeFromStore(ctx, env.repo, r.store, r.put.Digest, *r.runner.Evidence, filepath.Join(env.work, "stage4")); err != nil {
		t.Fatal(err)
	}

	var closure string
	for _, src := range r.s1.Plan.Sources {
		if src.Kind == sourceKindGoClosure {
			closure = src.Digest
		}
	}
	ev := r.runner.Evidence
	r.inputs = []stageInput{
		{1, r.s1.PlanFingerprint, nil, nil},
		{2, r.rep.PlanFingerprint, map[string]string{"first-success": r.rep.Digests[0], "second-success": r.rep.Digests[1], "stored": r.put.Digest}, nil},
		{3, r.s3.PlanFingerprint, map[string]string{"first-build": r.s3.FirstArtifactDigest, "second-build": r.s3.SecondArtifactDigest, "registered": r.s3.RegisteredArtifactDigest},
			map[string]string{"transform": r.s3.TransformEngineFingerprint, "static-response": r.s3.StaticResponseEngineFingerprint}},
		{4, r.s4.BuildPlanFingerprint, map[string]string{"retrieved": r.s4.RetrievedDigest, "invoked": r.s4.InvokedArtifactDigest},
			map[string]string{"static-response": r.s4.StaticResponseEngineFingerprint}},
	}
	details := []any{
		map[string]any{"revision": r.s1.Revision, "intentFingerprint": r.s1.IntentFingerprint, "resolutionLockFingerprint": r.s1.Lock.Fingerprint,
			"builder": r.s1.Plan.Builder, "goVersion": r.toolchain.Version, "sourceClosureDigest": closure,
			"sourceClosureMembers": len(r.s1.SourceClosure), "authoringIntentCLI": r.cli},
		map[string]any{"attempts": r.history.Attempts, "reproducibility": r.rep, "store": r.put},
		map[string]any{"binding": r.s3, "runnerSchema": ev.Schema, "repositoryTreeDigest": ev.RepositoryTreeDigest, "hostGoSettingsIgnored": r.poisonEmpty},
		r.s4,
	}
	c := NewChain()
	for i, in := range r.inputs {
		if err := c.Append(in.stage, in.plan, in.artifacts, in.engines, details[i]); err != nil {
			t.Fatal(err)
		}
	}
	r.chain = *c
	r.result = VerifyThrough(r.chain)
	if dir := os.Getenv(envOutDir); dir != "" {
		writeJSON(t, filepath.Join(dir, "chain.json"), map[string]any{"chain": r.chain, "through": r.result})
	}
	return r
}

func TestThroughPluginBuildSupply(t *testing.T) {
	r := throughFixture(t)
	if !r.result.Complete {
		t.Fatalf("through check failed: %+v", r.result)
	}
	if r.result.PlanFingerprint != r.s1.PlanFingerprint || r.result.ArtifactDigest != r.rep.ArtifactDigest {
		t.Fatalf("through identity: %+v", r.result)
	}
	if r.s4.Outcome != "accepted" || !r.s4.DigestVerified || !r.s4.ArgvPreserved || r.s4.ProtocolVersion != 1 {
		t.Fatalf("stage 4 call: %+v", r.s4)
	}
	if r.s4.PluginLockFingerprint != r.runner.Evidence.RuntimePluginLockFingerprint {
		t.Fatalf("stage 4 used another plugin lock: %s", r.s4.PluginLockFingerprint)
	}
	if !r.cli.NonSourceFieldsEqual || r.cli.ExitCode != 0 {
		t.Fatalf("authoring-intent CLI disagrees beyond source digests: %+v", r.cli)
	}
	observed(t, "through", r.result)
	observed(t, "stage1", map[string]any{"planFingerprint": r.s1.PlanFingerprint, "builder": r.s1.Plan.Builder, "lock": r.s1.Lock.Fingerprint, "revision": r.s1.Revision})
	observed(t, "stage1-authoring-intent-cli", r.cli)
	observed(t, "stage2", map[string]any{"reproducibility": r.rep, "store": r.put})
	observed(t, "stage3", r.s3)
	observed(t, "stage4", r.s4)
}

func TestPJOB01ConcurrentAttemptsKeepDistinctOccurrences(t *testing.T) {
	r := throughFixture(t)
	a, b, c := r.attempts["A"], r.attempts["B-killed"], r.attempts["C"]
	if r.killErr != nil {
		t.Fatalf("kill: %v", r.killErr)
	}
	if b.Outcome != OutcomeFailed || b.Termination.Signal != "SIGKILL" || b.ArtifactDigest != "" {
		t.Fatalf("the killed attempt must be a failure with the signal the OS reported: %+v", b)
	}
	if b.PartialOutput != nil && !b.PartialOutput.Quarantined {
		t.Fatalf("a partial output must be moved aside: %+v", b.PartialOutput)
	}
	ids := map[string]bool{a.OccurrenceID: true, b.OccurrenceID: true, c.OccurrenceID: true}
	if len(ids) != 3 || r.host.IssuedCount() != 3 {
		t.Fatalf("each attempt needs its own occurrence: %d distinct, %d issued", len(ids), r.host.IssuedCount())
	}
	if r.rep.Occurrences[0] != a.OccurrenceID || r.rep.Occurrences[1] != c.OccurrenceID {
		t.Fatalf("reproducibility must use A and C only: %+v", r.rep.Occurrences)
	}
	if err := r.history.Append(r.host, a); !errors.Is(err, ErrOccurrenceCounted) {
		t.Fatalf("a counted attempt must not be counted again: %v", err)
	}
	observed(t, "concurrent-attempts", map[string]any{
		"A":                          map[string]any{"occurrence": a.OccurrenceID, "ordinal": a.Ordinal, "outcome": a.Outcome, "digest": a.ArtifactDigest},
		"B":                          map[string]any{"occurrence": b.OccurrenceID, "ordinal": b.Ordinal, "outcome": b.Outcome, "termination": b.Termination, "partialOutput": b.PartialOutput},
		"C":                          map[string]any{"occurrence": c.OccurrenceID, "ordinal": c.Ordinal, "outcome": c.Outcome, "digest": c.ArtifactDigest},
		"reproducibilityOccurrences": r.rep.Occurrences,
	})
}

func TestPJOB01FailuresAndMismatchesDoNotYieldReproducibility(t *testing.T) {
	r := throughFixture(t)
	env := requireEnvironment(t)
	ctx := context.Background()
	host := NewHost(r.host.Executable, filepath.Join(env.work, "attempts-pjob"))
	base := AttemptSpec{Plan: r.s1.Plan, Lock: r.s1.Lock, Repo: env.repo, GoPath: r.toolchain.Executable}

	partialSpec := base
	partialSpec.Mode = ModePartial
	partial, err := host.Run(ctx, partialSpec)
	if err != nil {
		t.Fatal(err)
	}
	if partial.Outcome != OutcomeFailed || partial.Termination.ExitCode != 2 || partial.PartialOutput == nil || !partial.PartialOutput.Quarantined {
		t.Fatalf("a failed build with a partial output: %+v", partial)
	}
	h := History{PlanFingerprint: r.s1.PlanFingerprint}
	if err := h.Append(host, partial); err != nil {
		t.Fatal(err)
	}
	_, errPartial := DeriveReproducibility(host, h)
	if !errors.Is(errPartial, ErrInsufficientOccurrences) {
		t.Fatalf("a failure must not give reproducibility: %v", errPartial)
	}

	mismatchSpec := base
	mismatchSpec.Mode = ModeNondeterministic
	h2 := History{PlanFingerprint: r.s1.PlanFingerprint}
	var digests []string
	for i := 0; i < 2; i++ {
		rec, err := host.Run(ctx, mismatchSpec)
		if err != nil {
			t.Fatal(err)
		}
		if err := h2.Append(host, rec); err != nil {
			t.Fatal(err)
		}
		digests = append(digests, rec.ArtifactDigest)
	}
	mismatch, errMismatch := DeriveReproducibility(host, h2)
	if !errors.Is(errMismatch, ErrNotReproducible) || mismatch.Reproducible || mismatch.ArtifactDigest != "" {
		t.Fatalf("two different outputs must be a mismatch: %+v %v", mismatch, errMismatch)
	}
	out, _ := h2.Attempts[0].OutputOf()
	bytesOut, _ := os.ReadFile(out)
	if _, err := r.store.PutReproducible(mismatch, bytesOut); !errors.Is(err, ErrNotStorable) {
		t.Fatalf("a mismatched artifact must not be stored: %v", err)
	}

	// Claiming the single attempt A twice.
	a := r.attempts["A"]
	copyA := a
	copyA.Ordinal = a.Ordinal + 100
	copyA.Fingerprint, _ = attemptFingerprint(copyA)
	h3 := History{PlanFingerprint: r.s1.PlanFingerprint}
	if err := h3.Append(r.host, a); err != nil {
		t.Fatal(err)
	}
	errCopy := h3.Append(r.host, copyA)
	_, errTwice := DeriveReproducibility(r.host, History{PlanFingerprint: r.s1.PlanFingerprint, Attempts: []AttemptRecord{a, a}})
	_, errOnce := DeriveReproducibility(r.host, h3)
	if !errors.Is(errCopy, ErrNotIssued) || !errors.Is(errTwice, ErrOccurrenceCounted) || !errors.Is(errOnce, ErrInsufficientOccurrences) {
		t.Fatalf("one execution claimed twice: copy=%v twice=%v once=%v", errCopy, errTwice, errOnce)
	}
	observed(t, "failures-and-mismatches", map[string]any{
		"partial":       map[string]any{"termination": partial.Termination, "partialOutput": partial.PartialOutput, "derive": errPartial.Error()},
		"mismatch":      map[string]any{"digests": digests, "reproducible": mismatch.Reproducible, "derive": errMismatch.Error()},
		"claimedTwice":  map[string]any{"copyWithNewOrdinal": errCopy.Error(), "listedTwice": errTwice.Error(), "single": errOnce.Error()},
		"simulatedMode": "partial and mismatch use simulated builders; the main flow uses the real go build",
	})
}

func TestPGOV01ResolutionStopsAndRevocationRefusesUse(t *testing.T) {
	r := throughFixture(t)
	env := requireEnvironment(t)
	seed := r.catalog[0]
	alt := seed
	alt.ID, alt.Implementation = "tei.builder.go.alternative", "go.alternative"
	alt.ArtifactDigest = sha256Digest([]byte("simulated alternative Go builder"))
	state := filepath.Join(env.work, "pgov-state")

	_, errAB := Derive(env.repo, r.intent, DeriveOptions{Toolchain: r.toolchain, Catalog: []Provider{seed, alt}, StateDir: state})
	_, errBA := Derive(env.repo, r.intent, DeriveOptions{Toolchain: r.toolchain, Catalog: []Provider{alt, seed}, StateDir: state})
	if !errors.Is(errAB, ErrAmbiguousProvider) || errAB.Error() != errBA.Error() {
		t.Fatalf("two providers must stop the derivation whatever the order: %v / %v", errAB, errBA)
	}
	codeAB, _, stderrAB := runAuthoringIntent(t, env, r.intentPath, []Provider{seed, alt}, r.intent, "ambiguous-ab")
	codeBA, _, stderrBA := runAuthoringIntent(t, env, r.intentPath, []Provider{alt, seed}, r.intent, "ambiguous-ba")
	if codeAB == 0 || codeBA == 0 || !strings.Contains(stderrAB, "ambiguous") || stderrAB != stderrBA {
		t.Fatalf("authoring-intent CLI must stop on ambiguity in both orders: %d %q / %d %q", codeAB, stderrAB, codeBA, stderrBA)
	}

	replayed, err := Derive(env.repo, r.intent, DeriveOptions{Toolchain: r.toolchain, Catalog: []Provider{alt, seed}, Lock: &r.s1.Lock, StateDir: state})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(r.s1.Lock)
	after, _ := json.Marshal(replayed.Lock)
	if string(before) != string(after) || replayed.PlanFingerprint != r.s1.PlanFingerprint {
		t.Fatalf("the added provider changed the fixed record or the plan")
	}

	trust := TrustStore{Revoked: []Revocation{{ProviderID: seed.ID, ArtifactDigest: seed.ArtifactDigest, Reason: "injected revocation"}}}
	reproduced, errReproduce := Derive(env.repo, r.intent, DeriveOptions{Toolchain: r.toolchain, Catalog: []Provider{seed}, Lock: &r.s1.Lock, StateDir: state})
	host := NewHost(r.host.Executable, filepath.Join(env.work, "attempts-revoked"))
	_, errLaunch := host.Run(context.Background(), AttemptSpec{Plan: r.s1.Plan, Lock: r.s1.Lock, Trust: trust, Repo: env.repo, GoPath: r.toolchain.Executable, Mode: ModeBuild})
	_, errBind := BindStage3(r.s1, r.rep, *r.runner.Evidence, trust)
	if errReproduce != nil || reproduced.PlanFingerprint != r.s1.PlanFingerprint {
		t.Fatalf("reproduction after revocation: %v", errReproduce)
	}
	if !errors.Is(errLaunch, ErrBuilderRevoked) || host.IssuedCount() != 0 || !errors.Is(errBind, ErrBuilderRevoked) {
		t.Fatalf("use after revocation must be refused: launch=%v issued=%d bind=%v", errLaunch, host.IssuedCount(), errBind)
	}
	observed(t, "builder-resolution", map[string]any{
		"ambiguous":         errAB.Error(),
		"ambiguousCLI":      strings.TrimSpace(stderrAB),
		"lockUnchanged":     string(before) == string(after),
		"planAfterAdded":    replayed.PlanFingerprint,
		"reproducedPlan":    reproduced.PlanFingerprint,
		"launchRevoked":     errLaunch.Error(),
		"bindRevoked":       errBind.Error(),
		"occurrencesIssued": host.IssuedCount(),
	})
}

func TestClosureThatCannotBeFixedStops(t *testing.T) {
	r := throughFixture(t)
	env := requireEnvironment(t)
	ctx := context.Background()
	derive := func(repo string, tc GoToolchain, name string) (Stage1, error) {
		return Derive(repo, r.intent, DeriveOptions{
			Toolchain: tc, Catalog: []Provider{GoSeedProvider(tc, r.intent.Target, r.intent.Profile)},
			StateDir: filepath.Join(env.work, "closure-state-"+name),
		})
	}
	results := map[string]any{}

	dirty := copyRepository(t, env, "dirty", func(dir string) {
		if err := os.WriteFile(filepath.Join(dir, "tools", "reference-process-plugin", "untracked.go"), []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}, false)
	_, errDirty := derive(dirty, r.toolchain, "dirty")
	dirtyRun, err := RunSelfBuild(ctx, env.tools["self-build"], dirty, r.toolchain.Executable, os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	if errDirty == nil || dirtyRun.ExitCode == 0 || !strings.Contains(dirtyRun.Stderr, "clean") {
		t.Fatalf("a dirty working tree must stop stage 1 and the runner: %v / exit %d %s", errDirty, dirtyRun.ExitCode, dirtyRun.Stderr)
	}
	results["dirtyTree"] = map[string]any{"stage1": errDirty.Error(), "runnerExit": dirtyRun.ExitCode, "runner": lastLine(dirtyRun.Stderr)}

	external := copyRepository(t, env, "external-module", func(dir string) {
		main := filepath.Join(dir, "tools", "reference-process-plugin", "main.go")
		raw, err := os.ReadFile(main)
		if err != nil {
			t.Fatal(err)
		}
		edited := strings.Replace(string(raw), "import (\n", "import (\n\t_ \"example.invalid/teicache\"\n", 1)
		if err := os.WriteFile(main, []byte(edited), 0o644); err != nil {
			t.Fatal(err)
		}
		mod, err := os.OpenFile(filepath.Join(dir, "go.mod"), os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		mod.WriteString("\nrequire example.invalid/teicache v1.0.0\n")
		mod.Close()
	}, true)
	_, errExternal := derive(external, r.toolchain, "external")
	externalRun, err := RunSelfBuild(ctx, env.tools["self-build"], external, r.toolchain.Executable, os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	if errExternal == nil || externalRun.ExitCode == 0 || externalRun.Evidence != nil {
		t.Fatalf("an external module must stop stage 1 and the runner: %v / exit %d", errExternal, externalRun.ExitCode)
	}
	results["externalModule"] = map[string]any{"stage1": errExternal.Error(), "runnerExit": externalRun.ExitCode, "runner": lastLine(externalRun.Stderr)}

	changed := copyRepository(t, env, "source-changed", func(dir string) {
		main := filepath.Join(dir, "tools", "reference-process-plugin", "main.go")
		f, err := os.OpenFile(main, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		f.WriteString("\n// edited after the plan was confirmed\n")
		f.Close()
	}, true)
	sChanged, err := derive(changed, r.toolchain, "changed")
	if err != nil {
		t.Fatal(err)
	}
	_, errSource := BindStage3(sChanged, r.rep, *r.runner.Evidence, TrustStore{})
	if !errors.Is(errSource, ErrBindingMismatch) || !strings.Contains(errSource.Error(), "source closure") {
		t.Fatalf("a source version mismatch must stop the binding: %v", errSource)
	}
	results["sourceMismatch"] = map[string]any{"stage1Plan": sChanged.PlanFingerprint, "bind": errSource.Error()}

	wrapper := filepath.Join(env.work, "go-wrapper")
	script := "#!/bin/sh\nexec '" + r.toolchain.Executable + "' \"$@\"\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	wrapped, err := ObserveGoToolchain(wrapper)
	if err != nil {
		t.Fatal(err)
	}
	sWrapped, err := derive(env.repo, wrapped, "wrapped")
	if err != nil {
		t.Fatal(err)
	}
	_, errToolchain := BindStage3(sWrapped, r.rep, *r.runner.Evidence, TrustStore{})
	if wrapped.Version != r.toolchain.Version || wrapped.ToolchainDigest == r.toolchain.ToolchainDigest ||
		!errors.Is(errToolchain, ErrBindingMismatch) || !strings.Contains(errToolchain.Error(), "toolchain") {
		t.Fatalf("a toolchain identity mismatch must stop the binding: %v", errToolchain)
	}
	results["toolchainMismatch"] = map[string]any{"sameGoVersion": wrapped.Version, "planToolchain": wrapped.ToolchainDigest, "runnerToolchain": r.runner.Evidence.GoToolchainDigest, "bind": errToolchain.Error()}

	for key, empty := range r.poisonEmpty {
		if !empty {
			t.Fatalf("the runner wrote into the host %s", key)
		}
	}
	if r.s3.FirstArtifactDigest != r.rep.ArtifactDigest {
		t.Fatalf("host Go settings changed the runner's output")
	}
	results["implicitCache"] = map[string]any{"hostSettings": "GOFLAGS=-mod=mod GOCACHE GOMODCACHE GOPATH set to separate directories", "directoriesStayedEmpty": r.poisonEmpty,
		"runnerDigest": r.s3.FirstArtifactDigest, "stage2DigestWithoutThem": r.rep.ArtifactDigest}
	observed(t, "closure", results)
}

func TestThroughDetectsSwappedIdentityAndMissingStage(t *testing.T) {
	r := throughFixture(t)
	env := requireEnvironment(t)

	// Each stage asset's standalone check passes on its own fixed input,
	// whatever this run's identities are.
	standalone := map[string]any{}
	for _, name := range []string{"contract-resolution", "attempt-evidence", "artifact-store"} {
		code, stdout, _ := runTool(t, env.tools[name])
		var doc map[string]any
		_ = json.Unmarshal([]byte(stdout), &doc)
		standalone[name] = map[string]any{"exitCode": code, "schema": doc["schema"]}
	}

	other := r.attempts["A"].ArtifactDigest
	other = sha256Digest([]byte(other + "swapped"))
	swapped := append([]stageInput(nil), r.inputs...)
	swapped[2].artifacts = map[string]string{"first-build": other, "second-build": other, "registered": other}
	resSwap := VerifyThrough(buildChain(t, swapped))
	if resSwap.Complete || len(resSwap.SelfChecksPassed) != 4 || !strings.Contains(strings.Join(resSwap.Mismatches, "\n"), "stage 3") {
		t.Fatalf("a swapped artifact identity must fail only the through check: %+v", resSwap)
	}
	planSwap := append([]stageInput(nil), r.inputs...)
	planSwap[1].plan = strings.Repeat("0", 64)
	resPlan := VerifyThrough(buildChain(t, planSwap))
	if resPlan.Complete || len(resPlan.SelfChecksPassed) != 4 {
		t.Fatalf("a swapped plan fingerprint must fail the through check: %+v", resPlan)
	}
	var missingInputs []stageInput
	for _, in := range r.inputs {
		if in.stage != 2 {
			missingInputs = append(missingInputs, in)
		}
	}
	resMissing := VerifyThrough(buildChain(t, missingInputs))
	if resMissing.Complete || len(resMissing.MissingStages) != 1 || resMissing.MissingStages[0] != 2 {
		t.Fatalf("a missing stage must be named: %+v", resMissing)
	}
	observed(t, "negative-control", map[string]any{
		"standaloneAssetChecks": standalone,
		"swappedArtifact":       resSwap, "swappedPlan": resPlan, "missingStage2": resMissing,
	})
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}
