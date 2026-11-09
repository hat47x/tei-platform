package expectedstateupdate

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

// RCAS-03 (#2564): a real socket-level loss of the commit acknowledgement. The
// test connects through a TCP proxy that forwards the commit to PostgreSQL and
// then drops the server's COMMIT CommandComplete before it reaches the caller.
// The server has committed; the caller sees a transport error and cannot tell
// whether the effect was applied. This is a real connection failure, not a
// wrapper that fabricates an error.
func TestPGCommitAcknowledgementLossOverSocket(t *testing.T) {
	ctx := context.Background()
	direct := openPGStore(t, 0)
	if err := direct.Seed(ctx, "order:1", "secret-before"); err != nil {
		t.Fatal(err)
	}
	dsn := os.Getenv("TEI_EXPECTED_STATE_PG_DSN")
	proxyAddr, stop := startCommitResponseCutProxy(t, dsn)
	defer stop()
	proxyDSN := strings.Replace(dsn, postgresAddress(t, dsn), proxyAddr, 1)
	proxyDB, err := sql.Open("pgx", proxyDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer proxyDB.Close()
	proxied := NewPGStore(proxyDB)

	request := UpdateRequest{
		Key: "order:1", ExpectedPresent: true, Expected: "secret-before", NewValue: "secret-after",
		IdempotencyKey: "update-a", Authorized: true, Valid: true,
	}
	stage := NewRuntimeStage()
	if _, err := stage.Dispatch("cmd-ack", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := proxied.Apply(ctx, request); err == nil {
		t.Fatal("the lost commit acknowledgement was reported as a clean success")
	}

	// The server committed even though the caller saw a transport error.
	deadline := time.Now().Add(15 * time.Second)
	var value string
	for {
		value, _, err = direct.Value(ctx, "order:1")
		if err != nil {
			t.Fatal(err)
		}
		if value == "secret-after" || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if value != "secret-after" {
		t.Fatalf("the server did not commit the effect, value=%q", value)
	}
	if receipts := receiptCount(t, ctx, direct, "update-a"); receipts != 1 {
		t.Fatalf("receipts=%d, want 1", receipts)
	}

	// The caller cannot observe the outcome, so the stage becomes unknown and a
	// same-attempt replay is refused.
	if err := stage.MarkUnknown("cmd-ack", "owner-a", 1, "connection closed before the commit response"); err != nil {
		t.Fatal(err)
	}
	if stage.MayReplay("cmd-ack") {
		t.Fatal("an unacknowledged commit was reported as replayable")
	}
	if _, err := stage.Dispatch("cmd-ack", "owner-a", 1); err == nil {
		t.Fatal("the same attempt was replayed after an unacknowledged commit")
	}

	// Recovery uses the operation identity: the same key replays the receipt
	// instead of applying a second effect.
	if _, err := stage.Takeover("cmd-ack", "owner-b", 2); err != nil {
		t.Fatal(err)
	}
	recovered, err := direct.Apply(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Outcome != OutcomeIdempotentReplay {
		t.Fatalf("recovery outcome=%q, want %s", recovered.Outcome, OutcomeIdempotentReplay)
	}
	if value, _, err := direct.Value(ctx, "order:1"); err != nil || value != "secret-after" {
		t.Fatalf("recovery changed the value: %q err=%v", value, err)
	}
	if receipts := receiptCount(t, ctx, direct, "update-a"); receipts != 1 {
		t.Fatalf("recovery wrote a second receipt: %d", receipts)
	}
	if err := stage.Observe("cmd-ack", "owner-b", 2, recovered.Outcome); err != nil {
		t.Fatal(err)
	}
	t.Logf("socket-level ack loss: serverCommitted=true callerError=true recovery=%s receipts=1",
		recovered.Outcome)
}

func receiptCount(t *testing.T, ctx context.Context, store *PGStore, idempotencyKey string) int {
	t.Helper()
	var receipts int
	if err := store.db.QueryRowContext(ctx,
		`SELECT count(*) FROM `+PGAppliedTable+` WHERE idempotency_key = $1`, idempotencyKey).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	return receipts
}

// postgresAddress extracts host:port from the verification DSN so the proxy can
// be substituted into a copy of it.
func postgresAddress(t *testing.T, dsn string) string {
	t.Helper()
	withoutScheme := dsn
	if index := strings.Index(withoutScheme, "://"); index >= 0 {
		withoutScheme = withoutScheme[index+3:]
	}
	if index := strings.Index(withoutScheme, "@"); index >= 0 {
		withoutScheme = withoutScheme[index+1:]
	}
	if index := strings.IndexAny(withoutScheme, "/?"); index >= 0 {
		withoutScheme = withoutScheme[:index]
	}
	if withoutScheme == "" {
		t.Fatal("cannot extract the PostgreSQL address from the DSN")
	}
	return withoutScheme
}

// startCommitResponseCutProxy forwards PostgreSQL traffic and drops the client
// connection when the server's COMMIT CommandComplete arrives. The commit has
// already happened on the server; only the acknowledgement is lost.
func startCommitResponseCutProxy(t *testing.T, dsn string) (string, func()) {
	t.Helper()
	upstream := postgresAddress(t, dsn)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			client, err := listener.Accept()
			if err != nil {
				return
			}
			go proxyConnection(client, upstream)
		}
	}()
	return listener.Addr().String(), func() {
		_ = listener.Close()
		<-done
	}
}

func proxyConnection(client net.Conn, upstream string) {
	server, err := net.Dial("tcp", upstream)
	if err != nil {
		_ = client.Close()
		return
	}
	go func() {
		defer server.Close()
		_, _ = io.Copy(server, client)
	}()
	defer client.Close()
	defer server.Close()
	copyUntilCommitResponse(client, server)
}

// copyUntilCommitResponse copies typed PostgreSQL server messages to the
// client, then stops just before the COMMIT CommandComplete payload is written.
func copyUntilCommitResponse(client, server net.Conn) {
	header := make([]byte, 5)
	for {
		if _, err := io.ReadFull(server, header); err != nil {
			return
		}
		length := int(binary.BigEndian.Uint32(header[1:5]))
		if length < 4 || length > 1<<20 {
			return
		}
		payload := make([]byte, length-4)
		if _, err := io.ReadFull(server, payload); err != nil {
			return
		}
		if header[0] == 'C' && bytes.HasPrefix(payload, []byte("COMMIT")) {
			return
		}
		if _, err := client.Write(header); err != nil {
			return
		}
		if _, err := client.Write(payload); err != nil {
			return
		}
	}
}
