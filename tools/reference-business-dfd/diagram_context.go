package main

import (
	"fmt"
	"strconv"
	"strings"
)

// DOT is a disposable read-only view. In particular, an explicit Handoff
// carries conditional business responsibility, not an executable edge.
func appendBusinessContextDOT(b *strings.Builder, p projection) {
	if p.ActorCandidate.Label != "" {
		id := "actor-candidate:" + p.ProcedureID
		fmt.Fprintf(b, "  %s [label=%s, shape=plaintext];\n",
			strconv.Quote(id), strconv.Quote("actor candidate: "+p.ActorCandidate.Label))
		// A context note is not a new control-flow edge to any one Step.
	}
	if p.TriggerCandidate.Text != "" {
		id := "trigger-context:" + p.ProcedureID
		fmt.Fprintf(b, "  %s [label=%s, shape=plaintext];\n",
			strconv.Quote(id), strconv.Quote("trigger context candidate: "+p.TriggerCandidate.Text))
	}
	for _, h := range p.Handoffs {
		from, to := "step:"+h.From.Step, "step:"+h.To.Step
		if h.From.Procedure != p.ProcedureID {
			from = "other-procedure:" + h.From.Procedure + "/" + h.From.Step
			fmt.Fprintf(b, "  %s [label=%s, shape=note];\n", strconv.Quote(from),
				strconv.Quote("other Procedure: "+h.From.Procedure+" / "+h.From.Step))
		}
		if h.To.Procedure != p.ProcedureID {
			to = "other-procedure:" + h.To.Procedure + "/" + h.To.Step
			fmt.Fprintf(b, "  %s [label=%s, shape=note];\n", strconv.Quote(to),
				strconv.Quote("other Procedure: "+h.To.Procedure+" / "+h.To.Step))
		}
		fmt.Fprintf(b, "  %s -> %s [style=dotted, label=%s];\n",
			strconv.Quote(from), strconv.Quote(to),
			strconv.Quote("business Handoff (not runtime): "+h.Condition))
	}
}
