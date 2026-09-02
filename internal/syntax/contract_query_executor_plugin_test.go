package syntax_test

import (
	"testing"

	"tei/internal/environment"
	"tei/internal/plugin"
	"tei/internal/syntax"
)

func TestParseContractQueryExecutorCapability(t *testing.T) {
	definition, err := syntax.NewParser(`plugin tei.database.oracle v1.0.0 {
        runtime builtin
        provides database-provider database.oracle {
            provider oracle
        }
        provides contract-query-executor database.query.oracle {
            provider oracle
        }
    }`).ParsePlugin()
	if err != nil {
		t.Fatalf("Contract Query executor Capabilityを読み込めません: %v", err)
	}

	if len(definition.Provides) != 2 {
		t.Fatalf("Capability数が期待値と異なります: %#v", definition.Provides)
	}

	var queryExecutor *plugin.Capability
	for i := range definition.Provides {
		capability := &definition.Provides[i]
		if capability.Kind == plugin.CapabilityContractQueryExecutor {
			queryExecutor = capability
			break
		}
	}
	if queryExecutor == nil {
		t.Fatalf("contract-query-executor Capabilityがありません: %#v", definition.Provides)
	}
	if queryExecutor.ID != "database.query.oracle" {
		t.Fatalf("Contract Query executor ID=%q", queryExecutor.ID)
	}
	if !queryExecutor.SupportsDatabaseProvider(environment.ProviderOracle) {
		t.Fatalf("Oracle provider supportを保持できていません: %#v", queryExecutor.DatabaseProviders)
	}
}

func TestContractQueryExecutorIsIndependentFromDatabaseProviderCapability(t *testing.T) {
	definition, err := syntax.NewParser(`plugin tei.database.oracle v1.0.0 {
        runtime builtin
        provides database-provider database.oracle {
            provider oracle
        }
    }`).ParsePlugin()
	if err != nil {
		t.Fatal(err)
	}

	for _, capability := range definition.Provides {
		if capability.Kind == plugin.CapabilityContractQueryExecutor {
			t.Fatalf("database-providerだけの宣言からContract Query executor能力を暗黙に補っています: %#v", capability)
		}
	}
}
