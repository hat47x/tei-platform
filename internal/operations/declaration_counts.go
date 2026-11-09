package operations

import "tei/internal/ir"

// DeclarationCounts counts the declarations a Workspace itself authored. The
// in-process standard semantic prelude (`std`) is excluded: otherwise an empty
// Workspace would still report non-zero types and a caller could not tell
// "nothing was declared" from "the standard library was loaded" (#2822).
//
// These are counts of the normalized World, not proof that the declarations
// were accepted, that they are consistent, or that they were executed.
type DeclarationCounts struct {
	Namespaces          int `json:"namespaces"`
	Types               int `json:"types"`
	Data                int `json:"data"`
	Attributes          int `json:"attributes"`
	Capabilities        int `json:"capabilities"`
	Validations         int `json:"validations"`
	Periods             int `json:"periods"`
	EffectRules         int `json:"effectRules"`
	RunPlans            int `json:"runPlans"`
	Applications        int `json:"applications"`
	Patterns            int `json:"patterns"`
	Databases           int `json:"databases"`
	Environments        int `json:"environments"`
	Queries             int `json:"queries"`
	Plugins             int `json:"plugins"`
	Bundles             int `json:"bundles"`
	PersistenceMappings int `json:"persistenceMappings"`
	Total               int `json:"total"`
}

// CountDeclarations counts what the Workspace itself authored. The prelude
// namespace and the twelve types it declares are excluded; anything else is
// counted, including a type the Workspace put inside the std namespace.
//
// Excluding the whole `std` namespace by name was wrong (#3851): a Workspace
// that writes `namespace std { type email from std.text }` declared a model
// element, yet its declarationCounts stayed zero and validate warned that it
// declared nothing, while inspect happily resolved the type from its file.
func CountDeclarations(world ir.World) DeclarationCounts {
	counts := DeclarationCounts{}
	for _, ns := range world.Semantics {
		preludeNamespace := string(ns.ID) == stdlibNamespaceID
		types := len(ns.Types)
		if preludeNamespace {
			// Subtract only the types the prelude itself contributes.
			for _, typ := range ns.Types {
				if isPreludeType(string(typ.ID)) {
					types--
				}
			}
		}
		if preludeNamespace && types == 0 && len(ns.Data) == 0 && len(ns.Capabilities) == 0 &&
			len(ns.Validations) == 0 {
			// The prelude namespace with nothing added is not a declaration.
			continue
		}
		counts.Namespaces++
		counts.Types += types
		counts.Data += len(ns.Data)
		for _, data := range ns.Data {
			counts.Attributes += len(data.Attributes)
		}
		counts.Capabilities += len(ns.Capabilities)
		counts.Validations += len(ns.Validations)
		counts.Periods += len(ns.Periods)
		counts.EffectRules += len(ns.EffectRules)
		counts.RunPlans += len(ns.RunPlans)
	}
	counts.Applications = len(world.Applications)
	for _, app := range world.Applications {
		counts.Patterns += len(app.Patterns)
	}
	counts.Databases = len(world.Databases)
	counts.Environments = len(world.Environments)
	counts.Queries = len(world.Queries)
	counts.Plugins = len(world.Plugins)
	counts.Bundles = len(world.Bundles)
	counts.PersistenceMappings = len(world.Persistence)
	counts.Total = counts.Namespaces + counts.Types + counts.Data + counts.Attributes +
		counts.Capabilities + counts.Validations + counts.Periods + counts.EffectRules +
		counts.RunPlans + counts.Applications + counts.Patterns + counts.Databases +
		counts.Environments + counts.Queries + counts.Plugins + counts.Bundles +
		counts.PersistenceMappings
	return counts
}
