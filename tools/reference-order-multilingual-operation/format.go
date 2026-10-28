package ordermultilingualoperation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"

	multilingualformat "tei/reference/multilingual-format"
	messages "tei/reference/standard-messages"
)

// RenderCase is one count rendering request.
type RenderCase struct {
	ID       string            `json:"id"`
	Locale   string            `json:"locale"`
	Count    int64             `json:"count"`
	Variants map[string]string `json:"variants"`
}

// Rendered is the result of one rendering.
type Rendered struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Number   string `json:"number"`
	Text     string `json:"text"`
}

// RenderGo selects the plural variant with the Go probe of
// reference/multilingual-format and substitutes the grouped integer. The
// message contract (named-v0) has no plural selection; the variants and this
// substitution are the sample's (PlatformGaps).
func RenderGo(c RenderCase) (Rendered, error) {
	r, err := multilingualformat.Probe(c.Locale, c.Count)
	if err != nil {
		return Rendered{}, err
	}
	v, ok := c.Variants[r.Category]
	if !ok {
		v = c.Variants["other"]
	}
	return Rendered{ID: c.ID, Category: r.Category, Number: r.Number, Text: strings.ReplaceAll(v, "{count}", r.Number)}, nil
}

// RenderJS runs render.mjs (which imports probe.mjs unchanged) with Node.
func RenderJS(ctx context.Context, node, repo string, cases []RenderCase) ([]Rendered, map[string]string, error) {
	in, err := json.Marshal(map[string]any{"cases": cases})
	if err != nil {
		return nil, nil, err
	}
	cmd := exec.CommandContext(ctx, node, filepath.Join(repo, "tools", "reference-order-multilingual-operation", "render.mjs"))
	cmd.Stdin = bytes.NewReader(in)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, nil, fmt.Errorf("node: %v: %s", err, stderr.String())
	}
	var out struct {
		Node, ICU, CLDR, Unicode string
		Results                  []Rendered
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, nil, err
	}
	return out.Results, map[string]string{"node": out.Node, "icu": out.ICU, "cldr": out.CLDR, "unicode": out.Unicode}, nil
}

// xTextVersion reads the golang.org/x/text requirement of go.mod (a test
// binary does not always carry dependency build information).
func xTextVersion(repo string) string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, d := range bi.Deps {
			if d.Path == "golang.org/x/text" {
				return d.Version
			}
		}
	}
	raw, err := os.ReadFile(filepath.Join(repo, "go.mod"))
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && f[0] == "golang.org/x/text" {
			return f[1] + " (go.mod)"
		}
	}
	return "unknown"
}

// Counts are the integer arguments of the confirmation message.
var Counts = []int64{0, 1, 2, 999, 1000, 12345, 1234567, 2147483647}

// Stage6 connects the integer argument of the revised confirmation message
// to plural selection and digit grouping, in Go and in JavaScript.
func (c *Chain) Stage6(ctx context.Context, node string) error {
	const id = "order.batch_confirmed"
	s := c.S4
	def, err := s.definition(id)
	if err != nil {
		return err
	}
	argOK := messages.ValidateArguments(def, map[string]messages.ParameterKind{"count": messages.ParameterInteger})
	argText := messages.ValidateArguments(def, map[string]messages.ParameterKind{"count": messages.ParameterText})
	en, err := s.ResolveMessage(ctx, id, "en-US")
	if err != nil {
		return fmt.Errorf("stage 6 en-US: %w", err)
	}
	ja, err := s.ResolveMessage(ctx, id, "ja-JP")
	if err != nil {
		return fmt.Errorf("stage 6 ja-JP: %w", err)
	}
	_, deErr := s.ResolveMessage(ctx, id, "de-DE")
	variants := map[string]string{"other": en.Resource.Text}
	var variantErrs []string
	for _, t := range s.Conf.Current {
		if t.Kind == "message" && t.Target == id && t.Locale == "en-US" {
			if t.Text != en.Resource.Text || t.BasedOn != en.Resource.SourceRevision {
				variantErrs = append(variantErrs, "selected text is not the confirmed r2 record")
			}
			for cat, v := range t.Plural {
				if err := messages.ValidateText(def, v); err != nil {
					variantErrs = append(variantErrs, cat+": "+err.Error())
				}
				variants[cat] = v
			}
		}
	}
	var cases []RenderCase
	for _, loc := range []struct {
		locale   string
		variants map[string]string
	}{{"en-US", variants}, {"ja-JP", map[string]string{"other": ja.Resource.Text}}} {
		for _, n := range Counts {
			cases = append(cases, RenderCase{ID: fmt.Sprintf("%s-%d", loc.locale, n), Locale: loc.locale, Count: n, Variants: loc.variants})
		}
	}
	goOut := map[string]Rendered{}
	for _, cs := range cases {
		r, err := RenderGo(cs)
		if err != nil {
			return err
		}
		goOut[cs.ID] = r
	}
	jsOut, versions, jsErr := RenderJS(ctx, node, c.Repo, cases)
	var mismatches []string
	matched := 0
	for _, r := range jsOut {
		if goOut[r.ID] != r {
			mismatches = append(mismatches, fmt.Sprintf("%s go=%+v js=%+v", r.ID, goOut[r.ID], r))
		} else {
			matched++
		}
	}
	displays := map[string]string{}
	for _, d := range c.B2.Displays {
		if d.Locale == "en-US" {
			displays[d.Resource.Target.Subject.ID] = d.Text
		}
	}
	one, many := goOut["en-US-1"].Text, goOut["en-US-12345"].Text
	ok := argOK == nil && argText != nil && len(variantErrs) == 0 && jsErr == nil && len(mismatches) == 0 && matched == len(cases) &&
		one == "Confirmed 1 order and issued shipping instructions." &&
		many == "Confirmed 12,345 orders and issued shipping instructions." &&
		goOut["ja-JP-12345"].Text == "12,345件の受注を確定し、出荷を指示しました" &&
		len(c.U.CheckBundleAgainstSources(c.B2, c.Rel2)) == 0 && deErr != nil
	c.Rec.Conditional = append(c.Rec.Conditional, Check{ID: "stage6-plural-and-grouping-go-js", Stage: "6", Passed: ok,
		Observed: map[string]any{"cases": len(cases), "matched": matched, "mismatches": mismatches, "jsError": errText(jsErr),
			"samples": map[string]string{"en-US 1": one, "en-US 12345": many, "ja-JP 12345": goOut["ja-JP-12345"].Text,
				"en-US 2147483647": goOut["en-US-2147483647"].Text}}})
	c.Rec.AddStage(StageRecord{Stage: 6, Name: "plural and digit grouping of the count argument (conditional)", Conditional: true,
		LockFP: s.WS.LockFP, Release: c.Rel2.Release, ReleaseHash: c.Rel2.Digest,
		Assets: []AssetCall{{"reference/multilingual-format", "Probe (Go), probe.mjs (Node)"},
			{"reference/standard-messages", "ValidateArguments, ValidateText"},
			{"reference/message-localization", "ResolveWorkspace"}},
		Outputs: map[string]any{"passed": ok, "bundleDigest": c.B2.Digest, "messageRevision": messages.Revision(def),
			"messageEnUS": en.Resource.Text, "variants": variants, "variantErrors": variantErrs, "displaysEnUS": displays,
			"integerArgument": errText(argOK), "textArgumentRefused": errText(argText), "deDEAfterRevision": errText(deErr),
			"goXText": xTextVersion(c.Repo), "javascript": versions, "rendered": goOut}})
	return nil
}
