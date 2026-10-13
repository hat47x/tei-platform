package webapicontract

import (
	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// PublishedCommand only identifies an existing command Pattern and its
// underlying Capability contract. It does not grant execution permission,
// infer CRUD behavior or create an HTTP endpoint.
type PublishedCommand struct {
	ApplicationID core.CanonicalID
	PatternID     core.CanonicalID
	CapabilityID  core.CanonicalID
	Inputs        []semantic.Parameter
}

// ResolvePublishedCommand is a reference adapter for a trusted caller.
// Grant is a synthetic external authorization decision, NOT a permission
// inferred from a browser-supplied Action name or a Plugin declaration.
func ResolvePublishedCommand(
	world ir.World,
	applicationID, patternID core.CanonicalID,
	granted bool,
) (PublishedCommand, error) {
	if !granted {
		return PublishedCommand{}, ErrRejected
	}
	resolved, err := driver.NewResolver(world).ResolvePattern(applicationID, patternID)
	if err != nil || resolved.Pattern.Kind != application.PatternCommand ||
		resolved.Capability == nil || resolved.Capability.ID == "" {
		return PublishedCommand{}, ErrRejected
	}
	result := PublishedCommand{
		ApplicationID: applicationID,
		PatternID:     patternID,
		CapabilityID:  resolved.Capability.ID,
	}
	for _, input := range resolved.Capability.Inputs {
		if input.Name == "" || input.Type == "" {
			return PublishedCommand{}, ErrRejected
		}
		result.Inputs = append(result.Inputs, input)
	}
	return result, nil
}
