// reference-data-first-persistenceは、E0のData定義へ手書きの明示overlayを重ねた作業領域で、
// Command実行と別接続の読取りを実PostgreSQL上で一巡させる参照用ツールである。
//
// 実PostgreSQLを使う証明は、このmain関数が出力するJSON receiptである。go testは
// DBを必要としない純粋処理とmock処理だけを検査する。範囲と非主張はREADME.mdを参照する。
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database/physical"
	postgresprovider "tei/internal/database/providers/postgres"
	dbrealization "tei/internal/database/realization"
	"tei/internal/database/schemadiff"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/validation"
	"tei/internal/workspace"
	"tei/tools/internal/postgresdb"
	"tei/tools/internal/repositoryevidence"
)

const evidenceNature = "real-postgres-ephemeral-fixture"

const (
	baseWorkspaceDir = "examples/first-workspace"
	overlayDir       = "tools/reference-data-first-persistence/testdata/overlay"
	pluginSourceFile = "reference/workspace/plugins/tei-database-postgres/1.0.0/plugin.tei"
	pluginTargetFile = "plugins/tei-database-postgres/1.0.0/plugin.tei"
	requestSourceTEI = "semantic/intake/request.tei"
	realizeFile      = "applications/request-log/realizations/submit-request.tei"
)

var (
	applicationID     = core.MustCanonicalID("request_log")
	browsePatternID   = core.MustCanonicalID("request_log.list_requests")
	commandPatternID  = core.MustCanonicalID("request_log.submit_request")
	capabilityID      = core.MustCanonicalID("intake.request.submit")
	environmentID     = core.MustCanonicalID("development")
	datasourceAlias   = core.LocalName("store")
	bindingIdentity   = realization.BindingIdentity(applicationID, capabilityID)
	readIdentity      = "request_log.list_requests@independent-read-pool"
	observationNature = driver.ObservationEvidenceNature(evidenceNature)
)

// ---------------------------------------------------------------------------
// 実行結果の型
// ---------------------------------------------------------------------------

type stageStatus string

const (
	stageResolved    stageStatus = "resolved"
	stageUnresolved  stageStatus = "unresolved"
	stageUnsupported stageStatus = "unsupported"
)

// stageResultは、Command実行前の各段階を、解決済み・未解決・未対応の三値で表す。
type stageResult struct {
	Stage  string      `json:"stage"`
	Status stageStatus `json:"status"`
	Detail string      `json:"detail,omitempty"`
}

type scopeStatement struct {
	HandWrittenOverlay            bool   `json:"handWrittenOverlay"`
	HandWrittenInsertAndQuerySQL  bool   `json:"handWrittenInsertAndQuerySQL"`
	GeneratedRoundTrip            bool   `json:"generatedRoundTrip"`
	Authority                     string `json:"authority"`
	CommandToPersistenceLinkage   string `json:"commandToPersistenceLinkage"`
	ProductE2                     bool   `json:"productE2"`
	IndependenceKind              string `json:"independenceKind"`
	DeferredToNextSlice           string `json:"deferredToNextSlice"`
	ProviderBindingPoolInjection  string `json:"providerBindingPoolInjection"`
	DeclaredReferenceMissingList  string `json:"declaredReferenceMissingList"`
	ParameterBindingOfInsertQuery string `json:"parameterBindingOfInsertQuery"`
}

func fixedScope() scopeStatement {
	return scopeStatement{
		HandWrittenOverlay:            true,
		HandWrittenInsertAndQuerySQL:  true,
		GeneratedRoundTrip:            false,
		Authority:                     "none",
		CommandToPersistenceLinkage:   "unverified",
		ProductE2:                     false,
		IndependenceKind:              "connection-pool-separation-in-one-process",
		DeferredToNextSlice:           "S05-and-duplicate-outcome",
		ProviderBindingPoolInjection:  "database-identity-and-provider-derived-from-environment-binding; connection-pools-injected-by-tool",
		DeclaredReferenceMissingList:  "reference-standard-function-candidates-missing-list-is-static",
		ParameterBindingOfInsertQuery: "client-side-interpolation-by-pgx-simple-protocol",
	}
}

type overlayEvidence struct {
	RequestTeiSha256Unchanged bool `json:"requestTeiSha256Unchanged"`
	ValidationErrors          int  `json:"validationErrors"`
	PluginSourceCopiedAtRun   bool `json:"pluginSourceCopiedAtRun"`
}

type fixtureDatabaseSafetyEvidence struct {
	GuardPassed                     bool `json:"guardPassed"`
	DatabaseIdentityVerified        bool `json:"databaseIdentityVerified"`
	DatabaseIdentityPersisted       bool `json:"databaseIdentityPersisted"`
	NonDefaultDatabaseVerified      bool `json:"nonDefaultDatabaseVerified"`
	UserRelationsBeforeFixture      int  `json:"userRelationsBeforeFixture"`
	EventTriggersBeforeFixture      int  `json:"eventTriggersBeforeFixture"`
	FixtureObjectAbsentBeforeCreate bool `json:"fixtureObjectAbsentBeforeCreate"`
}

type fixtureCleanupSafetyEvidence struct {
	Verified                        bool `json:"verified"`
	UserRelationsAfterCleanup       int  `json:"userRelationsAfterCleanup"`
	EventTriggersAfterCleanup       int  `json:"eventTriggersAfterCleanup"`
	FixtureObjectAbsentAfterCleanup bool `json:"fixtureObjectAbsentAfterCleanup"`
}

type positiveEvidence struct {
	Preflight                 []stageResult `json:"preflight"`
	AllStagesResolvedAfterDDL bool          `json:"allStagesResolvedAfterFixtureDDL"`
	CommandID                 string        `json:"commandId"`
	BindingIdentity           string        `json:"bindingIdentity"`
	ResultMode                string        `json:"resultMode"`
	BackendPIDCommandPool     int64         `json:"backendPidCommandPool"`
	ReadIdentity              string        `json:"readIdentity"`
	ReadEvidenceNature        string        `json:"readEvidenceNature"`
	SubmittedCommandID        string        `json:"submittedCommandId"`
	BackendPIDReadPoolBefore  int64         `json:"backendPidReadPoolBefore"`
	BackendPIDReadPoolAfter   int64         `json:"backendPidReadPoolAfter"`
	BackendPIDsDiffer         bool          `json:"backendPidsDiffer"`
	RowsRead                  int           `json:"rowsRead"`
	RowMatchesSubmission      bool          `json:"rowMatchesSubmission"`
	MappedSemanticAttributes  []string      `json:"mappedSemanticAttributes"`
	ObservationStatus         string        `json:"observationStatus"`
	DatabaseIdentityFromEnv   string        `json:"databaseIdentitySource"`
	DatabaseProviderFromEnv   string        `json:"databaseProvider"`
	ParameterValuesPersisted  bool          `json:"parameterValuesPersisted"`
}

type negativeEvidence struct {
	Name                    string        `json:"name"`
	Preflight               []stageResult `json:"preflight,omitempty"`
	CommandRefusedByGate    bool          `json:"commandRefusedByGate"`
	DriverRefusedDirectCall *bool         `json:"driverRefusedDirectInvocation,omitempty"`
	CommandDispatchCount    int           `json:"commandDispatchCount"`
	RowsAfter               *int          `json:"rowsAfter,omitempty"`
	RelationAbsentAfter     *bool         `json:"relationAbsentAfter,omitempty"`
	Note                    string        `json:"note,omitempty"`
}

type n11Evidence struct {
	Name                    string `json:"name"`
	CommandDispatched       bool   `json:"commandDispatched"`
	CommandResultPresent    bool   `json:"commandResultPresent"`
	CommandID               string `json:"commandId"`
	ObservationErrorSeen    bool   `json:"observationErrorSeen"`
	ObservationStatus       string `json:"observationStatus"`
	RowsCommittedViaPoolB   *int   `json:"rowsCommittedViaIndependentPool,omitempty"`
	ReadFailureMechanism    string `json:"readFailureMechanism"`
	NeverReportedAsObserved bool   `json:"neverReportedAsObserved"`
}

type receipt struct {
	EvidenceNature       string                        `json:"evidenceNature"`
	RepositoryRevision   string                        `json:"repositoryRevision"`
	WorkingTreeClean     bool                          `json:"workingTreeClean"`
	ReproducibleRevision bool                          `json:"reproducibleRevision"`
	PostRunRepository    repositoryevidence.PostRun    `json:"postRunRepository"`
	Scope                scopeStatement                `json:"scope"`
	Overlay              overlayEvidence               `json:"overlay"`
	FixtureObject        string                        `json:"fixtureObject"`
	FixtureOwnedByTool   bool                          `json:"fixtureOwnedByTool"`
	FixtureDatabase      fixtureDatabaseSafetyEvidence `json:"fixtureDatabaseSafety"`
	FixtureCleanup       fixtureCleanupSafetyEvidence  `json:"fixtureCleanupSafety"`
	Positive             positiveEvidence              `json:"positive"`
	N08                  negativeEvidence              `json:"n08MissingBinding"`
	S02AnalogueAbsent    negativeEvidence              `json:"s02AnalogueRelationAbsent"`
	S02AnalogueColumn    negativeEvidence              `json:"s02AnalogueColumnMissing"`
	N11ClosedPool        n11Evidence                   `json:"n11ClosedReadPool"`
	N11DroppedTable      n11Evidence                   `json:"n11DroppedTable"`
}

// ---------------------------------------------------------------------------
// DSN入力
// ---------------------------------------------------------------------------

const postgresDSNEnv = "TEI_TEST_POSTGRES_DSN"
const postgresDSNFileEnv = "TEI_TEST_POSTGRES_DSN_FILE"

type dsnLookupEnvFunc func(string) (string, bool)
type dsnReadFileFunc func(string) ([]byte, error)

// loadPostgresDSNInputは、直接入力とsecret file入力のどちらか一方だけを受理する。
// reference-postgres-header-detail-runtimeの同名処理と同じ規則の複製である。
func loadPostgresDSNInput(lookupEnv dsnLookupEnvFunc, readFile dsnReadFileFunc) (string, error) {
	direct, directSet := lookupEnv(postgresDSNEnv)
	path, fileSet := lookupEnv(postgresDSNFileEnv)

	if directSet && fileSet {
		return "", fmt.Errorf("PostgreSQL DSNの直接入力とsecret file入力が同時に設定されています。どちらか一方だけを指定してください")
	}
	if directSet {
		if direct == "" {
			return "", fmt.Errorf("%sが空です", postgresDSNEnv)
		}
		return direct, nil
	}
	if !fileSet || path == "" {
		return "", fmt.Errorf("%sまたは%sのどちらか一方を指定してください", postgresDSNEnv, postgresDSNFileEnv)
	}
	content, err := readFile(path)
	if err != nil {
		return "", fmt.Errorf("PostgreSQL DSN secret fileを読み取れません")
	}
	return normalizePostgresDSNFile(content)
}

func normalizePostgresDSNFile(content []byte) (string, error) {
	if bytes.IndexByte(content, 0) >= 0 {
		return "", fmt.Errorf("PostgreSQL DSN secret fileにNULが含まれているため受理しません")
	}
	switch {
	case bytes.HasSuffix(content, []byte("\r\n")):
		content = content[:len(content)-2]
	case bytes.HasSuffix(content, []byte("\n")):
		content = content[:len(content)-1]
	}
	if len(content) == 0 {
		return "", fmt.Errorf("PostgreSQL DSN secret fileが空です")
	}
	if bytes.ContainsAny(content, "\r\n") {
		return "", fmt.Errorf("PostgreSQL DSN secret fileは一行でなければなりません")
	}
	return string(content), nil
}

func clearPostgresDSNEnvironment() error {
	if err := os.Unsetenv(postgresDSNEnv); err != nil {
		return fmt.Errorf("PostgreSQL DSN環境変数を消去できません: %w", err)
	}
	if err := os.Unsetenv(postgresDSNFileEnv); err != nil {
		return fmt.Errorf("PostgreSQL DSN secret file環境変数を消去できません: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// 作業領域の組立て
// ---------------------------------------------------------------------------

type preparedWorkspace struct {
	Root                string
	RequestTeiUnchanged bool
	PluginCopied        bool
}

// prepareWorkspaceは、first-workspaceを一時領域へ複製し、overlayを重ね、
// 参照作業領域のplugin定義を複製する。overlayはrequest.teiを置き換えない。
// omitは、overlayから除外する相対パスである（N08でrealizeを取り除くために使う）。
func prepareWorkspace(repoRoot, dst string, omit ...string) (preparedWorkspace, error) {
	baseDir := filepath.Join(repoRoot, filepath.FromSlash(baseWorkspaceDir))
	if err := copyTree(baseDir, dst, nil); err != nil {
		return preparedWorkspace{}, fmt.Errorf("first-workspaceを複製できません: %w", err)
	}
	omitSet := map[string]bool{}
	for _, o := range omit {
		omitSet[filepath.ToSlash(o)] = true
	}
	if err := copyTree(filepath.Join(repoRoot, filepath.FromSlash(overlayDir)), dst, omitSet); err != nil {
		return preparedWorkspace{}, fmt.Errorf("overlayを適用できません: %w", err)
	}
	pluginSrc, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(pluginSourceFile)))
	if err != nil {
		return preparedWorkspace{}, fmt.Errorf("plugin定義を読み取れません: %w", err)
	}
	pluginDst := filepath.Join(dst, filepath.FromSlash(pluginTargetFile))
	if err := os.MkdirAll(filepath.Dir(pluginDst), 0o755); err != nil {
		return preparedWorkspace{}, err
	}
	if err := os.WriteFile(pluginDst, pluginSrc, 0o644); err != nil {
		return preparedWorkspace{}, err
	}

	before, err := fileSha256(filepath.Join(baseDir, filepath.FromSlash(requestSourceTEI)))
	if err != nil {
		return preparedWorkspace{}, err
	}
	after, err := fileSha256(filepath.Join(dst, filepath.FromSlash(requestSourceTEI)))
	if err != nil {
		return preparedWorkspace{}, err
	}
	return preparedWorkspace{Root: dst, RequestTeiUnchanged: before == after, PluginCopied: true}, nil
}

func copyTree(src, dst string, omit map[string]bool) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		slash := filepath.ToSlash(rel)
		if omit[slash] {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("通常ファイル以外は複製しません: %s", slash)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	})
}

func fileSha256(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

func countValidationErrors(world ir.World) (int, []string) {
	count := 0
	var codes []string
	for _, d := range validation.Validate(world) {
		if d.Severity == validation.Error {
			count++
			codes = append(codes, d.Code)
		}
	}
	return count, codes
}

// ---------------------------------------------------------------------------
// 実行前の段階判定（DB効果の前）
// ---------------------------------------------------------------------------

// databaseTargetは、Worldのenvironment bindingから導出したDatabase識別とproviderである。
type databaseTarget struct {
	Alias      core.LocalName
	Definition core.CanonicalID
	Instance   environment.DatabaseInstance
}

// preflightWorldは、DBへ接続せずWorldだけで解決できる段階を判定する。
// browse PatternとcommandのBindingはDriverの解決処理をそのまま使い、
// Database識別とproviderはenvironmentのbindingから導出する。
func preflightWorld(world ir.World) ([]stageResult, *databaseTarget) {
	resolver := driver.NewResolver(world)
	var stages []stageResult

	var datasource *application.DatabaseRequirement
	if browse, err := resolver.ResolvePattern(applicationID, browsePatternID); err != nil {
		stages = append(stages, stageResult{Stage: "browse-pattern", Status: stageUnresolved, Detail: err.Error()})
	} else if browse.Query == nil || browse.Datasource == nil {
		stages = append(stages, stageResult{Stage: "browse-pattern", Status: stageUnresolved, Detail: "QueryまたはDatasourceが解決されていません"})
	} else {
		stages = append(stages, stageResult{Stage: "browse-pattern", Status: stageResolved})
		datasource = browse.Datasource
	}

	if _, err := resolver.ResolveCommandBinding(applicationID, commandPatternID); err != nil {
		stages = append(stages, stageResult{Stage: "command-pattern-binding", Status: stageUnresolved, Detail: err.Error()})
	} else {
		stages = append(stages, stageResult{Stage: "command-pattern-binding", Status: stageResolved})
	}

	target, stage := resolveDatabaseTarget(world, datasource)
	stages = append(stages, stage)
	return stages, target
}

// resolveDatabaseTargetは、DatasourceのaliasからenvironmentのDatabase instanceを導出する。
// providerがpostgres以外なら、このツールが実行できる範囲外として未対応にする。
func resolveDatabaseTarget(world ir.World, datasource *application.DatabaseRequirement) (*databaseTarget, stageResult) {
	const stage = "environment-database-binding"
	if datasource == nil {
		return nil, stageResult{Stage: stage, Status: stageUnresolved, Detail: "Datasourceが解決されていないため、environmentのbindingを選べません"}
	}
	var env *environment.Environment
	for i := range world.Environments {
		if world.Environments[i].ID == environmentID {
			env = &world.Environments[i]
		}
	}
	if env == nil {
		return nil, stageResult{Stage: stage, Status: stageUnresolved, Detail: fmt.Sprintf("environment %sがありません", environmentID)}
	}
	var matches []environment.ApplicationDatabaseBinding
	for _, b := range env.Bindings {
		if b.Application == applicationID && b.Alias == datasource.Alias {
			matches = append(matches, b)
		}
	}
	switch len(matches) {
	case 0:
		return nil, stageResult{Stage: stage, Status: stageUnresolved, Detail: fmt.Sprintf("environment %sに%s.%sのbindingがありません", env.ID, applicationID, datasource.Alias)}
	case 1:
	default:
		return nil, stageResult{Stage: stage, Status: stageUnresolved, Detail: fmt.Sprintf("environment %sの%s.%sのbindingが複数あります", env.ID, applicationID, datasource.Alias)}
	}
	instance, ok := env.InstanceByID(matches[0].Instance)
	if !ok {
		return nil, stageResult{Stage: stage, Status: stageUnresolved, Detail: fmt.Sprintf("Database instance %sが定義されていません", matches[0].Instance)}
	}
	if instance.Definition != datasource.TargetDatabase {
		return nil, stageResult{Stage: stage, Status: stageUnresolved, Detail: fmt.Sprintf("instance %sの定義%sとDatasourceの対象%sが一致しません", instance.ID, instance.Definition, datasource.TargetDatabase)}
	}
	if instance.Provider != environment.ProviderPostgres {
		return nil, stageResult{Stage: stage, Status: stageUnsupported, Detail: fmt.Sprintf("providerは%sです。このツールが実行できるのはpostgresだけです", instance.Provider)}
	}
	return &databaseTarget{Alias: datasource.Alias, Definition: instance.Definition, Instance: instance}, stageResult{Stage: stage, Status: stageResolved}
}

type introspectFunc func(context.Context) (physical.SchemaSnapshot, error)

// preflightSchemaは、Persistence MappingからRealizeDesiredで導いた物理schemaと、
// Introspectした実物をschemadiff.Compareで比べる。information_schemaを自前で読まない。
// 変更または未確定の比較が一つでもあれば、Commandを実行できる状態ではないとする。
func preflightSchema(ctx context.Context, world ir.World, target *databaseTarget, introspect introspectFunc) (stageResult, schemadiff.Diff) {
	const stage = "physical-schema"
	if target == nil {
		return stageResult{Stage: stage, Status: stageUnresolved, Detail: "Database識別が導出されていないため、物理schemaを比較できません"}, schemadiff.Diff{}
	}
	desired, err := dbrealization.RealizeDesired(world.Persistence, string(target.Definition), target.Instance.Provider)
	if err != nil {
		return stageResult{Stage: stage, Status: stageUnsupported, Detail: fmt.Sprintf("望ましい物理schemaを導出できません: %v", err)}, schemadiff.Diff{}
	}
	if len(desired.Tables) == 0 {
		return stageResult{Stage: stage, Status: stageUnresolved, Detail: "Persistence Mappingから導出されたtableがありません"}, schemadiff.Diff{}
	}
	actual, err := introspect(ctx)
	if err != nil {
		return stageResult{Stage: stage, Status: stageUnresolved, Detail: fmt.Sprintf("物理schemaを観測できません: %v", err)}, schemadiff.Diff{}
	}
	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		return stageResult{Stage: stage, Status: stageUnresolved, Detail: fmt.Sprintf("物理schemaを比較できません: %v", err)}, schemadiff.Diff{}
	}
	return classifyDiff(diff), diff
}

func classifyDiff(diff schemadiff.Diff) stageResult {
	const stage = "physical-schema"
	if len(diff.Changes) == 0 && len(diff.Unresolved) == 0 {
		return stageResult{Stage: stage, Status: stageResolved}
	}
	return stageResult{Stage: stage, Status: stageUnresolved, Detail: summarizeDiff(diff)}
}

func summarizeDiff(diff schemadiff.Diff) string {
	var parts []string
	for _, c := range diff.Changes {
		part := string(c.Kind) + " " + c.Table
		switch {
		case c.Column != nil:
			part += "." + c.Column.Name
		case c.BeforeColumn != nil:
			part += "." + c.BeforeColumn.Name
		}
		parts = append(parts, part)
	}
	for _, u := range diff.Unresolved {
		parts = append(parts, "unresolved "+u.Table+"."+u.ObjectName)
	}
	sort.Strings(parts)
	return strings.Join(parts, "; ")
}

func allResolved(stages []stageResult) bool {
	if len(stages) == 0 {
		return false
	}
	for _, s := range stages {
		if s.Status != stageResolved {
			return false
		}
	}
	return true
}

func unresolvedStages(stages []stageResult) []stageResult {
	var out []stageResult
	for _, s := range stages {
		if s.Status != stageResolved {
			out = append(out, s)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Command: 手書きのINSERT adapterと実行ゲート
// ---------------------------------------------------------------------------

type submission struct {
	CommandID string
	RequestID string
	Summary   string
	Status    string
}

type commandPayload struct {
	CommandID  string
	RequestID  string
	BackendPID int64
}

// insertFuncは、一行を書き込み、その接続のpg_backend_pidを返す。
type insertFunc func(context.Context, submission) (int64, error)

// insertAdapterは、検査用のcustom-go adapterである。Capabilityの単一output契約に従い、
// ResultOutputで結果を返す。DB障害はerrorであり、業務結果へ変換しない。
type insertAdapter struct {
	insert     insertFunc
	dispatched *int
}

func (a insertAdapter) Invoke(ctx context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	s, ok := input.(submission)
	if !ok {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("想定外の入力型です: %T", input)
	}
	*a.dispatched++
	pid, err := a.insert(ctx, s)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("手書きINSERTを完了できません: %w", err)
	}
	return capabilityinvoke.BusinessResult{
		Capability: capability,
		Mode:       capabilityinvoke.ResultOutput,
		Payload:    commandPayload{CommandID: s.CommandID, RequestID: s.RequestID, BackendPID: pid},
	}, nil
}

// adapterSetは、解決済みBindingのidentityが期待どおりのときだけadapterを返す。
type adapterSet struct {
	insert     insertFunc
	dispatched int
}

var _ driver.BindingAdapterResolver = (*adapterSet)(nil)

func (s *adapterSet) ResolveAdapter(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
	if binding.Identity() != bindingIdentity {
		return nil, fmt.Errorf("想定外のBindingです: %s", binding.Identity())
	}
	return insertAdapter{insert: s.insert, dispatched: &s.dispatched}, nil
}

type gateResult struct {
	Refused    bool
	Reasons    []stageResult
	Invocation *driver.CapabilityInvocation
	Err        error
}

// submitIfReadyは、実行前の段階がすべてresolvedのときだけDriver経由でCommandを実行する。
// 一つでも未解決・未対応があれば、adapterを呼ばずに拒否する。
func submitIfReady(ctx context.Context, resolver driver.Resolver, preflight []stageResult, adapters *adapterSet, in submission) gateResult {
	if !allResolved(preflight) {
		return gateResult{Refused: true, Reasons: unresolvedStages(preflight)}
	}
	invocation, err := resolver.InvokeCapability(ctx, applicationID, capabilityID, adapters, in)
	if err != nil {
		return gateResult{Err: err}
	}
	return gateResult{Invocation: &invocation}
}

// ---------------------------------------------------------------------------
// 独立読取りと観測状態の判定
// ---------------------------------------------------------------------------

type observationStatus string

const (
	observationObserved            observationStatus = "observed"
	observationReadFailed          observationStatus = "read-failed"
	observationMismatch            observationStatus = "mismatch"
	observationCommandNotCommitted observationStatus = "command-not-committed"
)

// classifyObservationは、Command結果とそれとは別に実行した読取りの結果から観測状態を決める。
// 読取りが失敗した場合や、読んだ行が提出内容と一致しない場合は、決してobservedにしない。
func classifyObservation(commandResultPresent bool, readErr error, rowMatches bool) observationStatus {
	switch {
	case !commandResultPresent:
		return observationCommandNotCommitted
	case readErr != nil:
		return observationReadFailed
	case !rowMatches:
		return observationMismatch
	default:
		return observationObserved
	}
}

// rowMatchesSubmissionは、写像済みの結果がちょうど一行で、提出内容と一致するかを返す。
// 列順ではなく、明示された意味属性の識別子で値を引く。
func rowMatchesSubmission(mapped queryresult.MappedResult, s submission) (bool, error) {
	if len(mapped.Rows) != 1 {
		return false, nil
	}
	index := map[core.CanonicalID]int{}
	for i, c := range mapped.Columns {
		index[c.Attribute] = i
	}
	want := map[core.CanonicalID]string{
		"intake.request.request_id": s.RequestID,
		"intake.request.summary":    s.Summary,
		"intake.request.status":     s.Status,
	}
	if len(mapped.Columns) != len(want) {
		return false, fmt.Errorf("写像済みの列数=%d want=%d", len(mapped.Columns), len(want))
	}
	for attribute, expected := range want {
		i, ok := index[attribute]
		if !ok {
			return false, fmt.Errorf("意味属性%sの列がありません", attribute)
		}
		got, err := textValue(mapped.Rows[0][i])
		if err != nil {
			return false, fmt.Errorf("意味属性%s: %w", attribute, err)
		}
		if got != expected {
			return false, nil
		}
	}
	return true, nil
}

func textValue(v any) (string, error) {
	switch x := v.(type) {
	case string:
		return x, nil
	case []byte:
		return string(x), nil
	default:
		return "", fmt.Errorf("文字列として読めない値の型です: %T", v)
	}
}

func mappedAttributeIDs(mapped queryresult.MappedResult) []string {
	out := make([]string, 0, len(mapped.Columns))
	for _, c := range mapped.Columns {
		out = append(out, string(c.Attribute))
	}
	sort.Strings(out)
	return out
}

// ---------------------------------------------------------------------------
// 安全guardとfixture（reference-postgres-header-detail-runtimeの表非依存部分の一時的な複製）
// ---------------------------------------------------------------------------
//
// 共通化は別途行う。ここでは表に依存しない確認（既定Databaseの拒否、user relation 0件、
// Event Trigger 0件、終了時の同数確認）だけを複製し、所有印の接頭辞はこのツール専用にする。

const fixtureObject = "public.requests"
const fixtureOwnershipPrefix = "tei:reference-data-first-persistence:"

// fixtureCreateSQLは、このツールが手書きしたDDLである。DataやPersistence Mappingから
// 描画したものではなく、物理schemaの生成を主張しない。
const fixtureCreateSQL = `
CREATE TABLE public.requests (
    request_id varchar(64) PRIMARY KEY,
    summary text NOT NULL,
    status varchar(32) NOT NULL
);
`

const fixtureLockSQL = `LOCK TABLE public.requests IN ACCESS EXCLUSIVE MODE;`
const fixtureMarkerQuery = `SELECT pg_catalog.obj_description('public.requests'::regclass, 'pg_class')`
const fixtureDropSQL = `DROP TABLE public.requests;`

const userRelationCountQuery = `
SELECT count(*)
FROM pg_catalog.pg_class AS c
JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
  AND n.nspname NOT LIKE 'pg_toast%'
  AND n.nspname NOT LIKE 'pg_temp_%'
  AND c.relkind IN ('r', 'p', 'v', 'm', 'f', 'S')
`

const eventTriggerCountQuery = `SELECT count(*) FROM pg_catalog.pg_event_trigger`

func validateDedicatedFixtureDatabase(databaseName string, userRelations, eventTriggers int) error {
	name := strings.TrimSpace(databaseName)
	if name == "" {
		return fmt.Errorf("安全のためツールを停止します: 接続先Database identityが空です")
	}
	switch name {
	case "postgres", "template0", "template1":
		return fmt.Errorf("安全のためツールを停止します: PostgreSQL既定Databaseは実行先にできません。専用の空Databaseを指定してください")
	}
	if userRelations != 0 {
		return fmt.Errorf("安全のためツールを停止します: 接続先Databaseには既存user relationが%d件あります。専用の空Databaseを指定してください", userRelations)
	}
	if eventTriggers != 0 {
		return fmt.Errorf("安全のためツールを停止します: 接続先DatabaseにはEvent Triggerが%d件あります。専用の空Databaseを指定してください", eventTriggers)
	}
	return nil
}

func validateFixtureDatabaseRestored(userRelations, eventTriggers int) error {
	if userRelations != 0 {
		return fmt.Errorf("終了時にuser relationが%d件残っています。tool-owned fixtureまたは並行変更の残存を確認してください", userRelations)
	}
	if eventTriggers != 0 {
		return fmt.Errorf("終了時にEvent Triggerが%d件存在します。並行変更の残存を確認してください", eventTriggers)
	}
	return nil
}

func countOne(ctx context.Context, db *sql.DB, query, what string) (int, error) {
	var n int
	if err := db.QueryRowContext(ctx, query).Scan(&n); err != nil {
		return 0, fmt.Errorf("%sを確認できません: %w", what, err)
	}
	return n, nil
}

func relationAbsent(ctx context.Context, db *sql.DB) (bool, error) {
	var object sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT to_regclass('public.requests')::text`).Scan(&object); err != nil {
		return false, fmt.Errorf("fixture objectの有無を確認できません: %w", err)
	}
	return !object.Valid || strings.TrimSpace(object.String) == "", nil
}

func ensureDedicatedEmptyDatabase(ctx context.Context, db *sql.DB) (fixtureDatabaseSafetyEvidence, error) {
	result := fixtureDatabaseSafetyEvidence{}
	var databaseName string
	if err := db.QueryRowContext(ctx, `SELECT current_database()`).Scan(&databaseName); err != nil {
		return result, fmt.Errorf("接続先Database identityを確認できません: %w", err)
	}
	userRelations, err := countOne(ctx, db, userRelationCountQuery, "既存user relation")
	if err != nil {
		return result, err
	}
	eventTriggers, err := countOne(ctx, db, eventTriggerCountQuery, "Event Trigger")
	if err != nil {
		return result, err
	}
	if err := validateDedicatedFixtureDatabase(databaseName, userRelations, eventTriggers); err != nil {
		return result, err
	}
	result.DatabaseIdentityVerified = true
	result.NonDefaultDatabaseVerified = true
	result.UserRelationsBeforeFixture = userRelations
	result.EventTriggersBeforeFixture = eventTriggers

	absent, err := relationAbsent(ctx, db)
	if err != nil {
		return result, err
	}
	if !absent {
		return result, fmt.Errorf("安全のためツールを停止します: %sが既に存在します", fixtureObject)
	}
	result.FixtureObjectAbsentBeforeCreate = true
	result.GuardPassed = true
	return result, nil
}

func verifyDatabaseRestored(ctx context.Context, db *sql.DB) (fixtureCleanupSafetyEvidence, error) {
	result := fixtureCleanupSafetyEvidence{}
	userRelations, err := countOne(ctx, db, userRelationCountQuery, "終了時のuser relation")
	if err != nil {
		return result, err
	}
	result.UserRelationsAfterCleanup = userRelations
	eventTriggers, err := countOne(ctx, db, eventTriggerCountQuery, "終了時のEvent Trigger")
	if err != nil {
		return result, err
	}
	result.EventTriggersAfterCleanup = eventTriggers
	if err := validateFixtureDatabaseRestored(userRelations, eventTriggers); err != nil {
		return result, err
	}
	absent, err := relationAbsent(ctx, db)
	if err != nil {
		return result, err
	}
	if !absent {
		return result, fmt.Errorf("終了時にfixture objectが残っています")
	}
	result.FixtureObjectAbsentAfterCleanup = true
	result.Verified = true
	return result, nil
}

func newFixtureOwnershipMarker() (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("fixture ownership markerを生成できません: %w", err)
	}
	return fixtureOwnershipPrefix + hex.EncodeToString(random[:]), nil
}

func validateFixtureOwnershipMarker(marker string) error {
	if !strings.HasPrefix(marker, fixtureOwnershipPrefix) {
		return fmt.Errorf("fixture ownership markerのprefixが不正です")
	}
	suffix := strings.TrimPrefix(marker, fixtureOwnershipPrefix)
	if len(suffix) != 32 {
		return fmt.Errorf("fixture ownership markerの長さが不正です")
	}
	if _, err := hex.DecodeString(suffix); err != nil {
		return fmt.Errorf("fixture ownership markerのrandom部が16進数ではありません")
	}
	return nil
}

func verifyObservedFixtureOwnership(observed sql.NullString, expectedMarker string) error {
	if err := validateFixtureOwnershipMarker(expectedMarker); err != nil {
		return fmt.Errorf("期待ownership markerが不正です: %w", err)
	}
	if !observed.Valid || observed.String == "" {
		return fmt.Errorf("DB側ownership markerを観測できません")
	}
	if observed.String != expectedMarker {
		return fmt.Errorf("DB側ownership markerがtool作成時のmarkerと一致しません")
	}
	return nil
}

func createOwnedFixtureTable(ctx context.Context, db *sql.DB) (string, error) {
	marker, err := newFixtureOwnershipMarker()
	if err != nil {
		return "", err
	}
	if err := validateFixtureOwnershipMarker(marker); err != nil {
		return "", err
	}
	// markerは固定prefixと小文字16進数だけで構成するため、SQL literalへ埋め込める。
	commentSQL := fmt.Sprintf("COMMENT ON TABLE public.requests IS '%s';", marker)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("fixture作成transactionを開始できません: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, fixtureCreateSQL); err != nil {
		return "", fmt.Errorf("fixture tableを作成できません: %w", err)
	}
	if _, err := tx.ExecContext(ctx, commentSQL); err != nil {
		return "", fmt.Errorf("fixtureへownership markerを設定できません: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("fixture作成transactionを確定できません: %w", err)
	}
	return marker, nil
}

func removeOwnedFixtureTable(ctx context.Context, db *sql.DB, expectedMarker string) error {
	if err := validateFixtureOwnershipMarker(expectedMarker); err != nil {
		return fmt.Errorf("安全のためfixtureを削除しません: 期待ownership markerが不正です: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("fixture cleanup transactionを開始できません: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, fixtureLockSQL); err != nil {
		return fmt.Errorf("安全のためfixtureを削除しません: ownership確認用lockを取得できません: %w", err)
	}
	var observed sql.NullString
	if err := tx.QueryRowContext(ctx, fixtureMarkerQuery).Scan(&observed); err != nil {
		return fmt.Errorf("安全のためfixtureを削除しません: DB側ownership markerを確認できません: %w", err)
	}
	if err := verifyObservedFixtureOwnership(observed, expectedMarker); err != nil {
		return fmt.Errorf("安全のためfixtureを削除しません: %w", err)
	}
	if _, err := tx.ExecContext(ctx, fixtureDropSQL); err != nil {
		return fmt.Errorf("ownership確認済みfixtureを削除できません: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("fixture cleanup transactionを確定できません: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// 実行
// ---------------------------------------------------------------------------

func newCommandID() (string, error) {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("Command idを生成できません: %w", err)
	}
	return "cmd-" + hex.EncodeToString(random[:]), nil
}

func openPool(dsn string) (*sql.DB, error) {
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		return nil, err
	}
	// 一つの接続だけを使わせ、pg_backend_pidをpoolの識別として固定する。
	db.SetMaxOpenConns(1)
	return db, nil
}

func backendPID(ctx context.Context, db *sql.DB) (int64, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	var pid int64
	if err := conn.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
		return 0, err
	}
	return pid, nil
}

// insertOnPoolは、手書きの値渡しINSERTを一つの接続で実行し、その接続のpg_backend_pidを返す。
// pgxのsimple protocolでは引数はクライアント側で展開されるため、サーバー側の引数束縛とは主張しない。
func insertOnPool(db *sql.DB) insertFunc {
	return func(ctx context.Context, s submission) (int64, error) {
		conn, err := db.Conn(ctx)
		if err != nil {
			return 0, err
		}
		defer conn.Close()
		var pid int64
		if err := conn.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
			return 0, err
		}
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO public.requests (request_id, summary, status) VALUES ($1, $2, $3)`,
			s.RequestID, s.Summary, s.Status,
		); err != nil {
			return pid, err
		}
		return pid, nil
	}
}

func main() {
	out, err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoded, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
}

func run() (rcpt receipt, err error) {
	dsn, err := loadPostgresDSNInput(os.LookupEnv, os.ReadFile)
	if err != nil {
		return rcpt, err
	}
	if err := clearPostgresDSNEnvironment(); err != nil {
		return rcpt, err
	}

	// 後から同じsourceへ戻れるclean revisionに固定する。dirty treeの診断実行は証拠にしない。
	source, err := repositoryevidence.InspectStart()
	if err != nil {
		return rcpt, err
	}
	if !source.WorkingTreeClean {
		return rcpt, fmt.Errorf("working treeに未commit差分があるため、DBへ接続しません。commitしてから実行してください")
	}

	// DBへ接続する前に、overlay適用後の作業領域を検証する。
	workDir, err := os.MkdirTemp("", "tei-data-first-persistence-")
	if err != nil {
		return rcpt, err
	}
	defer os.RemoveAll(workDir)
	prepared, err := prepareWorkspace(".", filepath.Join(workDir, "workspace"))
	if err != nil {
		return rcpt, err
	}
	if !prepared.RequestTeiUnchanged {
		return rcpt, fmt.Errorf("overlayが%sを変更しています", requestSourceTEI)
	}
	loaded, err := workspace.Load(prepared.Root)
	if err != nil {
		return rcpt, fmt.Errorf("overlay適用後の作業領域を読み込めません: %w", err)
	}
	if n, codes := countValidationErrors(loaded.World); n != 0 {
		return rcpt, fmt.Errorf("overlay適用後の作業領域にValidation Errorが%d件あります: %v", n, codes)
	}
	world := loaded.World

	// N08用: realizeを取り除いた作業領域。
	strippedPrepared, err := prepareWorkspace(".", filepath.Join(workDir, "workspace-without-realize"), realizeFile)
	if err != nil {
		return rcpt, err
	}
	strippedLoaded, err := workspace.Load(strippedPrepared.Root)
	if err != nil {
		return rcpt, fmt.Errorf("realizeを除いた作業領域を読み込めません: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	dbA, err := openPool(dsn)
	if err != nil {
		return rcpt, err
	}
	defer dbA.Close()
	dbB, err := openPool(dsn)
	if err != nil {
		return rcpt, err
	}
	defer dbB.Close()
	if err := dbA.PingContext(ctx); err != nil {
		return rcpt, fmt.Errorf("PostgreSQL接続確認に失敗しました（接続情報は表示しません）")
	}
	if err := dbB.PingContext(ctx); err != nil {
		return rcpt, fmt.Errorf("PostgreSQL接続確認(読取り用)に失敗しました（接続情報は表示しません）")
	}

	safety, err := ensureDedicatedEmptyDatabase(ctx, dbA)
	if err != nil {
		return rcpt, err
	}

	resolver := driver.NewResolver(world)
	worldStages, target := preflightWorld(world)
	if !allResolved(worldStages) || target == nil {
		return rcpt, fmt.Errorf("DBへ書き込む前の世界の解決が完了しません: %+v", worldStages)
	}
	introspectA := func(c context.Context) (physical.SchemaSnapshot, error) {
		return postgresprovider.Provider{Catalog: postgresprovider.SQLCatalogReader{DB: dbA}}.Introspect(c, target.Instance)
	}
	fullPreflight := func() ([]stageResult, schemadiff.Diff) {
		schemaStage, diff := preflightSchema(ctx, world, target, introspectA)
		return append(append([]stageResult(nil), worldStages...), schemaStage), diff
	}

	adapters := &adapterSet{insert: insertOnPool(dbA)}

	// --- S02類似(a): 物理relationが存在しない。 ---
	absentPreflight, _ := fullPreflight()
	absentGate := submitIfReady(ctx, resolver, absentPreflight, adapters, submission{})
	absentNow, err := relationAbsent(ctx, dbB)
	if err != nil {
		return rcpt, err
	}
	rcpt.S02AnalogueAbsent = negativeEvidence{
		Name:                 "s02-analogue-relation-absent",
		Preflight:            absentPreflight,
		CommandRefusedByGate: absentGate.Refused,
		CommandDispatchCount: adapters.dispatched,
		RelationAbsentAfter:  &absentNow,
		Note:                 "Accepted Sourceの変更を伴うS02ではなく、物理schemaだけの不一致である",
	}
	if !absentGate.Refused || adapters.dispatched != 0 || !absentNow || findStage(absentPreflight, "physical-schema").Status != stageUnresolved {
		return rcpt, fmt.Errorf("S02類似(relation不在)の期待結果になりません: %+v", rcpt.S02AnalogueAbsent)
	}

	// --- fixtureの作成（手書きDDL）。 ---
	marker, err := createOwnedFixtureTable(ctx, dbA)
	if err != nil {
		return rcpt, err
	}
	fixtureOwned := true
	defer func() {
		if !fixtureOwned {
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if cerr := removeOwnedFixtureTable(cleanupCtx, dbA, marker); cerr != nil {
			fmt.Fprintf(os.Stderr, "fixture cleanup warning: %v\n", cerr)
		}
	}()

	baseline, _ := fullPreflight()
	if !allResolved(baseline) {
		return rcpt, fmt.Errorf("手書きDDLの作成後も物理schemaが望ましい形に一致しません: %+v", baseline)
	}

	// --- N08: realizeを取り除くとBindingを解決できず、Commandを実行しない。 ---
	strippedStages, _ := preflightWorld(strippedLoaded.World)
	strippedAdapters := &adapterSet{insert: insertOnPool(dbA)}
	strippedResolver := driver.NewResolver(strippedLoaded.World)
	strippedGate := submitIfReady(ctx, strippedResolver, append(strippedStages, findStage(baseline, "physical-schema")), strippedAdapters, submission{})
	_, directErr := strippedResolver.InvokeCapability(ctx, applicationID, capabilityID, strippedAdapters, submission{})
	driverRefused := directErr != nil
	rowsN08, err := countOne(ctx, dbB, `SELECT count(*) FROM public.requests`, "N08後の行数")
	if err != nil {
		return rcpt, err
	}
	strippedErrors, _ := countValidationErrors(strippedLoaded.World)
	rcpt.N08 = negativeEvidence{
		Name:                    "n08-missing-realize-binding",
		Preflight:               append(strippedStages, findStage(baseline, "physical-schema")),
		CommandRefusedByGate:    strippedGate.Refused,
		DriverRefusedDirectCall: &driverRefused,
		CommandDispatchCount:    strippedAdapters.dispatched,
		RowsAfter:               &rowsN08,
		Note:                    fmt.Sprintf("realizeを除いた作業領域のValidation Error件数は%dである。拒否の根拠として検査したのは、DriverのBinding解決と実行ゲートである", strippedErrors),
	}
	if !strippedGate.Refused || !driverRefused || strippedAdapters.dispatched != 0 || rowsN08 != 0 {
		return rcpt, fmt.Errorf("N08の期待結果になりません: %+v", rcpt.N08)
	}

	// --- S02類似(b): 列が欠けた物理schema。 ---
	if _, err := dbA.ExecContext(ctx, `ALTER TABLE public.requests DROP COLUMN summary`); err != nil {
		return rcpt, fmt.Errorf("列欠落の準備に失敗しました: %w", err)
	}
	driftPreflight, _ := fullPreflight()
	driftGate := submitIfReady(ctx, resolver, driftPreflight, adapters, submission{})
	rowsDrift, err := countOne(ctx, dbB, `SELECT count(*) FROM public.requests`, "列欠落後の行数")
	if err != nil {
		return rcpt, err
	}
	rcpt.S02AnalogueColumn = negativeEvidence{
		Name:                 "s02-analogue-column-missing",
		Preflight:            driftPreflight,
		CommandRefusedByGate: driftGate.Refused,
		CommandDispatchCount: adapters.dispatched,
		RowsAfter:            &rowsDrift,
		Note:                 "Accepted Sourceの変更を伴うS02ではなく、物理schemaだけの不一致である",
	}
	if !driftGate.Refused || adapters.dispatched != 0 || rowsDrift != 0 || findStage(driftPreflight, "physical-schema").Status != stageUnresolved {
		return rcpt, fmt.Errorf("S02類似(列欠落)の期待結果になりません: %+v", rcpt.S02AnalogueColumn)
	}
	if _, err := dbA.ExecContext(ctx, `ALTER TABLE public.requests ADD COLUMN summary text NOT NULL`); err != nil {
		return rcpt, fmt.Errorf("列の復元に失敗しました: %w", err)
	}

	// --- 正常系: 実行前確認 -> Command(pool A) -> 独立読取り(pool B)。 ---
	positivePreflight, _ := fullPreflight()
	if !allResolved(positivePreflight) {
		return rcpt, fmt.Errorf("列の復元後に物理schemaが一致しません: %+v", positivePreflight)
	}
	commandID, err := newCommandID()
	if err != nil {
		return rcpt, err
	}
	sub := submission{CommandID: commandID, RequestID: "req-" + commandID, Summary: "first persistence round trip", Status: "received"}
	gate := submitIfReady(ctx, resolver, positivePreflight, adapters, sub)
	if gate.Refused || gate.Err != nil || gate.Invocation == nil {
		return rcpt, fmt.Errorf("Commandを実行できません: refused=%v err=%v", gate.Refused, gate.Err)
	}
	payload, ok := gate.Invocation.Result.Payload.(commandPayload)
	if !ok || gate.Invocation.Result.Mode != capabilityinvoke.ResultOutput {
		return rcpt, fmt.Errorf("Command結果の形が想定と異なります: %+v", gate.Invocation.Result)
	}

	executorB := postgresquery.Executor{DB: dbB, Database: target.Definition}
	pidBefore, err := backendPID(ctx, dbB)
	if err != nil {
		return rcpt, err
	}
	var browse driver.BrowseQueryExecution
	observation, readErr := driver.ExecuteObservationRead(ctx, readIdentity, observationNature,
		func(c context.Context) (any, error) {
			executed, err := resolver.ExecuteBrowseQuery(c, applicationID, browsePatternID, executorB, nil)
			if err != nil {
				return nil, err
			}
			browse = executed
			return executed, nil
		})
	pidAfter, err := backendPID(ctx, dbB)
	if err != nil {
		return rcpt, err
	}
	matches := false
	if readErr == nil {
		matches, err = rowMatchesSubmission(browse.Result.Mapped, sub)
		if err != nil {
			return rcpt, err
		}
	}
	status := classifyObservation(true, readErr, matches)
	rcpt.Positive = positiveEvidence{
		Preflight:                 positivePreflight,
		AllStagesResolvedAfterDDL: allResolved(baseline),
		CommandID:                 payload.CommandID,
		BindingIdentity:           gate.Invocation.Binding.Binding.Identity(),
		ResultMode:                string(gate.Invocation.Result.Mode),
		BackendPIDCommandPool:     payload.BackendPID,
		ReadIdentity:              observation.ReadIdentity,
		ReadEvidenceNature:        string(observation.Nature),
		SubmittedCommandID:        sub.CommandID,
		BackendPIDReadPoolBefore:  pidBefore,
		BackendPIDReadPoolAfter:   pidAfter,
		BackendPIDsDiffer:         payload.BackendPID != pidBefore && payload.BackendPID != pidAfter,
		RowsRead:                  len(browse.Result.Mapped.Rows),
		RowMatchesSubmission:      matches,
		MappedSemanticAttributes:  mappedAttributeIDs(browse.Result.Mapped),
		ObservationStatus:         string(status),
		DatabaseIdentityFromEnv:   fmt.Sprintf("environment %s: %s.%s -> %s", environmentID, applicationID, datasourceAlias, target.Instance.ID),
		DatabaseProviderFromEnv:   string(target.Instance.Provider),
		ParameterValuesPersisted:  false,
	}
	if status != observationObserved || pidBefore != pidAfter || !rcpt.Positive.BackendPIDsDiffer || adapters.dispatched != 1 {
		return rcpt, fmt.Errorf("正常系の期待結果になりません: status=%s pidBefore=%d pidAfter=%d pidA=%d dispatched=%d", status, pidBefore, pidAfter, payload.BackendPID, adapters.dispatched)
	}

	// --- N11(a): Commandをcommitした後、読取り側のpoolが使えない。 ---
	n11a, err := runN11(ctx, resolver, target, positivePreflight, adapters, dbB, "n11-closed-read-pool", func(c context.Context) (*sql.DB, error) {
		closed, err := openPool(dsn)
		if err != nil {
			return nil, err
		}
		if err := closed.Close(); err != nil {
			return nil, err
		}
		return closed, nil
	}, "読取り用poolを閉じてから実行する", nil)
	if err != nil {
		return rcpt, err
	}
	rcpt.N11ClosedPool = n11a

	// --- N11(b): Commandをcommitした後、物理tableが消える。 ---
	n11b, err := runN11(ctx, resolver, target, positivePreflight, adapters, dbB, "n11-dropped-table", func(c context.Context) (*sql.DB, error) {
		return dbB, nil
	}, "所有印を確認してfixture tableを削除してから読む", func(c context.Context) error {
		if err := removeOwnedFixtureTable(c, dbA, marker); err != nil {
			return err
		}
		fixtureOwned = false
		return nil
	})
	if err != nil {
		return rcpt, err
	}
	rcpt.N11DroppedTable = n11b

	cleanup, err := verifyDatabaseRestored(ctx, dbA)
	if err != nil {
		return rcpt, err
	}

	rcpt.EvidenceNature = evidenceNature
	rcpt.RepositoryRevision = source.RepositoryRevision
	rcpt.WorkingTreeClean = source.WorkingTreeClean
	rcpt.Scope = fixedScope()
	rcpt.Overlay = overlayEvidence{RequestTeiSha256Unchanged: prepared.RequestTeiUnchanged, ValidationErrors: 0, PluginSourceCopiedAtRun: prepared.PluginCopied}
	rcpt.FixtureObject = fixtureObject
	rcpt.FixtureOwnedByTool = true
	rcpt.FixtureDatabase = safety
	rcpt.FixtureCleanup = cleanup

	postRun, err := repositoryevidence.InspectPost(source.RepositoryRevision)
	if err != nil {
		return rcpt, err
	}
	rcpt.PostRunRepository = postRun
	rcpt.ReproducibleRevision = repositoryevidence.IsReproducible(source, postRun)
	if !rcpt.ReproducibleRevision {
		return rcpt, errors.New("実行の前後で単一のclean revisionに固定できません")
	}
	return rcpt, nil
}

// runN11は、Commandを一件commitした後に読取りを失敗させ、観測済みと報告されないことを確認する。
// breakはCommand後・読取り前に追加で行う破壊操作である（nilなら何もしない）。
func runN11(
	ctx context.Context,
	resolver driver.Resolver,
	target *databaseTarget,
	preflight []stageResult,
	adapters *adapterSet,
	committedCheckPool *sql.DB,
	name string,
	readPool func(context.Context) (*sql.DB, error),
	mechanism string,
	breakAfterCommit func(context.Context) error,
) (n11Evidence, error) {
	evidence := n11Evidence{Name: name, ReadFailureMechanism: mechanism}
	commandID, err := newCommandID()
	if err != nil {
		return evidence, err
	}
	sub := submission{CommandID: commandID, RequestID: "req-" + commandID, Summary: "post-commit read failure", Status: "received"}
	dispatchedBefore := adapters.dispatched
	gate := submitIfReady(ctx, resolver, preflight, adapters, sub)
	if gate.Refused || gate.Err != nil || gate.Invocation == nil {
		return evidence, fmt.Errorf("%s: Commandを実行できません: refused=%v err=%v", name, gate.Refused, gate.Err)
	}
	evidence.CommandResultPresent = true
	evidence.CommandID = sub.CommandID
	evidence.CommandDispatched = adapters.dispatched == dispatchedBefore+1

	var rows int
	if err := committedCheckPool.QueryRowContext(ctx, `SELECT count(*) FROM public.requests WHERE request_id = $1`, sub.RequestID).Scan(&rows); err != nil {
		return evidence, fmt.Errorf("commit済み行数を確認できません: %w", err)
	}
	evidence.RowsCommittedViaPoolB = &rows
	if rows != 1 {
		return evidence, fmt.Errorf("%s: commit済みのはずの行が独立poolから見えません: %d", name, rows)
	}

	if breakAfterCommit != nil {
		if err := breakAfterCommit(ctx); err != nil {
			return evidence, err
		}
	}
	pool, err := readPool(ctx)
	if err != nil {
		return evidence, err
	}
	var browse driver.BrowseQueryExecution
	_, readErr := driver.ExecuteObservationRead(ctx, readIdentity, observationNature,
		func(c context.Context) (any, error) {
			executed, err := resolver.ExecuteBrowseQuery(c, applicationID, browsePatternID, postgresquery.Executor{DB: pool, Database: target.Definition}, nil)
			if err != nil {
				return nil, err
			}
			browse = executed
			return executed, nil
		})
	matches := false
	if readErr == nil {
		matches, _ = rowMatchesSubmission(browse.Result.Mapped, sub)
	}
	status := classifyObservation(evidence.CommandResultPresent, readErr, matches)
	evidence.ObservationErrorSeen = readErr != nil
	evidence.ObservationStatus = string(status)
	evidence.NeverReportedAsObserved = status != observationObserved
	if readErr == nil || status != observationReadFailed {
		return evidence, fmt.Errorf("%s: 読取り失敗が観測失敗として報告されません: status=%s", name, status)
	}
	return evidence, nil
}

func findStage(stages []stageResult, name string) stageResult {
	for _, s := range stages {
		if s.Stage == name {
			return s
		}
	}
	return stageResult{Stage: name, Status: stageUnresolved, Detail: "段階が実行されていません"}
}
