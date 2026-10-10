package main

import (
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/realization"
	"tei/tools/reference-first-web-target/sharedprojection"
)

// Aliases preserve the existing combined First Web Target JSON shapes and
// in-package callers while using one standalone, reusable Go package.
// This is a source dependency boundary, not a second semantic authority.
type semanticRowProjection = sharedprojection.SemanticRowProjection
type semanticValueProjection = sharedprojection.SemanticValueProjection
type runtimeBindingProjection = sharedprojection.RuntimeBindingProjection

func projectEnvironmentBinding(
	pattern string,
	environmentID core.CanonicalID,
	executor realization.ResolvedContractQueryExecutor,
) runtimeBindingProjection {
	return sharedprojection.ProjectEnvironmentBinding(pattern, environmentID, executor)
}

func projectMappedRows(source queryresult.MappedResult) ([]semanticRowProjection, error) {
	return sharedprojection.ProjectMappedRows(source)
}

func projectContractRows(source []driver.ContractQueryRow) []semanticRowProjection {
	return sharedprojection.ProjectContractRows(source)
}
