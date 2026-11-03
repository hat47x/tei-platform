package main

import "testing"

func TestProjectKeepsMissingEmptyAndUnresolvedDistinct(t *testing.T) {
	cases := []struct {
		name  string
		input map[string]any
		want  map[string]any
	}{
		{"present", map[string]any{"text": "hello"}, map[string]any{"state": "present", "text": "hello"}},
		{"empty", map[string]any{"text": ""}, map[string]any{"state": "present", "text": ""}},
		{"missing", map[string]any{}, map[string]any{"state": "missing"}},
		{"unresolved", map[string]any{"text": nil}, map[string]any{"state": "unresolved"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := project("identity", tc.input, "text")
			if err != nil {
				t.Fatal(err)
			}
			if !equal(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNormalizeTrimsButKeepsEmptyPresent(t *testing.T) {
	got, err := project("normalize", map[string]any{"text": "  hi  "}, "text")
	if err != nil {
		t.Fatal(err)
	}
	if !equal(got, map[string]any{"state": "present", "text": "hi"}) {
		t.Fatalf("got %v", got)
	}
	got, err = project("normalize", map[string]any{"text": "   "}, "text")
	if err != nil {
		t.Fatal(err)
	}
	if !equal(got, map[string]any{"state": "present", "text": ""}) {
		t.Fatalf("got %v", got)
	}
}
