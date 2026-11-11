package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// studioShoDocument invokes the SHO draft document CLI as a child process.
//
// It is the Studio-side half of the TEI -> SHO document integration: the Studio
// opens a draft the TEI projection produced and shows the document state the
// real SHO CLI reports. It is a relay, not an editor and not a meaning owner:
//
//   - The Studio never writes the model and never ingests a document into it.
//   - Save is deliberately NOT exposed here. A save is an author's explicit act
//     in SHO, not a side effect of looking at a document in the Studio.
//   - The child process is the qualification boundary. If the SHO CLI is absent,
//     times out, exits non-zero or emits an unreadable receipt, the Studio says
//     so with a stable code instead of showing an empty document that looks like
//     an empty draft.
//
// The command path is configuration, not a discovery: a Studio build that has
// no SHO checkout to point at answers `sho-not-configured` rather than searching
// the filesystem for something that might be the right CLI.
type studioShoDocument struct {
	// CommandPath is the absolute path to the SHO `bin/sho-doc.cjs` entry point.
	CommandPath string
	// NodePath overrides the `node` executable on PATH. Empty uses "node".
	NodePath string
	// Timeout bounds one child invocation. Zero uses shoDocumentDefaultTimeout.
	Timeout time.Duration
	// MaxReceiptBytes bounds the receipt read from the child's standard output.
	MaxReceiptBytes int
}

// Failure codes. They are stable strings so a caller can branch without parsing
// prose, and they never imply a document state the Studio did not observe.
const (
	ShoNotConfigured     = "sho-not-configured"
	ShoInvalidArgument   = "sho-invalid-argument"
	ShoSpawnFailed       = "sho-spawn-failed"
	ShoTimeout           = "sho-timeout"
	ShoNonZeroExit       = "sho-nonzero-exit"
	ShoReceiptUnreadable = "sho-receipt-unreadable"
)

const (
	shoDocumentDefaultTimeout    = 60 * time.Second
	shoDocumentDefaultMaxReceipt = 1 << 20
	shoDocumentSchemaVersion     = "tei.studio.sho-document/v0"
)

// StudioShoDocumentRequest names one document operation. Operation is `open` or
// `resume`; save is not offered.
type StudioShoDocumentRequest struct {
	Operation    string `json:"operation"`
	DraftPath    string `json:"draftPath,omitempty"`
	DocumentPath string `json:"documentPath"`
}

// StudioShoDocumentResponse relays the child's receipt. Result is the child's
// own JSON object, unmodified: the Studio does not interpret document meaning,
// so it has nothing to add and nothing to complete.
type StudioShoDocumentResponse struct {
	SchemaVersion   string            `json:"schemaVersion"`
	Effect          string            `json:"effect"`
	OK              bool              `json:"ok"`
	Result          json.RawMessage   `json:"result,omitempty"`
	Failure         *studioShoFailure `json:"failure,omitempty"`
	AuthorityLimits []string          `json:"authorityLimits"`
}

type studioShoFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// StudioShoDocumentAuthorityLimits states what a relayed document state does not
// grant. It is a constant list, not a per-caller decision.
func StudioShoDocumentAuthorityLimits() []string {
	return []string{
		"studio-relays-sho-document-state-without-editing-it",
		"studio-does-not-write-the-tei-model",
		"studio-save-is-not-tei-model-ingestion",
		"sho-command-availability-is-not-product-qualification",
		"document-meaning-is-owned-by-the-tei-workspace-not-by-the-document",
	}
}

// Run invokes one document operation and relays the receipt.
func (s studioShoDocument) Run(request StudioShoDocumentRequest) StudioShoDocumentResponse {
	response := StudioShoDocumentResponse{
		SchemaVersion:   shoDocumentSchemaVersion,
		Effect:          "read-only-relay",
		AuthorityLimits: StudioShoDocumentAuthorityLimits(),
	}
	fail := func(code, message string) StudioShoDocumentResponse {
		response.OK = false
		response.Failure = &studioShoFailure{Code: code, Message: message}
		return response
	}

	commandPath := strings.TrimSpace(s.CommandPath)
	if commandPath == "" {
		return fail(ShoNotConfigured,
			"no SHO document command is configured for this Studio build")
	}
	if !filepath.IsAbs(commandPath) {
		return fail(ShoInvalidArgument,
			"the SHO document command path must be absolute")
	}
	if request.Operation != "open" && request.Operation != "resume" {
		return fail(ShoInvalidArgument,
			"operation must be open or resume; save is not offered by this surface")
	}
	// A document path the caller supplies must be absolute too: a relative path
	// would be resolved against whatever directory the Studio happens to run in.
	if !filepath.IsAbs(request.DocumentPath) {
		return fail(ShoInvalidArgument, "documentPath must be absolute")
	}
	args := []string{commandPath, request.Operation}
	switch request.Operation {
	case "open":
		if !filepath.IsAbs(request.DraftPath) {
			return fail(ShoInvalidArgument, "draftPath must be absolute for open")
		}
		args = append(args, request.DraftPath, request.DocumentPath)
	case "resume":
		args = append(args, request.DocumentPath)
	}
	nodePath := strings.TrimSpace(s.NodePath)
	if nodePath == "" {
		nodePath = "node"
	}
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = shoDocumentDefaultTimeout
	}
	maxReceipt := s.MaxReceiptBytes
	if maxReceipt <= 0 {
		maxReceipt = shoDocumentDefaultMaxReceipt
	}

	command := exec.Command(nodePath, args...)
	// The child inherits no workspace root: it only reads the paths the caller
	// named, and the Studio never hands it a model.
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return fail(ShoSpawnFailed, err.Error())
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				return fail(ShoNonZeroExit, strings.TrimSpace(stderr.String()))
			}
			return fail(ShoSpawnFailed, err.Error())
		}
	case <-time.After(timeout):
		// A timed-out child is killed; its partial output is never read as a
		// document state.
		_ = command.Process.Kill()
		<-done
		return fail(ShoTimeout, "the SHO document command did not answer within "+timeout.String())
	}
	if stdout.Len() > maxReceipt {
		return fail(ShoReceiptUnreadable, "the SHO document receipt exceeds the read budget")
	}
	// The child's whole standard output is the receipt: `bin/sho-doc.cjs` writes
	// one JSON object and nothing else. The Studio relays that object as the
	// result rather than looking for a nested key, because a receipt that has to
	// be unwrapped first is a receipt the Studio would have to interpret.
	var receipt map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &receipt); err != nil {
		return fail(ShoReceiptUnreadable, err.Error())
	}
	response.OK = true
	response.Result = json.RawMessage(stdout.Bytes())
	return response
}

// studioShoDocumentFromEnvironment reads the command path from the environment.
// An unset variable is `sho-not-configured`, never a guess.
func studioShoDocumentFromEnvironment() studioShoDocument {
	return studioShoDocument{CommandPath: os.Getenv("SHO_DOC_COMMAND")}
}
