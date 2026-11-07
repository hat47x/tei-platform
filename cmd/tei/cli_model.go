package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"strings"

	"tei/internal/operations"
)

// Modeler commands share the existing operations layer with future web and
// desktop targets. No model edit or XEA[D] acceptance occurs in this transport.
type modelCLIResponse struct {
	SchemaVersion string                   `json:"schemaVersion"`
	OK            bool                     `json:"ok"`
	Result        *operations.ModelCatalog `json:"result,omitempty"`
	Error         *cliInspectionFailure    `json:"error,omitempty"`
}

func runModelCommand(args []string, out io.Writer) (int, error) {
	response := modelCLIResponse{SchemaVersion: "tei.cli.model/v0"}
	fail := func(code string, message string, exit int) (int, error) {
		response.Error = &cliInspectionFailure{Code: code, Message: message}
		return writeModelResponse(out, response, exit)
	}
	if len(args) == 0 || args[0] != "catalog" {
		return fail("INVALID_ARGUMENT", "model requires the subcommand catalog", 2)
	}
	fs := flag.NewFlagSet("model catalog", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("workspace", "", "TEI workspace directory")
	namespace := fs.String("namespace", "", "optional exact semantic namespace ID")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "model catalog")
			return 0, nil
		}
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	if fs.NArg() != 0 || strings.TrimSpace(*root) == "" || *root != strings.TrimSpace(*root) ||
		*namespace != strings.TrimSpace(*namespace) {
		return fail("INVALID_ARGUMENT", "model catalog requires an exact -workspace and optional exact -namespace, without positional arguments", 2)
	}
	result, err := operations.CatalogWorkspace(*root, *namespace)
	if err != nil {
		code := "MODEL_CATALOG_FAILED"
		var failure *operations.ValidationFailure
		if errors.As(err, &failure) {
			switch failure.Stage {
			case operations.StageLoad:
				code = "WORKSPACE_LOAD_FAILED"
			case operations.StageSourceIndex:
				code = "SOURCE_INDEX_FAILED"
			}
		}
		return fail(code, err.Error(), 2)
	}
	if !result.NamespaceMatched {
		// A namespace that does not exist is not "a namespace with no entries".
		// Return a typed failure so a caller cannot read an empty success as
		// evidence that the namespace declares nothing.
		message := "namespace " + result.NamespaceFilter + " is not a semantic namespace in this workspace"
		switch result.NamespaceMatchKind {
		case "application":
			message = "application " + result.NamespaceFilter + " is not a semantic namespace; omit -namespace or use a namespace ID"
		case "query":
			message = "query " + result.NamespaceFilter + " is not a semantic namespace; omit -namespace or use a namespace ID"
		}
		return fail("NAMESPACE_NOT_FOUND", message, 1)
	}
	response.OK = true
	response.Result = &result
	return writeModelResponse(out, response, 0)
}

func writeModelResponse(out io.Writer, response modelCLIResponse, exit int) (int, error) {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		return 2, err
	}
	return exit, nil
}
