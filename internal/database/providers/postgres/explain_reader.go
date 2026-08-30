package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type ExplainRequest struct {
	SQL     string
	Args    []any
	Analyze bool
	Buffers bool
}

// ExplainReader isolates statement execution from plan normalization. Tests can
// supply fixtures; a live editor can use SQLExplainReader or another adapter.
type ExplainReader interface {
	ExplainJSON(context.Context, ExplainRequest) ([]byte, error)
}

type SQLExplainReader struct{ DB *sql.DB }

func (r SQLExplainReader) ExplainJSON(ctx context.Context, req ExplainRequest) ([]byte, error) {
	if r.DB == nil {
		return nil, fmt.Errorf("postgres explain database is nil")
	}
	stmt := strings.TrimSpace(req.SQL)
	if stmt == "" {
		return nil, fmt.Errorf("postgres explain SQL is empty")
	}
	options := []string{"FORMAT JSON"}
	if req.Analyze {
		options = append(options, "ANALYZE true")
	}
	if req.Buffers {
		options = append(options, "BUFFERS true")
	}
	sqlText := "EXPLAIN (" + strings.Join(options, ", ") + ") " + stmt
	var raw []byte
	if err := r.DB.QueryRowContext(ctx, sqlText, req.Args...).Scan(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}
