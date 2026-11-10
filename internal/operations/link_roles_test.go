package operations

import "testing"

// LinkRole is the label on every link a located Subject carries. A reader groups
// links by this value, so the set is part of the response contract (#4032).
func TestLinkRolesAreCovered(t *testing.T) {
	roles := []string{
		LinkRoleRelationTarget,
		LinkRoleAttributeType,
		LinkRolePatternData,
		LinkRolePatternRelation,
		LinkRolePatternQuery,
		LinkRolePatternCapability,
	}
	want := []string{
		"relation-target", "attribute-type", "pattern-data",
		"pattern-relation", "pattern-query", "pattern-capability",
	}
	if len(roles) != len(want) {
		t.Fatalf("a link role was dropped: %d values", len(roles))
	}
	seen := map[string]bool{}
	for i, role := range roles {
		if role != want[i] {
			t.Fatalf("roles[%d]=%q want %q", i, role, want[i])
		}
		if seen[role] {
			t.Fatalf("role %q is declared twice", role)
		}
		seen[role] = true
	}
}
