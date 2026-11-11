package main

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

// bareFlagName is what a machine-readable flag name looks like: no leading dash,
// no spaces, no parenthetical.
var bareFlagName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// catalogRecord returns one advertised record.
func catalogRecord(t *testing.T, name string) cliCommandRecord {
	t.Helper()
	for _, record := range commandCatalog().Commands {
		if record.Name == name {
			return record
		}
	}
	t.Fatalf("%q is not advertised", name)
	return cliCommandRecord{}
}

func sortedFlagNames(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestAdvertisedFlagNamesAreBareNames pins #4166. `required` and `optional`
// carry the rules as sentences, and one entry glued a note to a name, so a
// caller that passed an element as a flag got "flag provided but not defined".
// The new fields must hold names only.
func TestAdvertisedFlagNamesAreBareNames(t *testing.T) {
	for _, record := range commandCatalog().Commands {
		for field, names := range map[string][]string{
			"requiredFlags": record.RequiredFlags,
			"optionalFlags": record.OptionalFlags,
		} {
			for _, name := range names {
				if !bareFlagName.MatchString(name) {
					t.Fatalf("%s.%s contains %q, which is not a bare flag name",
						record.Name, field, name)
				}
				if strings.HasPrefix(name, "-") {
					t.Fatalf("%s.%s contains a leading dash: %q", record.Name, field, name)
				}
			}
		}
	}
}

// The advertised flags must be exactly the flags the command accepts. For the
// commands that use the shared allowlist the two sets have to match; nothing
// may be advertised that the command would reject.
func TestAdvertisedFlagNamesAreTheAcceptedSet(t *testing.T) {
	selfParsed := map[string]bool{
		"model catalog": true, "model locate": true, "model overview": true,
		"model document-draft": true, "model document-reconcile": true,
	}
	for _, record := range commandCatalog().Commands {
		if selfParsed[record.Name] {
			continue
		}
		allowed, ok := selectedReadOnlyFlagAllowlist[record.Name]
		if !ok {
			t.Fatalf("%q has no allowlist entry and is not a self-parsed command", record.Name)
		}
		advertised := append(sortedFlagNames(record.RequiredFlags), record.OptionalFlags...)
		sort.Strings(advertised)
		want := sortedFlagNames(allowed)
		if !sameStrings(advertised, want) {
			t.Fatalf("%s advertises %v but accepts %v", record.Name, advertised, want)
		}
	}
}

// The commands that parse their own arguments are named here, so a change to
// either side fails instead of drifting silently.
func TestSelfParsedFlagNames(t *testing.T) {
	want := map[string]struct{ required, optional []string }{
		"model catalog":            {[]string{"workspace"}, []string{"namespace"}},
		"model locate":             {[]string{"workspace", "subject-kind", "subject-id"}, []string{"limit"}},
		"model overview":           {[]string{"workspace"}, []string{"limit"}},
		"model document-draft":     {[]string{"workspace", "subject-kind", "subject-id"}, []string{"relation-id", "title", "document-kind", "application", "frozen", "section"}},
		"model document-reconcile": {[]string{"workspace", "draft"}, nil},
	}
	for name, expect := range want {
		record := catalogRecord(t, name)
		if !sameStrings(sortedFlagNames(record.RequiredFlags), sortedFlagNames(expect.required)) {
			t.Fatalf("%s requiredFlags = %v, want %v", name, record.RequiredFlags, expect.required)
		}
		if !sameStrings(sortedFlagNames(record.OptionalFlags), sortedFlagNames(expect.optional)) {
			t.Fatalf("%s optionalFlags = %v, want %v", name, record.OptionalFlags, expect.optional)
		}
		if _, ok := selectedReadOnlyFlagAllowlist[name]; ok {
			t.Fatalf("%s gained an allowlist entry; the self-parsed table is stale", name)
		}
	}
}

// A flag cannot be both required and optional.
func TestAdvertisedFlagNamesDoNotOverlap(t *testing.T) {
	for _, record := range commandCatalog().Commands {
		required := map[string]bool{}
		for _, name := range record.RequiredFlags {
			required[name] = true
		}
		for _, name := range record.OptionalFlags {
			if required[name] {
				t.Fatalf("%s lists %q as both required and optional", record.Name, name)
			}
		}
	}
}

// The two shapes #4166 named: `version` claimed a flag called "none", and
// `validate` glued "(workspace only)" to -json.
func TestVersionAndValidateNoLongerClaimWrongNames(t *testing.T) {
	version := catalogRecord(t, "version")
	if len(version.RequiredFlags) != 0 {
		t.Fatalf("version requiredFlags = %v, want none", version.RequiredFlags)
	}
	if !sameStrings(version.OptionalFlags, []string{"json"}) {
		t.Fatalf("version optionalFlags = %v, want [json]", version.OptionalFlags)
	}
	for _, element := range version.Required {
		if strings.TrimSpace(element) == "none" {
			t.Fatalf("version still advertises %q in a flag-name position", element)
		}
	}

	validate := catalogRecord(t, "validate")
	if !sameStrings(sortedFlagNames(validate.RequiredFlags), []string{"workspace", "world"}) {
		t.Fatalf("validate requiredFlags = %v", validate.RequiredFlags)
	}
	if !sameStrings(validate.OptionalFlags, []string{"json"}) {
		t.Fatalf("validate optionalFlags = %v, want [json]", validate.OptionalFlags)
	}
	for _, element := range validate.Optional {
		if strings.ContainsAny(element, "()") {
			t.Fatalf("validate optional still glues a note to a name: %q", element)
		}
	}
}

// The prose fields stay, because they carry the rules the name lists cannot.
func TestProseFieldsAreKept(t *testing.T) {
	view := catalogRecord(t, "view")
	if len(view.Optional) == 0 {
		t.Fatal("view lost its optional rules")
	}
	isProse := false
	for _, element := range view.Optional {
		if strings.Contains(element, " at most one of ") || strings.HasPrefix(element, "at most one of ") {
			isProse = true
		}
	}
	if !isProse {
		t.Fatalf("view no longer states the at-most-one rule: %v", view.Optional)
	}
}
