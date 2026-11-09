package sharedprojection

import (
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/realization"
)

// Package sharedprojection translates already-resolved Driver rows and
// Environment bindings to read-only target presentation values. It does not
// resolve Patterns, authorize commands, manage credentials, or own meaning.
// Both Application targets may import this package without importing each
// other's business runners.
type SemanticRowProjection struct {
	Attributes []SemanticValueProjection `json:"attributes"`
}

type SemanticValueProjection struct {
	Attribute string `json:"attribute"`
	Value     any    `json:"value"`
}

type RuntimeBindingProjection struct {
	Pattern            string `json:"pattern"`
	Environment        string `json:"environment,omitempty"`
	Instance           string `json:"instance,omitempty"`
	Provider           string `json:"provider,omitempty"`
	ExecutorCapability string `json:"executorCapability,omitempty"`
	ContractQuery      string `json:"contractQuery,omitempty"`
	Datasource         string `json:"datasource,omitempty"`
	Database           string `json:"database,omitempty"`
	Access             string `json:"access,omitempty"`
	Note               string `json:"note,omitempty"`
}

func ProjectEnvironmentBinding(
	pattern string,
	environmentID core.CanonicalID,
	executor realization.ResolvedContractQueryExecutor,
) RuntimeBindingProjection {
	return RuntimeBindingProjection{
		Pattern:            pattern,
		Environment:        string(environmentID),
		Instance:           string(executor.Instance.ID),
		Provider:           string(executor.Instance.Provider),
		ExecutorCapability: string(executor.Capability.Capability.ID),
		ContractQuery:      executor.Binding.Target.Identity(),
	}
}

func ProjectMappedRows(source queryresult.MappedResult) ([]SemanticRowProjection, error) {
	result := make([]SemanticRowProjection, len(source.Rows))
	for rowIndex, row := range source.Rows {
		if len(row) != len(source.Columns) {
			return nil, fmt.Errorf(
				"mapped result row %d has %d values for %d columns",
				rowIndex,
				len(row),
				len(source.Columns),
			)
		}
		projected := SemanticRowProjection{}
		for columnIndex, column := range source.Columns {
			projected.Attributes = append(projected.Attributes, SemanticValueProjection{
				Attribute: string(column.Attribute),
				Value:     row[columnIndex],
			})
		}
		result[rowIndex] = projected
	}
	return result, nil
}

func ProjectContractRows(source []driver.ContractQueryRow) []SemanticRowProjection {
	result := make([]SemanticRowProjection, len(source))
	for rowIndex, row := range source {
		keys := make([]string, 0, len(row.Attributes))
		values := make(map[string]any, len(row.Attributes))
		for attribute, value := range row.Attributes {
			key := string(attribute)
			keys = append(keys, key)
			values[key] = value
		}
		sort.Strings(keys)
		for _, key := range keys {
			result[rowIndex].Attributes = append(result[rowIndex].Attributes, SemanticValueProjection{
				Attribute: key,
				Value:     values[key],
			})
		}
	}
	return result
}
