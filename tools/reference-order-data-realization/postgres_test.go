package orderdatarealization

// Real PostgreSQL test. It needs the dedicated, empty, disposable database
// tei_odr, normally started by tools/verify-order-data-realization.py.
// Without the DSN it skips; a skip is not evidence that anything passed.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPostgresOrderDataRealizationChain(t *testing.T) {
	dsn := os.Getenv(DSNEnv)
	if dsn == "" {
		t.Skip("requires the dedicated PostgreSQL database (TEI_ODR_DSN); skip is not a pass")
	}
	if os.Getenv(ConfirmEnv) != ConfirmValue {
		t.Fatalf("%s must be %q to confirm the database is disposable", ConfirmEnv, ConfirmValue)
	}
	c := newTestChain(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	runErr := c.Run(ctx, dsn)
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
	for _, f := range c.Rec.Faults {
		if !f.Passed {
			obs, _ := json.Marshal(f.Observed)
			t.Errorf("fault injection %s did not pass: %s", f.ID, obs)
		}
	}
	for _, ch := range c.Rec.Completion {
		if !ch.Passed {
			obs, _ := json.Marshal(ch.Observed)
			t.Errorf("completion condition %s did not pass: %s", ch.ID, obs)
		}
	}
	if !c.Rec.Passed {
		t.Fatal("chain record is not passed")
	}
}
