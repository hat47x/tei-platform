package reportrenderer

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestEmptyReportStillHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, render := range []Renderer{CSV, JSON} {
		for _, rows := range [][]Row{nil, {}} {
			doc, err := render(ctx, rows)
			if !errors.Is(err, context.Canceled) || len(doc.Bytes) != 0 || doc.MediaType != "" {
				t.Fatalf("cancelled empty report produced output: %+v %v", doc, err)
			}
		}
	}
}

func TestRendererExchangePreservesDataAndCSVDefusesFormulas(t *testing.T) {
	rows := []Row{{OrderID: "1001", ItemCode: " \t=SUM(1,2)\n\"x\"", Quantity: "2"}}
	c, err := CSV(context.Background(), rows)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := csv.NewReader(strings.NewReader(string(c.Bytes))).ReadAll()
	if err != nil || parsed[1][2] != "'"+rows[0].ItemCode || parsed[1][0] != "1001" {
		t.Fatalf("unsafe or corrupt CSV: %v %v", parsed, err)
	}
	j, err := JSON(context.Background(), rows)
	if err != nil {
		t.Fatal(err)
	}
	var got []Row
	if err := json.Unmarshal(j.Bytes, &got); err != nil || got[0] != rows[0] {
		t.Fatal("JSON lost original data")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CSV(ctx, rows); err == nil {
		t.Fatal("cancelled renderer succeeded")
	}
}

func TestRendererBudgetsAccountForEncodedOutputAndKeepNoPartialDocument(t *testing.T) {
	rows := []Row{{OrderID: "1001", ItemCode: "<\"quoted\">", Quantity: "2"}}
	for _, format := range []string{"csv", "json"} {
		_, plain, err := Select(format)
		if err != nil {
			t.Fatal(err)
		}
		expected, err := plain(context.Background(), rows)
		if err != nil {
			t.Fatal(err)
		}
		_, exact, err := SelectBounded(format, Limits{MaxRows: 1, MaxBytes: len(expected.Bytes)})
		if err != nil {
			t.Fatal(err)
		}
		doc, err := exact(context.Background(), rows)
		if err != nil || string(doc.Bytes) != string(expected.Bytes) {
			t.Fatal("exact byte boundary changed report")
		}
		_, short, _ := SelectBounded(format, Limits{MaxRows: 1, MaxBytes: len(expected.Bytes) - 1})
		doc, err = short(context.Background(), rows)
		if !errors.Is(err, ErrReportBudget) || len(doc.Bytes) != 0 || doc.MediaType != "" {
			t.Fatal("over-budget output returned partial document")
		}
		doc, err = exact(context.Background(), append(rows, rows...))
		if !errors.Is(err, ErrReportBudget) || len(doc.Bytes) != 0 {
			t.Fatal("row budget ignored")
		}
	}
	if _, _, err := SelectBounded("csv", Limits{}); err == nil {
		t.Fatal("invalid limits accepted")
	}
	_, escaped, _ := SelectBounded("json", Limits{MaxRows: 1, MaxBytes: 100})
	if doc, err := escaped(context.Background(), []Row{{ItemCode: strings.Repeat("<", 50)}}); !errors.Is(err, ErrReportBudget) || len(doc.Bytes) != 0 {
		t.Fatal("escaped bytes not bounded")
	}
}

func TestLargeCSVCellsCannotBypassEncodedByteBudget(t *testing.T) {
	rows := []Row{{ItemCode: strings.Repeat("x", 9000)}}
	doc, err := CSV(context.Background(), rows)
	if err != nil {
		t.Fatal(err)
	}
	_, render, err := SelectBounded("csv", Limits{MaxRows: 1, MaxBytes: len(doc.Bytes) - 1})
	if err != nil {
		t.Fatal(err)
	}
	if partial, err := render(context.Background(), rows); !errors.Is(err, ErrReportBudget) || len(partial.Bytes) != 0 {
		t.Fatal("buffered large cell escaped budget")
	}
}
