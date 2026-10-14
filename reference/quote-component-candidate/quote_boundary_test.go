package quotecandidate

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// The WIT boundary is a candidate typed projection. These tests do not load,
// instantiate, or execute a WebAssembly Component, and do not grant authority.
type quoteCandidateCorpus struct {
	Schema   string `json:"schema"`
	Status   string `json:"status"`
	Contract string `json:"contract"`
	Vectors  []struct {
		Name    string `json:"name"`
		Request string `json:"request"`
		Reject  bool   `json:"reject"`
		Expect  *struct {
			Currency      string `json:"currency"`
			LineCount     int    `json:"lineCount"`
			SubtotalMinor uint64 `json:"subtotalMinor"`
		} `json:"expect"`
	} `json:"vectors"`
}

type typedQuoteCandidateRequest struct {
	Currency string `json:"currency"`
	Lines    []struct {
		SKU            string      `json:"sku"`
		Quantity       json.Number `json:"quantity"`
		UnitPriceMinor json.Number `json:"unitPriceMinor"`
	} `json:"lines"`
}

func readQuoteCandidateCorpus(t *testing.T) quoteCandidateCorpus {
	t.Helper()
	raw, err := os.ReadFile("../candidates/order-quote/semantic-vectors-v0.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus quoteCandidateCorpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.Schema != "tei.reference.quote-semantic-vectors/candidate-v0" ||
		corpus.Status != "unaccepted" || corpus.Contract != "reference.order.quote-lines/v1" ||
		len(corpus.Vectors) != 23 {
		t.Fatalf("oracle identity or vector count drifted: %+v", corpus)
	}
	return corpus
}

func TestWITQuoteCandidatePositiveProjectionDoesNotLoseIntegers(t *testing.T) {
	corpus := readQuoteCandidateCorpus(t)
	accepted, rejected := 0, 0
	for _, vector := range corpus.Vectors {
		if vector.Reject == (vector.Expect != nil) {
			t.Fatalf("invalid oracle status: %q", vector.Name)
		}
		if vector.Reject {
			rejected++
			continue
		}
		accepted++
		t.Run(vector.Name, func(t *testing.T) {
			// This is a feasibility check for *previously validated*
			// positive JSON. Decoder alone is NOT the ingress validator.
			decoder := json.NewDecoder(strings.NewReader(vector.Request))
			decoder.UseNumber()
			decoder.DisallowUnknownFields()
			var request typedQuoteCandidateRequest
			if err := decoder.Decode(&request); err != nil {
				t.Fatal(err)
			}
			var trailing any
			if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
				t.Fatalf("unexpected trailing JSON: %v", err)
			}
			if request.Currency != "JPY" || len(request.Lines) == 0 || len(request.Lines) > 50 {
				t.Fatal("invalid positive business request")
			}
			var subtotal uint64
			for _, line := range request.Lines {
				qty, err := strconv.ParseUint(string(line.Quantity), 10, 64)
				if err != nil || qty < 1 || qty > 100 {
					t.Fatalf("quantity cannot lower to candidate u64: %v", err)
				}
				price, err := strconv.ParseUint(string(line.UnitPriceMinor), 10, 64)
				if err != nil || price > 1000000000 {
					t.Fatalf("price cannot lower to candidate u64: %v", err)
				}
				if strings.TrimSpace(line.SKU) == "" || len([]byte(line.SKU)) > 64 {
					t.Fatal("invalid SKU bound")
				}
				// All accepted per-line products are <=1e11.
				subtotal += qty * price
				if subtotal > 1000000000000 {
					t.Fatal("positive quote overflows business limit")
				}
			}
			if vector.Expect == nil || request.Currency != vector.Expect.Currency ||
				len(request.Lines) != vector.Expect.LineCount || subtotal != vector.Expect.SubtotalMinor {
				t.Fatalf("typed projected result differs from fixed oracle: got %d, expected %+v", subtotal, vector.Expect)
			}
		})
	}
	if accepted != 8 || rejected != 15 {
		t.Fatalf("coverage changed: accepted=%d rejected=%d", accepted, rejected)
	}
}

// WIT's typed record cannot recover lexical JSON information that was
// discarded BEFORE conversion. This regression explicitly demonstrates why
// the Host/ingress boundary must use the existing strict validators.
func TestWITQuoteCandidateMustNotReplaceIngressValidation(t *testing.T) {
	corpus := readQuoteCandidateCorpus(t)
	var duplicate, caseFolded string
	for _, vector := range corpus.Vectors {
		switch vector.Name {
		case "duplicate quantity":
			if !vector.Reject {
				t.Fatal("duplicate-key oracle is not a rejection")
			}
			duplicate = vector.Request
		case "case folded field":
			if !vector.Reject {
				t.Fatal("case-folded field oracle is not a rejection")
			}
			caseFolded = vector.Request
		}
	}
	if duplicate == "" || caseFolded == "" {
		t.Fatal("missing independent ingress counterexamples")
	}
	for name, raw := range map[string]string{"duplicate": duplicate, "case-folded": caseFolded} {
		var lossy typedQuoteCandidateRequest
		if err := json.Unmarshal([]byte(raw), &lossy); err != nil {
			t.Fatalf("expected permissive standard decoder to lose %s distinction: %v", name, err)
		}
		if len(lossy.Lines) != 1 || lossy.Lines[0].Quantity != "1" && name == "case-folded" {
			t.Fatalf("unexpected normalized data for %s: %+v", name, lossy)
		}
	}
	var imprecise float64
	if err := json.Unmarshal([]byte("9007199254740993"), &imprecise); err != nil {
		t.Fatal(err)
	}
	if uint64(imprecise) == 9007199254740993 {
		t.Fatal("counterexample no longer exhibits JavaScript-number-equivalent float64 precision loss")
	}
	if parsed, err := strconv.ParseUint("9007199254740993", 10, 64); err != nil || parsed != 9007199254740993 {
		t.Fatalf("integer lexical token must remain exactly representable in u64: %d %v", parsed, err)
	}
}

func TestWITQuoteCandidateNoAmbientHostImports(t *testing.T) {
	raw, err := os.ReadFile("quote.wit")
	if err != nil {
		t.Fatal(err)
	}
	wit := string(raw)
	for _, required := range []string{
		"package tei:quote-preview-candidate@0.1.0;",
		"quantity: u64", "unit-price-minor: u64",
		"line-count: u32", "subtotal-minor: u64",
		"result<quote-result, quote-rejection>",
		"world quote-preview-pure", "export quote-preview;",
	} {
		if !strings.Contains(wit, required) {
			t.Fatalf("candidate WIT lost explicit type/world boundary: %s", required)
		}
	}
	if strings.Contains(wit, "\n    import ") || strings.Contains(wit, "\nimport ") {
		t.Fatal("pure quote world must not acquire ambient imports")
	}
	if tool, err := exec.LookPath("wasm-tools"); err == nil {
		command := exec.Command(tool, "component", "wit", "quote.wit")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("WIT parser rejected candidate: %v\n%s", err, output)
		}
	} else {
		t.Log("wasm-tools is unavailable: WIT grammar validation is not evidenced by this run")
	}
}
