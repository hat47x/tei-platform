package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"tei/internal/application"
	apprelease "tei/internal/application/release"
	"tei/internal/consumerimpact"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/migration"
	"tei/internal/database/physical"
	"tei/internal/database/plan"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/realization"
	"tei/internal/database/reconcile"
	"tei/internal/database/release"
	"tei/internal/database/schemadiff"
	"tei/internal/dependency"
	"tei/internal/deployment"
	deploymentdurable "tei/internal/deployment/durable"
	deploymentevidence "tei/internal/deployment/evidence"
	deploymentexec "tei/internal/deployment/execution"
	"tei/internal/deployment/executor"
	deploymentobservation "tei/internal/deployment/observation"
	deploymentrouting "tei/internal/deployment/routing"
	"tei/internal/environment"
	"tei/internal/index"
	"tei/internal/ir"
	"tei/internal/plugin"
	pluginruntime "tei/internal/plugin/runtime"
	"tei/internal/query/analyzer"
	"tei/internal/releasegraph"
	"tei/internal/releaseplan"
	"tei/internal/validation"
	databaseworkbench "tei/internal/workbench/database"
	"tei/internal/workspace"
)

type providerRoute struct {
	Kind     string           `json:"kind"`
	Provider core.CanonicalID `json:"provider"`
	ActionID string           `json:"actionId,omitempty"`
}

type deploymentProviderPlan struct {
	Executors     []deploymentrouting.RoutedCommand `json:"executors,omitempty"`
	Observations  []providerRoute                   `json:"observations,omitempty"`
	Evidence      []providerRoute                   `json:"evidence,omitempty"`
	PluginRuntime *pluginruntime.EnvironmentPlan    `json:"pluginRuntime,omitempty"`
}

type recoveryRunBundle struct {
	Parent      deploymentdurable.Run `json:"parent"`
	RecoveryRun deploymentdurable.Run `json:"recoveryRun"`
}

type pathList []string

func (p *pathList) String() string { return fmt.Sprint([]string(*p)) }
func (p *pathList) Set(v string) error {
	*p = append(*p, v)
	return nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: tei <workspace-files|source-index|knowledge-review-plan|knowledge-context|knowledge-qualify|validate|index|plugin-lock|plugin-resolve|plugin-runtime-plan|observed-index|workbench-model|world|requirements|normalize-postgres-plan|desired-schema|schema-diff|plan-migration|migration-fingerprint|database-reconcile|database-compatibility|application-release-compatibility|release-graph|solution-release-graph|consumer-impact|deployment-simulate|deployment-start|deployment-ready-commands|deployment-record-attempt|deployment-retry|deployment-pause|deployment-resume|deployment-abort|deployment-recovery|deployment-start-recovery|deployment-provider-plan|deployment-reconcile> [options]")
	}
	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	worldPath := fs.String("world", "", "normalized M0 IR JSON")
	workspacePath := fs.String("workspace", "", "TEI workspace directory")
	subjectKind := fs.String("subject-kind", "", "Source Index subject kind")
	subjectID := fs.String("subject-id", "", "Source Index subject ID")
	expectedDataDesignFingerprint := fs.String("expected-data-design-fingerprint", "", "expected Data Design Context fingerprint; knowledge-context fails on mismatch")
	expectedReviewPlanFingerprint := fs.String("expected-review-plan-fingerprint", "", "expected Builder Review Plan fingerprint; knowledge-review-plan fails on mismatch")
	knowledgeDisclosure := fs.String("disclose", "", "comma-separated Knowledge sections to disclose: source,definition,dependencies,data-design")
	expectedDisclosureFingerprint := fs.String("expected-disclosure-fingerprint", "", "expected selective Knowledge Disclosure fingerprint; requires -disclose")
	qualificationRequirementsPath := fs.String("qualification-requirements", "", "knowledge qualification requirements JSON")
	var analysisPaths pathList
	var schemaPaths pathList
	var planPaths pathList
	var evidencePaths pathList
	fs.Var(&analysisPaths, "analysis", "SQL analyzer result JSON; repeatable")
	fs.Var(&schemaPaths, "schema", "physical schema snapshot JSON; repeatable")
	fs.Var(&planPaths, "plan", "normalized execution plan JSON; repeatable")
	fs.Var(&evidencePaths, "evidence", "deployment evidence JSON; repeatable")
	queryID := fs.String("query", "", "query canonical ID (plan normalization)")
	databaseID := fs.String("database", "", "database canonical ID (plan normalization)")
	explainPath := fs.String("explain", "", "raw PostgreSQL EXPLAIN FORMAT JSON")
	desiredPath := fs.String("desired", "", "desired physical schema JSON")
	actualPath := fs.String("actual", "", "actual physical schema snapshot JSON")
	providerName := fs.String("provider", "", "database provider")
	migrationVersion := fs.String("version", "", "migration version/id")
	applicationID := fs.String("application", "", "application canonical ID")
	environmentID := fs.String("environment", "", "environment canonical ID")
	solutionID := fs.String("solution", "", "solution release canonical ID")
	historyPath := fs.String("history", "", "normalized migration history JSON")
	deploymentPath := fs.String("deployment", "", "observed deployment state JSON")
	runPath := fs.String("run", "", "durable deployment run JSON")
	pluginLockPath := fs.String("plugin-lock", "", "resolved TEI plugin lock JSON")
	targetPluginLockPath := fs.String("target-plugin-lock", "", "plugin lock for a new recovery run; defaults to -plugin-lock")
	runID := fs.String("run-id", "", "durable deployment run ID")
	attemptPath := fs.String("attempt", "", "deployment executor result JSON")
	actionID := fs.String("action", "", "durable deployment action ID")
	reason := fs.String("reason", "", "operator reason")
	recoveryStrategy := fs.String("strategy", "", "recovery strategy: retry|forward-fix|rollback|manual")
	targetSolutionID := fs.String("target-solution", "", "recovery target solution release ID")
	targetSolutionVersion := fs.String("target-version", "", "recovery target solution release version")
	expectedRevision := fs.Int("expected-revision", -1, "expected durable deployment run revision for optimistic concurrency")
	var statePaths pathList
	fs.Var(&statePaths, "state", "database compatibility state JSON; repeatable")
	_ = fs.Parse(os.Args[2:])
	if command == "workspace-files" {
		if *workspacePath == "" || *worldPath != "" {
			fmt.Fprintln(os.Stderr, "workspace-files requires -workspace and does not accept -world")
			os.Exit(2)
		}
		r, err := workspace.Load(*workspacePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		writeJSON(r.ManagedFiles)
		return
	}
	if command == "source-index" {
		if *workspacePath == "" || *worldPath != "" {
			fmt.Fprintln(os.Stderr, "source-index requires -workspace and does not accept -world")
			os.Exit(2)
		}
		inspection, err := buildSourceIndexInspection(*workspacePath, *subjectKind, *subjectID)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		writeJSON(inspection)
		return
	}
	if command == "knowledge-review-plan" {
		if *workspacePath == "" || *worldPath != "" {
			fmt.Fprintln(os.Stderr, "knowledge-review-plan requires -workspace and does not accept -world")
			os.Exit(2)
		}
		plan, err := buildBuilderReviewPlan(*workspacePath, *subjectKind, *subjectID)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if err := checkBuilderReviewPlanFingerprint(plan, *expectedReviewPlanFingerprint); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		writeJSON(plan)
		return
	}
	if command == "knowledge-context" {
		if *workspacePath == "" || *worldPath != "" {
			fmt.Fprintln(os.Stderr, "knowledge-context requires -workspace and does not accept -world")
			os.Exit(2)
		}
		context, err := buildKnowledgeContext(*workspacePath, *subjectKind, *subjectID)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if err := checkDataDesignFingerprint(context, *expectedDataDesignFingerprint); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if strings.TrimSpace(*knowledgeDisclosure) != "" {
			disclosure, err := buildKnowledgeDisclosure(context, *knowledgeDisclosure)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			if err := checkKnowledgeDisclosureFingerprint(disclosure, *expectedDisclosureFingerprint); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			writeJSON(disclosure)
			return
		}
		if strings.TrimSpace(*expectedDisclosureFingerprint) != "" {
			fmt.Fprintln(os.Stderr, "-expected-disclosure-fingerprint requires -disclose")
			os.Exit(2)
		}
		writeJSON(context)
		return
	}
	if command == "knowledge-qualify" {
		if *workspacePath == "" || *worldPath != "" {
			fmt.Fprintln(os.Stderr, "knowledge-qualify requires -workspace and does not accept -world")
			os.Exit(2)
		}
		qualification, err := buildKnowledgeQualificationFromFile(
			*workspacePath,
			*subjectKind,
			*subjectID,
			*qualificationRequirementsPath,
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		writeJSON(qualification)
		return
	}
	if command == "normalize-postgres-plan" {
		if *queryID == "" || *databaseID == "" || *explainPath == "" {
			fmt.Fprintln(os.Stderr, "-query, -database and -explain are required")
			os.Exit(2)
		}
		raw, err := os.ReadFile(*explainPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		var snap *physical.SchemaSnapshot
		if len(schemaPaths) > 0 {
			var s physical.SchemaSnapshot
			if err := decodeJSONFile(schemaPaths[0], &s); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			s = s.Normalize()
			snap = &s
		}
		p, err := postgres.NormalizeExplainJSON(core.MustCanonicalID(*queryID), core.MustCanonicalID(*databaseID), raw, snap)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		writeJSON(p)
		return
	}
	if command == "schema-diff" {
		if *desiredPath == "" || *actualPath == "" {
			fmt.Fprintln(os.Stderr, "-desired and -actual are required")
			os.Exit(2)
		}
		var desired, actual physical.SchemaSnapshot
		if err := decodeJSONFile(*desiredPath, &desired); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if err := decodeJSONFile(*actualPath, &actual); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		d, err := schemadiff.Compare(desired.Normalize(), actual.Normalize())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		writeJSON(d)
		return
	}
	if (*worldPath == "") == (*workspacePath == "") {
		fmt.Fprintln(os.Stderr, "exactly one of -workspace or -world is required")
		os.Exit(2)
	}

	if command == "validate" && *workspacePath != "" {
		code, err := validateWorkspace(os.Stdout, *workspacePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if code != 0 {
			os.Exit(code)
		}
		return
	}

	w, err := loadWorld(*worldPath, *workspacePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	switch command {
	case "world":
		writeJSON(w)
	case "requirements":
		validateOrExit(w)
		writeJSON(dependency.Derive(w))
	case "plugin-lock":
		validateOrExit(w)
		lock, err := plugin.BuildLock(plugin.Catalog{Plugins: w.Plugins, Bundles: w.Bundles}, w.Environments)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		writeJSON(lock)
	case "plugin-resolve":
		validateOrExit(w)
		if *environmentID == "" {
			fmt.Fprintln(os.Stderr, "-environment is required")
			os.Exit(2)
		}
		env, ok := findEnvironment(w, core.MustCanonicalID(*environmentID))
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown environment %s\n", *environmentID)
			os.Exit(2)
		}
		catalog := plugin.Catalog{Plugins: w.Plugins, Bundles: w.Bundles}
		var r plugin.ResolvedSet
		var err error
		if *pluginLockPath != "" {
			var lock plugin.Lock
			if err := decodeJSONFile(*pluginLockPath, &lock); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			r, err = catalog.ResolveEnvironmentWithLock(env, lock)
		} else {
			r, err = catalog.ResolveEnvironment(env)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		writeJSON(r)
	case "plugin-runtime-plan":
		validateOrExit(w)
		if *environmentID == "" || *pluginLockPath == "" {
			fmt.Fprintln(os.Stderr, "-environment and -plugin-lock are required")
			os.Exit(2)
		}
		env, ok := findEnvironment(w, core.MustCanonicalID(*environmentID))
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown environment %s\n", *environmentID)
			os.Exit(2)
		}
		var lock plugin.Lock
		if err := decodeJSONFile(*pluginLockPath, &lock); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		plan, err := pluginruntime.BuildPlan(plugin.Catalog{Plugins: w.Plugins, Bundles: w.Bundles}, env, lock)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		writeJSON(plan)
	case "validate":
		ds := validation.Validate(w)
		hasErr := false
		for _, d := range ds {
			fmt.Printf("%s\t%s\t%s\n", d.Severity, d.Code, d.Message)
			if d.Severity == validation.Error {
				hasErr = true
			}
		}
		if hasErr {
			os.Exit(1)
		}
		fmt.Println("ok")
	case "observed-index", "workbench-model":
		validateOrExit(w)
		analyses, snapshots, plans, err := loadObservations(analysisPaths, schemaPaths, planPaths)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		d := index.BuildWithAllObservations(w, analyses, snapshots, plans)
		if command == "workbench-model" {
			writeJSON(databaseworkbench.Build(d))
		} else {
			writeJSON(d)
		}
	case "desired-schema":
		validateOrExit(w)
		if *databaseID == "" || *providerName == "" {
			fmt.Fprintln(os.Stderr, "-database and -provider are required")
			os.Exit(2)
		}
		pid := environment.DatabaseProvider(*providerName)
		if !pid.Known() || pid == environment.ProviderGeneric {
			fmt.Fprintf(os.Stderr, "unsupported concrete database provider %q\n", pid)
			os.Exit(2)
		}
		if _, ok := findDatabase(w, core.MustCanonicalID(*databaseID)); !ok {
			fmt.Fprintf(os.Stderr, "unknown database %s\n", *databaseID)
			os.Exit(2)
		}
		snap, err := realization.RealizeDesired(w.Persistence, *databaseID, pid)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		writeJSON(snap)
	case "plan-migration":
		validateOrExit(w)
		if *databaseID == "" || *providerName == "" || *actualPath == "" || *migrationVersion == "" {
			fmt.Fprintln(os.Stderr, "-database, -provider, -actual and -version are required")
			os.Exit(2)
		}
		dbid := core.MustCanonicalID(*databaseID)
		db, ok := findDatabase(w, dbid)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown database %s\n", dbid)
			os.Exit(2)
		}
		if db.Ownership != database.OwnershipManagedHere {
			fmt.Fprintf(os.Stderr, "database %s is %s; TEI does not own its migration lifecycle\n", dbid, db.Ownership)
			os.Exit(2)
		}
		pid := environment.DatabaseProvider(*providerName)
		if !pid.Known() || pid == environment.ProviderGeneric {
			fmt.Fprintf(os.Stderr, "unsupported concrete database provider %q\n", pid)
			os.Exit(2)
		}
		desired, err := realization.RealizeDesired(w.Persistence, string(dbid), pid)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		var actual physical.SchemaSnapshot
		if err := decodeJSONFile(*actualPath, &actual); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		diff, err := schemadiff.Compare(desired.Normalize(), actual.Normalize())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		m, err := migration.Plan(*migrationVersion, dbid, actual.Fingerprint, diff, pid)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		writeJSON(m)
	case "migration-fingerprint":
		validateOrExit(w)
		if *databaseID == "" || *migrationVersion == "" {
			fmt.Fprintln(os.Stderr, "-database and -version are required")
			os.Exit(2)
		}
		dbid := core.MustCanonicalID(*databaseID)
		m, ok := findMigration(w, dbid, *migrationVersion)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown migration %s for database %s\n", *migrationVersion, dbid)
			os.Exit(2)
		}
		fp, err := migration.Fingerprint(m)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Println(fp)
	case "database-reconcile":
		validateOrExit(w)
		if *databaseID == "" || *providerName == "" || *actualPath == "" || *historyPath == "" {
			fmt.Fprintln(os.Stderr, "-database, -provider, -actual and -history are required")
			os.Exit(2)
		}
		dbid := core.MustCanonicalID(*databaseID)
		pid := environment.DatabaseProvider(*providerName)
		if !pid.Known() || pid == environment.ProviderGeneric {
			fmt.Fprintf(os.Stderr, "unsupported concrete database provider %q\n", pid)
			os.Exit(2)
		}
		desired, err := realization.RealizeDesired(w.Persistence, string(dbid), pid)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		var actual physical.SchemaSnapshot
		if err := decodeJSONFile(*actualPath, &actual); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		diff, err := schemadiff.Compare(desired.Normalize(), actual.Normalize())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		var h migration.HistorySnapshot
		if err := decodeJSONFile(*historyPath, &h); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		hc, err := migration.CompareHistory(dbid, migrationsFor(w, dbid), h)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		writeJSON(reconcile.Build(dbid, hc, diff))
	case "database-compatibility":
		validateOrExit(w)
		if *applicationID == "" {
			fmt.Fprintln(os.Stderr, "-application is required")
			os.Exit(2)
		}
		app, ok := findApplication(w, core.MustCanonicalID(*applicationID))
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown application %s\n", *applicationID)
			os.Exit(2)
		}
		states := uniqueManagedReleaseStates(w, app)
		for _, path := range statePaths {
			var st release.StateSnapshot
			if err := decodeJSONFile(path, &st); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			states = upsertState(states, st)
		}
		if *deploymentPath != "" {
			var dep deployment.Snapshot
			if err := decodeJSONFile(*deploymentPath, &dep); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			validateDeploymentOrExit(w, dep)
			for _, st := range dep.Databases {
				states = upsertState(states, st)
			}
		}
		writeJSON(release.CheckApplication(app, states))
	case "application-release-compatibility", "release-graph":
		validateOrExit(w)
		if *applicationID == "" || *migrationVersion == "" {
			fmt.Fprintln(os.Stderr, "-application and -version are required")
			os.Exit(2)
		}
		ar, ok := findApplicationRelease(w, core.MustCanonicalID(*applicationID), *migrationVersion)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown application release %s@%s\n", *applicationID, *migrationVersion)
			os.Exit(2)
		}
		states := uniqueManagedReleaseStatesForRelease(w, ar)
		for _, path := range statePaths {
			var st release.StateSnapshot
			if err := decodeJSONFile(path, &st); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			states = upsertState(states, st)
		}
		if *deploymentPath != "" {
			var dep deployment.Snapshot
			if err := decodeJSONFile(*deploymentPath, &dep); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			validateDeploymentOrExit(w, dep)
			for _, st := range dep.Databases {
				states = upsertState(states, st)
			}
		}
		if command == "release-graph" {
			writeJSON(releasegraph.Build(ar, states))
		} else {
			writeJSON(release.CheckApplication(ar.AsApplication(), states))
		}
	case "solution-release-graph":
		validateOrExit(w)
		if *solutionID == "" || *migrationVersion == "" {
			fmt.Fprintln(os.Stderr, "-solution and -version are required")
			os.Exit(2)
		}
		sr, ok := findSolutionRelease(w, core.MustCanonicalID(*solutionID), *migrationVersion)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown solution release %s@%s\n", *solutionID, *migrationVersion)
			os.Exit(2)
		}
		if *deploymentPath != "" {
			var state deployment.Snapshot
			if err := decodeJSONFile(*deploymentPath, &state); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			validateDeploymentOrExit(w, state)
			writeJSON(releaseplan.CompileWithDeployment(sr, w.ApplicationReleases, w.DatabaseReleases, state))
		} else {
			writeJSON(releaseplan.Compile(sr, w.ApplicationReleases, w.DatabaseReleases))
		}
	case "deployment-start":
		validateOrExit(w)
		if *solutionID == "" || *migrationVersion == "" || *deploymentPath == "" || *runID == "" || *pluginLockPath == "" {
			fmt.Fprintln(os.Stderr, "-solution, -version, -deployment, -run-id and -plugin-lock are required")
			os.Exit(2)
		}
		sr, ok := findSolutionRelease(w, core.MustCanonicalID(*solutionID), *migrationVersion)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown solution release %s@%s\n", *solutionID, *migrationVersion)
			os.Exit(2)
		}
		var state deployment.Snapshot
		if err := decodeJSONFile(*deploymentPath, &state); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		validateDeploymentOrExit(w, state)
		compiled := releaseplan.CompileWithDeployment(sr, w.ApplicationReleases, w.DatabaseReleases, state)
		lockFP, policyFP, err := runtimeFingerprintsForEnvironment(w, *pluginLockPath, state.Environment)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		writeJSON(deploymentdurable.StartWithRuntimePolicy(*runID, sr, compiled, state, lockFP, policyFP))

	case "deployment-ready-commands":
		validateOrExit(w)
		run, _, _, err := loadCheckedDeploymentRun(w, *runPath, *solutionID, *migrationVersion, *expectedRevision, *pluginLockPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		writeJSON(deploymentdurable.ReadyCommands(run))
	case "deployment-record-attempt":
		validateOrExit(w)
		if *attemptPath == "" {
			fmt.Fprintln(os.Stderr, "-attempt is required")
			os.Exit(2)
		}
		run, _, _, err := loadCheckedDeploymentRun(w, *runPath, *solutionID, *migrationVersion, *expectedRevision, *pluginLockPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var result executor.Result
		if err := decodeJSONFile(*attemptPath, &result); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		env, ok := findEnvironment(w, run.Environment)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown environment %s\n", run.Environment)
			os.Exit(1)
		}
		rec, ok := findRunAction(run, result.ActionID)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown deployment action %s\n", result.ActionID)
			os.Exit(1)
		}
		cmd, ok := deploymentdurable.CommandForAction(rec)
		if !ok {
			fmt.Fprintf(os.Stderr, "deployment action %s is not a command\n", result.ActionID)
			os.Exit(1)
		}
		if err := deploymentrouting.ValidateResult(env, cmd, result); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if result.PluginInvocation != nil {
			var lock plugin.Lock
			if err := decodeJSONFile(*pluginLockPath, &lock); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			runtimePlan, err := pluginruntime.BuildPlan(plugin.Catalog{Plugins: w.Plugins, Bundles: w.Bundles}, env, lock)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if err := pluginruntime.ValidateDeploymentInvocation(runtimePlan, core.CanonicalID(result.ExecutorID), result.PluginInvocation); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		writeJSON(deploymentdurable.RecordExecutorResult(run, result))
	case "deployment-retry":
		validateOrExit(w)
		if *actionID == "" {
			fmt.Fprintln(os.Stderr, "-action is required")
			os.Exit(2)
		}
		run, _, _, err := loadCheckedDeploymentRun(w, *runPath, *solutionID, *migrationVersion, *expectedRevision, *pluginLockPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		writeJSON(deploymentdurable.Retry(run, *actionID, *reason))
	case "deployment-pause":
		validateOrExit(w)
		run, _, _, err := loadCheckedDeploymentRun(w, *runPath, *solutionID, *migrationVersion, *expectedRevision, *pluginLockPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		writeJSON(deploymentdurable.Pause(run, *reason))
	case "deployment-resume":
		validateOrExit(w)
		run, _, _, err := loadCheckedDeploymentRun(w, *runPath, *solutionID, *migrationVersion, *expectedRevision, *pluginLockPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		writeJSON(deploymentdurable.Resume(run))
	case "deployment-abort":
		validateOrExit(w)
		run, _, _, err := loadCheckedDeploymentRun(w, *runPath, *solutionID, *migrationVersion, *expectedRevision, *pluginLockPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		writeJSON(deploymentdurable.Abort(run, *reason))
	case "deployment-recovery":
		validateOrExit(w)
		run, _, _, err := loadCheckedDeploymentRun(w, *runPath, *solutionID, *migrationVersion, *expectedRevision, *pluginLockPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		decision := deploymentdurable.RecoveryDecision{
			ID: "recovery-1", Strategy: deploymentdurable.RecoveryStrategy(*recoveryStrategy), ActionID: *actionID,
			TargetSolutionID: core.CanonicalID(*targetSolutionID), TargetSolutionVersion: *targetSolutionVersion, Reason: *reason,
		}
		writeJSON(deploymentdurable.PlanRecovery(run, decision))
	case "deployment-provider-plan":
		validateOrExit(w)
		run, _, _, err := loadCheckedDeploymentRun(w, *runPath, *solutionID, *migrationVersion, *expectedRevision, *pluginLockPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		env, ok := findEnvironment(w, run.Environment)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown environment %s\n", run.Environment)
			os.Exit(1)
		}
		routed, err := deploymentrouting.RouteCommands(env, deploymentdurable.ReadyCommands(run))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		plan := deploymentProviderPlan{Executors: routed}
		if *pluginLockPath != "" {
			var lock plugin.Lock
			if err := decodeJSONFile(*pluginLockPath, &lock); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			runtimePlan, err := pluginruntime.BuildPlan(plugin.Catalog{Plugins: w.Plugins, Bundles: w.Bundles}, env, lock)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			plan.PluginRuntime = &runtimePlan
		}
		seenObs, seenEv := map[string]bool{}, map[string]bool{}
		for _, actionID := range run.WaitingActions {
			rec, ok := findRunAction(run, actionID)
			if !ok {
				continue
			}
			if rec.Mode == deploymentdurable.ModeEvidence {
				pid, err := deploymentevidence.Binding(env, rec.EvidenceKind)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				k := string(rec.EvidenceKind) + "|" + string(pid) + "|" + rec.ID
				if !seenEv[k] {
					plan.Evidence = append(plan.Evidence, providerRoute{Kind: string(rec.EvidenceKind), Provider: pid, ActionID: rec.ID})
					seenEv[k] = true
				}
				continue
			}
			if kind, ok := deployment.ObservationForAction(rec.Kind); ok {
				pid, err := deploymentobservation.Binding(env, kind)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				k := string(kind) + "|" + string(pid)
				if !seenObs[k] {
					plan.Observations = append(plan.Observations, providerRoute{Kind: string(kind), Provider: pid})
					seenObs[k] = true
				}
			}
		}
		writeJSON(plan)
	case "deployment-start-recovery":
		validateOrExit(w)
		if *runID == "" {
			fmt.Fprintln(os.Stderr, "-run-id is required for the recovery run")
			os.Exit(2)
		}
		parent, _, _, err := loadCheckedDeploymentRun(w, *runPath, *solutionID, *migrationVersion, *expectedRevision, *pluginLockPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if parent.Recovery == nil {
			fmt.Fprintln(os.Stderr, "parent run has no recovery decision")
			os.Exit(1)
		}
		target, ok := findSolutionRelease(w, parent.Recovery.TargetSolutionID, parent.Recovery.TargetSolutionVersion)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown recovery solution release %s@%s\n", parent.Recovery.TargetSolutionID, parent.Recovery.TargetSolutionVersion)
			os.Exit(1)
		}
		compiled := releaseplan.CompileWithDeployment(target, w.ApplicationReleases, w.DatabaseReleases, parent.LastObserved)
		childLockPath := *targetPluginLockPath
		if childLockPath == "" {
			childLockPath = *pluginLockPath
		}
		childLockFP, childPolicyFP := mustRuntimeFingerprints(w, childLockPath, parent.Environment)
		updated, child, err := deploymentdurable.StartRecoveryWithRuntimePolicy(parent, *runID, target, compiled, parent.LastObserved, childLockFP, childPolicyFP)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		writeJSON(recoveryRunBundle{Parent: updated, RecoveryRun: child})
	case "deployment-reconcile":
		validateOrExit(w)
		if *solutionID == "" || *migrationVersion == "" || *deploymentPath == "" || *runPath == "" {
			fmt.Fprintln(os.Stderr, "-solution, -version, -deployment and -run are required")
			os.Exit(2)
		}
		sr, ok := findSolutionRelease(w, core.MustCanonicalID(*solutionID), *migrationVersion)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown solution release %s@%s\n", *solutionID, *migrationVersion)
			os.Exit(2)
		}
		var run deploymentdurable.Run
		if err := decodeJSONFile(*runPath, &run); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if *expectedRevision >= 0 && run.Revision != *expectedRevision {
			fmt.Fprintf(os.Stderr, "deployment run revision conflict: expected %d, found %d\n", *expectedRevision, run.Revision)
			os.Exit(1)
		}
		compiled := releaseplan.CompileWithDeployment(sr, w.ApplicationReleases, w.DatabaseReleases, run.Initial)
		if err := deploymentdurable.CheckRun(run, sr, compiled); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		lockFP, policyFP, err := runtimeFingerprintsForEnvironment(w, *pluginLockPath, run.Environment)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := deploymentdurable.CheckPluginLock(run, lockFP); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := deploymentdurable.CheckRuntimePolicy(run, policyFP); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var state deployment.Snapshot
		if err := decodeJSONFile(*deploymentPath, &state); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		validateDeploymentOrExit(w, state)
		var evidence []deploymentdurable.Evidence
		for _, path := range evidencePaths {
			var e deploymentdurable.Evidence
			if err := decodeJSONFile(path, &e); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			evidence = append(evidence, e)
		}
		writeJSON(deploymentdurable.Reconcile(run, state, evidence))
	case "deployment-simulate":
		validateOrExit(w)
		if *solutionID == "" || *migrationVersion == "" || *deploymentPath == "" {
			fmt.Fprintln(os.Stderr, "-solution, -version and -deployment are required")
			os.Exit(2)
		}
		sr, ok := findSolutionRelease(w, core.MustCanonicalID(*solutionID), *migrationVersion)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown solution release %s@%s\n", *solutionID, *migrationVersion)
			os.Exit(2)
		}
		var state deployment.Snapshot
		if err := decodeJSONFile(*deploymentPath, &state); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		validateDeploymentOrExit(w, state)
		compiled := releaseplan.CompileWithDeployment(sr, w.ApplicationReleases, w.DatabaseReleases, state)
		writeJSON(deploymentexec.Simulate(sr, compiled, state, w.DatabaseReleases))
	case "consumer-impact":
		validateOrExit(w)
		if *databaseID == "" || *migrationVersion == "" || *deploymentPath == "" {
			fmt.Fprintln(os.Stderr, "-database, -version and -deployment are required")
			os.Exit(2)
		}
		target, ok := findDatabaseRelease(w, core.MustCanonicalID(*databaseID), *migrationVersion)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown database release %s@%s\n", *databaseID, *migrationVersion)
			os.Exit(2)
		}
		var state deployment.Snapshot
		if err := decodeJSONFile(*deploymentPath, &state); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		validateDeploymentOrExit(w, state)
		writeJSON(consumerimpact.AnalyzeTarget(target, state, w.ApplicationReleases))
	case "index":
		validateOrExit(w)
		writeJSON(index.Build(w))
	default:
		flag.Usage()
		os.Exit(2)
	}
}

func loadCheckedDeploymentRun(w ir.World, runPath, solutionID, version string, expectedRevision int, pluginLockPath string) (deploymentdurable.Run, releaseplan.Definition, releaseplan.Compiled, error) {
	if runPath == "" || solutionID == "" || version == "" {
		return deploymentdurable.Run{}, releaseplan.Definition{}, releaseplan.Compiled{}, fmt.Errorf("-run, -solution and -version are required")
	}
	sr, ok := findSolutionRelease(w, core.MustCanonicalID(solutionID), version)
	if !ok {
		return deploymentdurable.Run{}, releaseplan.Definition{}, releaseplan.Compiled{}, fmt.Errorf("unknown solution release %s@%s", solutionID, version)
	}
	var run deploymentdurable.Run
	if err := decodeJSONFile(runPath, &run); err != nil {
		return run, sr, releaseplan.Compiled{}, err
	}
	if expectedRevision >= 0 && run.Revision != expectedRevision {
		return run, sr, releaseplan.Compiled{}, fmt.Errorf("deployment run revision conflict: expected %d, found %d", expectedRevision, run.Revision)
	}
	compiled := releaseplan.CompileWithDeployment(sr, w.ApplicationReleases, w.DatabaseReleases, run.Initial)
	if err := deploymentdurable.CheckRun(run, sr, compiled); err != nil {
		return run, sr, compiled, err
	}
	lockFP, policyFP, err := runtimeFingerprintsForEnvironment(w, pluginLockPath, run.Environment)
	if err != nil {
		return run, sr, compiled, err
	}
	if err := deploymentdurable.CheckPluginLock(run, lockFP); err != nil {
		return run, sr, compiled, err
	}
	if err := deploymentdurable.CheckRuntimePolicy(run, policyFP); err != nil {
		return run, sr, compiled, err
	}
	return run, sr, compiled, nil
}

func runtimeFingerprintsForEnvironment(w ir.World, path string, environmentID core.CanonicalID) (string, string, error) {
	if path == "" {
		return "", "", fmt.Errorf("-plugin-lock is required for durable deployment run operations")
	}
	var lock plugin.Lock
	if err := decodeJSONFile(path, &lock); err != nil {
		return "", "", err
	}
	env, ok := findEnvironment(w, environmentID)
	if !ok {
		return "", "", fmt.Errorf("unknown environment %s", environmentID)
	}
	plan, err := pluginruntime.BuildPlan(plugin.Catalog{Plugins: w.Plugins, Bundles: w.Bundles}, env, lock)
	if err != nil {
		return "", "", fmt.Errorf("plugin lock/runtime policy is not valid for environment %s: %w", environmentID, err)
	}
	return plan.PluginLockFingerprint, plan.RuntimePolicyFingerprint, nil
}

func pluginLockFingerprintForEnvironment(w ir.World, path string, environmentID core.CanonicalID) (string, error) {
	lockFP, _, err := runtimeFingerprintsForEnvironment(w, path, environmentID)
	return lockFP, err
}

func mustRuntimeFingerprints(w ir.World, path string, environmentID core.CanonicalID) (string, string) {
	lockFP, policyFP, err := runtimeFingerprintsForEnvironment(w, path, environmentID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return lockFP, policyFP
}

func mustPluginLockFingerprint(w ir.World, path string, environmentID core.CanonicalID) string {
	lockFP, _ := mustRuntimeFingerprints(w, path, environmentID)
	return lockFP
}

func loadObservations(analysisPaths, schemaPaths, planPaths []string) ([]analyzer.Result, []physical.SchemaSnapshot, []plan.IR, error) {
	var analyses []analyzer.Result
	for _, path := range analysisPaths {
		var a analyzer.Result
		if err := decodeJSONFile(path, &a); err != nil {
			return nil, nil, nil, err
		}
		analyses = append(analyses, a)
	}
	var snapshots []physical.SchemaSnapshot
	for _, path := range schemaPaths {
		var snap physical.SchemaSnapshot
		if err := decodeJSONFile(path, &snap); err != nil {
			return nil, nil, nil, err
		}
		snapshots = append(snapshots, snap)
	}
	var plans []plan.IR
	for _, path := range planPaths {
		var p plan.IR
		if err := decodeJSONFile(path, &p); err != nil {
			return nil, nil, nil, err
		}
		plans = append(plans, p)
	}
	return analyses, snapshots, plans, nil
}

func formatProjectedDiagnostic(projected workspace.ProjectedDiagnostic) string {
	diagnostic := projected.Diagnostic
	if len(projected.Sources) == 0 {
		return fmt.Sprintf("%s\t%s\t%s", diagnostic.Severity, diagnostic.Code, diagnostic.Message)
	}
	source := projected.Sources[0]
	return fmt.Sprintf("%s:%d:%d\t%s\t%s\t%s", source.ArtifactPath, source.Range.Start.Line, source.Range.Start.Column, diagnostic.Severity, diagnostic.Code, diagnostic.Message)
}

func formatSourceDiagnostic(diagnostic workspace.SourceDiagnostic) string {
	return fmt.Sprintf("%s:%d:%d\t%s\t%s\t%s", diagnostic.Source.ArtifactPath, diagnostic.Source.Range.Start.Line, diagnostic.Source.Range.Start.Column, diagnostic.Severity, diagnostic.Code, diagnostic.Message)
}

func validateOrExit(w ir.World) {
	ds := validation.Validate(w)
	if !validation.HasErrors(ds) {
		return
	}
	for _, d := range ds {
		if d.Severity == validation.Error {
			fmt.Fprintf(os.Stderr, "%s\t%s\t%s\n", d.Severity, d.Code, d.Message)
		}
	}
	os.Exit(1)
}

func validateDeploymentOrExit(w ir.World, state deployment.Snapshot) {
	ds := validation.ValidateDeployment(w, state)
	if !validation.HasErrors(ds) {
		return
	}
	for _, d := range ds {
		if d.Severity == validation.Error {
			fmt.Fprintf(os.Stderr, "%s\t%s\t%s\n", d.Severity, d.Code, d.Message)
		}
	}
	os.Exit(1)
}

func writeJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func decodeJSONFile(path string, dst any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(dst)
}

func loadWorld(worldPath, workspacePath string) (ir.World, error) {
	if workspacePath != "" {
		r, err := workspace.Load(workspacePath)
		if err != nil {
			return ir.World{}, err
		}
		return r.World, nil
	}
	f, err := os.Open(worldPath)
	if err != nil {
		return ir.World{}, err
	}
	defer f.Close()
	var w ir.World
	if err := json.NewDecoder(f).Decode(&w); err != nil {
		return ir.World{}, err
	}
	return w, nil
}

func findEnvironment(w ir.World, id core.CanonicalID) (environment.Environment, bool) {
	for _, e := range w.Environments {
		if e.ID == id {
			return e, true
		}
	}
	return environment.Environment{}, false
}

func findRunAction(run deploymentdurable.Run, id string) (deploymentdurable.ActionRecord, bool) {
	for _, a := range run.Actions {
		if a.ID == id {
			return a, true
		}
	}
	return deploymentdurable.ActionRecord{}, false
}

func findDatabase(w ir.World, id core.CanonicalID) (database.Database, bool) {
	for _, d := range w.Databases {
		if d.ID == id {
			return d, true
		}
	}
	return database.Database{}, false
}

func findMigration(w ir.World, database core.CanonicalID, id string) (migration.Definition, bool) {
	for _, m := range w.Migrations {
		if m.Database == database && m.ID == id {
			return m, true
		}
	}
	return migration.Definition{}, false
}
func migrationsFor(w ir.World, database core.CanonicalID) []migration.Definition {
	var out []migration.Definition
	for _, m := range w.Migrations {
		if m.Database == database {
			out = append(out, m)
		}
	}
	return out
}
func findApplication(w ir.World, id core.CanonicalID) (application.Application, bool) {
	for _, a := range w.Applications {
		if a.ID == id {
			return a, true
		}
	}
	return application.Application{}, false
}
func findDatabaseRelease(w ir.World, id core.CanonicalID, version string) (release.Definition, bool) {
	for _, r := range w.DatabaseReleases {
		if r.Database == id && r.Version == version {
			return r, true
		}
	}
	return release.Definition{}, false
}

func findSolutionRelease(w ir.World, id core.CanonicalID, version string) (releaseplan.Definition, bool) {
	for _, r := range w.SolutionReleases {
		if r.ID == id && r.Version == version {
			return r, true
		}
	}
	return releaseplan.Definition{}, false
}

func findApplicationRelease(w ir.World, applicationID core.CanonicalID, version string) (apprelease.Definition, bool) {
	for _, r := range w.ApplicationReleases {
		if r.Application == applicationID && r.Version == version {
			return r, true
		}
	}
	return apprelease.Definition{}, false
}

func uniqueManagedReleaseStatesForRelease(w ir.World, ar apprelease.Definition) []release.StateSnapshot {
	return uniqueManagedReleaseStates(w, ar.AsApplication())
}

func uniqueManagedReleaseStates(w ir.World, app application.Application) []release.StateSnapshot {
	var out []release.StateSnapshot
	for _, req := range app.Databases {
		db, ok := findDatabase(w, req.TargetDatabase)
		if !ok || db.Ownership != database.OwnershipManagedHere {
			continue
		}
		var matches []release.Definition
		for _, r := range w.DatabaseReleases {
			if r.Database == db.ID {
				matches = append(matches, r)
			}
		}
		if len(matches) == 1 {
			out = upsertState(out, release.StateFromDefinition(matches[0]))
		}
	}
	return out
}
func upsertState(states []release.StateSnapshot, state release.StateSnapshot) []release.StateSnapshot {
	for i := range states {
		if states[i].Database == state.Database {
			states[i] = state
			return states
		}
	}
	return append(states, state)
}
