package provider

import (
	"context"
	"fmt"

	"tei/internal/database/physical"
	"tei/internal/database/plan"
	"tei/internal/environment"
	"tei/internal/query"
)

type ExplainOptions struct {
	Analyze bool
	Buffers bool
}

// Explainer is an optional extension to the small Database Provider SPI. A
// provider may support metadata introspection without exposing plan execution.
type Explainer interface {
	Provider
	Explain(context.Context, environment.DatabaseInstance, query.EmbeddedSQL, *physical.SchemaSnapshot, ExplainOptions) (plan.IR, error)
}

func Explain(ctx context.Context, p Explainer, instance environment.DatabaseInstance, q query.EmbeddedSQL, snapshot *physical.SchemaSnapshot, opts ExplainOptions) (plan.IR, error) {
	if p.ID() != instance.Provider {
		return plan.IR{}, fmt.Errorf("database provider mismatch: instance %s requires %s, provider is %s", instance.ID, instance.Provider, p.ID())
	}
	caps, err := p.Capabilities(ctx, instance)
	if err != nil {
		return plan.IR{}, err
	}
	required := CapabilityExplain
	if opts.Analyze {
		required = CapabilityExplainAnalyze
	}
	if !caps.Has(required) {
		return plan.IR{}, fmt.Errorf("database provider %s does not expose %s", p.ID(), required)
	}
	out, err := p.Explain(ctx, instance, q, snapshot, opts)
	if err != nil {
		return plan.IR{}, err
	}
	if out.Database != instance.Definition {
		return plan.IR{}, fmt.Errorf("execution plan database %s does not match instance definition %s", out.Database, instance.Definition)
	}
	if out.Query != q.ID {
		return plan.IR{}, fmt.Errorf("execution plan query %s does not match requested query %s", out.Query, q.ID)
	}
	return out, nil
}
