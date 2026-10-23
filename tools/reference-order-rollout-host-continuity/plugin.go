package orderrollouthost

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"tei/internal/deployment/executor"
	pruntime "tei/internal/plugin/runtime"
)

// Environment of the migration tool process. The host builds it explicitly
// (HostProcessEnvironmentProfile plus the injected secret); nothing is
// inherited from the host.
const (
	// The credential is injected by the host secret injector under the
	// target the environment definition declares.
	EnvCredential = "TEI_DATABASE_CREDENTIAL"
	// Opaque, non-secret runtime handle assigned by the host.
	EnvRuntimeHandle = "TEI_ATLAS_RUNTIME_HANDLE"
	// Milliseconds the tool keeps the expand transaction open after the
	// schema step, so concurrent legacy writes overlap the apply (P-MIG-03).
	EnvApplyHold = "TEI_ATLAS_APPLY_HOLD_MS"
	// The migration directory below the working directory (the mount).
	MigrationDir = "migrations"
)

// MigrationBundle is what the host places in the working directory: the
// migration of the target release, as read from the workspace in step 1.
type MigrationBundle struct {
	Release           string          `json:"release"`
	MigrationID       string          `json:"migrationId"`
	SchemaFingerprint string          `json:"schemaFingerprint"`
	Steps             []MigrationStep `json:"steps"`
}

// WriteMigrationBundle writes the bundle into dir/migrations/bundle.json.
func WriteMigrationBundle(dir string, s Start) error {
	b := MigrationBundle{Release: s.TargetRelease, MigrationID: s.MigrationID, SchemaFingerprint: s.TargetSchemaFP, Steps: s.Migration}
	raw, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, MigrationDir), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, MigrationDir, "bundle.json"), raw, 0o644)
}

// runAtlas is the stand-in executable of the migration.atlas deployment
// executor (there is no real one). It reads one process-plugin request from
// stdin, registers the execution in tei_atlas.job (lifecycle running,
// runtime token = the lease fencing token), waits until the job's gate is
// opened, applies the migration (schema, data, verify) in one transaction,
// records the release in tei_atlas.revision and answers "accepted". It never
// prints the credential or the connection target.
func runAtlas(stdin io.Reader, stdout, stderr io.Writer) int {
	fail := func(msg string) int { fmt.Fprintln(stderr, "atlas:", msg); return 1 }
	var req pruntime.Request
	if err := json.NewDecoder(stdin).Decode(&req); err != nil {
		return fail("request decode failed")
	}
	var cmd executor.Command
	if err := json.Unmarshal(req.Payload, &cmd); err != nil {
		return fail("command decode failed")
	}
	if req.ResourceLease == nil {
		return fail("no resource lease")
	}
	dsn, handle := os.Getenv(EnvCredential), os.Getenv(EnvRuntimeHandle)
	if dsn == "" || handle == "" {
		return fail("credential or runtime handle not injected")
	}
	raw, err := os.ReadFile(filepath.Join(MigrationDir, "bundle.json"))
	if err != nil {
		return fail("migration bundle not in the working directory")
	}
	var bundle MigrationBundle
	if err := json.Unmarshal(raw, &bundle); err != nil || bundle.Release != cmd.Version {
		return fail("migration bundle does not match the command")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fail("cannot connect to the target database") // no target or credential in the message
	}
	defer conn.Close(ctx)
	hb, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fail("cannot connect to the target database")
	}
	defer hb.Close(ctx)
	if _, err := conn.Exec(ctx, `INSERT INTO tei_atlas.job(runtime_handle,action_id,idempotency_key,lifecycle,runtime_token,os_pid,accepted_at) VALUES ($1,$2,$3,'running',$4,$5,clock_timestamp())`,
		handle, cmd.ActionID, cmd.IdempotencyKey, int64(req.ResourceLease.FencingToken), os.Getpid()); err != nil {
		// The idempotency key is unique: a second execution of the same
		// command is refused here, not applied twice.
		return fail("execution registration refused")
	}
	stopHB := make(chan struct{})
	hbDone := make(chan struct{})
	go func() {
		defer close(hbDone)
		t := time.NewTicker(150 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-stopHB:
				return
			case <-t.C:
				_, _ = hb.Exec(ctx, `UPDATE tei_atlas.job SET heartbeat_seq=heartbeat_seq+1 WHERE runtime_handle=$1`, handle)
			}
		}
	}()
	defer func() { close(stopHB); <-hbDone }()
	setState := func(lc, msg string) {
		_, _ = conn.Exec(ctx, `UPDATE tei_atlas.job SET lifecycle=$2, message=$3, finished_at=CASE WHEN $2 IN ('succeeded','failed') THEN clock_timestamp() END WHERE runtime_handle=$1`, handle, lc, msg)
	}
	for {
		var open bool
		if err := conn.QueryRow(ctx, `SELECT gate_open FROM tei_atlas.job WHERE runtime_handle=$1`, handle).Scan(&open); err != nil {
			return fail("execution registry lost")
		}
		if open {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if _, err := conn.Exec(ctx, `UPDATE tei_atlas.job SET lifecycle='applying', max_order_at_apply_start=(SELECT coalesce(max(order_id),0) FROM public.orders) WHERE runtime_handle=$1`, handle); err != nil {
		return fail("execution registry lost")
	}
	hold, _ := strconv.Atoi(os.Getenv(EnvApplyHold))
	applyErr := func() error {
		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		for _, st := range bundle.Steps {
			switch st.Kind {
			case "schema", "data":
				if _, err := tx.Exec(ctx, st.SQL); err != nil {
					return fmt.Errorf("%s step failed", st.Kind)
				}
				if st.Kind == "schema" && hold > 0 {
					time.Sleep(time.Duration(hold) * time.Millisecond)
				}
			case "verify":
				var n int64
				if err := tx.QueryRow(ctx, st.SQL).Scan(&n); err != nil || n != 0 {
					return fmt.Errorf("verify step did not return 0")
				}
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO tei_atlas.revision(release,migration_id,schema_fingerprint,runtime_handle) VALUES ($1,$2,$3,$4)`, bundle.Release, bundle.MigrationID, bundle.SchemaFingerprint, handle); err != nil {
			return fmt.Errorf("revision record failed")
		}
		return tx.Commit(ctx)
	}()
	if applyErr != nil {
		setState("failed", applyErr.Error())
		return fail(applyErr.Error())
	}
	_, _ = conn.Exec(ctx, `UPDATE tei_atlas.job SET max_order_at_apply_end=(SELECT coalesce(max(order_id),0) FROM public.orders) WHERE runtime_handle=$1`, handle)
	setState("succeeded", "migration "+bundle.MigrationID+" applied; release "+bundle.Release+" recorded")
	result := executor.Result{AttemptID: "attempt:" + handle, ActionID: cmd.ActionID, IdempotencyKey: cmd.IdempotencyKey, Outcome: executor.OutcomeAccepted,
		Message: "migration applied by the tool; completion requires observation", ExternalRef: handle}
	payload, _ := json.Marshal(result)
	resp := pruntime.Response{ProtocolVersion: req.ProtocolVersion, RequestID: req.RequestID, Plugin: req.Plugin, CapabilityID: req.CapabilityID,
		DescriptorFingerprint: req.DescriptorFingerprint, Status: pruntime.ResponseOK, Payload: payload}
	if err := json.NewEncoder(stdout).Encode(resp); err != nil {
		return 1
	}
	return 0
}
