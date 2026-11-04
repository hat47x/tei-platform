package migration_test

import (
	"strings"
	"testing"

	"tei/internal/database/migration"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

// #2561 の正式採用（PostgreSQL以外の提供元の明示拒否）を、提供元ごとに固定する。
// 移行SQLの描画はPostgreSQLだけを実装しており、他の提供元は空の結果とエラーを返す。
func TestRenderRefusesEveryNonPostgresProvider(t *testing.T) {
	for _, provider := range []environment.DatabaseProvider{
		environment.ProviderGeneric,
		environment.ProviderOracle,
		environment.ProviderMySQL,
		environment.ProviderMariaDB,
		environment.ProviderSQLServer,
		environment.ProviderSQLite,
		environment.ProviderDB2,
		environment.ProviderClickHouse,
	} {
		t.Run(string(provider), func(t *testing.T) {
			statements, err := migration.Render(provider, schemadiff.Diff{})
			if err == nil {
				t.Fatalf("provider %q was accepted for migration rendering", provider)
			}
			if !strings.Contains(err.Error(), "not implemented for provider") {
				t.Fatalf("provider %q error=%v, want an explicit unsupported-provider error", provider, err)
			}
			if !strings.Contains(err.Error(), string(provider)) {
				t.Fatalf("provider %q is not named in the error: %v", provider, err)
			}
			if len(statements) != 0 {
				t.Fatalf("provider %q returned statements despite refusal: %v", provider, statements)
			}
		})
	}
}
