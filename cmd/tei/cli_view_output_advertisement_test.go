package main

import (
	"strings"
	"testing"
)

// `tei commands` advertises what a caller can ask for. After `-format html`
// was added to `view -environment`, the command's `output` field still said
// only text and json, so a caller reading the catalogue would not know the page
// exists (#5408).
//
// The expected string below is written from the formats the command actually
// accepts: text and json everywhere, html for the runtime read and the browse
// patterns, print for the browse patterns. The check is deliberately about the
// advertisement agreeing with `argumentValues.format`, not about the exact
// wording of the prose.
func TestViewAdvertisesEveryFormatItAccepts(t *testing.T) {
	var view *cliCommandRecord
	for i := range commandCatalog().Commands {
		if commandCatalog().Commands[i].Name == "view" {
			view = &commandCatalog().Commands[i]
			break
		}
	}
	if view == nil {
		t.Fatal("view is not in the command catalogue")
	}

	accepted := map[string]bool{}
	for _, value := range view.ArgumentValues["format"] {
		accepted[value] = true
	}
	// The catalogue must still name the four formats this command accepts.
	for _, want := range []string{"text", "json", "html", "print"} {
		if !accepted[want] {
			t.Errorf("view does not accept %q, which the output column describes", want)
		}
	}

	// Every accepted format except print must appear in the output column, so
	// a caller can discover it without reading the source. print is a browse
	// pattern's print layout and is described by the optional rules instead.
	output := view.Output
	for _, want := range []string{"text", "json", "html"} {
		if !strings.Contains(output, want) {
			t.Errorf("view accepts %q but its output column %q does not name it", want, output)
		}
	}

	// A generic format list cannot imply that every format works in every
	// mode: the environment-backed read supports html but must refuse print.
	optional := strings.Join(view.Optional, "\n")
	for _, want := range []string{
		"text|json|html for -environment with -pattern",
		"print rejected",
		"text|json|html|print for a browse -pattern",
		"text|json for a command -pattern",
	} {
		if !strings.Contains(optional, want) {
			t.Errorf("view's optional rules do not explain mode-specific format %q: %s", want, optional)
		}
	}
}
