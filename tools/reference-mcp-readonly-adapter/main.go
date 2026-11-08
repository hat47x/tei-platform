// Command reference-mcp-readonly-adapter exposes the existing read-only TEI
// operations to MCP clients over stdio JSON-RPC. It is a Reference Adapter: it
// calls internal/operations exactly as the CLI does and introduces no second
// canonical interpretation, knowledge store, validation rule, or permission
// model (docs/architecture.md, #3400).
//
//	go build -o /tmp/tei-mcp ./tools/reference-mcp-readonly-adapter
//	/tmp/tei-mcp -workspace reference/workspace
//
// The Workspace is fixed at startup on purpose: the model never receives a
// path argument, so a tool call cannot widen the read set beyond what the
// Host already opened. Every tool is read-only; the adapter has no write,
// acceptance, deployment, or grant path.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"tei/internal/modelref"
	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

const (
	serverName    = "tei-reference-mcp-readonly-adapter"
	serverVersion = "0.1.0"
	// protocolVersion is the MCP revision this adapter implements. It is not a
	// claim that later revisions behave the same.
	protocolVersion = "2024-11-05"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// envelope mirrors the CLI's schemaVersion/ok/result/error document so a client
// can compare the MCP answer and the CLI answer without translating shapes.
type envelope struct {
	SchemaVersion string    `json:"schemaVersion"`
	OK            bool      `json:"ok"`
	Result        any       `json:"result,omitempty"`
	Error         *rpcError `json:"error,omitempty"`
}

func okEnvelope(schemaVersion string, result any) envelope {
	return envelope{SchemaVersion: schemaVersion, OK: true, Result: result}
}

func errorEnvelope(schemaVersion string, err error) envelope {
	code := "OPERATION_FAILED"
	message := err.Error()
	var inspection *operations.InspectionError
	var trace *modelref.TraceError
	var failure *operations.ValidationFailure
	var missing *workspaceLoadFailure
	switch {
	case errors.As(err, &inspection):
		code = inspection.Code
	case errors.As(err, &trace):
		code = trace.Code
	case errors.As(err, &failure):
		switch failure.Stage {
		case operations.StageLoad:
			code = "WORKSPACE_LOAD_FAILED"
		case operations.StageSourceIndex:
			code = "SOURCE_INDEX_FAILED"
		}
	case errors.As(err, &missing):
		code = "WORKSPACE_LOAD_FAILED"
	}
	return envelope{SchemaVersion: schemaVersion, OK: false,
		Error: &rpcError{Code: 1, Message: code + ": " + message}}
}

// workspaceLoadFailure is unused today; it keeps the switch above readable when
// an operation starts returning a typed load error.
type workspaceLoadFailure struct{ error }

type toolResult struct {
	Content []textContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type textContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func main() {
	workspace := flag.String("workspace", "", "TEI workspace directory fixed for this adapter process")
	flag.Parse()
	if *workspace == "" {
		fmt.Fprintln(os.Stderr, "reference-mcp-readonly-adapter requires -workspace")
		os.Exit(2)
	}
	if err := serve(os.Stdin, os.Stdout, *workspace); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// serve runs the JSON-RPC loop. It reads one request per line, replies on
// standard output, and never writes anything else there.
func serve(in io.Reader, out io.Writer, workspace string) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var request rpcRequest
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			if err := encoder.Encode(rpcResponse{JSONRPC: "2.0",
				Error: &rpcError{Code: -32700, Message: "parse error"}}); err != nil {
				return err
			}
			continue
		}
		if request.ID == nil {
			// A notification (for example notifications/initialized) has no reply.
			continue
		}
		if err := encoder.Encode(handle(request, workspace)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func handle(request rpcRequest, workspace string) rpcResponse {
	response := rpcResponse{JSONRPC: "2.0", ID: request.ID}
	switch request.Method {
	case "initialize":
		response.Result = map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": serverName, "version": serverVersion},
		}
	case "ping":
		response.Result = map[string]any{}
	case "tools/list":
		response.Result = map[string]any{"tools": toolList()}
	case "tools/call":
		result, err := callTool(request.Params, workspace)
		if err != nil {
			response.Error = &rpcError{Code: -32602, Message: err.Error()}
			return response
		}
		response.Result = result
	default:
		response.Error = &rpcError{Code: -32601, Message: "method not found: " + request.Method}
	}
	return response
}

func toolList() []tool {
	subjectProps := map[string]any{
		"subjectKind": map[string]any{"type": "string"},
		"subjectId":   map[string]any{"type": "string"},
	}
	traceProps := map[string]any{
		"scope":               map[string]any{"type": "string"},
		"subjectKind":         map[string]any{"type": "string"},
		"subjectId":           map[string]any{"type": "string"},
		"direction":           map[string]any{"type": "string", "enum": []string{"outgoing", "incoming", "both"}},
		"mode":                map[string]any{"type": "string", "enum": []string{"summary", "full"}},
		"offset":              map[string]any{"type": "integer"},
		"limit":               map[string]any{"type": "integer"},
		"expectedFingerprint": map[string]any{"type": "string"},
	}
	return []tool{
		{
			Name:        "tei_model_catalog",
			Description: "List the model elements of the fixed Workspace (same result as `tei model catalog`).",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"namespace": map[string]any{"type": "string"}},
			},
		},
		{
			Name:        "tei_inspect",
			Description: "Read one exact canonical subject (same result as `tei inspect`).",
			InputSchema: map[string]any{
				"type":       "object",
				"required":   []string{"subjectKind", "subjectId"},
				"properties": subjectProps,
			},
		},
		{
			Name:        "tei_trace",
			Description: "Trace declared direct references of one subject (same result as `tei trace`).",
			InputSchema: map[string]any{
				"type":       "object",
				"required":   []string{"scope", "subjectKind", "subjectId"},
				"properties": traceProps,
			},
		},
		{
			Name:        "tei_validate",
			Description: "Validate the fixed Workspace (same result as `tei validate -workspace ... -json`).",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		},
	}
}

func callTool(params json.RawMessage, workspace string) (toolResult, error) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return toolResult{}, fmt.Errorf("invalid tools/call params: %w", err)
	}
	var arguments map[string]json.RawMessage
	if len(call.Arguments) > 0 {
		if err := json.Unmarshal(call.Arguments, &arguments); err != nil {
			return toolResult{}, fmt.Errorf("invalid arguments: %w", err)
		}
	}
	payload, isError, err := dispatch(call.Name, arguments, workspace)
	if err != nil {
		return toolResult{}, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return toolResult{}, err
	}
	return toolResult{Content: []textContent{{Type: "text", Text: string(raw)}}, IsError: isError}, nil
}

// dispatch maps a tool name to one operations call. The envelope fields match
// the CLI's schemaVersion/ok/result/error contract for that command.
func dispatch(name string, arguments map[string]json.RawMessage, workspace string) (any, bool, error) {
	switch name {
	case "tei_model_catalog":
		var namespace string
		if raw, ok := arguments["namespace"]; ok {
			if err := json.Unmarshal(raw, &namespace); err != nil {
				return nil, false, fmt.Errorf("namespace must be a string")
			}
		}
		result, err := operations.CatalogWorkspace(workspace, namespace)
		if err != nil {
			return errorEnvelope("tei.cli.model/v0", err), true, nil
		}
		return okEnvelope("tei.cli.model/v0", result), false, nil
	case "tei_inspect":
		kind, id, err := requireSubject(arguments)
		if err != nil {
			return nil, false, err
		}
		result, err := operations.InspectSubject(workspace, sourceindex.Subject{Kind: kind, ID: id})
		if err != nil {
			return errorEnvelope("tei.cli.inspect/v0", err), true, nil
		}
		return okEnvelope("tei.cli.inspect/v0", result), false, nil
	case "tei_trace":
		scope, kind, id, err := requireTrace(arguments)
		if err != nil {
			return nil, false, err
		}
		options := modelref.TraceOptions{
			Direction:           optionalString(arguments, "direction", "both"),
			Mode:                optionalString(arguments, "mode", "summary"),
			ExpectedFingerprint: optionalString(arguments, "expectedFingerprint", ""),
		}
		if raw, ok := arguments["offset"]; ok {
			if err := json.Unmarshal(raw, &options.Offset); err != nil {
				return nil, false, fmt.Errorf("offset must be an integer")
			}
		}
		if raw, ok := arguments["limit"]; ok {
			if err := json.Unmarshal(raw, &options.Limit); err != nil {
				return nil, false, fmt.Errorf("limit must be an integer")
			}
		}
		result, err := operations.TraceWorkspace(workspace, modelref.Key{
			Scope: scope, Subject: subject.Subject{Kind: kind, ID: id},
		}, options)
		if err != nil {
			return errorEnvelope("tei.cli.trace/v0", err), true, nil
		}
		return okEnvelope("tei.cli.trace/v0", result), false, nil
	case "tei_validate":
		result, err := operations.ValidateWorkspace(workspace)
		if err != nil {
			return errorEnvelope("tei.cli.validate/v0", err), true, nil
		}
		return validateEnvelope(result), false, nil
	default:
		return nil, false, fmt.Errorf("unknown tool %q", name)
	}
}

// validateEnvelope mirrors cmd/tei's validate projection for the fields the
// adapter exposes. Unlike the CLI, the declared Workspace carries no syntax
// finding path here because a Direct finding means no World was built; the
// adapter reports it as an error instead of a partial document.
func validateEnvelope(result operations.WorkspaceValidation) envelope {
	if result.Direct != nil {
		return envelope{SchemaVersion: "tei.cli.validate/v0", OK: false,
			Error: &rpcError{Code: 1, Message: result.Direct.Code + ": " + result.Direct.Message}}
	}
	return okEnvelope("tei.cli.validate/v0", map[string]any{
		"worldFingerprint":       result.WorldFingerprint,
		"sourceIndexFingerprint": result.SourceIndexFingerprint,
		"diagnostics":            result.Diagnostics,
		"declarationCounts":      result.Declarations,
		"sourceRootFindings":     result.SourceRootFindings,
	})
}

func requireSubject(arguments map[string]json.RawMessage) (string, string, error) {
	var kind, id string
	if raw, ok := arguments["subjectKind"]; ok {
		if err := json.Unmarshal(raw, &kind); err != nil {
			return "", "", fmt.Errorf("subjectKind must be a string")
		}
	}
	if raw, ok := arguments["subjectId"]; ok {
		if err := json.Unmarshal(raw, &id); err != nil {
			return "", "", fmt.Errorf("subjectId must be a string")
		}
	}
	if kind == "" || id == "" {
		return "", "", fmt.Errorf("subjectKind and subjectId are required")
	}
	return kind, id, nil
}

func requireTrace(arguments map[string]json.RawMessage) (string, string, string, error) {
	var scope string
	if raw, ok := arguments["scope"]; ok {
		if err := json.Unmarshal(raw, &scope); err != nil {
			return "", "", "", fmt.Errorf("scope must be a string")
		}
	}
	kind, id, err := requireSubject(arguments)
	if err != nil {
		return "", "", "", err
	}
	if scope == "" {
		return "", "", "", fmt.Errorf("scope is required")
	}
	return scope, kind, id, nil
}

func optionalString(arguments map[string]json.RawMessage, key, fallback string) string {
	raw, ok := arguments[key]
	if !ok {
		return fallback
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || value == "" {
		return fallback
	}
	return value
}
