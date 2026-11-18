package main

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/validationread"
	"tei/internal/realization"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// webRelationCoverageProof is minted only by the PostgreSQL executor while the
// actual Contract Query's read-only transaction is open. The Target ties the
// claim to that execution; synthetic or unrelated reads receive no proof.
type webRelationCoverageProof struct {
	Target   realization.ContractQueryTarget
	Complete bool
}

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
// Coverage defaults to partial. A relation-wide rule can be satisfied only
// when the selected PostgreSQL executor checked the Query/storage mapping and
// physical table visibility in the same transaction that returned the rows.
// A covers declaration, a matching value or a separate read is not proof.
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
	coverage := validationeval.CoveragePartial
	if proof, ok := execution.Result.Payload.(webRelationCoverageProof); ok &&
		proof.Complete && proof.Target == execution.Resolution.Binding.Target {
		coverage = validationeval.CoverageComplete
	}
	detail, err := validationread.FromHeaderDetail(semantics, execution, coverage)
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

// requireReadRowAnswerAlignment is the boundary for the first Web template's
// positional join. Rows and answers must be produced by the SAME execution
// (the caller passes lines.Execution to both projections); this check rejects
// any different cardinality or position before HTML can attach a result to
// the wrong displayed row. Position is transport state, not business identity.
func requireReadRowAnswerAlignment(rows []semanticRowProjection, answers []rowAnswerProjection) error {
	if len(rows) != len(answers) {
		return fmt.Errorf("read has %d rows but %d answer rows", len(rows), len(answers))
	}
	for index, answer := range answers {
		if answer.RowIndex != index {
			return fmt.Errorf("answer at position %d belongs to position %d", index, answer.RowIndex)
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
