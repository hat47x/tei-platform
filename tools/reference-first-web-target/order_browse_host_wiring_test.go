//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"tei/internal/workspace"
)

// A Host (here a test acting as one, in the explicit diagnostic mode) injects
// the real browse executor and the real store. The running Web then selects
// the DB row and submits it. The shipped CLI injects neither.
func TestHostWiredPostgresBrowseDrivesTheRunningWebSubmit(t *testing.T) {
	dsn := orderSubmitTestDSN(t)
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	db := openDeclaredOrderFixtureDB(t, dsn, loaded.World)
	if _, err := db.Exec(`INSERT INTO public.orders (order_id, customer_id, ordered_at, status, total_amount, external_id)
VALUES (2001, 11, '2026-09-19T10:00:00Z', 'draft', 9900.00, NULL)`); err != nil {
		t.Fatal(err)
	}
	submit, err := newPostgresOrderSubmitStore(dsn, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = submit.Close() })

	reservation, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := reservation.Addr().String()
	if err := reservation.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runCombinedWebTargetWithHost(ctx, nil, nil,
			webHostWiring{submitStore: submit, browse: postgresBrowseExecutor{db: db}},
			[]string{
				"-workspace", referenceWorkspacePath(t), "-application", "order_entry",
				"-baseline-id", "B", "-baseline-revision", testBaselineRevision,
				"-target-revision", testTargetRevision, "-listen", address,
				"-legacy-users", legacyFixturePath(t, "users.json"),
				"-legacy-authorization", legacyFixturePath(t, "authorization.json"),
				"-diagnostic-unaccepted-source",
			})
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("web target did not stop with its context")
		}
	})

	client := &http.Client{Timeout: 10 * time.Second}
	base := "http://" + address
	for deadline := time.Now().Add(15 * time.Second); ; {
		response, err := client.Get(base + "/healthz")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				break
			}
		}
		select {
		case err := <-done:
			t.Fatalf("web target stopped before publishing HTTP: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("web target did not publish HTTP")
		}
		time.Sleep(100 * time.Millisecond)
	}

	response, err := client.Post(base+"/scenario/submit", "application/json", strings.NewReader(
		`{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":0}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.StatusCode, raw)
	}
	var envelope webCommandScenarioResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("response is not a command result: %v\n%s", err, raw)
	}
	result := envelope.Result
	if result.Stage != "completed" || result.Command == nil || result.Command.Payload["orderId"] != float64(2001) {
		t.Fatalf("result=%+v\n%s", result, raw)
	}
	if got := orderStatus(t, db, 2001); got != "submitted" {
		t.Fatalf("order 2001 status=%q", got)
	}
	for _, untouched := range []int64{1001, 1002} {
		if got := orderStatus(t, db, untouched); got != "draft" {
			t.Fatalf("order %d status=%q", untouched, got)
		}
	}
}
