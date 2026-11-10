package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// viewEnvelopeError returns error.code and error.message, or empty strings.
func viewEnvelopeError(t *testing.T, raw []byte) (string, string) {
	t.Helper()
	var body struct {
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("not one JSON document: %v (%q)", err, string(raw))
	}
	if body.Error == nil {
		return "", ""
	}
	return body.Error.Code, body.Error.Message
}

// headOf trims a rendered document for an error message.
func headOf(text string) string {
	if len(text) > 80 {
		return text[:80] + "..."
	}
	return text
}

// runViewForForm runs one view form with one format and returns the envelope.
func runViewForForm(t *testing.T, args ...string) []byte {
	t.Helper()
	root := filepath.Join("..", "..", "reference", "workspace")
	var out bytes.Buffer
	_, _ = runViewCommand(append([]string{"-workspace", root}, args...), &out)
	return out.Bytes()
}

// TestViewFormatDependsOnTheForm records where the advertisement and the gate
// disagreed: the catalogue advertised
// `-format` as one set (text, json, html, print) for the whole command, but the
// command narrows it per form: `-application` alone and `-package` take text or
// json only, and html/print are reachable only through a browse `-pattern`. An
// agent reading `tei commands` would try `-application X -format html` and be
// rejected, so the advertisement now states the restriction and this test keeps
// the two in step.
func TestViewFormatDependsOnTheForm(t *testing.T) {
	for _, format := range []string{"html", "print"} {
		code, message := viewEnvelopeError(t, runViewForForm(t, "-application", "order_entry", "-format", format))
		if code != resultCodeInvalidArgument {
			t.Fatalf("-application -format %s: code=%q, want %q", format, code, resultCodeInvalidArgument)
		}
		if !strings.Contains(message, "-application supports -format text or json") {
			t.Fatalf("-application -format %s: message=%q", format, message)
		}

		code, message = viewEnvelopeError(t, runViewForForm(t,
			"-application", "order_entry", "-package", "order_entry", "-format", format))
		if code != resultCodeInvalidArgument {
			t.Fatalf("-package -format %s: code=%q, want %q", format, code, resultCodeInvalidArgument)
		}
		if !strings.Contains(message, "-package supports -format text or json") {
			t.Fatalf("-package -format %s: message=%q", format, message)
		}
	}

	// A browse pattern does accept html and print, and renders them rather than
	// emitting the JSON envelope. This is the half the advertisement was right
	// about, so it is asserted positively.
	html := string(runViewForForm(t,
		"-application", "order_entry", "-pattern", "order_entry.recent_orders", "-format", "html"))
	if !strings.HasPrefix(strings.ToLower(html), "<!doctype html>") {
		t.Fatalf("a browse pattern -format html did not render HTML: %q", headOf(html))
	}
	printed := string(runViewForForm(t,
		"-application", "order_entry", "-pattern", "order_entry.recent_orders", "-format", "print"))
	if !strings.Contains(printed, "tei.print/v0") {
		t.Fatalf("a browse pattern -format print did not render the print document: %q", headOf(printed))
	}
}

// The advertised prose must say which form takes which formats, so a reader of
// `tei commands` alone does not have to discover the rule by failing.
func TestViewAdvertisementStatesTheFormatRestriction(t *testing.T) {
	catalog := commandCatalog()
	var view *cliCommandRecord
	for i := range catalog.Commands {
		if catalog.Commands[i].Name == "view" {
			view = &catalog.Commands[i]
		}
	}
	if view == nil {
		t.Fatal("view is not advertised")
	}
	joined := strings.Join(view.Optional, " | ")
	for _, want := range []string{"text|json for -application alone or -package", "browse -pattern"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("the advertisement does not say %q: %s", want, joined)
		}
	}
}
