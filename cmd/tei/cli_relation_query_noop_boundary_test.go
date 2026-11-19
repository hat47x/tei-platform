package main

import (
    "context"
    "reflect"
    "testing"

    "tei/internal/application"
    "tei/internal/driver/queryresult"
    "tei/internal/query"
)

// This command currently resolves SQL without reading a database. Keep that
// boundary explicit until a real Provider and Consumer are wired in (#2586).
func TestRelationQueryNoopExecutorDoesNotFabricateReadRows(t *testing.T) {
    const sql = "SELECT id, amount FROM orders"
    got, err := (noopRelationQueryExecutor{}).ExecuteRelationQuery(
        context.Background(),
        query.ResolvedRelationQuery{SQL: sql},
        application.DatabaseRequirement{},
        map[string]any{"id": 1},
    )
    if err != nil {
        t.Fatalf("noop query: %v", err)
    }
    if !reflect.DeepEqual(got.Raw, queryresult.RawResult{}) {
        t.Fatalf("no-op query unexpectedly reported raw database rows: %+v", got.Raw)
    }
    if !reflect.DeepEqual(got.Mapped, queryresult.MappedResult{}) {
        t.Fatalf("no-op query unexpectedly reported mapped database rows: %+v", got.Mapped)
    }
    if !reflect.DeepEqual(got.Payload, map[string]string{"sql": sql}) {
        t.Fatalf("no-op query must report only resolved SQL: %#v", got.Payload)
    }
}
