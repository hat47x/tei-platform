package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"tei/internal/threeelement"
)

type threeElementResponse struct {
	SchemaVersion string                `json:"schemaVersion"`
	OK            bool                  `json:"ok"`
	OKScope       string                `json:"okScope,omitempty"`
	Result        interface{}           `json:"result,omitempty"`
	Error         *cliInspectionFailure `json:"error,omitempty"`
}

func writeThreeElementResponse(out io.Writer, response threeElementResponse, exit int) (int, error) {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		return 2, err
	}
	return exit, nil
}

func runThreeElementCommand(args []string, out io.Writer) (int, error) {
	if len(args) == 0 {
		return failThreeElement("INVALID_ARGUMENT", "three-element requires a subcommand: validate|crud|cross-reference|diff|rea-intake", 2, out)
	}
	sub := args[0]
	switch sub {
	case "validate":
		return runThreeElementValidate(args[1:], out)
	case "crud":
		return runThreeElementCRUD(args[1:], out)
	case "cross-reference":
		return runThreeElementCrossReference(args[1:], out)
	case "diff":
		return runThreeElementDiff(args[1:], out)
	case "rea-intake":
		// The executor existed but was never routed here, so every rea-intake
		// invocation answered "unknown subcommand" while the tests expected it
		// to run. The missing case is why that gap went unnoticed: a test that
		// calls the executor directly never touches the dispatcher.
		return runThreeElementReaIntake(args[1:], out)
	default:
		return failThreeElement("INVALID_ARGUMENT", fmt.Sprintf("unknown three-element subcommand %q", sub), 2, out)
	}
}

func failThreeElement(code, message string, exit int, out io.Writer) (int, error) {
	response := threeElementResponse{
		SchemaVersion: "tei.cli.three-element/v0",
		Error: &cliInspectionFailure{
			Code:    code,
			Message: message,
		},
	}
	return writeThreeElementResponse(out, response, exit)
}

func parseThreeElementModelPath(args []string) (string, error) {
	fs := flag.NewFlagSet("three-element", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	modelPath := fs.String("model", "", "path to three-element model JSON")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return "", nil
		}
		return "", err
	}
	if strings.TrimSpace(*modelPath) == "" {
		return "", fmt.Errorf("three-element requires -model")
	}
	return *modelPath, nil
}

func loadThreeElementModel(path string) (threeelement.ThreeElementModel, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return threeelement.ThreeElementModel{}, err
	}
	var model threeelement.ThreeElementModel
	if err := json.Unmarshal(raw, &model); err != nil {
		return threeelement.ThreeElementModel{}, err
	}
	model.ModelFingerprint = threeelement.ModelFingerprint(model)
	return model, nil
}

func runThreeElementValidate(args []string, out io.Writer) (int, error) {
	path, err := parseThreeElementModelPath(args)
	if err != nil {
		return failThreeElement("INVALID_ARGUMENT", err.Error(), 2, out)
	}
	if path == "" {
		printHelp(out, flag.NewFlagSet("three-element validate", flag.ContinueOnError), "three-element validate")
		return 0, nil
	}
	model, err := loadThreeElementModel(path)
	if err != nil {
		return failThreeElement("MODEL_LOAD_FAILED", err.Error(), 2, out)
	}
	report := threeelement.ValidateModel(model)
	response := threeElementResponse{
		SchemaVersion: "tei.cli.three-element.validate/v0",
		OK:            report.OK,
		OKScope:       report.OKScope,
		Result:        report,
	}
	return writeThreeElementResponse(out, response, 0)
}

func runThreeElementCRUD(args []string, out io.Writer) (int, error) {
	path, err := parseThreeElementModelPath(args)
	if err != nil {
		return failThreeElement("INVALID_ARGUMENT", err.Error(), 2, out)
	}
	if path == "" {
		printHelp(out, flag.NewFlagSet("three-element crud", flag.ContinueOnError), "three-element crud")
		return 0, nil
	}
	model, err := loadThreeElementModel(path)
	if err != nil {
		return failThreeElement("MODEL_LOAD_FAILED", err.Error(), 2, out)
	}
	rows := threeelement.GenerateCRUDTable(model)
	report := threeelement.ValidateModel(model)
	response := threeElementResponse{
		SchemaVersion: "tei.cli.three-element.crud/v0",
		// This boolean reports completion of the read-only projection.
		// It does not attest CRUD consistency or Owner meaning acceptance.
		OK:      true,
		OKScope: "projection-produced-only",
		Result: struct {
			SchemaVersion                string                         `json:"schemaVersion"`
			Rows                         []threeelement.CRUDTable       `json:"rows"`
			StructuralOK                 bool                           `json:"structuralOk"`
			CRUDConflictStatus           string                         `json:"crudConflictStatus"`
			ContradictoryCRUDDefinitions []threeelement.CRUDDefinition  `json:"contradictoryCrudDefinitions,omitempty"`
		}{
			SchemaVersion:                "tei.three-element.crud/v0",
			Rows:                         rows,
			StructuralOK:                 report.OK,
			CRUDConflictStatus:           report.CRUDConflictStatus,
			ContradictoryCRUDDefinitions: report.ContradictoryCRUDs,
		},
	}
	return writeThreeElementResponse(out, response, 0)
}

func runThreeElementCrossReference(args []string, out io.Writer) (int, error) {
	path, err := parseThreeElementModelPath(args)
	if err != nil {
		return failThreeElement("INVALID_ARGUMENT", err.Error(), 2, out)
	}
	if path == "" {
		printHelp(out, flag.NewFlagSet("three-element cross-reference", flag.ContinueOnError), "three-element cross-reference")
		return 0, nil
	}
	model, err := loadThreeElementModel(path)
	if err != nil {
		return failThreeElement("MODEL_LOAD_FAILED", err.Error(), 2, out)
	}
	response := threeElementResponse{
		SchemaVersion: "tei.cli.three-element.cross-reference/v0",
		OK:            true,
		Result: struct {
			SchemaVersion    string                                        `json:"schemaVersion"`
			BusinessFunction []threeelement.BusinessFunctionCrossReference `json:"businessFunction"`
			FunctionData     []threeelement.FunctionDataCrossReference     `json:"functionData"`
			BusinessData     []threeelement.BusinessDataCrossReference     `json:"businessData"`
			DataData         []threeelement.DataDataCrossReference         `json:"dataData"`
		}{
			SchemaVersion:    "tei.three-element.cross-reference/v0",
			BusinessFunction: threeelement.GenerateBusinessFunctionCrossReference(model),
			FunctionData:     threeelement.GenerateFunctionDataCrossReference(model),
			BusinessData:     threeelement.GenerateBusinessDataCrossReference(model),
			DataData:         threeelement.GenerateDataDataCrossReference(model),
		},
	}
	return writeThreeElementResponse(out, response, 0)
}

func runThreeElementDiff(args []string, out io.Writer) (int, error) {
	fs := flag.NewFlagSet("three-element diff", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	before := fs.String("before", "", "path to before model JSON")
	after := fs.String("after", "", "path to after model JSON")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "three-element diff")
			return 0, nil
		}
		return failThreeElement("INVALID_ARGUMENT", err.Error(), 2, out)
	}
	if strings.TrimSpace(*before) == "" || strings.TrimSpace(*after) == "" {
		return failThreeElement("INVALID_ARGUMENT", "three-element diff requires -before and -after", 2, out)
	}
	beforeModel, err := loadThreeElementModel(*before)
	if err != nil {
		return failThreeElement("MODEL_LOAD_FAILED", err.Error(), 2, out)
	}
	afterModel, err := loadThreeElementModel(*after)
	if err != nil {
		return failThreeElement("MODEL_LOAD_FAILED", err.Error(), 2, out)
	}
	diff := threeelement.DiffModels(beforeModel, afterModel)
	response := threeElementResponse{
		SchemaVersion: "tei.cli.three-element.diff/v0",
		OK:            true,
		Result:        diff,
	}
	return writeThreeElementResponse(out, response, 0)
}
