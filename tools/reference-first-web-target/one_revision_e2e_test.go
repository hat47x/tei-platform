//go:build !tei_order_only && !tei_repair_only

package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tei/internal/sourceacceptance/hoststore"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// TestOneRevisionOrderEntryE2EPacket records ONE run for #1163 (provisional
// delegated decision D6): a Shared Acceptance R1, signed by a per-run Ed25519
// issuer key and read from the Host Acceptance Store by a running Web process,
// drives a real PostgreSQL write that another OS process reads back.
//
// What stays synthetic, and is labelled so in the packet: the Host effect Grant
// (syntheticEffectReader), the issuer (a per-run key, not a human), the order
// rows the Command validates (the reference read fixture), and the zero unit
// price of the negative case. The physical table is built from the accepted
// declaration, and the Web process checks it at startup and before each write
// (S02); two drifted tables in a second disposable database are the S02
// controls. The audit can never return a Product Gate; this run is not a G2
// PASS.
//
// It needs TEI_TEST_POSTGRES_DSN (an empty, disposable database) and
// TEI_E2E_PACKET_OUT (a directory outside the repository); otherwise it skips.
//
// TEI_E2E_FULL_HEALTH_RECEIPT optionally names a JSON file from
// `gh run view <id> --json databaseId,headSha,event,workflowName,status,conclusion,url`
// for the repository health workflow run on push to main. It is accepted only
// when that run checked out exactly this HEAD (which must equal origin/main)
// and succeeded; the workflow checks a clean tree before and after
// tools/repository-health-check.sh. The receipt is caller-supplied: the audit
// does not dereference it, and a human reviews it with the packet.

const (
	e2eOutEnv       = "TEI_E2E_PACKET_OUT"
	e2eHealthEnv    = "TEI_E2E_FULL_HEALTH_RECEIPT"
	e2eHealthFlow   = "TEI PR・main 健全性確認"
	e2eRunURLPrefix = "https://github.com/hat47x/tei-platform/actions/runs/"
	e2eChildStore   = "TEI_E2E_CHILD_STORE"
	e2eChildTrust   = "TEI_E2E_CHILD_TRUST"
	e2eChildGrant   = "TEI_E2E_CHILD_GRANT"
	e2eChildReadDSN = "TEI_E2E_CHILD_READ_DSN"
	e2eChildDSN     = "TEI_E2E_CHILD_DSN"
	e2eChildZero    = "TEI_E2E_CHILD_ZERO_PRICE"
	e2eDecisionRef  = "e2e-1163-d1"
	e2eIssuer       = "provisional-per-run-issuer"
	e2eKeyID        = "run-key-1"
)

// zeroFirstLinePrice is the running-server form of the existing price-0
// negative: the first order line of the read fixture gets unit price 0.
type zeroFirstLinePrice struct{ inner readScenarioRunner }

func (z zeroFirstLinePrice) Run(ctx context.Context, input readScenarioInput) (readScenarioResult, error) {
	result, err := z.inner.Run(ctx, input)
	if err != nil || len(result.OrderLineRows) == 0 {
		return result, err
	}
	rows := append([]semanticRowProjection(nil), result.OrderLineRows...)
	row := rows[0]
	row.Attributes = append(row.Attributes[:0:0], row.Attributes...)
	if !setSemanticRowValue(&row, "sales.order_line.unit_price", "0") {
		return result, errors.New("unit price not present in the read fixture")
	}
	rows[0] = row
	result.OrderLineRows = rows
	return result, nil
}

// TestOneRevisionE2EWebChild is the Web process of the run above.
func TestOneRevisionE2EWebChild(t *testing.T) {
	dir := os.Getenv(e2eChildStore)
	if dir == "" {
		return
	}
	separator := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separator = i
			break
		}
	}
	fail := func(code string) { fmt.Fprintln(os.Stderr, code); os.Exit(3) }
	if separator < 0 {
		fail("E2E_CHILD_ARGS")
	}
	policy, err := hoststore.LoadTrustPolicy(os.Getenv(e2eChildTrust))
	if err != nil {
		fail("E2E_CHILD_TRUST")
	}
	store, err := hoststore.Open(dir, policy.Verifier())
	if err != nil {
		fail("E2E_CHILD_STORE")
	}
	dsn := os.Getenv(orderSubmitDSNEnv)
	if override := os.Getenv(e2eChildDSN); override != "" {
		dsn = override
	}
	readDSN := dsn
	if override := os.Getenv(e2eChildReadDSN); override != "" {
		readDSN = override
	}
	submit, err := newPostgresOrderSubmitStore(dsn, readDSN)
	if err != nil {
		fail("E2E_CHILD_DB")
	}
	defer submit.Close()
	wiring := webHostWiring{submitStore: submit}
	if os.Getenv(e2eChildZero) == "1" {
		wiring.wrapCommandRead = func(inner readScenarioRunner) readScenarioRunner { return zeroFirstLinePrice{inner} }
	}
	grant := &syntheticEffectReader{current: os.Getenv(e2eChildGrant) != "revoked"}
	if err := runCombinedWebTargetWithHost(context.Background(), store, grant, wiring, os.Args[separator+1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

type e2eRun struct {
	t        *testing.T
	repoRoot string
	out      string
	env      []string
	exe      string
}

type e2eChild struct {
	base string
	pid  int
	stop func()
}

func (r *e2eRun) command(ctx context.Context, root, address string, extraEnv ...string) *exec.Cmd {
	args := []string{"-test.run=^TestOneRevisionE2EWebChild$", "--", "-workspace", root, "-application", "order_entry",
		"-baseline-id", "B", "-baseline-revision", testBaselineRevision, "-target-revision", testTargetRevision, "-listen", address}
	command := exec.CommandContext(ctx, r.exe, args...)
	command.Dir = r.repoRoot
	command.Env = append(append(os.Environ(), r.env...), extraEnv...)
	return command
}

func (r *e2eRun) start(root string, extraEnv ...string) e2eChild {
	t := r.t
	t.Helper()
	address := freeAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	command := r.command(ctx, root, address, extraEnv...)
	var log bytes.Buffer
	command.Stdout, command.Stderr = &log, &log
	if err := command.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	stopped := false
	stop := func() {
		if !stopped {
			stopped = true
			cancel()
			<-done
		}
	}
	t.Cleanup(stop)
	base := "http://" + address
	client := &http.Client{Timeout: time.Second}
	for deadline := time.Now().Add(20 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		if response, err := client.Get(base + "/healthz"); err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return e2eChild{base: base, pid: command.Process.Pid, stop: stop}
			}
		}
		if time.Now().After(deadline) {
			stop()
			t.Fatalf("Web child did not serve HTTP: %s", redactDSN(log.String()))
		}
	}
}

func (r *e2eRun) refuse(root, code string, extraEnv ...string) []byte {
	t := r.t
	t.Helper()
	address := freeAddress(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := r.command(ctx, root, address, extraEnv...)
	command.Args = append(command.Args, "-legacy-users", "does-not-exist", "-legacy-authorization", "does-not-exist")
	output, err := command.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 2 || !bytes.Contains(output, []byte(code)) ||
		bytes.Contains(output, []byte("TEI first Web Target projection:")) {
		t.Fatalf("expected the Web process to refuse with %s before HTTP: err=%v output=%s", code, err, redactDSN(string(output)))
	}
	if response, err := (&http.Client{Timeout: 300 * time.Millisecond}).Get("http://" + address + "/healthz"); err == nil {
		response.Body.Close()
		t.Fatal("refused Source published HTTP")
	}
	return output
}

func submitRow(t *testing.T, base string, row int) (int, []byte) {
	t.Helper()
	body := fmt.Sprintf(`{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":%d}`, row)
	response, err := (&http.Client{Timeout: 10 * time.Second}).Post(base+"/scenario/submit", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("editor-secret")) {
		t.Fatal("credential echoed")
	}
	return response.StatusCode, raw
}

func decodeCommand(t *testing.T, raw []byte) (commandScenarioResult, string) {
	t.Helper()
	var observed struct {
		Result         commandScenarioResult `json:"result"`
		ExecutionError string                `json:"executionError"`
	}
	if err := json.Unmarshal(raw, &observed); err != nil {
		t.Fatalf("not a Command response: %v %s", err, raw)
	}
	return observed.Result, observed.ExecutionError
}

// write stores a receipt and returns its locator: file name and sha256.
func (r *e2eRun) write(name string, value any) string {
	t := r.t
	t.Helper()
	raw, ok := value.([]byte)
	if !ok {
		var err error
		raw, err = json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
	}
	raw = []byte(redactDSN(string(raw)))
	if err := os.WriteFile(filepath.Join(r.out, name), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	return name + "@sha256:" + hex.EncodeToString(sum[:])
}

var e2eDSNs []string

func redactDSN(s string) string {
	for _, dsn := range e2eDSNs {
		if dsn != "" {
			s = strings.ReplaceAll(s, dsn, "<dsn>")
		}
	}
	return s
}

func gitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	output, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(output))
}

func TestOneRevisionOrderEntryE2EPacket(t *testing.T) {
	dsn := orderSubmitTestDSN(t)
	out := os.Getenv(e2eOutEnv)
	if out == "" {
		t.Skip(e2eOutEnv + " is not set")
	}
	repoRoot := filepath.Dir(filepath.Dir(referenceWorkspacePath(t)))
	if rel, err := filepath.Rel(repoRoot, out); err == nil && !strings.HasPrefix(rel, "..") {
		t.Fatalf("%s must be outside the repository", e2eOutEnv)
	}
	if err := os.MkdirAll(out, 0o700); err != nil {
		t.Fatal(err)
	}
	head := gitOutput(t, repoRoot, "rev-parse", "HEAD")
	mainSHA := gitOutput(t, repoRoot, "rev-parse", "origin/main")
	cleanBefore := gitOutput(t, repoRoot, "status", "--porcelain") == ""
	missingRead, err := url.Parse(dsn)
	if err != nil {
		t.Fatal("DSN is not a URL")
	}
	missingRead.Path = "/tei_e2e_missing_read_database"
	e2eDSNs = []string{dsn, missingRead.String()}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	run := &e2eRun{t: t, repoRoot: repoRoot, out: out, exe: exe}
	started := time.Now().UTC()

	// 1. Shared Acceptance R0 -> R1 (test-only acceptance on a scratch copy).
	r1, receipts := workspaceWithUnitPriceRuleReceipts(t)
	r0 := filepath.Join(t.TempDir(), "r0")
	if err := os.CopyFS(r0, os.DirFS(referenceWorkspacePath(t))); err != nil {
		t.Fatal(err)
	}
	r2 := filepath.Join(t.TempDir(), "r2")
	if err := os.CopyFS(r2, os.DirFS(r1)); err != nil {
		t.Fatal(err)
	}
	appFile, err := os.OpenFile(filepath.Join(r2, "applications", "order-entry", "application.tei"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := appFile.WriteString("\n")
	if closeErr := appFile.Close(); writeErr != nil || closeErr != nil {
		t.Fatal(writeErr, closeErr)
	}
	loaded, err := workspace.Load(r1)
	if err != nil {
		t.Fatal(err)
	}
	index1, err := workspace.BuildSourceIndex(r1, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if sourceindex.ModelRevision(index1) != receipts.R1 || receipts.R0 == receipts.R1 {
		t.Fatalf("Source Index revision %s is not the accepted R1 %s", sourceindex.ModelRevision(index1), receipts.R1)
	}

	// 2. A per-run issuer signs R1; the Host Store ingests it under a trust file.
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	trustRaw, err := json.Marshal(hoststore.TrustPolicy{Schema: "tei.host-acceptance-trust/v0", Issuers: []hoststore.TrustedIssuer{{
		Issuer: e2eIssuer, KeyID: e2eKeyID, PublicKey: base64.StdEncoding.EncodeToString(public), Applications: []string{"order_entry"},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	hostDir := t.TempDir()
	trustPath, storeDir := filepath.Join(hostDir, "trust.json"), filepath.Join(hostDir, "store")
	if err := os.WriteFile(trustPath, trustRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	policy, err := hoststore.ParseTrustPolicy(trustRaw)
	if err != nil {
		t.Fatal(err)
	}
	store, err := hoststore.Open(storeDir, policy.Verifier())
	if err != nil {
		t.Fatal(err)
	}
	signed, err := hoststore.Sign(hoststore.Decision{Issuer: e2eIssuer,
		Record: hoststore.RecordForIndex("order_entry", e2eDecisionRef, "provisional-issuer-revision-1", index1)}, e2eKeyID, private)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Ingest(context.Background(), signed); err != nil {
		t.Fatal(err)
	}
	run.env = []string{e2eChildStore + "=" + storeDir, e2eChildTrust + "=" + trustPath}

	// 3. The disposable PostgreSQL fixture, built from the accepted R1
	// declaration: orders 1001 and 1002, both draft. Its readiness is the S02
	// positive evidence.
	db := openDeclaredOrderFixtureDB(t, dsn, loaded.World)
	readinessStore, err := newPostgresOrderSubmitStore(dsn, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = readinessStore.Close() })
	readiness, err := readinessStore.checkOrderSubmitSchema(context.Background(), loaded.World)
	if err != nil || !readiness.Compatible {
		t.Fatalf("the declared fixture is not ready: %+v %v", readiness, err)
	}

	// 4. Web refuses R0 and the meaning-equivalent R2 itself, before HTTP.
	refusalR0 := run.refuse(r0, "SOURCE_FINGERPRINT_MISMATCH")
	refusalR2 := run.refuse(r2, "SOURCE_FINGERPRINT_MISMATCH")

	// 5. Selected rule, running server: unit price 0 is refused; no write.
	child := run.start(r1, e2eChildZero+"=1")
	status, negativeRaw := submitRow(t, child.base, 0)
	negative, execErr := decodeCommand(t, negativeRaw)
	child.stop()
	refusedByRule := false
	for _, v := range negative.Validations {
		refusedByRule = refusedByRule || (v.Validation == "sales.order_line.unit_price_positive" && v.Result == "not-satisfied")
	}
	if status != http.StatusOK || negative.Stage != "validation-not-satisfied" || !refusedByRule ||
		negative.Command != nil || execErr != "" || orderStatus(t, db, 1001) != "draft" {
		t.Fatalf("selected-rule negative: status=%d stage=%s err=%q row=%s", status, negative.Stage, execErr, orderStatus(t, db, 1001))
	}

	// 6. S05 control: the effect Grant is revoked; order 1001 is not written.
	// (The read fixture offers one order row: order 1001.)
	child = run.start(r1, e2eChildGrant+"=revoked")
	status, s05Raw := submitRow(t, child.base, 0)
	child.stop()
	if status != http.StatusUnprocessableEntity || !bytes.Contains(s05Raw, []byte("HOST_EFFECT_GRANT_UNAVAILABLE")) ||
		orderStatus(t, db, 1001) != "draft" || orderStatus(t, db, 1002) != "draft" {
		t.Fatalf("S05 control: status=%d body=%s", status, s05Raw)
	}

	// 7. Positive: order 1001 is submitted through the running Web process.
	child = run.start(r1)
	webPID := child.pid
	status, positiveRaw := submitRow(t, child.base, 0)
	positive, execErr := decodeCommand(t, positiveRaw)
	child.stop()
	if status != http.StatusOK || positive.Stage != "completed" || positive.Command == nil || execErr != "" ||
		positive.Observation == nil || positive.Observation.Nature != "external-observed" || positive.Observation.Result != "submitted" {
		t.Fatalf("positive: status=%d body=%s", status, positiveRaw)
	}
	if orderStatus(t, db, 1001) != "submitted" || orderStatus(t, db, 1002) != "draft" {
		t.Fatal("the positive Command did not change exactly order 1001")
	}

	// 8. Independent read by another OS process after the Web process stopped.
	reader := exec.Command(exe, "-test.run=^TestHelperReadOrderStatus$", "-test.v")
	reader.Env = append(os.Environ(), orderSubmitHelperEnv+"=1001")
	var readOut bytes.Buffer
	reader.Stdout, reader.Stderr = &readOut, &readOut
	if err := reader.Run(); err != nil || !strings.Contains(readOut.String(), "ORDER_STATUS status=submitted") {
		t.Fatalf("independent read: %v %s", err, redactDSN(readOut.String()))
	}
	if reader.Process.Pid == webPID {
		t.Fatal("the read reused the Web process")
	}

	// 9. N11 control: the write commits but the Observation read fails. The
	// only selectable order is 1001, so the disposable fixture row is reset
	// to draft first; the reset is recorded in the N11 receipt.
	if _, err := db.Exec(`UPDATE public.orders SET status = 'draft' WHERE order_id = 1001`); err != nil {
		t.Fatal(err)
	}
	child = run.start(r1, e2eChildReadDSN+"="+missingRead.String())
	status, n11Raw := submitRow(t, child.base, 0)
	n11, _ := decodeCommand(t, n11Raw)
	child.stop()
	parsedDSN, _ := url.Parse(dsn)
	for _, detail := range []string{parsedDSN.Host, parsedDSN.User.Username() + " ", "user=", "tei_e2e_missing_read_database"} {
		if bytes.Contains(n11Raw, []byte(detail)) {
			t.Fatalf("N11 response discloses database connection detail %q", detail)
		}
	}
	if n11.Stage == "completed" || (n11.Observation != nil && n11.Observation.Result == "submitted") ||
		orderStatus(t, db, 1001) != "submitted" {
		t.Fatalf("N11: an unobserved write was reported: status=%d body=%s row=%s", status, n11Raw, orderStatus(t, db, 1001))
	}
	// 10. S02 controls in a second disposable database of the same server.
	// (a) The declared index is missing: the Web process does not publish HTTP.
	// (b) A column is dropped after startup: the submit is refused before the
	// UPDATE, and order 1001 and the row count stay.
	s02DSN := otherDatabaseDSN(t, dsn, "tei_e2e_s02")
	e2eDSNs = append(e2eDSNs, s02DSN)
	db2 := openDeclaredOrderFixtureDB(t, s02DSN, loaded.World)
	s02Store, err := newPostgresOrderSubmitStore(s02DSN, s02DSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s02Store.Close() })
	if _, err := db2.Exec(`DROP INDEX public.orders_customer_idx`); err != nil {
		t.Fatal(err)
	}
	startupDrift, _ := s02Store.checkOrderSubmitSchema(context.Background(), loaded.World)
	s02Startup := run.refuse(r1, "PHYSICAL_SCHEMA_NOT_READY", e2eChildDSN+"="+s02DSN)
	resetDeclaredOrders(t, db2, loaded.World)
	child = run.start(r1, e2eChildDSN+"="+s02DSN)
	if _, err := db2.Exec(`ALTER TABLE public.orders DROP COLUMN external_id`); err != nil {
		t.Fatal(err)
	}
	runtimeDrift, _ := s02Store.checkOrderSubmitSchema(context.Background(), loaded.World)
	status, s02Raw := submitRow(t, child.base, 0)
	s02Result, s02Err := decodeCommand(t, s02Raw)
	child.stop()
	var s02Rows int
	if err := db2.QueryRow(`SELECT count(*) FROM public.orders`).Scan(&s02Rows); err != nil {
		t.Fatal(err)
	}
	if status != http.StatusUnprocessableEntity || !strings.Contains(s02Err, "PHYSICAL_SCHEMA_NOT_READY") ||
		s02Result.Stage != "physical-schema-not-ready" || s02Result.Command != nil ||
		orderStatus(t, db2, 1001) != "draft" || s02Rows != 2 ||
		bytes.Contains(s02Raw, []byte("public.orders")) || bytes.Contains(s02Raw, []byte("tei_e2e_s02")) {
		t.Fatalf("S02 runtime control: status=%d stage=%s err=%q rows=%d body=%s", status, s02Result.Stage, s02Err, s02Rows, redactDSN(string(s02Raw)))
	}
	if startupDrift.Compatible || runtimeDrift.Compatible {
		t.Fatal("the drifted tables were reported compatible")
	}

	cleanAfter := gitOutput(t, repoRoot, "status", "--porcelain") == ""
	fullHealth, healthRaw := e2eFullHealth(t, head, mainSHA)

	// 11. Receipts and the packet.
	locator := map[string]string{
		"candidate": run.write("acceptance-candidate.json", receipts.Candidate),
		"review":    run.write("acceptance-review.json", receipts.Review),
		"decision":  run.write("acceptance-decision.json", receipts.Decision),
		"reload":    run.write("acceptance-reload.json", receipts.Reload),
		"signed":    run.write("host-signed-decision.json", signed),
		"trust":     run.write("host-trust.json", trustRaw),
		"refusalR0": run.write("web-refusal-r0.txt", refusalR0),
		"refusalR2": run.write("web-refusal-r2.txt", refusalR2),
		"negative":  run.write("web-negative-unit-price-zero.json", negativeRaw),
		"s05":       run.write("web-s05-effect-grant-revoked.json", s05Raw),
		"positive":  run.write("web-positive.json", positiveRaw),
		"command":   run.write("physical-command.json", map[string]any{"orderId": 1001, "command": positive.Command, "webPid": webPID, "rowStatusAfter": "submitted"}),
		"read":      run.write("physical-independent-read.json", map[string]any{"orderId": 1001, "readerPid": reader.Process.Pid, "output": readOut.String()}),
		"n11": run.write("web-n11-unobserved.json", map[string]any{"fixtureReset": "public.orders order_id=1001 set to draft before this control",
			"status": status, "response": json.RawMessage(n11Raw), "rowStatusAfter": "submitted"}),
		"schema": run.write("physical-schema-readiness.json", map[string]any{"scope": "the submit store's write set (public.orders)",
			"declaredFrom": "accepted R1 Persistence Mapping", "readiness": readiness}),
		"s02": run.write("physical-s02-controls.json", map[string]any{
			"startup": map[string]any{"drift": "DROP INDEX public.orders_customer_idx", "readiness": startupDrift, "webOutput": string(s02Startup)},
			"runtime": map[string]any{"drift": "ALTER TABLE public.orders DROP COLUMN external_id (after startup)", "readiness": runtimeDrift,
				"status": status, "response": json.RawMessage(s02Raw), "order1001After": "draft", "rowCountAfter": s02Rows},
			"database": "second disposable database in the same server"}),
		"workspace": run.write("web-workspace-revision.json", map[string]any{"acceptedR1": receipts.R1, "loadedSourceIndexRevision": sourceindex.ModelRevision(index1), "members": len(index1.Artifacts), "hostDecisionRef": e2eDecisionRef}),
		"targetLaunch": run.write("web-target-launch.json", map[string]any{"executable": "go test binary of tools/reference-first-web-target",
			"head": head, "children": []string{"zero-price", "grant-revoked", "positive", "read-dsn-missing", "s02-startup-refused", "s02-runtime"}, "positiveWebPid": webPID,
			"startedAt": started.Format(time.RFC3339), "finishedAt": time.Now().UTC().Format(time.RFC3339)}),
	}
	if healthRaw != nil {
		locator["fullHealth"] = run.write("qualification-full-health-ci-run.json", healthRaw)
	}
	packet := map[string]any{
		"schema": "tei.critical-path-evidence/v0",
		"claim":  "E2",
		"contract": map[string]any{
			"producerPath": "internal/sourceacceptance/hoststore/store.go", "consumerPath": "tools/reference-first-web-target/main.go",
			"positiveTest": "TestOneRevisionOrderEntryE2EPacket", "negativeTest": "TestOneRevisionOrderEntryE2EPacket (R0/R2/unit-price-zero/S05/N11/S02 controls)",
			"evidenceClass": "actual-observed",
		},
		"qualification": map[string]any{"head": head, "main": mainSHA, "fullHealth": fullHealth,
			"cleanBefore": cleanBefore, "cleanAfter": cleanAfter, "receipt": locator["fullHealth"]},
		"acceptance": map[string]any{
			"r0": receipts.R0, "r1": receipts.R1, "candidateFingerprint": receipts.CandidateFingerprint,
			"reviewFingerprint": receipts.ReviewFingerprint, "reviewCandidateFingerprint": receipts.ReviewCandidateFingerprint,
			"decisionReceipt": locator["decision"], "reloadReceipt": locator["reload"],
			"delta": []string{"add-validation", "add-application-validation-use"},
		},
		"web": map[string]any{
			"acceptedRevision": receipts.R1, "application": "order_entry", "target": "web",
			"selectedValidation": "sales.order_line.unit_price_positive", "negativeValidation": "sales.order_line.unit_price_positive",
			"negativeEvidenceMode": "running-server", "providerKind": "actual-db", "runMode": "running-server",
			"workspaceRevisionReceipt": locator["workspace"], "targetLaunchReceipt": locator["targetLaunch"],
			"positiveReceipt": locator["positive"], "negativeReceipt": locator["negative"], "unobservedControlReceipt": locator["n11"],
		},
		"physical": map[string]any{
			"acceptedRevision": receipts.R1, "datasource": "disposable PostgreSQL 16 (public.orders built from the R1 declaration)",
			"schemaCompatibility": "compatible", "schemaEvidence": locator["schema"],
			// D6: the effect Grant is synthetic. This value is a label, not authority.
			"hostAuthorityEvidence": "synthetic-effect-grant:test-only",
			"commandReceipt":        locator["command"], "independentReadReceipt": locator["read"], "recordId": "public.orders:order_id=1001",
			"schemaFailureControl": locator["s02"], "hostFailureControl": locator["s05"],
		},
	}
	packetPath := filepath.Join(out, "packet.json")
	packetRaw, err := json.MarshalIndent(packet, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(packetPath, packetRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"packet.json"} {
		raw, _ := os.ReadFile(filepath.Join(out, name))
		for _, secret := range e2eDSNs {
			if bytes.Contains(raw, []byte(secret)) {
				t.Fatal("DSN leaked into the packet")
			}
		}
	}

	// 12. The existing audit. It never returns a Product Gate.
	audit := exec.Command("go", "run", "./tools/reference-critical-path-evidence-audit", "-packet", packetPath, "-head", head, "-main", mainSHA)
	audit.Dir = repoRoot
	reportRaw, auditErr := audit.Output()
	var report struct {
		Status      string   `json:"status"`
		ProductGate string   `json:"productGate"`
		Diagnostics []string `json:"diagnostics"`
	}
	if err := json.Unmarshal(reportRaw, &report); err != nil {
		t.Fatalf("audit report: %v %v %s", err, auditErr, reportRaw)
	}
	run.write("audit-report.json", reportRaw)
	t.Logf("AUDIT status=%s productGate=%s diagnostics=%v", report.Status, report.ProductGate, report.Diagnostics)
	if report.ProductGate != "UNQUALIFIED" {
		t.Fatalf("the audit must not qualify a Product Gate: %s", report.ProductGate)
	}
	has := map[string]bool{}
	for _, d := range report.Diagnostics {
		has[d] = true
	}
	if healthRaw != nil {
		// With a matching health receipt nothing declarative is missing. The
		// audit still asks for human review and never grants a Product Gate.
		if len(report.Diagnostics) != 0 || report.Status != "REVIEW_REQUIRED" {
			t.Errorf("expected REVIEW_REQUIRED with no diagnostics: status=%s %v", report.Status, report.Diagnostics)
		}
	} else if len(report.Diagnostics) != 1 || !has["TRUSTED_FULL_HEALTH_NOT_RECORDED"] || report.Status != "INCOMPLETE" {
		// Only the trusted full health of an exact checkout remains.
		t.Errorf("expected only TRUSTED_FULL_HEALTH_NOT_RECORDED to remain: status=%s %v", report.Status, report.Diagnostics)
	}
	for _, filled := range []string{"S02_REAL_DATASOURCE_SCHEMA_READINESS_MISSING", "S02_NEGATIVE_CONTROL_MISSING",
		"ACCEPTED_R0_R1_LINEAGE_MISSING", "WEB_NOT_BOUND_TO_ACCEPTED_R1_RULE_SELECTION",
		"E2_COMMAND_AND_INDEPENDENT_READ_NOT_CORRELATED", "N11_UNOBSERVED_CONTROL_MISSING", "S05_NEGATIVE_CONTROL_MISSING",
		"E1_NEGATIVE_SELECTED_RULE_RUNNING_WEB_EVIDENCE_REQUIRED", "ACTUAL_WEB_POSITIVE_AND_NEGATIVE_RECEIPTS_REQUIRED"} {
		if has[filled] {
			t.Errorf("this run should have supplied evidence for %s", filled)
		}
	}
}

// e2eFullHealth returns NOT_RUN without a receipt. With one, it fails the
// test unless the receipt is the health workflow's successful push run for
// exactly this HEAD, and HEAD is origin/main.
func e2eFullHealth(t *testing.T, head, mainSHA string) (string, []byte) {
	t.Helper()
	path := os.Getenv(e2eHealthEnv)
	if path == "" {
		return "NOT_RUN", nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v", e2eHealthEnv, err)
	}
	var run struct {
		DatabaseID   int64  `json:"databaseId"`
		HeadSHA      string `json:"headSha"`
		Event        string `json:"event"`
		WorkflowName string `json:"workflowName"`
		Status       string `json:"status"`
		Conclusion   string `json:"conclusion"`
		URL          string `json:"url"`
	}
	if err := json.Unmarshal(raw, &run); err != nil {
		t.Fatalf("%s is not a run receipt: %v", e2eHealthEnv, err)
	}
	if head != mainSHA || run.HeadSHA != head || run.Event != "push" || run.WorkflowName != e2eHealthFlow ||
		run.Status != "completed" || run.Conclusion != "success" || run.DatabaseID == 0 ||
		run.URL != fmt.Sprintf("%s%d", e2eRunURLPrefix, run.DatabaseID) {
		t.Fatalf("health receipt does not match this exact checkout: head=%s main=%s run=%+v", head, mainSHA, run)
	}
	return "PASS", raw
}
