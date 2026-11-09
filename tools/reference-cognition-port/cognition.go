// Package cognitionport is a reference for treating lightweight cognition as a
// bounded proposal port. A provider may propose candidates, but it never
// accepts meaning, grants authority, or applies a change.
package cognitionport

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	RequestSchema  = "tei.reference.cognition-request/v0"
	ProviderSchema = "tei.reference.cognition-provider/v0"
	ProposalSchema = "tei.reference.cognition-proposal/v0"
	ReceiptSchema  = "tei.reference.cognition-receipt/v0"
)

const (
	ProfileOffline   = "offline"
	ProfileConnected = "connected"
)

const (
	ExecutionLocal    = "local"
	ExecutionExternal = "external"
)

const (
	NetworkNone     = "none"
	NetworkRequired = "required"
)

const (
	StateProposed    = "proposed"
	StateUnsupported = "unsupported"
	StateUnknown     = "unknown"
	StateFailed      = "failed"
	StateBlocked     = "blocked"
)

type Limits struct {
	MaxContextBytes int `json:"maxContextBytes"`
	MaxOutputBytes  int `json:"maxOutputBytes"`
	TimeoutSeconds  int `json:"timeoutSeconds"`
}

type Observation struct {
	ID        string `json:"id"`
	Statement string `json:"statement"`
	Reference string `json:"reference,omitempty"`
}

type Context struct {
	WorkspaceRevision  string        `json:"workspaceRevision"`
	AcceptedReferences []string      `json:"acceptedReferences"`
	Observations       []Observation `json:"observations"`
	Constraints        []string      `json:"constraints,omitempty"`
}

type Request struct {
	Schema  string  `json:"schema"`
	ID      string  `json:"id"`
	Kind    string  `json:"kind"`
	Profile string  `json:"profile"`
	Context Context `json:"context"`
}

type Provider struct {
	Schema       string         `json:"schema"`
	ID           string         `json:"id"`
	Model        string         `json:"model"`
	Version      string         `json:"version"`
	Execution    string         `json:"execution"`
	Network      string         `json:"network"`
	Capabilities []string       `json:"capabilities"`
	Limits       Limits         `json:"limits"`
	Settings     map[string]any `json:"settings,omitempty"`
}

type Candidate struct {
	ID          string         `json:"id"`
	Kind        string         `json:"kind"`
	Subject     string         `json:"subject"`
	Payload     map[string]any `json:"payload"`
	Basis       []string       `json:"basis"`
	Assumptions []string       `json:"assumptions,omitempty"`
}

type Proposal struct {
	Schema      string      `json:"schema"`
	State       string      `json:"state"`
	Candidates  []Candidate `json:"candidates"`
	Diagnostics []string    `json:"diagnostics,omitempty"`
}

type ProviderIdentity struct {
	ID        string `json:"id"`
	Model     string `json:"model"`
	Version   string `json:"version"`
	Execution string `json:"execution"`
	Network   string `json:"network"`
}

type Receipt struct {
	Schema           string           `json:"schema"`
	RequestID        string           `json:"requestId"`
	Kind             string           `json:"kind"`
	Profile          string           `json:"profile"`
	Provider         ProviderIdentity `json:"provider"`
	RequestDigest    string           `json:"requestDigest"`
	ContextDigest    string           `json:"contextDigest"`
	ProviderDigest   string           `json:"providerDigest"`
	SettingsDigest   string           `json:"settingsDigest"`
	InputFingerprint string           `json:"inputFingerprint"`
	State            string           `json:"state"`
	Proposal         *Proposal        `json:"proposal,omitempty"`
	OutputDigest     string           `json:"outputDigest,omitempty"`
	Diagnostics      []string         `json:"diagnostics,omitempty"`
	Limits           Limits           `json:"limits"`
	GrantsAuthority  bool             `json:"grantsAuthority"`
	Accepted         bool             `json:"accepted"`
	Applied          bool             `json:"applied"`
	Notes            []string         `json:"notes"`
}

func digest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func readJSON(path string, into any) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(into); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if decoder.More() {
		return nil, fmt.Errorf("%s: trailing JSON value", path)
	}
	return raw, nil
}

func validateRequest(request Request) error {
	if request.Schema != RequestSchema {
		return fmt.Errorf("request schema %q is not %q", request.Schema, RequestSchema)
	}
	if strings.TrimSpace(request.ID) == "" || strings.TrimSpace(request.Kind) == "" {
		return fmt.Errorf("request id and kind are required")
	}
	if request.Profile != ProfileOffline && request.Profile != ProfileConnected {
		return fmt.Errorf("request profile must be %q or %q", ProfileOffline, ProfileConnected)
	}
	if strings.TrimSpace(request.Context.WorkspaceRevision) == "" {
		return fmt.Errorf("context workspaceRevision is required")
	}
	if len(request.Context.AcceptedReferences) == 0 && len(request.Context.Observations) == 0 {
		return fmt.Errorf("context must contain at least one accepted reference or observation")
	}
	seenRef := map[string]bool{}
	for _, ref := range request.Context.AcceptedReferences {
		if strings.TrimSpace(ref) == "" {
			return fmt.Errorf("accepted reference must not be empty")
		}
		if seenRef[ref] {
			return fmt.Errorf("duplicate accepted reference %q", ref)
		}
		seenRef[ref] = true
	}
	seenObs := map[string]bool{}
	for _, observation := range request.Context.Observations {
		if strings.TrimSpace(observation.ID) == "" || strings.TrimSpace(observation.Statement) == "" {
			return fmt.Errorf("observation id and statement are required")
		}
		if seenObs[observation.ID] {
			return fmt.Errorf("duplicate observation id %q", observation.ID)
		}
		seenObs[observation.ID] = true
	}
	return nil
}

func validateProvider(provider Provider) error {
	if provider.Schema != ProviderSchema {
		return fmt.Errorf("provider schema %q is not %q", provider.Schema, ProviderSchema)
	}
	for name, value := range map[string]string{
		"id": provider.ID, "model": provider.Model, "version": provider.Version,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("provider %s is required", name)
		}
	}
	if provider.Execution != ExecutionLocal && provider.Execution != ExecutionExternal {
		return fmt.Errorf("provider execution must be %q or %q", ExecutionLocal, ExecutionExternal)
	}
	if provider.Network != NetworkNone && provider.Network != NetworkRequired {
		return fmt.Errorf("provider network must be %q or %q", NetworkNone, NetworkRequired)
	}
	if provider.Execution == ExecutionLocal && provider.Network != NetworkNone {
		return fmt.Errorf("local provider must declare network %q", NetworkNone)
	}
	if len(provider.Capabilities) == 0 {
		return fmt.Errorf("provider capabilities must not be empty")
	}
	if provider.Limits.MaxContextBytes < 1 || provider.Limits.MaxOutputBytes < 1 || provider.Limits.TimeoutSeconds < 1 {
		return fmt.Errorf("provider limits must be positive")
	}
	return nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// Propose runs one bounded provider against one request and returns a receipt.
// Provider failure, unsupported capability, offline blocking, and an unknown
// proposal state are returned as distinct states; they are not collapsed into
// an empty proposal or a success.
func Propose(ctx context.Context, requestPath, providerPath, command string) (Receipt, error) {
	var request Request
	requestRaw, err := readJSON(requestPath, &request)
	if err != nil {
		return Receipt{}, err
	}
	if err := validateRequest(request); err != nil {
		return Receipt{}, err
	}
	var provider Provider
	providerRaw, err := readJSON(providerPath, &provider)
	if err != nil {
		return Receipt{}, err
	}
	if err := validateProvider(provider); err != nil {
		return Receipt{}, err
	}
	contextRaw, err := json.Marshal(request.Context)
	if err != nil {
		return Receipt{}, err
	}
	settingsRaw, err := json.Marshal(provider.Settings)
	if err != nil {
		return Receipt{}, err
	}
	receipt := Receipt{
		Schema: ReceiptSchema, RequestID: request.ID, Kind: request.Kind, Profile: request.Profile,
		Provider: ProviderIdentity{
			ID: provider.ID, Model: provider.Model, Version: provider.Version,
			Execution: provider.Execution, Network: provider.Network,
		},
		RequestDigest: digest(requestRaw), ContextDigest: digest(contextRaw),
		ProviderDigest: digest(providerRaw), SettingsDigest: digest(settingsRaw),
		InputFingerprint: digest(append(append([]byte(request.Kind+"\x00"), contextRaw...), 0)),
		Limits:           provider.Limits,
		GrantsAuthority:  false, Accepted: false, Applied: false,
		Notes: []string{
			"認知ポートは提案だけを返し、意味の受理・権限・実行を決めません。",
			"receiptはShared Acceptanceの代わりではありません。候補は別の受理境界へ渡す必要があります。",
			"providerの自己申告や提案の成功は、正しさ・権限・本番動作の証拠ではありません。",
		},
	}
	if !contains(provider.Capabilities, request.Kind) {
		receipt.State = StateUnsupported
		receipt.Diagnostics = []string{"provider does not declare capability " + request.Kind}
		return receipt, nil
	}
	if request.Profile == ProfileOffline && provider.Network != NetworkNone {
		receipt.State = StateBlocked
		receipt.Diagnostics = []string{"offline request requires a provider that declares no network access"}
		return receipt, nil
	}
	if len(contextRaw) > provider.Limits.MaxContextBytes {
		receipt.State = StateFailed
		receipt.Diagnostics = []string{fmt.Sprintf("context is %d bytes, limit is %d", len(contextRaw), provider.Limits.MaxContextBytes)}
		return receipt, nil
	}
	var proposal *Proposal
	switch provider.Execution {
	case ExecutionLocal:
		proposal, err = proposeLocal(request, provider)
	case ExecutionExternal:
		proposal, err = proposeExternal(ctx, request, provider, command, contextRaw)
	default:
		err = fmt.Errorf("unsupported execution %q", provider.Execution)
	}
	if err != nil {
		receipt.State = StateFailed
		receipt.Diagnostics = []string{err.Error()}
		return receipt, nil
	}
	if err := validateProposal(*proposal, request); err != nil {
		receipt.State = StateFailed
		receipt.Diagnostics = []string{"proposal is not valid: " + err.Error()}
		return receipt, nil
	}
	proposalRaw, err := json.Marshal(proposal)
	if err != nil {
		return Receipt{}, err
	}
	receipt.Proposal = proposal
	receipt.OutputDigest = digest(proposalRaw)
	receipt.State = proposal.State
	receipt.Diagnostics = proposal.Diagnostics
	return receipt, nil
}

func proposeLocal(request Request, provider Provider) (*Proposal, error) {
	strategy, _ := provider.Settings["strategy"].(string)
	if strategy == "" {
		strategy = "first"
	}
	if strategy != "first" && strategy != "all" {
		return nil, fmt.Errorf("unknown local strategy %q", strategy)
	}
	switch request.Kind {
	case "validation-candidate":
		var candidates []Candidate
		for _, observation := range request.Context.Observations {
			if !looksLikeNegativeValue(observation.Statement) || strings.TrimSpace(observation.Reference) == "" {
				continue
			}
			candidates = append(candidates, Candidate{
				ID:      "candidate-" + observation.ID,
				Kind:    request.Kind,
				Subject: observation.Reference,
				Payload: map[string]any{
					"expression": observation.Reference + " >= 0",
					"severity":   "error",
				},
				Basis: []string{"observation:" + observation.ID},
				Assumptions: []string{
					"the accepted type is numeric",
					"the observation is a design signal, not an accepted fact",
				},
			})
			if strategy == "first" {
				break
			}
		}
		if len(candidates) == 0 {
			return &Proposal{Schema: ProposalSchema, State: StateUnknown, Diagnostics: []string{"no observation matched the local validation rule"}}, nil
		}
		return &Proposal{Schema: ProposalSchema, State: StateProposed, Candidates: candidates}, nil
	case "diagnosis":
		if len(request.Context.Observations) == 0 {
			return &Proposal{Schema: ProposalSchema, State: StateUnknown, Diagnostics: []string{"no observation is available for diagnosis"}}, nil
		}
		observation := request.Context.Observations[0]
		candidate := Candidate{
			ID:      "diagnosis-" + observation.ID,
			Kind:    request.Kind,
			Subject: observation.Reference,
			Payload: map[string]any{
				"cause":  "missing or incomplete constraint",
				"signal": observation.Statement,
			},
			Basis:       []string{"observation:" + observation.ID},
			Assumptions: []string{"the observation is a design signal, not an accepted fact"},
		}
		return &Proposal{Schema: ProposalSchema, State: StateProposed, Candidates: []Candidate{candidate}}, nil
	default:
		return &Proposal{Schema: ProposalSchema, State: StateUnsupported, Diagnostics: []string{"local provider does not implement kind " + request.Kind}}, nil
	}
}

func looksLikeNegativeValue(statement string) bool {
	lower := strings.ToLower(statement)
	return strings.Contains(lower, "negative") || strings.Contains(statement, "負") || strings.Contains(statement, "未満")
}

func proposeExternal(ctx context.Context, request Request, provider Provider, command string, contextRaw []byte) (*Proposal, error) {
	if strings.TrimSpace(command) == "" {
		return nil, fmt.Errorf("external provider requires a command")
	}
	tempDir, err := os.MkdirTemp("", "tei-cognition-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)
	contextPath := filepath.Join(tempDir, "context.json")
	outputPath := filepath.Join(tempDir, "proposal.json")
	if err := os.WriteFile(contextPath, contextRaw, 0o600); err != nil {
		return nil, err
	}
	runCtx := ctx
	if provider.Limits.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(provider.Limits.TimeoutSeconds)*time.Second)
		defer cancel()
	}
	cmd := exec.CommandContext(runCtx, "bash", "-c", command)
	cmd.Env = append(os.Environ(),
		"TEI_COGNITION_CONTEXT="+contextPath,
		"TEI_COGNITION_OUTPUT="+outputPath,
		"TEI_COGNITION_PROVIDER="+provider.ID,
		"TEI_COGNITION_MODEL="+provider.Model,
		"TEI_COGNITION_VERSION="+provider.Version,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("cognition command failed: %w: %s", err, output)
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		return nil, fmt.Errorf("cognition command did not write TEI_COGNITION_OUTPUT: %w", err)
	}
	if info.Size() == 0 {
		return nil, fmt.Errorf("cognition command wrote an empty proposal")
	}
	if info.Size() > int64(provider.Limits.MaxOutputBytes) {
		return nil, fmt.Errorf("proposal is %d bytes, limit is %d", info.Size(), provider.Limits.MaxOutputBytes)
	}
	var proposal Proposal
	if _, err := readJSON(outputPath, &proposal); err != nil {
		return nil, err
	}
	return &proposal, nil
}

func validateProposal(proposal Proposal, request Request) error {
	if proposal.Schema != ProposalSchema {
		return fmt.Errorf("schema %q is not %q", proposal.Schema, ProposalSchema)
	}
	switch proposal.State {
	case StateProposed:
		if len(proposal.Candidates) == 0 {
			return fmt.Errorf("proposed state requires at least one candidate")
		}
	case StateUnsupported, StateUnknown, StateFailed:
		if len(proposal.Candidates) != 0 {
			return fmt.Errorf("%s state must not contain candidates", proposal.State)
		}
	default:
		return fmt.Errorf("unknown proposal state %q", proposal.State)
	}
	basisPattern := regexp.MustCompile(`^(observation|reference|constraint):[A-Za-z0-9._:-]+$`)
	seen := map[string]bool{}
	for _, candidate := range proposal.Candidates {
		if strings.TrimSpace(candidate.ID) == "" || strings.TrimSpace(candidate.Kind) == "" || strings.TrimSpace(candidate.Subject) == "" {
			return fmt.Errorf("candidate id, kind, and subject are required")
		}
		if seen[candidate.ID] {
			return fmt.Errorf("duplicate candidate id %q", candidate.ID)
		}
		seen[candidate.ID] = true
		if candidate.Kind != request.Kind {
			return fmt.Errorf("candidate kind %q does not match request kind %q", candidate.Kind, request.Kind)
		}
		if len(candidate.Payload) == 0 {
			return fmt.Errorf("candidate %s has no payload", candidate.ID)
		}
		if len(candidate.Basis) == 0 {
			return fmt.Errorf("candidate %s has no basis", candidate.ID)
		}
		for _, basis := range candidate.Basis {
			if !basisPattern.MatchString(basis) {
				return fmt.Errorf("candidate %s has invalid basis %q", candidate.ID, basis)
			}
			if !basisIsPresent(basis, request) {
				return fmt.Errorf("candidate %s cites absent basis %q", candidate.ID, basis)
			}
		}
	}
	return nil
}

func basisIsPresent(basis string, request Request) bool {
	parts := strings.SplitN(basis, ":", 2)
	if len(parts) != 2 {
		return false
	}
	switch parts[0] {
	case "observation":
		for _, observation := range request.Context.Observations {
			if observation.ID == parts[1] {
				return true
			}
		}
	case "reference":
		for _, reference := range request.Context.AcceptedReferences {
			if reference == parts[1] {
				return true
			}
		}
	case "constraint":
		for i := range request.Context.Constraints {
			if fmt.Sprintf("%d", i) == parts[1] {
				return true
			}
		}
	}
	return false
}
