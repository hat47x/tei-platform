package validationread_test

import (
	"math/big"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic/validationeval"
)

// #5902 added ReadRelationNumbers and pinned it with a synthetic Detail. The
// values a real database produces are not pinned there: whether a decimal
// column arrives exactly, whether a page reports partial, and whether a column
// the query did not return is not-fetched are all properties of the product
// read path, not of a hand-built Detail.
//
// The expected values are taken from the seed rows, not from the run:
//
//	D1: allocation '1', '0.5', '0', null
//	D2: allocation '1', '1'
func TestRelationNumbersOverARealDatabase(t *testing.T) {
	executor := membershipExecutor(t)
	allocation, err := core.ParseCanonicalID("organization.membership.allocation")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("D1 keeps the exact decimals and the null apart", func(t *testing.T) {
		detail := readMembers(t, executor, "members", "D1", validationeval.CoverageComplete)
		reading, err := detail.ReadRelationNumbers(membersRelation.ID, allocation)
		if err != nil {
			t.Fatalf("the relation numbers were refused: %v", err)
		}
		if reading.Coverage != validationeval.CoverageComplete {
			t.Fatalf("coverage=%q, want complete", reading.Coverage)
		}
		if len(reading.Rows) != 4 {
			t.Fatalf("rows=%d, want the four rows of D1", len(reading.Rows))
		}
		want := []struct {
			state validationeval.ValueState
			value string
		}{
			{validationeval.ValuePresent, "1"},
			{validationeval.ValuePresent, "1/2"},
			{validationeval.ValuePresent, "0"},
			{validationeval.ValueNull, ""},
		}
		for i, tc := range want {
			got := reading.Rows[i]
			if got.State != tc.state {
				t.Fatalf("row %d state=%q, want %q", i, got.State, tc.state)
			}
			if tc.value == "" {
				if got.Value != nil {
					t.Fatalf("row %d reports a null with the value %v", i, got.Value)
				}
				continue
			}
			// '0.5' is one half: the decimal is kept, and no binary
			// floating point value was produced for it.
			expected, ok := new(big.Rat).SetString(tc.value)
			if !ok {
				t.Fatalf("the expected value %q is not a number", tc.value)
			}
			if got.Value == nil || got.Value.Cmp(expected) != 0 {
				t.Fatalf("row %d value=%v, want %v", i, got.Value, expected)
			}
		}
	})

	t.Run("D2 reads its two rows", func(t *testing.T) {
		detail := readMembers(t, executor, "members", "D2", validationeval.CoverageComplete)
		reading, err := detail.ReadRelationNumbers(membersRelation.ID, allocation)
		if err != nil {
			t.Fatalf("the relation numbers were refused: %v", err)
		}
		if len(reading.Rows) != 2 {
			t.Fatalf("rows=%d, want the two rows of D2", len(reading.Rows))
		}
		for i, row := range reading.Rows {
			if row.State != validationeval.ValuePresent || row.Value == nil || row.Value.Cmp(big.NewRat(1, 1)) != 0 {
				t.Fatalf("row %d = %+v, want the present value 1", i, row)
			}
		}
	})

	t.Run("a page reports partial", func(t *testing.T) {
		detail := readMembers(t, executor, "members_first_page", "D1", validationeval.CoveragePartial)
		reading, err := detail.ReadRelationNumbers(membersRelation.ID, allocation)
		if err != nil {
			t.Fatalf("the relation numbers were refused: %v", err)
		}
		if reading.Coverage != validationeval.CoveragePartial {
			t.Fatalf("coverage=%q, want partial", reading.Coverage)
		}
		if len(reading.Rows) != 2 {
			t.Fatalf("rows=%d, want the two rows the page returned", len(reading.Rows))
		}
	})

	t.Run("a column the query did not return is not fetched", func(t *testing.T) {
		detail := readMembers(t, executor, "members_narrow", "D1", validationeval.CoverageComplete)
		reading, err := detail.ReadRelationNumbers(membersRelation.ID, allocation)
		if err != nil {
			t.Fatalf("the relation numbers were refused: %v", err)
		}
		for i, row := range reading.Rows {
			if row.State != validationeval.ValueNotFetched {
				t.Fatalf("row %d state=%q, want not-fetched", i, row.State)
			}
		}
	})

	t.Run("one row does not answer for the relation", func(t *testing.T) {
		detail := readMembers(t, executor, "members", "D1", validationeval.CoverageComplete)
		row, err := detail.Row(0)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := row.ReadRelationNumbers(membersRelation.ID, allocation); err == nil {
			t.Fatal("one row answered for the rows of the relation")
		}
	})
}
