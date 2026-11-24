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
//
// Basis and Reason are carried so the page can say what the verdict rests on.
// Without them the page shows a relation-wide answer with no statement of
// whether every row was confirmed, and a reader cannot tell a confirmed
// 'satisfied' from one that only looked satisfied.
type webRelationCoverageProof struct {
	Target           realization.ContractQueryTarget
	Complete         bool
	Basis            string
	Reason           string
	ProviderEvidence any // original non-secret SQL runner evidence, not replaced by coverage
}

// readCoverage is the coverage verdict this page reports, with the ground of it.
// It is the same statement `tei view -environment` makes about the same read:
// complete only when the selected executor confirmed the Query, the relation's
// persistence mapping and the physical table inside the read's transaction.
type readCoverage struct {
	Verdict string `json:"verdict"`
	Reason  string `json:"reason,omitempty"`
	Basis   string `json:"basis,omitempty"`
}

// relationCoverage decides the verdict these relation-wide answers rest on, and
// the ground of it. A proof counts only for the target it was minted for: an
// unrelated read's proof, a covers declaration or a matching value is not
// evidence. When the proof did not complete, the reason it stopped at is
// reported so the page can print it instead of leaving the verdict unexplained.
func relationCoverage(execution driver.HeaderDetailContractQueryExecution) (validationeval.Coverage, readCoverage) {
	partial := readCoverage{Verdict: string(validationeval.CoveragePartial)}
	proof, ok := execution.Result.Payload.(webRelationCoverageProof)
	if !ok || proof.Target != execution.Resolution.Binding.Target {
		return validationeval.CoveragePartial, partial
	}
	if proof.Complete {
		return validationeval.CoverageComplete, readCoverage{
			Verdict: string(validationeval.CoverageComplete),
			Basis:   proof.Basis,
		}
	}
	partial.Reason = proof.Reason
	return validationeval.CoveragePartial, partial
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
) ([]rowAnswerProjection, []string, readCoverage, error) {
	rules, err := resolver.ResolveApplicationValidations(applicationID)
	if err != nil {
		return nil, nil, readCoverage{}, err
	}
	coverage, reported := relationCoverage(execution)
	detail, err := validationread.FromHeaderDetail(semantics, execution, coverage)
	if err != nil {
		return nil, nil, readCoverage{}, err
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
			return nil, nil, readCoverage{}, fmt.Errorf("cannot decide rule %s: %w", rule.ID, err)
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
			return nil, nil, readCoverage{}, fmt.Errorf("rule %s did not return one answer for each observed row", rule.ID)
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
	return rows, relationAnswers, reported, nil
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
