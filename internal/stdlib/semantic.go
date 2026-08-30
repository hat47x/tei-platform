package stdlib

import (
	"tei/internal/core"
	"tei/internal/semantic"
)

// SemanticNamespace is the M0 in-process representation of TEI's standard
// semantic type library. Keeping it outside the semantic core makes the
// distinction explicit: these are standard-library meanings, not kernel kinds.
// A later package/registry implementation can replace this prelude without
// changing the normalized IR.
func SemanticNamespace() semantic.Namespace {
	mk := func(name string) semantic.Type {
		return semantic.Type{
			ID:     core.MustCanonicalID("std." + name),
			Symbol: name,
			Kind:   semantic.TypePrimitive,
		}
	}
	return semantic.Namespace{
		ID: core.MustCanonicalID("std"),
		Types: []semantic.Type{
			mk("binary"),
			mk("boolean"),
			mk("code"),
			mk("date"),
			mk("datetime"),
			mk("decimal"),
			mk("identifier"),
			mk("integer"),
			mk("json"),
			mk("money"),
			mk("text"),
			mk("time"),
		},
	}
}
