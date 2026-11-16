// Package businessprocedure projects a one-level business procedure read-only.
//
// A business procedure holds what the Function and Data models cannot: who does
// the work, which steps a person performs, what starts the work, which branches
// a decision opens, and which existing Function a step refers to. The projection
// reuses the declared Application and Semantic identities and never restates
// them.
//
// It is a reading aid. It does not accept meaning, authorize execution, or
// create a second source of truth for any identity it names.
package businessprocedure

// FixtureSchemaVersion is the only accepted input schema. A caller that writes
// a different version is rejected rather than read optimistically.
const FixtureSchemaVersion = "business-procedure-model-reference-v0"

// FixtureArtifactRole marks a reference authoring fixture. The projection is
// built from a Reference-local document, not from an accepted Canonical source.
const FixtureArtifactRole = "reference-authoring-fixture"

// ProjectionSchemaVersion names the output shape.
const ProjectionSchemaVersion = "tei.business-procedure.projection/v0"

// EffectReadOnly marks a projection that authorizes no source write.
const EffectReadOnly = "read-only"

// Realization is how one business step is carried out.
type Realization string

const (
	// RealizationManual is work a person performs. It has no Function and never
	// becomes one; a manual step is business meaning the Function model cannot
	// express.
	RealizationManual Realization = "manual"
	// RealizationApplication is work done through a declared Application Pattern.
	RealizationApplication Realization = "application"
	// RealizationSystem is work done by a declared Semantic Capability or
	// Validation.
	RealizationSystem Realization = "system"
)

// realizations lists every accepted realization in report order.
var realizations = []Realization{
	RealizationManual,
	RealizationApplication,
	RealizationSystem,
}

// Valid reports whether the realization is one this package understands.
func (r Realization) Valid() bool {
	for _, accepted := range realizations {
		if r == accepted {
			return true
		}
	}
	return false
}

// ReferenceKind names the existing identity a step or the authorization clause
// points at. It is deliberately a small closed set: the projection can resolve
// these against the normalized World, and it cannot invent a target kind the
// World does not declare.
type ReferenceKind string

const (
	ReferenceCapability ReferenceKind = "capability"
	ReferenceValidation ReferenceKind = "validation"
	ReferencePattern    ReferenceKind = "application-pattern"
	ReferenceData       ReferenceKind = "data"
)

// subjectKind maps a reference kind to the Source Index subject kind that
// declares it. A kind with no entry cannot be located, which is why the set is
// closed.
func (k ReferenceKind) subjectKind() (string, bool) {
	switch k {
	case ReferenceCapability:
		return "semantic-capability", true
	case ReferenceValidation:
		return "semantic-validation", true
	case ReferencePattern:
		return "application-pattern", true
	case ReferenceData:
		return "semantic-data", true
	}
	return "", false
}

// Fixture is the authored input document.
type Fixture struct {
	SchemaVersion string `json:"schemaVersion"`
	ArtifactRole  string `json:"artifactRole"`
	// Canonical states that the document is an accepted Canonical source. The
	// projection refuses that claim rather than accepting it on the fixture's
	// word.
	Canonical   bool      `json:"canonical"`
	Application string    `json:"application,omitempty"`
	Procedure   Procedure `json:"procedure"`
	Steps       []Step    `json:"steps"`
	Branches    []Branch  `json:"branches,omitempty"`
}

// Procedure is the procedure identity and the business context around it.
type Procedure struct {
	ID      string  `json:"id"`
	Name    string  `json:"name,omitempty"`
	Purpose string  `json:"purpose,omitempty"`
	Trigger Trigger `json:"trigger"`
	// Actor is the business actor: a human or organizational role in business
	// words. It is not an authorization role and carries no grant.
	Actor         Actor         `json:"actor"`
	Authorization Authorization `json:"authorization"`
}

// Trigger is what starts the business work. It is business wording, not a
// runtime event: the projection never turns it into an invocation.
type Trigger struct {
	Status string `json:"status"`
	Text   string `json:"text"`
}

// Actor is a business actor label.
type Actor struct {
	Status string `json:"status"`
	Label  string `json:"label"`
}

// Authorization names the separate authorization concern. RoleCode is a
// dispatch label for the authorization system and must stay distinct from the
// business actor label; Capability names the declared authorization capability
// the projection resolves.
type Authorization struct {
	Status     string `json:"status"`
	RoleCode   string `json:"roleCode,omitempty"`
	Capability string `json:"capability,omitempty"`
}

// Step is one business step.
type Step struct {
	ID          string      `json:"id"`
	Action      string      `json:"action"`
	Realization Realization `json:"realization"`
	// ActorLabel overrides the procedure actor for this step. Empty inherits the
	// procedure actor.
	ActorLabel string `json:"actorLabel,omitempty"`
	// ManualInstruction is the wording a person needs. A manual step without it
	// is still kept; the projection does not invent one.
	ManualInstruction string `json:"manualInstruction,omitempty"`
	// AutoExecute is the fixture's claim that the step runs by itself. The
	// projection refuses it for a manual step: a business event is not a
	// Function launch.
	AutoExecute bool `json:"autoExecute,omitempty"`
	// Function is the existing identity this step uses. A manual step must not
	// carry one.
	Function *Reference `json:"function,omitempty"`
}

// Reference points at an existing declared identity. Kind and ID are both
// required: a bare ID is not enough to say which model the step uses.
type Reference struct {
	Kind ReferenceKind `json:"kind"`
	ID   string        `json:"id"`
}

// Branch is one business branch a decision opens. Approval and rejection are
// separate branches with separate identities, never one branch with two labels.
type Branch struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	OnStep  string `json:"onStep"`
	Outcome string `json:"outcome"`
}
