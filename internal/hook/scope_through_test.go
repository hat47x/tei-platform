package hook_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/syntax"
	"tei/internal/validation"
)

// この試験は「宣言 → 検証 → 実行」を1本で通します。
//
// 範囲（scope）を層ごとに足したとき、各層の試験は緑のまま**貫通する経路だけが
// 存在しない**状態が起きました（`row`は宣言できず、実装全体が到達不能でした）。
// 層ごとの試験では、その隙間が見えません。ここではソースから書き起こした宣言が、
// 検証を通り、実際にランナーで動くところまでを1本で確かめます。

// scopeThroughSource is an application pattern with one hook whose scope the test
// parameterises. The semantic namespace declares the attribute the hook reads.
func scopeThroughSource(scope string) string {
	scopeLine := ""
	if scope != "" {
		scopeLine = "\n            scope " + scope
	}
	return `pattern order_entry.order_record record {
    data sales.order
    interaction {
        hook before-update {` + scopeLine + `
            reads sales.order.amount
        }
    }
}`
}

// scopeThroughSemantic is the semantic half. The parser reads one namespace per
// file, so the semantic source and the pattern source are parsed separately.
const scopeThroughSemantic = `namespace sales

data order {
    attribute order_id type std.identifier
    attribute amount type std.decimal
}
`

// scopeThroughWorld parses the source into a world with the standard prelude, so the
// attribute type resolves.
func scopeThroughWorld(t *testing.T, source string) (ir.World, application.Pattern) {
	t.Helper()
	semanticNamespace, err := syntax.NewParser(scopeThroughSemantic).ParseSemanticFile()
	if err != nil {
		t.Fatalf("parse semantic: %v", err)
	}
	pattern, err := syntax.NewParser(source).ParseApplicationPattern()
	if err != nil {
		t.Fatalf("parse pattern: %v", err)
	}
	world := ir.World{
		Semantics: []semantic.Namespace{
			{ID: "std", Types: []semantic.Type{
				{ID: "std.identifier", Symbol: "identifier", Kind: semantic.TypePrimitive},
				{ID: "std.decimal", Symbol: "decimal", Kind: semantic.TypePrimitive},
			}},
			semanticNamespace,
		},
		Applications: []application.Application{{
			ID:           "order_entry",
			SemanticUses: []core.CanonicalID{"sales"},
			Patterns:     []application.Pattern{pattern},
		}},
	}
	return world, pattern
}

// throughImplementation records the values each call was given.
type throughImplementation struct {
	seen [][]string
}

func (i *throughImplementation) Run(_ context.Context, invocation hook.Invocation) error {
	i.seen = append(i.seen, []string{invocation.Reads["sales.order.amount"]})
	return nil
}

// TestScopeIsDeclarableValidatedAndRun checks the three layers in one test: the
// declaration parses, validation accepts it, and the runner applies it.
//
// A scope that parses and validates but cannot be run would pass the first two
// layers and fail the third, which is what this test is for.
func TestScopeIsDeclarableValidatedAndRun(t *testing.T) {
	for _, scope := range []string{"", "record", "row"} {
		t.Run("scope="+scope, func(t *testing.T) {
			world, pattern := scopeThroughWorld(t, scopeThroughSource(scope))
			// Layer 2: validation.
			for _, diagnostic := range validation.Validate(world) {
				if diagnostic.Severity == validation.Error {
					t.Fatalf("validation refused the declaration: %s %s", diagnostic.Code, diagnostic.Message)
				}
			}
			if pattern.Interaction == nil || len(pattern.Interaction.Hooks) != 1 {
				t.Fatalf("the hook was not parsed: %+v", pattern.Interaction)
			}
			if pattern.Interaction.Hooks[0].Scope != scope {
				t.Fatalf("scope=%q, want %q", pattern.Interaction.Hooks[0].Scope, scope)
			}

			// Layer 3: the runner. A row scope needs the row set; a record scope
			// runs once.
			implementation := &throughImplementation{}
			runner := hook.Runner{World: world, Implementation: implementation}
			if scope == "row" {
				runner.Rows = []map[core.CanonicalID]string{
					{"sales.order.amount": "10"},
					{"sales.order.amount": "20"},
				}
			}
			if err := runner.Run(context.Background(), hook.Invocation{
				Application: "order_entry",
				Pattern:     "order_entry.order_record",
				Phase:       "before-update",
				Reads:       map[core.CanonicalID]string{"sales.order.amount": "10"},
			}); err != nil {
				t.Fatalf("the runner refused a declaration validation accepted: %v", err)
			}
			want := 1
			if scope == "row" {
				want = 2
			}
			if len(implementation.seen) != want {
				t.Fatalf("calls=%d, want %d", len(implementation.seen), want)
			}
		})
	}
}

// TestScopeValidationAndRunAgreeOnRefusal checks the other direction: a scope the
// runner will not apply is also refused by validation, so the two layers do not
// disagree about what is declarable.
func TestScopeValidationAndRunAgreeOnRefusal(t *testing.T) {
	// item is accepted by validation for a command pattern only, and a command
	// pattern cannot declare an interaction. So on a record pattern validation
	// refuses it, and the runner must not silently accept it either.
	world, _ := scopeThroughWorld(t, scopeThroughSource("item"))
	var refused bool
	for _, diagnostic := range validation.Validate(world) {
		if diagnostic.Code == "APPLICATION_PATTERN_INTERACTION_HOOK_SCOPE_MISMATCH" {
			refused = true
		}
	}
	if !refused {
		t.Fatal("validation accepted a scope it cannot check")
	}

	// The runner refuses it too, so a caller that skipped validation does not get a
	// silently inert hook.
	implementation := &throughImplementation{}
	runner := hook.Runner{World: world, Implementation: implementation}
	err := runner.Run(context.Background(), hook.Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Phase:       "before-update",
	})
	if err == nil || !strings.Contains(err.Error(), "HOOK_SCOPE_UNAPPLIED") {
		t.Fatalf("the runner did not refuse an unapplied scope: %v", err)
	}
	if len(implementation.seen) != 0 {
		t.Fatalf("the hook ran with an unapplied scope: %d calls", len(implementation.seen))
	}
}
