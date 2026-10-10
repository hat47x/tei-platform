package main

import (
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/realization"
)

// The projection types and translation helpers are shared by order_entry and
// repair_intake; no Application-specific runner, authentication, or business
// operation belongs in this file. Extracting them is a prerequisite for
// compile-time source selection, not yet separate package/binary evidence.
type semanticRowProjection struct {
	Attributes []semanticValueProjection `json:"attributes"`
}

type semanticValueProjection struct {
	Attribute string `json:"attribute"`
	Value     any    `json:"value"`
}

type runtimeBindingProjection struct {
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

func projectEnvironmentBinding(
	pattern string,
	environmentID core.CanonicalID,
	executor realization.ResolvedContractQueryExecutor,
) runtimeBindingProjection {
	return runtimeBindingProjection{
		Pattern:            pattern,
		Environment:        string(environmentID),
		Instance:           string(executor.Instance.ID),
		Provider:           string(executor.Instance.Provider),
		ExecutorCapability: string(executor.Capability.Capability.ID),
		ContractQuery:      executor.Binding.Target.Identity(),
	}
}

func projectMappedRows(source queryresult.MappedResult) ([]semanticRowProjection, error) {
	result := make([]semanticRowProjection, len(source.Rows))
	for rowIndex, row := range source.Rows {
		if len(row) != len(source.Columns) {
			return nil, fmt.Errorf(
				"mapped result row %d has %d values for %d columns",
				rowIndex,
				len(row),
				len(source.Columns),
			)
		}
		projected := semanticRowProjection{}
		for columnIndex, column := range source.Columns {
			projected.Attributes = append(projected.Attributes, semanticValueProjection{
				Attribute: string(column.Attribute),
				Value:     row[columnIndex],
			})
		}
		result[rowIndex] = projected
	}
	return result, nil
}

func projectContractRows(source []driver.ContractQueryRow) []semanticRowProjection {
	result := make([]semanticRowProjection, len(source))
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
			result[rowIndex].Attributes = append(result[rowIndex].Attributes, semanticValueProjection{
				Attribute: key,
				Value:     values[key],
			})
		}
	}
	return result
}

