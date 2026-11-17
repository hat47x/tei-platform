package main

import (
	"fmt"
	"io"
	"sort"
)

// writeEnvironmentReadText renders the read-only runtime result as lines a
// person reads: the row's key, then each answered rule with the answer the
// Workspace's rule produced. The JSON form carries the same facts; this form
// exists because plan/NOW.md asks for rows and answers to be readable without
// a JSON reader (#2586).
//
// Nothing is inferred here. A rule that has no answer is not printed as
// satisfied, and an undecided rule keeps the reason the evaluation recorded.
func writeEnvironmentReadText(out io.Writer, read destinationRead) (int, error) {
	if _, err := fmt.Fprintf(out, "pattern %s (%s)\n", read.Pattern, read.Application); err != nil {
		return 2, err
	}
	if _, err := fmt.Fprintf(out, "relation %s\n", read.Relation); err != nil {
		return 2, err
	}
	if _, err := fmt.Fprintf(out, "environment %s\n", read.Environment); err != nil {
		return 2, err
	}
	if _, err := fmt.Fprintf(out, "coverage %s\n", read.Coverage); err != nil {
		return 2, err
	}
	if len(read.Rows) == 0 {
		if _, err := fmt.Fprintln(out, "rows: (none)"); err != nil {
			return 2, err
		}
	} else {
		if _, err := fmt.Fprintf(out, "rows: %d\n", len(read.Rows)); err != nil {
			return 2, err
		}
	}
	// Stable order: a reader comparing two runs must not see the rows shuffle.
	keys := make([]string, 0, len(read.Rows))
	for _, row := range read.Rows {
		keys = append(keys, row.Key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, err := fmt.Fprintf(out, "  %s\n", key); err != nil {
			return 2, err
		}
		for _, rule := range sortedRuleNames(read.PerRow[key], read.PerRowUndecided[key]) {
			line := "    " + ruleAnswerLine(rule, read.PerRow[key][rule], read.PerRowUndecided[key][rule])
			if _, err := fmt.Fprintln(out, line); err != nil {
				return 2, err
			}
		}
	}
	for _, rule := range sortedRuleNames(read.PerRelation, read.PerRelationUndecided) {
		line := "  " + ruleAnswerLine(rule, read.PerRelation[rule], read.PerRelationUndecided[rule])
		if _, err := fmt.Fprintln(out, line); err != nil {
			return 2, err
		}
	}
	return 0, nil
}

// sortedRuleNames is the union of the answered and the undecided rule names, in
// one order, so a rule that has an answer and one that does not cannot be
// confused by their position.
func sortedRuleNames(answered, undecided map[string]string) []string {
	names := make([]string, 0, len(answered)+len(undecided))
	for name := range answered {
		names = append(names, name)
	}
	for name := range undecided {
		if _, ok := answered[name]; !ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// ruleAnswerLine says what the evaluation decided. An answer with a reason keeps
// both; a rule with no answer is not printed as a blank one, so the reason that
// explains why it could not be decided becomes the line (#2586).
func ruleAnswerLine(rule, answer, reason string) string {
	switch {
	case answer == "" && reason == "":
		return rule + ": (no answer recorded)"
	case answer == "":
		return rule + ": " + reason
	case reason != "":
		return rule + ": " + answer + " (" + reason + ")"
	}
	return rule + ": " + answer
}
