package partition

import (
	"math"
	"reflect"
	"testing"
)

func fn(id string, allowed ...string) Function {
	return Function{ID: id, Allowed: allowed}
}

func wr(id, function, data, owner string, weight int64) Write {
	return Write{
		ID: id, Function: function, Data: data, Owner: owner,
		Operation: "U", Source: "candidate:" + id, Weight: weight,
	}
}

// bruteForce is an independent test oracle: it enumerates all assignments and
// does not use component contraction or the evaluator's scoring helper.
func bruteForce(p Problem) (int64, bool) {
	if len(p.Functions) == 0 {
		return 0, false
	}
	assigned := map[string]string{}
	var best int64
	found := false
	var walk func(int)
	walk = func(i int) {
		if i == len(p.Functions) {
			for _, link := range p.MustLinks {
				if assigned[link.Left] != assigned[link.Right] {
					return
				}
			}
			var total int64
			for _, e := range p.Writes {
				if assigned[e.Function] != e.Owner {
					total += e.Weight
				}
			}
			if !found || total < best {
				found, best = true, total
			}
			return
		}
		f := p.Functions[i]
		for _, s := range f.Allowed {
			assigned[f.ID] = s
			walk(i + 1)
		}
		delete(assigned, f.ID)
	}
	walk(0)
	return best, found
}

func TestExactExamples(t *testing.T) {
	cases := []struct {
		name     string
		input    Problem
		optimal  int64
		feasible bool
	}{
		{"independent", Problem{
			Functions: []Function{fn("fA", "S1", "S2"), fn("fB", "S1", "S2")},
			Writes:    []Write{wr("wA", "fA", "A", "S1", 1), wr("wB", "fB", "B", "S2", 1)},
		}, 0, true},
		{"must-link", Problem{
			Functions: []Function{fn("fA", "S1", "S2"), fn("fB", "S1", "S2")},
			Writes:    []Write{wr("wA", "fA", "A", "S1", 1), wr("wB", "fB", "B", "S2", 1)},
			MustLinks: []MustLink{{"fA", "fB"}},
		}, 1, true},
		{"transitive-must-link", Problem{
			Functions: []Function{
				fn("fA", "S1", "S2", "S3"), fn("fB", "S1", "S2"),
				fn("fC", "S1", "S2", "S3"),
			},
			Writes: []Write{
				wr("wA", "fA", "A", "S1", 2),
				wr("wB", "fB", "B", "S2", 3),
				wr("wC", "fC", "C", "S3", 4),
			},
			MustLinks: []MustLink{{"fA", "fB"}, {"fB", "fC"}},
		}, 6, true},
		{"no-common-allowed", Problem{
			Functions: []Function{fn("fA", "S1"), fn("fB", "S2")},
			Writes:    []Write{wr("wA", "fA", "A", "S1", 1), wr("wB", "fB", "B", "S2", 1)},
			MustLinks: []MustLink{{"fA", "fB"}},
		}, 0, false},
		{"no-writes", Problem{
			Functions: []Function{fn("fA", "S2", "S1")},
		}, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Evaluate(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			oracle, feasible := bruteForce(tc.input)
			if feasible != tc.feasible {
				t.Fatal("test fixture inconsistent with brute-force oracle")
			}
			if !feasible {
				if got.Status != "infeasible" || got.MinimumKnownCrossWeight != nil || len(got.Placements) != 0 {
					t.Fatalf("an infeasible case must not return an optimum: %+v", got)
				}
				return
			}
			if got.Status != "exact-for-supplied-graph" ||
				got.MinimumKnownCrossWeight == nil ||
				*got.MinimumKnownCrossWeight != tc.optimal ||
				*got.MinimumKnownCrossWeight != oracle {
				t.Fatalf("got=%+v oracle=%d", got, oracle)
			}
			if got.AuthorityLimit != AuthorityLimit || len(got.Placements) != len(tc.input.Functions) {
				t.Fatal("authority limit and placement witness must be explicit")
			}
		})
	}
}

func TestMustLinkCounterexampleIndependentLowerBoundZero(t *testing.T) {
	p := Problem{
		Functions: []Function{fn("a", "S1", "S2"), fn("b", "S1", "S2")},
		Writes:    []Write{wr("wa", "a", "A", "S1", 1), wr("wb", "b", "B", "S2", 1)},
	}
	without, _ := Evaluate(p)
	if *without.MinimumKnownCrossWeight != 0 {
		t.Fatal("independent functions should reach zero")
	}
	p.MustLinks = []MustLink{{"a", "b"}}
	with, _ := Evaluate(p)
	if *with.MinimumKnownCrossWeight != 1 || len(with.Components) != 1 ||
		with.Components[0].Selected != "S1" ||
		!reflect.DeepEqual(with.Components[0].CrossWriteIDs, []string{"wb"}) {
		t.Fatalf("must-link exact optimum and deterministic witness: %+v", with)
	}
}

func TestComponentContractionAgainstBruteForce(t *testing.T) {
	// All nonempty allowed-subsystem masks across three functions and four
	// must-link topologies: 7^3*4 small, independent assignment enumerations.
	s := []string{"S1", "S2", "S3"}
	allowed := func(mask int) []string {
		out := []string{}
		for bit, name := range s {
			if mask&(1<<bit) != 0 {
				out = append(out, name)
			}
		}
		return out
	}
	links := [][]MustLink{
		nil, {{"f1", "f2"}}, {{"f2", "f3"}},
		{{"f1", "f2"}, {"f2", "f3"}},
	}
	for a := 1; a < 8; a++ {
		for b := 1; b < 8; b++ {
			for c := 1; c < 8; c++ {
				for _, link := range links {
					p := Problem{
						Functions: []Function{
							fn("f1", allowed(a)...),
							fn("f2", allowed(b)...),
							fn("f3", allowed(c)...),
						},
						Writes: []Write{
							wr("w1", "f1", "D1", "S1", 2),
							wr("w2", "f2", "D2", "S2", 3),
							wr("w3", "f3", "D3", "S3", 4),
						},
						MustLinks: link,
					}
					got, err := Evaluate(p)
					if err != nil {
						t.Fatal(err)
					}
					expected, feasible := bruteForce(p)
					if !feasible {
						if got.Status != "infeasible" || got.MinimumKnownCrossWeight != nil {
							t.Fatalf("must be infeasible: %+v", got)
						}
					} else if got.MinimumKnownCrossWeight == nil ||
						*got.MinimumKnownCrossWeight != expected {
						t.Fatalf("independent oracle=%d got=%+v", expected, got)
					}
				}
			}
		}
	}
}

func TestOrderDoesNotChangeResult(t *testing.T) {
	a := Problem{
		Functions: []Function{fn("z", "S2", "S1"), fn("a", "S2", "S1")},
		Writes:    []Write{wr("wz", "z", "Z", "S2", 1), wr("wa", "a", "A", "S1", 1)},
		MustLinks: []MustLink{{"z", "a"}},
	}
	before := Problem{
		Functions: append([]Function(nil), a.Functions...),
		Writes:    append([]Write(nil), a.Writes...),
		MustLinks: append([]MustLink(nil), a.MustLinks...),
	}
	r1, e1 := Evaluate(a)
	a.Functions[0], a.Functions[1] = a.Functions[1], a.Functions[0]
	a.Writes[0], a.Writes[1] = a.Writes[1], a.Writes[0]
	r2, e2 := Evaluate(a)
	if e1 != nil || e2 != nil || !reflect.DeepEqual(r1, r2) {
		t.Fatalf("non-determinism: %+v / %+v", r1, r2)
	}
	if !reflect.DeepEqual(before.Functions[0].Allowed, []string{"S2", "S1"}) {
		t.Fatal("test fixture unexpectedly changed")
	}
}

func TestRejectUnqualifiedAndInvalidInput(t *testing.T) {
	valid := Problem{
		Functions: []Function{fn("f", "S1")},
		Writes:    []Write{wr("w", "f", "A", "S1", 1)},
	}
	cases := []struct {
		name string
		p    Problem
	}{
		{"empty", Problem{}},
		{"missing-allowed", Problem{Functions: []Function{fn("f")}}},
		{"duplicate-function", Problem{Functions: []Function{fn("f", "S1"), fn("f", "S1")}}},
		{"duplicate-allowed", Problem{Functions: []Function{fn("f", "S1", "S1")}}},
		{"unknown-must-link", Problem{Functions: []Function{fn("f", "S1")}, MustLinks: []MustLink{{"f", "missing"}}}},
		{"unknown-writer", Problem{Functions: valid.Functions, Writes: []Write{wr("w", "missing", "A", "S1", 1)}}},
		{"read-not-write", func() Problem {
			p := valid
			p.Writes = append([]Write(nil), p.Writes...)
			p.Writes[0].Operation = "R"
			return p
		}()},
		{"zero-weight", func() Problem {
			p := valid
			p.Writes = append([]Write(nil), p.Writes...)
			p.Writes[0].Weight = 0
			return p
		}()},
		{"missing-provenance", func() Problem {
			p := valid
			p.Writes = append([]Write(nil), p.Writes...)
			p.Writes[0].Source = ""
			return p
		}()},
		{"duplicate-write", Problem{Functions: valid.Functions, Writes: []Write{wr("w", "f", "A", "S1", 1), wr("w", "f", "A", "S1", 1)}}},
		{"conflicting-owners", Problem{Functions: valid.Functions, Writes: []Write{wr("w", "f", "A", "S1", 1), wr("w2", "f", "A", "S2", 1)}}},
		{"overflow", Problem{Functions: valid.Functions, Writes: []Write{wr("w", "f", "A", "S1", math.MaxInt64), wr("w2", "f", "B", "S2", 1)}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, err := Evaluate(tc.p)
			if err == nil || report.MinimumKnownCrossWeight != nil {
				t.Fatalf("invalid input must not produce a numeric optimum: %+v err=%v", report, err)
			}
		})
	}
}
