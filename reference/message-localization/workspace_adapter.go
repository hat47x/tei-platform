package messagelocalization

import (
"fmt"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
	"tei/internal/workspace"
	messages "tei/reference/standard-messages"
)

// WorkspaceRequest carries extension data not yet supported by authoritative
// Workspace IR. Host acceptance remains mandatory for these declarations.
type WorkspaceRequest struct {
	Root        string
	Environment core.CanonicalID
	Lock        plugin.Lock
	Messages    messages.Catalog
	Stores      []StoreDeclaration
	Bindings    []ApplicationBinding
	Records     []TranslationRecord
	Grants      []Grant
	Overrides   []Override
	Policy      Policy
	Request     Request
	Review      ReviewGate
}

// ResolveWorkspace uses the real Workspace loader for plugin and environment
// declarations. The caller must provide an existing lock and host review.
func ResolveWorkspace(req WorkspaceRequest) (Output, error) {
	if req.Review == nil {
		return Output{}, fmt.Errorf("workspace-message: host-review-required")
	}
	loaded, err := workspace.Load(req.Root)
	if err != nil {
		return Output{}, fmt.Errorf("workspace-message: load: %w", err)
	}
	var selected environment.Environment
	found := false
	for _, env := range loaded.World.Environments {
		if env.ID != req.Environment {
			continue
		}
		if found {
			return Output{}, fmt.Errorf("workspace-message: ambiguous-environment %s", req.Environment)
		}
		selected = env
		found = true
	}
	if !found {
		return Output{}, fmt.Errorf("workspace-message: missing-environment %s", req.Environment)
	}
	return Resolve(Input{
		Plugins:     plugin.Catalog{Plugins: loaded.World.Plugins, Bundles: loaded.World.Bundles},
		Environment: selected,
		Lock:        req.Lock,
		Messages:    req.Messages,
		Stores:      req.Stores,
		Bindings:    req.Bindings,
		Records:     req.Records,
		Grants:      req.Grants,
		Overrides:   req.Overrides,
		Policy:      req.Policy,
		Request:     req.Request,
		Review:      req.Review,
	})
}
