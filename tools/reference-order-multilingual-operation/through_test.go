package ordermultilingualoperation

// The through test needs the dedicated, empty, disposable PostgreSQL
// database tei_omo, Python with sqlite3 and Node, normally provided by
// tools/verify-order-multilingual-operation.py. Without the DSN it skips; a
// skip is not evidence that anything passed.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestThroughOrderMultilingualOperation(t *testing.T) {
	dsn := os.Getenv(DSNEnv)
	if dsn == "" {
		t.Skip("requires the dedicated PostgreSQL database (TEI_OMO_DSN); skip is not a pass")
	}
	if os.Getenv(ConfirmEnv) != ConfirmValue {
		t.Fatalf("%s must be %q to confirm the database is disposable", ConfirmEnv, ConfirmValue)
	}
	cfg := Config{DSN: dsn, Python: os.Getenv(PythonEnv), Node: os.Getenv(NodeEnv)}
	if cfg.Python == "" || cfg.Node == "" {
		t.Fatalf("%s and %s are required", PythonEnv, NodeEnv)
	}
	c := newTestChain(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	runErr := c.Run(ctx, cfg)
	raw, err := json.MarshalIndent(c.Rec, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv(OutDirEnv)
	if dir == "" {
		dir = t.TempDir()
	}
	if err := os.WriteFile(filepath.Join(dir, "chain-record.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if runErr != nil {
		t.Fatalf("chain stopped: %v", runErr)
	}
	for _, list := range [][]Check{c.Rec.Faults, c.Rec.Completion, c.Rec.Conditional} {
		for _, ch := range list {
			if !ch.Passed {
				t.Errorf("%s did not pass: %s", ch.ID, dump(ch.Observed))
			}
		}
	}
	if !c.Rec.Through.Complete {
		t.Errorf("through: %+v", c.Rec.Through)
	}
	if !c.Rec.Passed {
		t.Fatal("chain record is not passed")
	}
}
