package main

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/validationread"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// rowAnswerProjection is one row of the read surface with the answer of every
// rule the Application declares, for that row.
//
// The answer is the same three-valued result the CLI reports: satisfied,
// not-satisfied, or undecided with the reason it is undecided. A row is named
// by its position in the read, because the read is what the page shows; the
// position is transport state, not a business identity.
type rowAnswerProjection struct {
	RowIndex int                    `json:"rowIndex"`
	Answers  []ruleAnswerProjection `json:"answers"`
}

type ruleAnswerProjection struct {
	Rule   string `json:"rule"`
	Result string `json:"result"`
	Reason string `json:"reason,omitempty"`
}

// ruleAnswers projects the answer of every declared rule onto every row that
// was read.
//
// It reuses the two steps the product CLI uses: the rows come from the
// already-executed header-detail Contract Query, validationread reads them as
// the rows of one relation, and ResolveApplicationValidations decides which
// rules to ask (ADR-0029). This file holds no rule list of its own.
//
// Coverage is partial: the current Contract Query binding does not certify that
// every row of the relation was fetched, so a rule about all the rows stays
// undecided rather than being read as satisfied. That is the same conservative
// statement `tei view -environment` makes.
func ruleAnswers(
	resolver driver.Resolver,
	semantics []semantic.Namespace,
	applicationID core.CanonicalID,
	execution driver.HeaderDetailContractQueryExecution,
) ([]rowAnswerProjection, []string, error) {
	rules, err := resolver.ResolveApplicationValidations(applicationID)
	if err != nil {
		return nil, nil, err
	}
	detail, err := validationread.FromHeaderDetail(semantics, execution, validationeval.CoveragePartial)
	if err != nil {
		return nil, nil, err
	}
	// The CLI passes an empty operator set for the same read; a rule of kind
	// operator is then refused rather than answered. Passing declarations
	// without implementations would be a different statement, so this keeps
	// what the product CLI does.
	operators := validationeval.Operators{}

	rows := make([]rowAnswerProjection, 0, detail.Len())
	for index := 0; index < detail.Len(); index++ {
		rows = append(rows, rowAnswerProjection{RowIndex: index, Answers: []ruleAnswerProjection{}})
	}
	var relationAnswers []string
	for _, rule := range rules {
		decisions, err := detail.Decide(rule, operators)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot decide rule %s: %w", rule.ID, err)
		}
		if decisions.Relation != nil {
			answer, reason := answerOf(*decisions.Relation)
			line := fmt.Sprintf("%s %s", rule.ID, answer)
			if reason != "" {
				line += " (" + reason + ")"
			}
			relationAnswers = append(relationAnswers, line)
			continue
		}
		if len(decisions.Rows) != detail.Len() {
			return nil, nil, fmt.Errorf("rule %s did not return one answer for each observed row", rule.ID)
		}
		for index, decision := range decisions.Rows {
			answer, reason := answerOf(decision)
			rows[index].Answers = append(rows[index].Answers, ruleAnswerProjection{
				Rule:   string(rule.ID),
				Result: answer,
				Reason: reason,
			})
		}
	}
	return rows, relationAnswers, nil
}

// correspondRowAnswers checks that the answers describe the rows the page will
// show. The page binds them by position ("index $.Result.Read.RowAnswers $index"
// beside "range $index, $row := .Result.Read.OrderLineRows"), which is only
// sound when the two sequences came from the same read in the same order.
//
// plan/NOW.md requires exactly this: the per-row answers must be tied to the rows
// actually read, by row count, order and target read, and a screen position must
// never be treated as a business key. This function is that check for the first
// Web target; it does not invent a second correspondence, it refuses to proceed
// when the one the page relies on does not hold.
func correspondRowAnswers(rows []semanticRowProjection, answers []rowAnswerProjection) error {
	if len(answers) != len(rows) {
		return fmt.Errorf(
			"the read returned %d order-line rows but answers were produced for %d; the page joins them by position, so this read cannot be shown",
			len(rows),
			len(answers),
		)
	}
	for index, answer := range answers {
		if answer.RowIndex != index {
			return fmt.Errorf(
				"answer %d is labelled row %d; the page shows it beside order-line row %d, so a screen position would be joined to another row's answers",
				index,
				answer.RowIndex,
				index,
			)
		}
	}
	return nil
}

// answerOf reports a decision in the three words the CLI uses: a decided rule
// is satisfied or not-satisfied, an undecided one says so and why.
func answerOf(decision validationeval.Decision) (answer, reason string) {
	if decision.Undecided != nil {
		return "undecided", decision.Undecided.String()
	}
	return string(decision.Result), ""
}
