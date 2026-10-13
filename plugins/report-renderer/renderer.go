// Package reportrenderer implements pure document formatting, without storage,
// database access, or business authorization.
package reportrenderer

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
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

type Limits struct {
	MaxRows  int `json:"maxRows"`
	MaxBytes int `json:"maxBytes"`
}

var ErrReportBudget = errors.New("report exceeds host row or byte budget")

func DefaultLimits() Limits { return Limits{MaxRows: 10000, MaxBytes: 8 << 20} }

// Limits belong to the host's resource policy, not the business model.
func SelectBounded(format string, limits Limits) (string, Renderer, error) {
	if limits.MaxRows <= 0 || limits.MaxBytes <= 0 {
		return "", nil, fmt.Errorf("positive report budgets are required")
	}
	provider, _, err := Select(format)
	if err != nil {
		return "", nil, err
	}
	return provider, func(ctx context.Context, rows []Row) (Document, error) {
		return renderBounded(ctx, rows, format, limits)
	}, nil
}

type boundedBuffer struct {
	buffer bytes.Buffer
	ctx    context.Context
	limit  int
}

func (b *boundedBuffer) Len() int      { return b.buffer.Len() }
func (b *boundedBuffer) Bytes() []byte { return b.buffer.Bytes() }

func (b *boundedBuffer) Write(raw []byte) (int, error) {
	if err := b.ctx.Err(); err != nil {
		return 0, err
	}
	if len(raw) > b.limit-b.Len() {
		return 0, ErrReportBudget
	}
	return b.buffer.Write(raw)
}

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
	return renderBounded(ctx, rows, "csv", DefaultLimits())
}
func JSON(ctx context.Context, rows []Row) (Document, error) {
	return renderBounded(ctx, rows, "json", DefaultLimits())
}

func renderBounded(ctx context.Context, rows []Row, format string, limits Limits) (Document, error) {
	if ctx == nil {
		return Document{}, fmt.Errorf("report context is required")
	}
	if err := ctx.Err(); err != nil {
		return Document{}, err
	}
	if len(rows) > limits.MaxRows {
		return Document{}, ErrReportBudget
	}
	// Reject oversized source text before escaping allocates a row's encoding.
	total := 0
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return Document{}, err
		}
		for _, value := range []string{row.OrderID, row.LineNumber, row.ItemCode, row.Quantity, row.UnitPrice} {
			if len(value) > limits.MaxBytes-total {
				return Document{}, ErrReportBudget
			}
			total += len(value)
		}
	}
	buf := &boundedBuffer{ctx: ctx, limit: limits.MaxBytes}
	media := "application/json"
	if format == "csv" {
		media = "text/csv; charset=utf-8"
		writer := csv.NewWriter(buf)
		if err := writer.Write([]string{"order_id", "line_number", "item_code", "quantity", "unit_price"}); err != nil {
			return Document{}, err
		}
		for _, row := range rows {
			if err := ctx.Err(); err != nil {
				return Document{}, err
			}
			if err := writer.Write([]string{cell(row.OrderID), cell(row.LineNumber), cell(row.ItemCode), cell(row.Quantity), cell(row.UnitPrice)}); err != nil {
				return Document{}, err
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return Document{}, err
		}
	} else if rows == nil {
		if _, err := buf.Write([]byte("null\n")); err != nil {
			return Document{}, err
		}
	} else {
		if _, err := buf.Write([]byte{'['}); err != nil {
			return Document{}, err
		}
		for i, row := range rows {
			if i > 0 {
				if _, err := buf.Write([]byte{','}); err != nil {
					return Document{}, err
				}
			}
			encoded, err := json.Marshal(row)
			if err != nil {
				return Document{}, err
			}
			if _, err := buf.Write(encoded); err != nil {
				return Document{}, err
			}
		}
		if _, err := buf.Write([]byte("]\n")); err != nil {
			return Document{}, err
		}
	}
	return Document{Bytes: buf.Bytes(), MediaType: media}, nil
}
