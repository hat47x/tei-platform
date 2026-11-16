package main

import (
	"io"

	"tei/internal/inspectoradapter"
	"tei/internal/operations"
	"tei/internal/sourceindex"
)

// inspectWorkspaceJSONSelected is the normal CLI entry for the explicitly
// compiled local reference Inspector. It does not silently fall back to the
// old direct Operation if Host composition cannot select/bind its Provider.
// expectedFingerprint is the caller's snapshot precondition; an empty value
// asks for no check.
func inspectWorkspaceJSONSelected(
	out io.Writer, root, world, kind, id, expectedFingerprint string, positional int,
) (int, error) {
	return inspectWorkspaceJSONFromHost(out, inspectoradapter.NewLocalInspectorHost,
		root, world, kind, id, expectedFingerprint, positional)
}

// inspectWorkspaceJSONFromHost runs the CLI transport against whichever
// Inspector the composing Host selected. The Host, not a command-line
// argument, decides the Provider; a selection that cannot be composed or
// bound stops here with a classified failure.
func inspectWorkspaceJSONFromHost(
	out io.Writer, compose func() (*inspectoradapter.LocalInspectorHost, error),
	root, world, kind, id, expectedFingerprint string, positional int,
) (int, error) {
	// Keep the CLI transport's syntactic rejection ahead of Host setup.
	if root == "" || world != "" || kind == "" || id == "" || positional != 0 {
		return inspectWorkspaceJSONWithService(out, root, world, kind, id, positional, nil)
	}
	host, err := compose()
	if err != nil {
		return selectedInspectorUnavailable(out)
	}
	selected, err := host.Bind(root)
	if err != nil {
		return selectedInspectorUnavailable(out)
	}
	// The precondition is applied by this transport, around whichever Provider
	// the Host selected, rather than by extending the Inspector Port. A Port
	// that took the expected fingerprint would force every registered
	// implementation to accept one, and a Provider that ignored it would answer
	// a question the caller asked. Comparing the fingerprints the selected
	// service already returned keeps the rule in one place and fails closed for
	// a Provider that reports none.
	var service cliInspectionService = selected
	if expectedFingerprint != "" {
		service = inspectSnapshotPrecondition{service: service, expected: expectedFingerprint}
	}
	return inspectWorkspaceJSONWithService(out, root, world, kind, id, positional, service)
}

// inspectSnapshotPrecondition wraps a selected inspection service with the
// caller's expected snapshot fingerprint. It compares the two fingerprints the
// service reported, so a Provider cannot satisfy the precondition by asserting
// a snapshot value of its own, and it never re-reads the Workspace: a match
// means this read agrees with the earlier one, not that the Workspace is
// unchanged now.
type inspectSnapshotPrecondition struct {
	service  cliInspectionService
	expected string
}

func (p inspectSnapshotPrecondition) Inspect(subject sourceindex.Subject) (operations.SubjectInspection, error) {
	result, err := p.service.Inspect(subject)
	if err != nil {
		return operations.SubjectInspection{}, err
	}
	if err := operations.CheckInspectionSnapshot(
		result.WorldFingerprint, result.SourceIndexFingerprint, p.expected,
	); err != nil {
		return operations.SubjectInspection{}, err
	}
	return result, nil
}

// CanonicalInspectionMessages reports whether the wrapped service forwards the
// canonical Operation's own error text. The precondition's failure is raised
// here, not by the Provider, but its text is the Operation's, so the transport
// may show it verbatim exactly when the wrapped implementation is trusted to.
func (p inspectSnapshotPrecondition) CanonicalInspectionMessages() bool {
	return canonicalInspectionMessages(p.service)
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
