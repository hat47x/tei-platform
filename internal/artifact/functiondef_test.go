package artifact

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/semantic"
)

func functionDefinitionInput() FunctionDefinitionInput {
	return FunctionDefinitionInput{
		InputRevision: "sha256:input",
		Application: application.Application{
			ID:          "order_entry",
			Validations: []core.CanonicalID{"sales.order.order_id_positive"},
			Patterns: []application.Pattern{
				{ID: "order_entry.recent_orders", Kind: "browse"},
				{ID: "order_entry.order_record", Kind: "record"},
			},
		},
		Capabilities: map[core.CanonicalID]semantic.Capability{
			"sales.order.submit": {
				ID:     "sales.order.submit",
				Inputs: []semantic.Parameter{{Name: "order", Type: "sales.order"}},
				Output: "sales.order",
				Effect: &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"},
			},
		},
		Validations: map[core.CanonicalID]semantic.Validation{
			"sales.order.order_id_positive": {ID: "sales.order.order_id_positive", Kind: "positive"},
		},
	}
}

func TestRenderFunctionDefinitionIsDeterministic(t *testing.T) {
	first, err := RenderFunctionDefinition(functionDefinitionInput())
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	second, err := RenderFunctionDefinition(functionDefinitionInput())
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if first.Content != second.Content || first.Digest != second.Digest {
		t.Fatalf("the same input must render the same bytes")
	}
	if err := first.Verify(); err != nil {
		t.Fatalf("rendered artifact must verify: %v", err)
	}
	if first.Kind != string(KindFunctionDefinition) || first.Rule.ID != FunctionDefinitionRuleID {
		t.Fatalf("artifact identity is incomplete: %+v", first)
	}

	for _, line := range []string{
		"application order_entry\n",
		"capability sales.order.submit\n",
		"input order sales.order\n",
		"output sales.order\n",
		"effect create sales.order\n",
		"validation sales.order.order_id_positive positive\n",
		"pattern order_entry.order_record record\n",
	} {
		if !strings.Contains(first.Content, line) {
			t.Fatalf("content lost %q:\n%s", line, first.Content)
		}
	}
	if len(first.Unresolved) != 0 {
		t.Fatalf("a complete application must have no unresolved entries: %+v", first.Unresolved)
	}
}

func TestRenderFunctionDefinitionReportsIncompleteCapabilities(t *testing.T) {
	input := functionDefinitionInput()
	input.Capabilities["sales.order.cancel"] = semantic.Capability{ID: "sales.order.cancel"}

	rendered, err := RenderFunctionDefinition(input)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(rendered.Unresolved) != 3 {
		t.Fatalf("expected three unresolved entries: %+v", rendered.Unresolved)
	}
	for _, unresolved := range rendered.Unresolved {
		if unresolved.Subject != "sales.order.cancel" || unresolved.Detail == "" {
			t.Fatalf("unexpected unresolved entry: %+v", unresolved)
		}
	}
}

func TestRenderFunctionDefinitionRequiresAnEntryPoint(t *testing.T) {
	input := functionDefinitionInput()
	input.Application.Patterns = nil

	rendered, err := RenderFunctionDefinition(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(rendered.Unresolved) != 1 || rendered.Unresolved[0].Subject != "order_entry" {
		t.Fatalf("an application without a pattern must be unresolved: %+v", rendered.Unresolved)
	}
}
func TestRenderFunctionDefinitionShowsPeriods(t *testing.T) {
	input := functionDefinitionInput()
	input.Periods = map[core.CanonicalID]semantic.Period{
		"sales.month": {
			ID:     "sales.month",
			Data:   "sales.order",
			States: []string{"open", "closing", "closed"},
		},
	}

	rendered, err := RenderFunctionDefinition(input)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "period sales.month data sales.order states open,closing,closed\n"
	if !strings.Contains(rendered.Content, want) {
		t.Fatalf("content lost the period:\n%s", rendered.Content)
	}
}
func TestRenderFunctionDefinitionShowsEffectRules(t *testing.T) {
	input := functionDefinitionInput()
	input.EffectRules = []semantic.EffectRule{{
		ID:            "sales.order.stock",
		OnCapability:  "sales.order.register",
		Stage:         "after-commit",
		EffectKind:    "update",
		EffectData:    "sales.order",
		ViaCapability: "sales.order.register",
		OnFailure:     "compensate",
	}}

	rendered, err := RenderFunctionDefinition(input)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "effect-rule sales.order.stock on sales.order.register stage after-commit effect update sales.order via sales.order.register on-failure compensate\n"
	if !strings.Contains(rendered.Content, want) {
		t.Fatalf("content lost the effect rule:\n%s", rendered.Content)
	}
}
