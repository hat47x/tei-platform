package dependency

import "tei/internal/core"

type Scope string

const (
	ScopeEditor    Scope = "editor"
	ScopeBuild     Scope = "build"
	ScopeTest      Scope = "test"
	ScopeRuntime   Scope = "runtime"
	ScopeMigration Scope = "migration"
)

type Kind string

const (
	KindToolchain       Kind = "toolchain"
	KindRuntime         Kind = "runtime"
	KindLanguagePackage Kind = "language-package"
	KindTool            Kind = "tool"
)

type Origin struct {
	IntroducedBy string `json:"introducedBy"`
	Reason       string `json:"reason"`
}

// Requirement is normalized dependency intent. Requirement identity is
// environment/scope/kind/name; multiple origins are retained losslessly so a
// provider can resolve one package/tool while Cockpit can still explain every
// reason it is needed.
type Requirement struct {
	Environment core.CanonicalID `json:"environment"`
	Scope       Scope            `json:"scope"`
	Kind        Kind             `json:"kind"`
	Name        string           `json:"name"`
	Origins     []Origin         `json:"origins"`
}
