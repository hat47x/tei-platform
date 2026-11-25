package main

import (
	"context"
	"fmt"
	"testing"

	port "tei/tools/reference-cognition-port"
)

// The comparison harness must classify output by the port's own rule, so a
// provider that returns an undeclared option is a format violation rather than
// "wrong": it did not answer at all, and merging the two would hide a
// capability failure behind an accuracy figure.
func TestMeasureCaseClassifiesTheFourOutcomes(t *testing.T) {
	set := CaseSet{
		Options:     []port.ChoiceOption{{ID: "billing", Description: "請求"}, {ID: "contract", Description: "契約"}},
		Constraints: []string{"主な目的を分類する"},
	}
	for _, tc := range []struct {
		name    string
		c       Case
		reply   func() ([]byte, error)
		verdict string
		count   func(Counts) int
	}{
		{
			name: "declared answer matching the expectation is correct",
			c:    Case{ID: "c1", Kind: "clear", Statement: "請求の確認", Expected: "billing"},
			reply: func() ([]byte, error) {
				return []byte(`{"schema":"tei.reference.cognition-proposal/v0","state":"proposed","candidates":[{"id":"choice-1","kind":"choice","subject":"c1","payload":{"optionId":"billing"},"basis":["observation:c1"]}]}`), nil
			},
			verdict: "correct",
			count:   func(c Counts) int { return c.Correct },
		},
		{
			name: "declared answer against the expectation is wrong",
			c:    Case{ID: "c2", Kind: "clear", Statement: "請求の確認", Expected: "contract"},
			reply: func() ([]byte, error) {
				return []byte(`{"schema":"tei.reference.cognition-proposal/v0","state":"proposed","candidates":[{"id":"choice-1","kind":"choice","subject":"c2","payload":{"optionId":"billing"},"basis":["observation:c2"]}]}`), nil
			},
			verdict: "wrong",
			count:   func(c Counts) int { return c.Wrong },
		},
		{
			name: "undeclared option is a format violation, not a wrong answer",
			c:    Case{ID: "c3", Kind: "clear", Statement: "請求の確認", Expected: "billing"},
			reply: func() ([]byte, error) {
				return []byte(`{"schema":"tei.reference.cognition-proposal/v0","state":"proposed","candidates":[{"id":"choice-1","kind":"choice","subject":"c3","payload":{"optionId":"invented"},"basis":["observation:c3"]}]}`), nil
			},
			verdict: "format-violation",
			count:   func(c Counts) int { return c.FormatViolation },
		},
		{
			name: "abstaining where an answer was expected is an abstention",
			c:    Case{ID: "c4", Kind: "clear", Statement: "請求の確認", Expected: "billing"},
			reply: func() ([]byte, error) {
				return []byte(`{"schema":"tei.reference.cognition-proposal/v0","state":"unknown","candidates":[],"diagnostics":["no match"]}`), nil
			},
			verdict: "abstained-wrongly",
			count:   func(c Counts) int { return c.AbstainedWrongly },
		},
		{
			name:    "a provider error is a failure, not an abstention",
			c:       Case{ID: "c5", Kind: "clear", Statement: "請求の確認", Expected: "billing"},
			reply:   func() ([]byte, error) { return nil, fmt.Errorf("provider refused") },
			verdict: "failed",
			count:   func(c Counts) int { return c.Failed },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := provider{
				name:    "test",
				model:   "test-model",
				version: "1",
				invoke:  func(context.Context, port.Provider, []byte) ([]byte, error) { return tc.reply() },
			}
			declared := port.Provider{
				Schema: port.ProviderSchema, ID: "test", Model: "test-model", Version: "1",
				Execution: port.ExecutionExternal, Network: port.NetworkNone,
				Capabilities: []string{"choice"},
				Limits:       port.Limits{MaxContextBytes: 8192, MaxOutputBytes: 8192, TimeoutSeconds: 60},
			}
			outcome, counts := measureCase(set, tc.c, p, declared)
			if outcome.Verdict != tc.verdict {
				t.Fatalf("verdict = %q, want %q (%s)", outcome.Verdict, tc.verdict, outcome.Detail)
			}
			if tc.count(counts) != 1 {
				t.Fatalf("counts did not record the outcome: %+v", counts)
			}
			if counts.Cases != 1 {
				t.Fatalf("each case must be counted once: %+v", counts)
			}
		})
	}
}
