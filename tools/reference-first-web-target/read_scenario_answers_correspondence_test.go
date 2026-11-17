package main

import (
	"strings"
	"testing"
)

// The page binds answers to order-line rows by position. That binding is only
// sound when the two sequences are the same read, the same length, in the same
// order, so a mismatch must stop the read rather than show a wrong pairing.
func TestCorrespondRowAnswersRefusesAMismatch(t *testing.T) {
	rows := []semanticRowProjection{{}, {}, {}}

	t.Run("matching sequences are accepted", func(t *testing.T) {
		answers := []rowAnswerProjection{
			{RowIndex: 0}, {RowIndex: 1}, {RowIndex: 2},
		}
		if err := correspondRowAnswers(rows, answers); err != nil {
			t.Fatalf("a matching read was refused: %v", err)
		}
	})

	t.Run("fewer answers than rows is refused", func(t *testing.T) {
		// This is the case the page would silently render: the third row would
		// get the "no rows" branch and show no answers at all.
		answers := []rowAnswerProjection{{RowIndex: 0}, {RowIndex: 1}}
		err := correspondRowAnswers(rows, answers)
		if err == nil {
			t.Fatal("a read with fewer answers than rows was accepted")
		}
		if !strings.Contains(err.Error(), "3 order-line rows") {
			t.Fatalf("the refusal does not name the row count: %v", err)
		}
	})

	t.Run("more answers than rows is refused", func(t *testing.T) {
		answers := []rowAnswerProjection{
			{RowIndex: 0}, {RowIndex: 1}, {RowIndex: 2}, {RowIndex: 3},
		}
		if err := correspondRowAnswers(rows, answers); err == nil {
			t.Fatal("a read with more answers than rows was accepted")
		}
	})

	t.Run("an answer labelled with another row is refused", func(t *testing.T) {
		// Row 1's answers placed at position 0: the page would show line 0's
		// values beside line 1's answers.
		answers := []rowAnswerProjection{
			{RowIndex: 1}, {RowIndex: 0}, {RowIndex: 2},
		}
		err := correspondRowAnswers(rows, answers)
		if err == nil {
			t.Fatal("a mislabelled answer was accepted")
		}
		if !strings.Contains(err.Error(), "labelled row 1") {
			t.Fatalf("the refusal does not name the mislabelled index: %v", err)
		}
	})

	t.Run("an empty read with no answers is accepted", func(t *testing.T) {
		if err := correspondRowAnswers(nil, nil); err != nil {
			t.Fatalf("an empty read was refused: %v", err)
		}
	})
}
