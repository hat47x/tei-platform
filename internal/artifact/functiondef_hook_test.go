package artifact

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
)

// hookFunctionDefinitionInput declares a record pattern with one hook, so the
// rendering of the hook can be checked without a wider fixture.
func hookFunctionDefinitionInput() FunctionDefinitionInput {
	input := functionDefinitionInput()
	input.Application.Patterns = []application.Pattern{{
		ID:   "order_entry.order_record",
		Kind: application.PatternRecord,
		Interaction: &application.Interaction{
			Hooks: []application.Hook{{
				Phase:    "before-update",
				Reads:    []core.CanonicalID{"sales.order.quantity"},
				Uses:     []core.CanonicalID{"sales.order.submit"},
				Produces: []core.CanonicalID{"sales.order.amount"},
				Failure:  "reject",
			}},
		},
	}}
	return input
}

// TestRenderFunctionDefinitionShowsHooks checks that a declared hook appears in
// the function definition, under its pattern, with its phase and declared reads,
// capability, proposal and failure treatment.
func TestRenderFunctionDefinitionShowsHooks(t *testing.T) {
	rendered, err := RenderFunctionDefinition(hookFunctionDefinitionInput())
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "pattern order_entry.order_record record\n  hook before-update reads sales.order.quantity uses sales.order.submit produces sales.order.amount on-failure reject\n"
	if !strings.Contains(rendered.Content, want) {
		t.Fatalf("content lost the hook declaration:\n%s", rendered.Content)
	}
	if len(rendered.Unresolved) != 0 {
		t.Fatalf("unresolved=%+v", rendered.Unresolved)
	}
}

// TestRenderFunctionDefinitionOrdersHooksByPhase checks that several hooks of a
// pattern are listed in a fixed order, not in declaration order.
func TestRenderFunctionDefinitionOrdersHooksByPhase(t *testing.T) {
	input := hookFunctionDefinitionInput()
	input.Application.Patterns[0].Interaction.Hooks = []application.Hook{
		{Phase: "after-update", Failure: "warn"},
		{Phase: "before-read"},
	}
	rendered, err := RenderFunctionDefinition(input)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	before := strings.Index(rendered.Content, "  hook before-read\n")
	after := strings.Index(rendered.Content, "  hook after-update on-failure warn\n")
	if before < 0 || after < 0 || before > after {
		t.Fatalf("hooks are not listed in phase order:\n%s", rendered.Content)
	}
}

// TestRenderFunctionDefinitionWithoutHooksHasNoHookLine checks that a pattern
// without a hook gains no line, so an empty declaration is not implied.
func TestRenderFunctionDefinitionWithoutHooksHasNoHookLine(t *testing.T) {
	input := functionDefinitionInput()
	input.Application.Patterns = []application.Pattern{{ID: "order_entry.recent_orders", Kind: "browse"}}
	rendered, err := RenderFunctionDefinition(input)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(rendered.Content, "hook ") {
		t.Fatalf("content claims a hook it does not declare:\n%s", rendered.Content)
	}
}
