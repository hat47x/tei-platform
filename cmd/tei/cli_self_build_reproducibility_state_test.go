package main

import "testing"

// "In separate state" is what makes two builds independent. The entry passes a
// fresh -state; these tests pin that the child's own report is checked against it
// rather than assumed to agree.
func TestReproducibilityStateVerdict(t *testing.T) {
	cases := []struct {
		name             string
		passed, reported string
		want             string
	}{
		{
			name:     "the child used the state directory it was given",
			passed:   "/tmp/run/state",
			reported: "/tmp/run/state",
			want:     "",
		},
		{
			// Same directory spelled with a trailing separator is the same place.
			name:     "the same directory written differently",
			passed:   "/tmp/run/state",
			reported: "/tmp/run/state/",
			want:     "",
		},
		{
			name:     "the child used its own state directory",
			passed:   "/tmp/run/state",
			reported: "/tmp/run/other",
			want:     "the child used a different state directory than the one this entry passed",
		},
		{
			// An unrecorded state is not agreement (SC0-18's rule, applied here).
			name:     "the child reported no state directory",
			passed:   "/tmp/run/state",
			reported: "",
			want:     "the child did not report the state directory it used, so the separation of the two runs is not shown",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := reproducibilityStateVerdict(tc.passed, tc.reported); got != tc.want {
				t.Fatalf("verdict = %q, want %q", got, tc.want)
			}
		})
	}
}
