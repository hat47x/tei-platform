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
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
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
// Its Error is the CLI's string code; the JSON-RPC numeric error stays in
// rpcResponse, because the two carry different meanings (#3420).
type envelope struct {
	SchemaVersion string                              `json:"schemaVersion"`
	OK            bool                                `json:"ok"`
	Result        any                                 `json:"result,omitempty"`
	Error         *operations.ValidationReportFailure `json:"error,omitempty"`
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
	var catalog *operations.CatalogError
	switch {
	case errors.As(err, &catalog):
		code = catalog.Code
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
		Error: &operations.ValidationReportFailure{Code: code, Message: message}}
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

// maxRequestBytes bounds one request line. A larger line is refused as an
// invalid request instead of ending the loop, so one oversized call cannot
// stop the server for every later request (#3420).
const maxRequestBytes = 4 * 1024 * 1024

// serve runs the JSON-RPC loop. It reads one request per line, replies on
// standard output, and never writes anything else there.
func serve(in io.Reader, out io.Writer, workspace string) error {
	reader := bufio.NewReaderSize(in, 64*1024)
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	for {
		line, oversized, err := readRequestLine(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if oversized {
			// JSON-RPC -32600: the request could not be read as one request.
			if err := encoder.Encode(rpcResponse{JSONRPC: "2.0",
				Error: &rpcError{Code: -32600, Message: "request exceeds the 4 MiB line limit"}}); err != nil {
				return err
			}
			continue
		}
		line = strings.TrimSpace(line)
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
}

// readRequestLine reads one line without buffering more than maxRequestBytes
// plus a little. It reports oversized instead of returning a partial line, and
// consumes the rest of an oversized line so the next request starts cleanly.
func readRequestLine(reader *bufio.Reader) (line string, oversized bool, err error) {
	var builder strings.Builder
	for {
		chunk, readErr := reader.ReadSlice('\n')
		if len(chunk) > 0 {
			if builder.Len()+len(chunk) > maxRequestBytes {
				// The chunk may already hold the newline; either way the rest of
				// this line must be drained before the next request is read.
				if !bytes.HasSuffix(chunk, []byte("\n")) {
					return "", true, drainLine(reader, readErr)
				}
				return "", true, nil
			}
			builder.Write(chunk)
		}
		if readErr == nil {
			return builder.String(), false, nil
		}
		if errors.Is(readErr, bufio.ErrBufferFull) {
			continue
		}
		if errors.Is(readErr, io.EOF) {
			if builder.Len() == 0 {
				return "", false, io.EOF
			}
			return builder.String(), false, nil
		}
		return "", false, readErr
	}
}

// drainLine consumes the remainder of an oversized line. A read error other
// than EOF ends the loop, because the stream state is unknown.
func drainLine(reader *bufio.Reader, pending error) error {
	for {
		chunk, err := reader.ReadSlice('\n')
		if len(chunk) > 0 && bytes.HasSuffix(chunk, []byte("\n")) {
			return nil
		}
		if err == nil {
			return nil
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if errors.Is(err, io.EOF) {
			if pending == nil || errors.Is(pending, io.EOF) {
				return nil
			}
			return pending
		}
		return err
	}
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
		if err := rejectUnknownArguments(arguments, "namespace"); err != nil {
			return nil, false, err
		}
		namespace, err := optionalString(arguments, "namespace", "")
		if err != nil {
			return nil, false, err
		}
		result, err := operations.CatalogWorkspace(workspace, namespace)
		if err != nil {
			return errorEnvelope("tei.cli.model/v0", err), true, nil
		}
		// A namespace that names nothing is a failed answer, not an empty
		// catalog. The projection owns the decision so the CLI and MCP adapter
		// answer the same code and message (#3420).
		if failure := result.NamespaceFilterError(); failure != nil {
			return operationFailureEnvelope("tei.cli.model/v0", failure.Code, failure.Message), true, nil
		}
		return okEnvelope("tei.cli.model/v0", result), false, nil
	case "tei_inspect":
		if err := rejectUnknownArguments(arguments, "subjectKind", "subjectId"); err != nil {
			return nil, false, err
		}
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
		if err := rejectUnknownArguments(arguments, "scope", "subjectKind", "subjectId",
			"direction", "mode", "offset", "limit", "expectedFingerprint"); err != nil {
			return nil, false, err
		}
		scope, kind, id, err := requireTrace(arguments)
		if err != nil {
			return nil, false, err
		}
		options := modelref.TraceOptions{}
		if options.Direction, err = optionalString(arguments, "direction", "both"); err != nil {
			return nil, false, err
		}
		if options.Mode, err = optionalString(arguments, "mode", "summary"); err != nil {
			return nil, false, err
		}
		if options.ExpectedFingerprint, err = optionalString(arguments, "expectedFingerprint", ""); err != nil {
			return nil, false, err
		}
		if options.Offset, err = optionalInt(arguments, "offset"); err != nil {
			return nil, false, err
		}
		if options.Limit, err = optionalInt(arguments, "limit"); err != nil {
			return nil, false, err
		}
		result, err := operations.TraceWorkspace(workspace, modelref.Key{
			Scope: scope, Subject: subject.Subject{Kind: kind, ID: id},
		}, options)
		if err != nil {
			return errorEnvelope("tei.cli.trace/v0", err), true, nil
		}
		return okEnvelope("tei.cli.trace/v0", result), false, nil
	case "tei_validate":
		if err := rejectUnknownArguments(arguments); err != nil {
			return nil, false, err
		}
		result, err := operations.ValidateWorkspace(workspace)
		if err != nil {
			return validationFailureEnvelope(err), true, nil
		}
		// The same projection the CLI writes, so a client compares documents
		// instead of translating two shapes (#3420). A validation finding that
		// is not an operation error is a failed answer here too, not a success
		// with isError false.
		report := operations.WorkspaceValidationReportFrom(result)
		return report, !report.OK, nil
	default:
		return nil, false, fmt.Errorf("unknown tool %q", name)
	}
}

// operationFailureEnvelope carries a typed operation failure that is not a load
// error, using the CLI's string code so a client compares the two answers.
func operationFailureEnvelope(schemaVersion, code, message string) envelope {
	return envelope{SchemaVersion: schemaVersion, OK: false,
		Error: &operations.ValidationReportFailure{Code: code, Message: message}}
}

// validationFailureEnvelope carries the CLI's validate failure contract: the
// document keeps `diagnostics` as an empty array and puts the string code in
// `error.code`, so a client compares it with the CLI answer directly (#3420).
func validationFailureEnvelope(err error) operations.WorkspaceValidationReport {
	code := "WORKSPACE_VALIDATION_FAILED"
	var failure *operations.ValidationFailure
	if errors.As(err, &failure) {
		switch failure.Stage {
		case operations.StageLoad:
			code = "WORKSPACE_LOAD_FAILED"
		case operations.StageSourceIndex:
			code = "SOURCE_INDEX_FAILED"
		}
	}
	return operations.WorkspaceValidationFailureReport(code, err.Error())
}

func requireSubject(arguments map[string]json.RawMessage) (string, string, error) {
	kind, err := requiredString(arguments, "subjectKind")
	if err != nil {
		return "", "", err
	}
	id, err := requiredString(arguments, "subjectId")
	if err != nil {
		return "", "", err
	}
	return kind, id, nil
}

func requireTrace(arguments map[string]json.RawMessage) (string, string, string, error) {
	scope, err := requiredString(arguments, "scope")
	if err != nil {
		return "", "", "", err
	}
	kind, id, err := requireSubject(arguments)
	if err != nil {
		return "", "", "", err
	}
	return scope, kind, id, nil
}

// rejectUnknownArguments refuses a member the tool's input schema does not
// declare. A misspelled key would otherwise be dropped and the call would
// succeed with the default, which hides the caller's mistake (#3420).
func rejectUnknownArguments(arguments map[string]json.RawMessage, allowed ...string) error {
	var unknown []string
	for key := range arguments {
		known := false
		for _, name := range allowed {
			if key == name {
				known = true
				break
			}
		}
		if !known {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	return fmt.Errorf("unknown argument %q", unknown[0])
}

// stringArgument decodes one string member. null, a non-string and an empty
// string are invalid values; they are never replaced by a default, so an
// unspecified member and an invalid one stay distinguishable (#3420).
func stringArgument(arguments map[string]json.RawMessage, key string) (string, bool, error) {
	raw, ok := arguments[key]
	if !ok {
		return "", false, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", true, fmt.Errorf("%s must be a string", key)
	}
	if value == "" {
		return "", true, fmt.Errorf("%s must not be an empty string", key)
	}
	return value, true, nil
}

func requiredString(arguments map[string]json.RawMessage, key string) (string, error) {
	value, present, err := stringArgument(arguments, key)
	if err != nil {
		return "", err
	}
	if !present {
		return "", fmt.Errorf("%s is required", key)
	}
	return value, nil
}

func optionalString(arguments map[string]json.RawMessage, key, fallback string) (string, error) {
	value, present, err := stringArgument(arguments, key)
	if err != nil {
		return "", err
	}
	if !present {
		return fallback, nil
	}
	return value, nil
}

func optionalInt(arguments map[string]json.RawMessage, key string) (int, error) {
	raw, ok := arguments[key]
	if !ok {
		return 0, nil
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	return value, nil
}
