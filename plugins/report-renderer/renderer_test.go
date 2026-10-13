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
