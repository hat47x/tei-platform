package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

// viewFingerprints decodes the two read-only observations that `view -format
// json` reports at the envelope level (#3345).
type cliViewFingerprints struct {
	WorldFingerprint       string `json:"worldFingerprint"`
	SourceIndexFingerprint string `json:"sourceIndexFingerprint"`
}

func decodeViewFingerprints(t *testing.T, raw []byte) cliViewFingerprints {
	t.Helper()
	var got cliViewFingerprints
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("not one JSON object: %v (%q)", err, string(raw))
	}
	return got
}

// TestViewJSONReportsTheSameFingerprintsAsTheOtherReadCommands pins #3345:
// `view -format json` previously carried only `result.semanticFingerprint`,
// which covers the namespaces the Application uses, so a caller could not
// compare a view result with `model catalog` or `validate -json` at all.
func TestViewJSONReportsTheSameFingerprintsAsTheOtherReadCommands(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")

	var catalog bytes.Buffer
	if code, err := runModelCommand([]string{"catalog", "-workspace", root}, &catalog); err != nil || code != 0 {
		t.Fatalf("model catalog: code=%d err=%v body=%s", code, err, catalog.String())
	}
	var catalogBody struct {
		Result cliViewFingerprints `json:"result"`
	}
	if err := json.Unmarshal(catalog.Bytes(), &catalogBody); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"-workspace", root, "-application", "order_entry", "-format", "json"},
		{"-workspace", root, "-application", "order_entry", "-pattern", "order_entry.order_record", "-format", "json"},
	} {
		var out bytes.Buffer
		code, err := runViewCommand(args, &out)
		if err != nil || code != 0 {
			t.Fatalf("view %q: code=%d err=%v body=%s", args, code, err, out.String())
		}
		got := decodeViewFingerprints(t, out.Bytes())
		if got.WorldFingerprint != catalogBody.Result.WorldFingerprint ||
			got.SourceIndexFingerprint != catalogBody.Result.SourceIndexFingerprint {
			t.Fatalf("view %q disagrees with model catalog:\n view=%+v\n model=%+v",
				args, got, catalogBody.Result)
		}
		// The narrow value stays: it is a different observation, not a rename.
		var narrow struct {
			Result struct {
				SemanticFingerprint string `json:"semanticFingerprint"`
			} `json:"result"`
		}
		if err := json.Unmarshal(out.Bytes(), &narrow); err != nil {
			t.Fatal(err)
		}
		if narrow.Result.SemanticFingerprint != "" &&
			narrow.Result.SemanticFingerprint == got.WorldFingerprint {
			t.Fatalf("semanticFingerprint collapsed into worldFingerprint: %s", got.WorldFingerprint)
		}
	}
}

// TestViewTextOutputDoesNotClaimFingerprints keeps the addition to the JSON
// contract: the text rendering is unchanged and does not print the two new
// values, so a text caller never reads a value the JSON contract owns.
func TestViewTextOutputDoesNotClaimFingerprints(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	var out bytes.Buffer
	if code, err := runViewCommand([]string{"-workspace", root, "-application", "order_entry"}, &out); err != nil || code != 0 {
		t.Fatalf("view text: code=%d err=%v body=%s", code, err, out.String())
	}
	if bytes.Contains(out.Bytes(), []byte("worldFingerprint")) ||
		bytes.Contains(out.Bytes(), []byte("sourceIndexFingerprint")) {
		t.Fatalf("text output claims the JSON-only fingerprints: %s", out.String())
	}
	var probe struct {
		WorldFingerprint string `json:"worldFingerprint"`
	}
	if err := json.Unmarshal(out.Bytes(), &probe); err == nil {
		t.Fatalf("text output parsed as JSON: %s", out.String())
	}
}
