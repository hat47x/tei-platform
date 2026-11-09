package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/database/physical"
	postgresprovider "tei/internal/database/providers/postgres"
	dbrealization "tei/internal/database/realization"
	"tei/internal/database/schemadiff"
	"tei/internal/driver"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/semantic/capabilityinvoke"
)

var (
	applicationID    = core.MustCanonicalID("order_entry")
	commandPatternID = core.MustCanonicalID("order_entry.submit_order")
	capabilityID     = core.MustCanonicalID("sales.order.submit")
	environmentID    = core.MustCanonicalID("development")
	datasourceAlias  = core.LocalName("sales")
	bindingIdentity  = realization.BindingIdentity(applicationID, capabilityID)
)

type stageStatus string

const (
	stageResolved    stageStatus = "resolved"
	stageUnresolved  stageStatus = "unresolved"
	stageUnsupported stageStatus = "unsupported"
)

type stageResult struct {
	Stage  string      `json:"stage"`
	Status stageStatus `json:"status"`
	Detail string      `json:"detail,omitempty"`
}

type databaseTarget struct {
	Alias      core.LocalName
	Definition core.CanonicalID
	Instance   environment.DatabaseInstance
}

// preflightWorldは、DBへ接続せずWorldだけで解決できる段階を判定する。
// CommandのBindingはDriverの解決処理をそのまま使い、Database識別とproviderは
// 環境のbindingから導出する。呼び出し側が指定した接続先を信用しない。
func preflightWorld(world ir.World) ([]stageResult, *databaseTarget) {
	resolver := driver.NewResolver(world)
	var stages []stageResult
	if _, err := resolver.ResolveCommandBinding(applicationID, commandPatternID); err != nil {
		stages = append(stages, stageResult{Stage: "command-pattern-binding", Status: stageUnresolved, Detail: err.Error()})
	} else {
		stages = append(stages, stageResult{Stage: "command-pattern-binding", Status: stageResolved})
	}
	target, stage := resolveDatabaseTarget(world)
	return append(stages, stage), target
}

func resolveDatabaseTarget(world ir.World) (*databaseTarget, stageResult) {
	const stage = "environment-database-binding"
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
		if b.Application == applicationID && b.Alias == datasourceAlias {
			matches = append(matches, b)
		}
	}
	if len(matches) != 1 {
		return nil, stageResult{Stage: stage, Status: stageUnresolved, Detail: fmt.Sprintf("environment %sの%s.%sのbindingが%d件です", env.ID, applicationID, datasourceAlias, len(matches))}
	}
	instance, ok := env.InstanceByID(matches[0].Instance)
	if !ok {
		return nil, stageResult{Stage: stage, Status: stageUnresolved, Detail: fmt.Sprintf("Database instance %sが定義されていません", matches[0].Instance)}
	}
	if instance.Provider != environment.ProviderPostgres {
		return nil, stageResult{Stage: stage, Status: stageUnsupported, Detail: fmt.Sprintf("providerは%sです。このツールが実行できるのはpostgresだけです", instance.Provider)}
	}
	return &databaseTarget{Alias: datasourceAlias, Definition: instance.Definition, Instance: instance}, stageResult{Stage: stage, Status: stageResolved}
}

func desiredSchema(world ir.World, target *databaseTarget) (physical.SchemaSnapshot, error) {
	desired, err := dbrealization.RealizeDesired(world.Persistence, string(target.Definition), target.Instance.Provider)
	if err != nil {
		return physical.SchemaSnapshot{}, fmt.Errorf("望ましい物理schemaを導出できません: %w", err)
	}
	if len(desired.Tables) == 0 {
		return physical.SchemaSnapshot{}, fmt.Errorf("Persistence Mappingから導出されたtableがありません")
	}
	return desired, nil
}

// renderCreateDDLは、Persistence Mappingから導いた望ましい物理schemaを空のschemaと比べ、
// 既存のPostgreSQL Providerの描画でDDLにする。手書きDDLではない。
func renderCreateDDL(world ir.World, target *databaseTarget) ([]string, error) {
	desired, err := desiredSchema(world, target)
	if err != nil {
		return nil, err
	}
	diff, err := schemadiff.Compare(desired, physical.SchemaSnapshot{Database: target.Definition})
	if err != nil {
		return nil, fmt.Errorf("空schemaとの比較に失敗しました: %w", err)
	}
	return postgresprovider.RenderMigrationSQL(diff)
}

type introspectFunc func(context.Context) (physical.SchemaSnapshot, error)

// preflightSchemaは、望ましい物理schemaと観測した実物をschemadiff.Compareで比べる。
// 変更または未確定の比較が一つでもあれば、Commandを実行できる状態ではない。
// これは物理schemaの不一致(S02の物理側)だけを見る。Source revisionの不一致は別に拒否する。
func preflightSchema(ctx context.Context, world ir.World, target *databaseTarget, introspect introspectFunc) stageResult {
	const stage = "physical-schema"
	if target == nil {
		return stageResult{Stage: stage, Status: stageUnresolved, Detail: "Database識別が導出されていないため、物理schemaを比較できません"}
	}
	desired, err := desiredSchema(world, target)
	if err != nil {
		return stageResult{Stage: stage, Status: stageUnsupported, Detail: err.Error()}
	}
	actual, err := introspect(ctx)
	if err != nil {
		return stageResult{Stage: stage, Status: stageUnresolved, Detail: "物理schemaを観測できません"}
	}
	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		return stageResult{Stage: stage, Status: stageUnresolved, Detail: "物理schemaを比較できません"}
	}
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

func findStage(stages []stageResult, name string) stageResult {
	for _, s := range stages {
		if s.Stage == name {
			return s
		}
	}
	return stageResult{Stage: name, Status: stageUnresolved, Detail: "段階が実行されていません"}
}

// orderStoreは、submitがDBへ与える唯一の更新である。実装は実PostgreSQL(pgStore)で、
// 単体試験ではこの境界の偽物を使って、書込みが呼ばれたか(dispatch)を数える。
type orderStore interface {
	// SubmitOrderは、pendingの注文をsubmittedにし、その接続のpg_backend_pidを返す。
	SubmitOrder(ctx context.Context, orderID int64) (backendPID int64, err error)
}

// effectGrantは、書込みの直前に呼ぶ現在の効果許可の確認である。
// 返すerrorは「いま許可されていない」を表す。モデルの宣言は許可ではない。
type effectGrant func(context.Context) error

// submitAdapterは、解決済みBindingのcustom-go adapterである。許可確認は更新の直前にあり、
// 拒否されたらstoreを呼ばない(書込みdispatch 0)。DB障害はerrorで、業務結果へ変換しない。
type submitAdapter struct {
	store    orderStore
	grant    effectGrant
	adapters *adapterSet
}

func (a submitAdapter) Invoke(ctx context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	values, ok := input.(map[string]any)
	if !ok {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("想定外の入力型です: %T", input)
	}
	orderID, ok := values["orderId"].(int64)
	if !ok {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("orderIdがint64ではありません")
	}
	a.adapters.invoked++
	if a.grant != nil {
		if err := a.grant(ctx); err != nil {
			return capabilityinvoke.BusinessResult{}, fmt.Errorf("EFFECT_GRANT_UNAVAILABLE")
		}
	}
	a.adapters.writeAttempts++
	pid, err := a.store.SubmitOrder(ctx, orderID)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("注文の更新を完了できません: %w", err)
	}
	a.adapters.backendPID = pid
	return capabilityinvoke.BusinessResult{
		Capability: capability,
		Mode:       capabilityinvoke.ResultOutput,
		Payload:    map[string]any{"accepted": true, "orderId": orderID},
	}, nil
}

// adapterSetは、解決済みBindingのidentityが期待どおりのときだけadapterを返す。
type adapterSet struct {
	store         orderStore
	grant         effectGrant
	invoked       int // Driverがadapterを呼んだ回数
	writeAttempts int // storeの更新を試みた回数。許可拒否のときは増えない
	backendPID    int64
}

var _ driver.BindingAdapterResolver = (*adapterSet)(nil)

func (s *adapterSet) ResolveAdapter(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
	if binding.Identity() != bindingIdentity {
		return nil, fmt.Errorf("想定外のBindingです: %s", binding.Identity())
	}
	return submitAdapter{store: s.store, grant: s.grant, adapters: s}, nil
}

type gateResult struct {
	Refused    bool
	Reasons    []stageResult
	Invocation *driver.CapabilityInvocation
	Err        error
}

// submitIfReadyは、実行前の段階がすべてresolvedのときだけDriver経由でCommandを実行する。
// 一つでも未解決・未対応があれば、adapterを呼ばずに拒否する。
func submitIfReady(ctx context.Context, resolver driver.Resolver, preflight []stageResult, adapters *adapterSet, orderID int64) gateResult {
	if !allResolved(preflight) {
		return gateResult{Refused: true, Reasons: unresolvedStages(preflight)}
	}
	invocation, err := resolver.InvokeCapability(ctx, applicationID, capabilityID, adapters, map[string]any{"orderId": orderID})
	if err != nil {
		return gateResult{Err: err}
	}
	return gateResult{Invocation: &invocation}
}
