package driver_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/query"
)

// countingBrowseExecutor records every call. The negative tests assert zero calls:
// a refusal must happen before any Executor can reach a database.
type countingBrowseExecutor struct {
	calls    int
	provider environment.DatabaseProvider // empty: does not declare a provider
	gotSrc   string
}

func (c *countingBrowseExecutor) ExecuteBrowseQuery(
	_ context.Context,
	q query.EmbeddedSQL,
	_ application.DatabaseRequirement,
	_ map[string]any,
) (driver.BrowseQueryResult, error) {
	c.calls++
	c.gotSrc = q.Source
	columns := make([]string, 0, len(q.ResultColumns))
	row := make([]any, 0, len(q.ResultColumns))
	for _, column := range q.ResultColumns {
		columns = append(columns, column.Column)
		row = append(row, "x")
	}
	return driver.BrowseQueryResult{Raw: queryresult.RawResult{Columns: columns, Rows: [][]any{row}}}, nil
}

type providerDeclaringExecutor struct {
	*countingBrowseExecutor
}

func (p providerDeclaringExecutor) DatabaseProvider() environment.DatabaseProvider {
	return p.provider
}

var (
	browseEnvDev = core.MustCanonicalID("development")
	browseApp    = core.MustCanonicalID("order_entry")
	browsePat    = core.MustCanonicalID("order_entry.recent_orders")
	browseInst   = core.MustCanonicalID("sales_core.dev")
)

// worldWithEnvironments returns a copy of the Reference World whose Environments are
// edited by mutate. The shared World is never modified.
func worldWithEnvironments(t *testing.T, mutate func([]environment.Environment) []environment.Environment) ir.World {
	t.Helper()
	world := referenceWorld(t)
	copied := make([]environment.Environment, len(world.Environments))
	for i, env := range world.Environments {
		env.Bindings = append([]environment.ApplicationDatabaseBinding(nil), env.Bindings...)
		env.Instances = append([]environment.DatabaseInstance(nil), env.Instances...)
		copied[i] = env
	}
	world.Environments = mutate(copied)
	return world
}

func devEnvironment(envs []environment.Environment) *environment.Environment {
	for i := range envs {
		if envs[i].ID == browseEnvDev {
			return &envs[i]
		}
	}
	return nil
}

func TestBrowseQueryInEnvironmentRunsOnlyTheExecutorRegisteredForTheResolvedInstance(t *testing.T) {
	resolver := driver.NewResolver(referenceWorld(t))
	exec := &countingBrowseExecutor{}
	registry := driver.BrowseQueryExecutorMap{
		{Environment: browseEnvDev, Instance: browseInst}: exec,
	}

	got, err := resolver.ExecuteBrowseQueryInEnvironment(
		context.Background(), browseEnvDev, browseApp, browsePat, registry, map[string]any{"from": "x"})
	if err != nil {
		t.Fatalf("実行できません: %v", err)
	}
	if exec.calls != 1 {
		t.Fatalf("Executor呼出=%d、期待=1", exec.calls)
	}
	if got.Environment != browseEnvDev || got.Instance.ID != browseInst ||
		got.Instance.Definition != "sales_core" || got.Instance.Provider != environment.ProviderPostgres {
		t.Fatalf("束縛の宣言側の事実が違います: %+v", got)
	}
	sum := sha256.Sum256([]byte(exec.gotSrc))
	if got.QuerySourceSHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("digestはExecutorが実際に受け取ったbytesのものであるべきです: %s", got.QuerySourceSHA256)
	}
	if len(got.Execution.Result.Mapped.Rows) != 1 {
		t.Fatalf("既存のMapが適用されていません: %+v", got.Execution.Result.Mapped)
	}
}

func TestBrowseQueryInEnvironmentRefusesBeforeCallingAnyExecutor(t *testing.T) {
	good := func(envs []environment.Environment) []environment.Environment { return envs }
	cases := []struct {
		name      string
		mutate    func([]environment.Environment) []environment.Environment
		env       core.CanonicalID
		registry  func(*countingBrowseExecutor) driver.BrowseQueryExecutorRegistry
		wantInErr string
	}{
		{
			name: "Environment不明", mutate: good, env: core.MustCanonicalID("staging"),
			registry: func(e *countingBrowseExecutor) driver.BrowseQueryExecutorRegistry {
				return mapFor(browseEnvDev, browseInst, e)
			},
			wantInErr: "見つかりません",
		},
		{
			name: "束縛なし",
			mutate: func(envs []environment.Environment) []environment.Environment {
				dev := devEnvironment(envs)
				var kept []environment.ApplicationDatabaseBinding
				for _, b := range dev.Bindings {
					if b.Alias != "sales" {
						kept = append(kept, b)
					}
				}
				dev.Bindings = kept
				return envs
			},
			env: browseEnvDev,
			registry: func(e *countingBrowseExecutor) driver.BrowseQueryExecutorRegistry {
				return mapFor(browseEnvDev, browseInst, e)
			},
			wantInErr: "Bindingがありません",
		},
		{
			name: "束縛が二重",
			mutate: func(envs []environment.Environment) []environment.Environment {
				dev := devEnvironment(envs)
				for _, b := range dev.Bindings {
					if b.Alias == "sales" {
						dev.Bindings = append(dev.Bindings, b)
						break
					}
				}
				return envs
			},
			env: browseEnvDev,
			registry: func(e *countingBrowseExecutor) driver.BrowseQueryExecutorRegistry {
				return mapFor(browseEnvDev, browseInst, e)
			},
			wantInErr: "複数",
		},
		{
			name: "Definitionが別DB",
			mutate: func(envs []environment.Environment) []environment.Environment {
				dev := devEnvironment(envs)
				for i := range dev.Instances {
					if dev.Instances[i].ID == browseInst {
						dev.Instances[i].Definition = "master_data"
					}
				}
				return envs
			},
			env: browseEnvDev,
			registry: func(e *countingBrowseExecutor) driver.BrowseQueryExecutorRegistry {
				return mapFor(browseEnvDev, browseInst, e)
			},
			wantInErr: "target Database",
		},
		{
			name: "Providerが別",
			mutate: func(envs []environment.Environment) []environment.Environment {
				dev := devEnvironment(envs)
				for i := range dev.Instances {
					if dev.Instances[i].ID == browseInst {
						dev.Instances[i].Provider = environment.ProviderOracle
					}
				}
				return envs
			},
			env: browseEnvDev,
			registry: func(e *countingBrowseExecutor) driver.BrowseQueryExecutorRegistry {
				return mapFor(browseEnvDev, browseInst, e)
			},
			wantInErr: "dialect",
		},
		{
			name: "別InstanceのExecutorしか無い", mutate: good, env: browseEnvDev,
			registry: func(e *countingBrowseExecutor) driver.BrowseQueryExecutorRegistry {
				return mapFor(browseEnvDev, core.MustCanonicalID("sales_core.prod"), e)
			},
			wantInErr: "Registryにありません",
		},
		{
			name: "別EnvironmentのExecutorしか無い", mutate: good, env: browseEnvDev,
			registry: func(e *countingBrowseExecutor) driver.BrowseQueryExecutorRegistry {
				return mapFor(core.MustCanonicalID("production"), browseInst, e)
			},
			wantInErr: "Registryにありません",
		},
		{
			name: "Registryがnil", mutate: good, env: browseEnvDev,
			registry:  func(*countingBrowseExecutor) driver.BrowseQueryExecutorRegistry { return nil },
			wantInErr: "Registry",
		},
		{
			name: "Executorが別Providerを申告", mutate: good, env: browseEnvDev,
			registry: func(e *countingBrowseExecutor) driver.BrowseQueryExecutorRegistry {
				e.provider = environment.ProviderOracle
				return mapFor(browseEnvDev, browseInst, providerDeclaringExecutor{e})
			},
			wantInErr: "が一致しません",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolver := driver.NewResolver(worldWithEnvironments(t, tc.mutate))
			exec := &countingBrowseExecutor{}
			_, err := resolver.ExecuteBrowseQueryInEnvironment(
				context.Background(), tc.env, browseApp, browsePat, tc.registry(exec), map[string]any{"from": "x"})
			if err == nil {
				t.Fatal("拒否されるべき構成が受理されました")
			}
			if !strings.Contains(err.Error(), tc.wantInErr) {
				t.Fatalf("理由が違います: %v（期待: %q を含む）", err, tc.wantInErr)
			}
			if exec.calls != 0 {
				t.Fatalf("拒否の前にExecutorが%d回呼ばれました", exec.calls)
			}
		})
	}
}

func TestBrowseQueryInEnvironmentAcceptsExecutorDeclaringTheMatchingProvider(t *testing.T) {
	resolver := driver.NewResolver(referenceWorld(t))
	exec := &countingBrowseExecutor{provider: environment.ProviderPostgres}
	registry := mapFor(browseEnvDev, browseInst, providerDeclaringExecutor{exec})
	if _, err := resolver.ExecuteBrowseQueryInEnvironment(
		context.Background(), browseEnvDev, browseApp, browsePat, registry, map[string]any{"from": "x"}); err != nil {
		t.Fatal(err)
	}
	if exec.calls != 1 {
		t.Fatalf("Executor呼出=%d", exec.calls)
	}
}

func TestBrowseQueryInEnvironmentIsRefusedForAWorkspaceWithoutEnvironments(t *testing.T) {
	resolver := driver.NewResolver(worldWithEnvironments(t, func([]environment.Environment) []environment.Environment { return nil }))
	exec := &countingBrowseExecutor{}
	_, err := resolver.ExecuteBrowseQueryInEnvironment(
		context.Background(), browseEnvDev, browseApp, browsePat, mapFor(browseEnvDev, browseInst, exec), map[string]any{"from": "x"})
	if err == nil || exec.calls != 0 {
		t.Fatalf("Environmentが無いWorkspaceは拒否されるべきです: err=%v calls=%d", err, exec.calls)
	}
}

func mapFor(env, instance core.CanonicalID, executor driver.BrowseQueryExecutor) driver.BrowseQueryExecutorRegistry {
	return driver.BrowseQueryExecutorMap{{Environment: env, Instance: instance}: executor}
}
