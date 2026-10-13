// Package reportrenderer implements pure document formatting, without storage,
// database access, or business authorization.
package reportrenderer

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"tei/internal/composition"
)

var Contract = composition.Contract{ID: "tei.reference.report.render", Version: "1", Shape: "order-line-text-rows -> document-bytes-and-media-type/v1; pure; no-storage"}

type Row struct {
	OrderID    string `json:"orderId"`
	LineNumber string `json:"lineNumber"`
	ItemCode   string `json:"itemCode"`
	Quantity   string `json:"quantity"`
	UnitPrice  string `json:"unitPrice"`
}
type Document struct {
	Bytes     []byte
	MediaType string
}
type Renderer func(context.Context, []Row) (Document, error)

func Select(format string) (string, Renderer, error) {
	switch format {
	case "csv":
		return "report-csv", CSV, nil
	case "json":
		return "report-json", JSON, nil
	default:
		return "", nil, fmt.Errorf("unsupported report format")
	}
}

// Spreadsheet programs may interpret leading whitespace followed by a formula.
// CSV is a viewing artifact; JSON preserves the original text for interchange.
func cell(s string) string {
	t := strings.TrimLeft(s, " \t\r\n")
	if len(t) > 0 && strings.ContainsRune("=+-@", rune(t[0])) {
		return "'" + s
	}
	return s
}
func CSV(ctx context.Context, rows []Row) (Document, error) {
	if err := ctx.Err(); err != nil {
		return Document{}, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"order_id", "line_number", "item_code", "quantity", "unit_price"}); err != nil {
		return Document{}, err
	}
	for _, r := range rows {
		if err := ctx.Err(); err != nil {
			return Document{}, err
		}
		if err := w.Write([]string{cell(r.OrderID), cell(r.LineNumber), cell(r.ItemCode), cell(r.Quantity), cell(r.UnitPrice)}); err != nil {
			return Document{}, err
		}
	}
	w.Flush()
	return Document{buf.Bytes(), "text/csv; charset=utf-8"}, w.Error()
}
func JSON(ctx context.Context, rows []Row) (Document, error) {
	if err := ctx.Err(); err != nil {
		return Document{}, err
	}
	b, err := json.Marshal(rows)
	return Document{append(b, '\n'), "application/json"}, err
}
