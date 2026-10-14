package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntentAndCatalogRejectAmbiguousInputJSON(t *testing.T) {
	for _, source := range []string{"intent.json", "catalog.json"} {
		t.Run(source, func(t *testing.T) {
			var value any
			if err := json.Unmarshal(readTestdata(t, source), &value); err != nil {
				t.Fatal(err)
			}
			compact, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			base := string(compact)
			parse := func(raw []byte) error {
				if source == "intent.json" {
					_, err := ParseIntent(raw)
					return err
				}
				_, err := ParseFixtureCatalog(raw)
				return err
			}
			if err := parse(compact); err != nil {
				t.Fatalf("valid control failed: %v", err)
			}
			inject := func(field string) string { return "{" + field + "," + base[1:] }
			cases := map[string]string{
				"duplicate root field": inject(`"formatVersion":99`),
				"escaped alias":        inject(`"format\u0056ersion":99`),
				"case folded alias":    inject(`"FORMATVERSION":99`),
				"unicode folded alias": inject(`"formatVerſion":99`),
				"trailing object":      base + `{}`, "trailing null": base + `null`,
			}
			if source == "intent.json" {
				cases["nested duplicate"] = strings.Replace(base, `"compatibleMajor":1`, `"compatibleMajor":2,"compatibleMajor":1`, 1)
				cases["array object duplicate"] = strings.Replace(base, `"name":"optimization"`, `"name":"overridden","name":"optimization"`, 1)
				cases["nested folded alias"] = strings.Replace(base, `"compatibleMajor":1`, `"COMPATIBLEMAJOR":2,"compatibleMajor":1`, 1)
				cases["unpaired surrogate"] = strings.Replace(base, `"value":"size"`, `"value":"\ud800"`, 1)
				cases["invalid utf8"] = strings.Replace(base, `"value":"size"`, "\"value\":\""+string([]byte{0xff})+"\"", 1)
			} else {
				cases["array object duplicate"] = strings.Replace(base, `"contractMajor":1`, `"contractMajor":2,"contractMajor":1`, 1)
				cases["nested escaped alias"] = strings.Replace(base, `"contractMajor":1`, `"contract\u004dajor":2,"contractMajor":1`, 1)
				cases["nested folded alias"] = strings.Replace(base, `"contractMajor":1`, `"CONTRACTMAJOR":2,"contractMajor":1`, 1)
				cases["unpaired surrogate"] = strings.Replace(base, `"implementation":"go.builder"`, `"implementation":"\udc00"`, 1)
				cases["invalid utf8"] = strings.Replace(base, `"implementation":"go.builder"`, "\"implementation\":\""+string([]byte{0xff})+"\"", 1)
			}
			for name, raw := range cases {
				t.Run(name, func(t *testing.T) {
					if raw == base {
						t.Fatal("negative fixture did not change input")
					}
					if err := parse([]byte(raw)); err == nil {
						t.Fatalf("ambiguous/invalid JSON was accepted: %q", raw)
					}
				})
			}
		})
	}
}

func TestStrictInputPreservesResolutionReceiptAndCLIStopsBeforeOutput(t *testing.T) {
	intentRaw := readTestdata(t, "intent.json")
	catalogRaw := readTestdata(t, "catalog.json")
	expected := mustMarshal(t, mustResolve(t, loadIntent(t), filepath.Join("testdata", "src"), loadCatalog(t)))
	// Reorder keys through a map and change whitespace without changing meaning.
	reorder := func(raw []byte) []byte {
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatal(err)
		}
		result, err := json.MarshalIndent(v, "", "\t")
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	intent, err := ParseIntent(reorder(intentRaw))
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := ParseFixtureCatalog(reorder(catalogRaw))
	if err != nil {
		t.Fatal(err)
	}
	actual := mustMarshal(t, mustResolve(t, intent, filepath.Join("testdata", "src"), catalog))
	if !bytes.Equal(expected, actual) {
		t.Fatal("JSON spelling changed receipt bytes")
	}
	// Well-formed supplementary Unicode remains valid, unlike unpaired escapes.
	validUnicode := bytes.Replace(intentRaw, []byte(`"value": "size"`), []byte(`"value": "\ud83d\ude00"`), 1)
	if bytes.Equal(validUnicode, intentRaw) {
		t.Fatal("Unicode control did not change input")
	}
	if _, err := ParseIntent(validUnicode); err != nil {
		t.Fatalf("valid surrogate pair rejected: %v", err)
	}
	for _, invalid := range []string{"intent", "catalog"} {
		t.Run(invalid, func(t *testing.T) {
			dir := t.TempDir()
			i := filepath.Join(dir, "intent.json")
			c := filepath.Join(dir, "catalog.json")
			ir, cr := intentRaw, catalogRaw
			if invalid == "intent" {
				ir = append([]byte(`{"formatVersion":99,`), bytes.TrimSpace(intentRaw)[1:]...)
			} else {
				cr = append(append([]byte(nil), catalogRaw...), []byte(`{}`)...)
			}
			if err := os.WriteFile(i, ir, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(c, cr, 0600); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			code := run([]string{"-intent", i, "-catalog", c, "-source-root", filepath.Join("testdata", "src")}, &stdout, &stderr)
			if code != 1 || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("invalid %s produced a receipt: code=%d out=%s err=%s", invalid, code, &stdout, &stderr)
			}
		})
	}
}
