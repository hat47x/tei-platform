package operations

import (
    "os"
    "path/filepath"
    "strings"
    "testing"

    "tei/internal/workspace"
)

// TestOrgStagedAsOfQuery checks the REAL workspace loader's query declaration
// and the reference SQL text against explicit bitemporal boundary requirements.
// It is not a database execution test, nor evidence of historical persistence.
func TestOrgStagedAsOfQuery(t *testing.T) {
    root := filepath.Join("..", "..", "reference", "org-staged-axis")
    loaded, err := workspace.Load(root)
    if err != nil {
        t.Fatalf("load staged axis workspace: %v", err)
    }

    var found bool
    for _, q := range loaded.World.Queries {
        if q.ID != "org_axes_view.as_of_membership" {
            continue
        }
        found = true
        if q.Dialect != "postgres" {
            t.Fatalf("unexpected dialect: %+v", q)
        }
        if len(q.Inputs) != 3 {
            t.Fatalf("expected valid_at/known_at/allocation_min, got %+v", q.Inputs)
        }
        names := map[string]bool{}
        for _, in := range q.Inputs {
            names[string(in.Name)] = true
        }
        if !names["valid_at"] || !names["known_at"] || !names["allocation_min"] {
            t.Fatalf("as-of inputs must name evaluation instants, got %+v", q.Inputs)
        }
        if names["valid_from"] || names["known_from"] {
            t.Fatalf("query parameters should not be named as row interval boundaries: %+v", q.Inputs)
        }
    }
    if !found {
        t.Fatal("org_axes_view.as_of_membership query not loaded")
    }

    queryPath := filepath.Join(root, "applications", "org-axes-view", "queries", "as_of_membership.tei")
    sqlFile, err := os.ReadFile(queryPath)
    if err != nil {
        t.Fatalf("read query source for structural boundary check: %v", err)
    }
    sql := string(sqlFile)
    for _, want := range []string{
        "FROM public.org_memberships m",
        "m.valid_from <= :valid_at",
        "m.valid_to IS NULL OR :valid_at < m.valid_to",
        "m.known_from <= :known_at",
        "m.known_to IS NULL OR :known_at < m.known_to",
        "m.allocation >= :allocation_min",
    } {
        if !strings.Contains(sql, want) {
            t.Errorf("reference SQL lacks necessary temporal clause %q", want)
        }
    }
    if strings.Contains(sql, "m.known_from >= :known_at") {
        t.Fatal("knownFrom must not be compared in the opposite direction")
    }

    mappingPath := filepath.Join(root, "databases", "org-store", "persistence", "membership.tei")
    physical, err := os.ReadFile(mappingPath)
    if err != nil {
        t.Fatalf("read physical mapping: %v", err)
    }
    for _, want := range []string{
        "table public.org_memberships",
        "column organization.membership.valid_from -> valid_from",
        "column organization.membership.valid_to -> valid_to",
        "column organization.membership.known_from -> known_from",
        "column organization.membership.known_to -> known_to",
    } {
        if !strings.Contains(string(physical), want) {
            t.Errorf("physical mapping lacks %q", want)
        }
    }
    // The current mapping PK is still membership_id only. This check must
    // never be reported as real two-version persistence or an actual Provider read.
}
