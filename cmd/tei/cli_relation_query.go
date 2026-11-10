package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/workspace"
)

type cliRelationQueryResponse struct {
	SchemaVersion string                        `json:"schemaVersion"`
	OK            bool                          `json:"ok"`
	Query         *driver.RelationQueryExecution `json:"query,omitempty"`
	Error         *cliRelationQueryFailure       `json:"error,omitempty"`
}

type cliRelationQueryFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func runRelationQueryCommand(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("relation-query", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	workspacePath := fs.String("workspace", "", "TEI workspace directory")
	applicationID := fs.String("application", "", "application canonical ID")
	queryID := fs.String("query", "", "relation query canonical ID")
	worldPath := fs.String("world", "", "normalized M0 IR JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if requestedHelp(args) {
		fs.SetOutput(out)
		fs.Usage()
		return nil
	}

	response := cliRelationQueryResponse{SchemaVersion: "tei.cli.relation-query/v0"}
	if *workspacePath == "" && *worldPath == "" {
		response.Error = &cliRelationQueryFailure{
			Code:    "INVALID_ARGUMENT",
			Message: "relation-query requires -workspace or -world",
		}
		response.OK = false
		return writeJSONResponse(out, response)
	}
	if *applicationID == "" || *queryID == "" {
		response.Error = &cliRelationQueryFailure{
			Code:    "INVALID_ARGUMENT",
			Message: "relation-query requires -application and -query",
		}
		response.OK = false
		return writeJSONResponse(out, response)
	}

	var w ir.World
	if *worldPath != "" {
		data, err := os.ReadFile(*worldPath)
		if err != nil {
			response.Error = &cliRelationQueryFailure{Code: "INVALID_WORLD", Message: err.Error()}
			response.OK = false
			return writeJSONResponse(out, response)
		}
		if err := json.Unmarshal(data, &w); err != nil {
			response.Error = &cliRelationQueryFailure{Code: "INVALID_WORLD", Message: err.Error()}
			response.OK = false
			return writeJSONResponse(out, response)
		}
	} else {
		loaded, err := workspace.Load(*workspacePath)
		if err != nil {
			response.Error = &cliRelationQueryFailure{Code: "WORKSPACE_LOAD_FAILED", Message: err.Error()}
			response.OK = false
			return writeJSONResponse(out, response)
		}
		w = loaded.World
	}

	resolver := driver.NewResolver(w)
	execution, err := resolver.ExecuteRelationQuery(context.Background(), core.MustCanonicalID(*applicationID), core.MustCanonicalID(*queryID), noopRelationQueryExecutor{}, nil)
	if err != nil {
		response.Error = &cliRelationQueryFailure{Code: "RESOLUTION_FAILED", Message: err.Error()}
		response.OK = false
		return writeJSONResponse(out, response)
	}

	response.Query = &execution
	response.OK = true
	return writeJSONResponse(out, response)
}

type noopRelationQueryExecutor struct{}

func (noopRelationQueryExecutor) ExecuteRelationQuery(
	ctx context.Context,
	resolved query.ResolvedRelationQuery,
	_ application.DatabaseRequirement,
	parameters map[string]any,
) (driver.RelationQueryResult, error) {
	_ = parameters
	return driver.RelationQueryResult{
		Raw:     queryresult.RawResult{},
		Mapped:  queryresult.MappedResult{},
		Payload: map[string]string{"sql": resolved.SQL},
	}, nil
}

func writeJSONResponse(out io.Writer, response cliRelationQueryResponse) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(response)
}
