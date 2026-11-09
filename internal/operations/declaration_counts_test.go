package operations

import (
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// TestCountDeclarationsExcludesTheStandardPrelude pins the distinction an
// empty Workspace must preserve: the std namespace is loaded, but it is not a
// declaration the Workspace authored (#2822).
func TestCountDeclarationsExcludesTheStandardPrelude(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{
		{ID: "std", Types: []semantic.Type{{ID: "std.text"}, {ID: "std.integer"}}},
		{ID: "sales", Data: []semantic.Data{{
			ID:         "sales.order",
			Attributes: []semantic.Attribute{{ID: "sales.order.order_id"}, {ID: "sales.order.total"}},
		}}, Capabilities: []semantic.Capability{{ID: "sales.submit"}}},
	}, Applications: []application.Application{{ID: "order_entry", Patterns: []application.Pattern{{ID: "order_entry.list"}}}}}
	counts := CountDeclarations(world)
	if counts.Namespaces != 1 || counts.Types != 0 || counts.Data != 1 ||
		counts.Attributes != 2 || counts.Capabilities != 1 || counts.Applications != 1 || counts.Patterns != 1 {
		t.Fatalf("counts=%+v", counts)
	}
	if counts.Total != 7 {
		t.Fatalf("total=%d want 7 (%+v)", counts.Total, counts)
	}
}

func TestCountDeclarationsOfAnEmptyWorldIsZero(t *testing.T) {
	counts := CountDeclarations(ir.World{Semantics: []semantic.Namespace{{ID: "std", Types: []semantic.Type{{ID: "std.text"}}}}})
	if counts.Total != 0 || counts.Namespaces != 0 || counts.Types != 0 {
		t.Fatalf("empty workspace counts=%+v", counts)
	}
}
