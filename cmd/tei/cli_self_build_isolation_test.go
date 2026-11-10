package main

import "testing"

// §4.45: the entry reads what the builder said about its build tree. It stays a
// statement — the five conditions do not move — but a reader no longer has to
// know that the builder's own report says it.

func TestReadChildIsolation(t *testing.T) {
	cases := []struct {
		name  string
		child *selfBuildChildIsolation
		want  string
	}{
		{
			name:  "the builder did not report a tree",
			child: nil,
			want:  "",
		},
		{
			name:  "the tree is made only of planned sources",
			child: &selfBuildChildIsolation{Root: "/tmp/tree", MemberCount: 3},
			want:  childIsolationIsolated,
		},
		{
			// Module context taken from the working tree is the exception §4.43
			// names, and the record has to show it rather than round to "isolated".
			name:  "the tree also took the module context",
			child: &selfBuildChildIsolation{Root: "/tmp/tree", MemberCount: 1, ModuleFiles: []string{"go.mod:sha256:aa"}},
			want:  childIsolationWithContext,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := readChildIsolation(tc.child)
			if tc.want == "" {
				if record != nil {
					t.Fatalf("a missing statement produced a record: %+v", record)
				}
				return
			}
			if record == nil {
				t.Fatal("no record was produced")
			}
			if record.State != tc.want {
				t.Fatalf("state = %q, want %q", record.State, tc.want)
			}
			if record.Boundary == "" {
				t.Fatal("the record has no boundary")
			}
			if record.State == childIsolationWithContext && record.Reason == "" {
				t.Fatal("the exception is not explained")
			}
			if record.State == childIsolationIsolated && record.Reason != "" {
				t.Fatalf("an isolated tree carries a reason: %q", record.Reason)
			}
		})
	}
}
