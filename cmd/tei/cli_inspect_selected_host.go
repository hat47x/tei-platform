package main

import (
	"io"

	"tei/internal/inspectoradapter"
)

// inspectWorkspaceJSONSelected is the normal CLI entry for the explicitly
// compiled local reference Inspector. It does not silently fall back to the
// old direct Operation if Host composition cannot select/bind its Provider.
func inspectWorkspaceJSONSelected(out io.Writer, root, world, kind, id string, positional int) (int, error) {
	// Keep the CLI transport's syntactic rejection ahead of Host setup.
	if root == "" || world != "" || kind == "" || id == "" || positional != 0 {
		return inspectWorkspaceJSONWithService(out, root, world, kind, id, positional, nil)
	}
	host, err := inspectoradapter.NewLocalInspectorHost()
	if err != nil {
		return selectedInspectorUnavailable(out)
	}
	selected, err := host.Bind(root)
	if err != nil {
		return selectedInspectorUnavailable(out)
	}
	return inspectWorkspaceJSONWithService(out, root, world, kind, id, positional, selected)
}

func selectedInspectorUnavailable(out io.Writer) (int, error) {
	return writeInspectionReport(out, cliInspectionReport{
		SchemaVersion: "tei.cli.inspect/v0",
		Error: &cliInspectionFailure{
			Code:    "INSPECTION_PROVIDER_UNAVAILABLE",
			Message: "the selected local Inspector could not be initialized",
		},
	}, 2)
}

// Compile-time proof that the selected implementation has the CLI port.
// The actual Selection/Host binding is performed in inspectoradapter.
var _ cliInspectionService = (*inspectoradapter.Selected)(nil)
