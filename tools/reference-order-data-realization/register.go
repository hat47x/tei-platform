package orderdatarealization

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/semantic/capabilityinvoke"
)

const (
	applicationID    core.CanonicalID = "order_entry"
	browsePatternID  core.CanonicalID = "order_entry.recent_orders"
	commandPatternID core.CanonicalID = "order_entry.submit_order"
	detailPatternID  core.CanonicalID = "order_entry.order_lines"
	capabilityID     core.CanonicalID = "sales.order.submit"
	environmentID    core.CanonicalID = "development"
)

// listFrom is the lower bound passed to the list query; it is a parameter of
// the query, not a judgement on the wall clock.
var listFrom = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// OrderLine is one line of a submission.
type OrderLine struct {
	ID         int64  `json:"orderLineId"`
	LineNumber int64  `json:"lineNumber"`
	ItemCode   string `json:"itemCode"`
	Quantity   string `json:"quantity"`
	UnitPrice  string `json:"unitPrice"`
}

// Submission is a registration under one model revision.
type Submission struct {
	ModelRevision string      `json:"modelRevision"`
	OrderID       int64       `json:"orderId"`
	CustomerID    int64       `json:"customerId"`
	OrderedAt     time.Time   `json:"orderedAt"`
	Status        string      `json:"status"`
	Total         string      `json:"total"`
	ExternalID    *string     `json:"externalId"`
	DeliveryNote  *string     `json:"deliveryNote"`
	Lines         []OrderLine `json:"lines"`
}

type stageResult struct {
	Stage  string `json:"stage"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// preflight decides, before any adapter is called, whether the registration
// can run: the revision, the browse and command bindings, the environment
// binding and the physical schema against the expected state of the final
// version.
func (c *Chain) preflight(ctx context.Context, sub Submission) []stageResult {
	var out []stageResult
	add := func(stage string, err error) {
		if err != nil {
			out = append(out, stageResult{Stage: stage, Status: "unresolved", Detail: err.Error()})
		} else {
			out = append(out, stageResult{Stage: stage, Status: "resolved"})
		}
	}
	if sub.ModelRevision != c.final {
		add("model-revision", fmt.Errorf("submission revision %s differs from the final revision %s", sub.ModelRevision, c.final))
	} else {
		add("model-revision", nil)
	}
	resolver := driver.NewResolver(c.worldFinal)
	browse, err := resolver.ResolvePattern(applicationID, browsePatternID)
	if err == nil && (browse.Query == nil || browse.Datasource == nil) {
		err = fmt.Errorf("query or datasource unresolved")
	}
	add("browse-pattern", err)
	_, err = resolver.ResolveCommandBinding(applicationID, commandPatternID)
	add("command-pattern-binding", err)
	add("environment-database-binding", envBinding(c.worldFinal))
	diff, err := converged(ctx, c.db.db, c.db.instance, c.db.desiredFinal)
	if err == nil && (len(diff.Changes) > 0 || len(diff.Unresolved) > 0) {
		var parts []string
		for _, ch := range diff.Changes {
			name := ""
			if ch.Column != nil {
				name = "." + ch.Column.Name
			} else if ch.UniqueConstraint != nil {
				name = " " + ch.UniqueConstraint.Name
			}
			parts = append(parts, string(ch.Kind)+" "+ch.Table+name)
		}
		sort.Strings(parts)
		err = fmt.Errorf("physical schema differs from the expected state: %s", strings.Join(parts, "; "))
	}
	add("physical-schema", err)
	return out
}

func envBinding(world ir.World) error {
	for _, env := range world.Environments {
		if env.ID != environmentID {
			continue
		}
		for _, b := range env.Bindings {
			if b.Application == applicationID && b.Alias == "sales" {
				inst, ok := env.InstanceByID(b.Instance)
				if !ok {
					return fmt.Errorf("instance %s not defined", b.Instance)
				}
				if inst.Provider != environment.ProviderPostgres || inst.Definition != databaseID {
					return fmt.Errorf("instance %s is %s/%s", inst.ID, inst.Provider, inst.Definition)
				}
				return nil
			}
		}
		return fmt.Errorf("no binding order_entry.sales in %s", env.ID)
	}
	return fmt.Errorf("environment %s not found", environmentID)
}

func allResolved(s []stageResult) bool {
	for _, x := range s {
		if x.Status != "resolved" {
			return false
		}
	}
	return len(s) > 0
}

// insertHooks let the chain inject faults inside the adapter's transaction.
type insertHooks struct {
	onBegin      func(pid int64)
	beforeCommit func(ctx context.Context, tx *sql.Tx, pid int64) error
}

// orderAdapter is the hand-written custom-go adapter for sales.order.submit.
// It writes through the final version's storage mapping on its own pool.
type orderAdapter struct {
	db         *sql.DB
	w          rowWriter
	attr       string
	hooks      insertHooks
	mu         sync.Mutex
	dispatched int
	binding    string
}

var _ driver.BindingAdapterResolver = (*orderAdapter)(nil)

func (a *orderAdapter) ResolveAdapter(_ context.Context, b realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
	if b.Capability != capabilityID {
		return nil, fmt.Errorf("unexpected binding %s", b.Identity())
	}
	a.binding = b.Identity()
	return adapterFunc(a.invoke), nil
}

type adapterFunc func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error)

func (f adapterFunc) Invoke(ctx context.Context, id core.CanonicalID, in any) (capabilityinvoke.BusinessResult, error) {
	return f(ctx, id, in)
}

type commandPayload struct {
	OrderID    int64 `json:"orderId"`
	BackendPID int64 `json:"backendPid"`
}

func (a *orderAdapter) invoke(ctx context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	sub, ok := input.(Submission)
	if !ok {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("unexpected input %T", input)
	}
	a.mu.Lock()
	a.dispatched++
	a.mu.Unlock()
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	defer tx.Rollback()
	var pid int64
	if err := tx.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	if a.hooks.onBegin != nil {
		a.hooks.onBegin(pid)
	}
	values := map[string]any{
		"sales.order.order_id": sub.OrderID, "sales.order.customer_id": sub.CustomerID, "sales.order.ordered_at": sub.OrderedAt,
		"sales.order.status": sub.Status, "sales.order.total": sub.Total, "sales.order.external_id": ptrValue(sub.ExternalID),
		a.attr: ptrValue(sub.DeliveryNote),
	}
	q, args, err := insertFor(a.w.order, values, nil)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	if _, err := tx.ExecContext(ctx, q, args...); err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	for _, l := range sub.Lines {
		q, args, err := insertFor(a.w.line, map[string]any{
			"sales.order_line.order_line_id": l.ID, "sales.order_line.line_number": l.LineNumber, "sales.order_line.item_code": l.ItemCode,
			"sales.order_line.quantity": l.Quantity, "sales.order_line.unit_price": l.UnitPrice,
		}, map[string]any{"sales.order.order_id": sub.OrderID})
		if err != nil {
			return capabilityinvoke.BusinessResult{}, err
		}
		if _, err := tx.ExecContext(ctx, q, args...); err != nil {
			return capabilityinvoke.BusinessResult{}, err
		}
	}
	if a.hooks.beforeCommit != nil {
		if err := a.hooks.beforeCommit(ctx, tx, pid); err != nil {
			return capabilityinvoke.BusinessResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("commit: %w", err)
	}
	return capabilityinvoke.BusinessResult{Capability: capability, Mode: capabilityinvoke.ResultOutput, Payload: commandPayload{OrderID: sub.OrderID, BackendPID: pid}}, nil
}

func ptrValue(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

// Outcome kinds of one registration.
const (
	OutcomeRefused          = "refused-before-adapter"
	OutcomeCommitted        = "committed"
	OutcomeConfirmedFailure = "confirmed-failure"
	OutcomeUnknown          = "result-unknown"
	OutcomeCancelled        = "cancelled"
)

var errCancelled = errors.New("registration cancelled by the user before commit")

// classifyRegistration separates a failure the server confirmed (the
// transaction did not commit) from a result that is unknown (the connection
// was lost, so the commit may or may not have happened).
func classifyRegistration(err error) (string, string) {
	if err == nil {
		return OutcomeCommitted, ""
	}
	if errors.Is(err, errCancelled) {
		return OutcomeCancelled, ""
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		if strings.HasPrefix(pg.Code, "08") || strings.HasPrefix(pg.Code, "57P") {
			return OutcomeUnknown, pg.Code
		}
		return OutcomeConfirmedFailure, pg.Code
	}
	return OutcomeUnknown, ""
}

// Registration is the record of one attempt: the command result and the
// independent observation are separate pieces of evidence.
type Registration struct {
	Label       string        `json:"label"`
	OrderID     int64         `json:"orderId,omitempty"`
	ExternalID  *string       `json:"externalId"`
	Preflight   []stageResult `json:"preflight"`
	Outcome     string        `json:"outcome"`
	SQLState    string        `json:"sqlState,omitempty"`
	Error       string        `json:"error,omitempty"`
	AdapterCall int           `json:"adapterCalls"`
	CommandPID  int64         `json:"commandBackendPid,omitempty"`
	Binding     string        `json:"binding,omitempty"`
}

func (c *Chain) submit(ctx context.Context, label string, a *orderAdapter, sub Submission) Registration {
	r := Registration{Label: label, OrderID: sub.OrderID, ExternalID: sub.ExternalID}
	r.Preflight = c.preflight(ctx, sub)
	before := a.dispatched
	if !allResolved(r.Preflight) {
		r.Outcome = OutcomeRefused
		r.AdapterCall = a.dispatched - before
		return r
	}
	inv, err := driver.NewResolver(c.worldFinal).InvokeCapability(ctx, applicationID, capabilityID, a, sub)
	r.AdapterCall = a.dispatched - before
	r.Binding = a.binding
	r.Outcome, r.SQLState = classifyRegistration(err)
	r.Error = errString(err)
	if err == nil {
		if p, ok := inv.Result.Payload.(commandPayload); ok {
			r.CommandPID = p.BackendPID
		}
	}
	return r
}

// Observation statuses of the independent re-read.
const (
	ObsObserved     = "observed"
	ObsReadFailed   = "read-failed"
	ObsMismatch     = "mismatch"
	ObsNotCommitted = "command-not-committed"
)

func classifyObservation(commandCommitted bool, readErr error, matches bool) string {
	switch {
	case !commandCommitted:
		return ObsNotCommitted
	case readErr != nil:
		return ObsReadFailed
	case !matches:
		return ObsMismatch
	default:
		return ObsObserved
	}
}

// listOrders runs the list (browse) query through the driver on the given
// pool.
func (c *Chain) listOrders(ctx context.Context, db *sql.DB) (queryresult.MappedResult, error) {
	exec, err := driver.NewResolver(c.worldFinal).ExecuteBrowseQuery(ctx, applicationID, browsePatternID,
		postgresquery.Executor{DB: db, Database: databaseID}, map[string]any{"from": listFrom})
	if err != nil {
		return queryresult.MappedResult{}, err
	}
	return exec.Result.Mapped, nil
}

// reread observes a registration through the list query on a separate pool,
// independent of the command result.
func (c *Chain) reread(ctx context.Context, db *sql.DB, sub Submission) (string, int, error) {
	var mapped queryresult.MappedResult
	execution, err := driver.ExecuteObservationRead(ctx, string(browsePatternID), driver.ObservationEvidenceManagedHere, func(ctx context.Context) (any, error) {
		return c.listOrders(ctx, db)
	})
	if err != nil {
		return "", 0, err
	}
	mapped = execution.Result.(queryresult.MappedResult)
	row, ok := findRow(mapped, "sales.order.order_id", sub.OrderID)
	if !ok {
		return "", len(mapped.Rows), nil
	}
	want := map[string]any{
		"sales.order.external_id": ptrValue(sub.ExternalID), c.Decl.Attribute.Attribute: ptrValue(sub.DeliveryNote),
		"sales.order.status": sub.Status, "sales.order.total": sub.Total,
	}
	for attr, w := range want {
		got := cell(mapped, row, attr)
		if canonicalText(got) != canonicalText(w) {
			return fmt.Sprintf("%s: %v != %v", attr, got, w), len(mapped.Rows), nil
		}
	}
	return "match", len(mapped.Rows), nil
}

func findRow(m queryresult.MappedResult, attr string, id int64) (int, bool) {
	for i := range m.Rows {
		if v, ok := cell(m, i, attr).(int64); ok && v == id {
			return i, true
		}
	}
	return 0, false
}

func cell(m queryresult.MappedResult, row int, attr string) any {
	for j, col := range m.Columns {
		if string(col.Attribute) == attr {
			return m.Rows[row][j]
		}
	}
	return nil
}

// canonicalText renders a value for comparison; NULL and the empty string
// stay different.
func canonicalText(v any) string {
	switch x := v.(type) {
	case nil:
		return "<null>"
	case string:
		return "s:" + x
	case []byte:
		return "s:" + string(x)
	case time.Time:
		return "t:" + x.UTC().Format("2006-01-02T15:04:05.000000Z")
	default:
		return fmt.Sprintf("v:%v", x)
	}
}

// issuer is the external number source: an HTTP service that can be stopped.
type issuer struct {
	mu     sync.Mutex
	next   int
	srv    *httptest.Server
	client *http.Client
	issued []string
}

func newIssuer(first int) *issuer {
	i := &issuer{next: first, client: &http.Client{Timeout: 2 * time.Second}}
	i.start()
	return i
}

func (i *issuer) start() {
	i.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i.mu.Lock()
		n := i.next
		i.next++
		i.mu.Unlock()
		fmt.Fprintf(w, "%07d", n)
	}))
}

func (i *issuer) stop() { i.srv.Close() }

func (i *issuer) issue(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, i.srv.URL+"/issue", nil)
	if err != nil {
		return "", err
	}
	resp, err := i.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("issuer: %v status %d", err, resp.StatusCode)
	}
	i.mu.Lock()
	i.issued = append(i.issued, string(raw))
	i.mu.Unlock()
	return string(raw), nil
}

var ErrIssuerUnavailable = errors.New("external number issuer unavailable; registration stopped by the declared policy")

// externalNumber applies the declared policy when the issuer is unavailable.
func externalNumber(ctx context.Context, iss *issuer, policy, prefix string, orderID int64) (string, string, error) {
	n, err := iss.issue(ctx)
	if err == nil {
		return n, "issued", nil
	}
	switch policy {
	case PolicyProvisional:
		return fmt.Sprintf("%s%d", prefix, orderID), "provisional", nil
	default:
		return "", "", fmt.Errorf("%w: %v", ErrIssuerUnavailable, err)
	}
}

// numbering allocates order numbers from the sequence and records every event
// in the ledger, outside the business transaction, so that a rolled-back
// registration still leaves its reason.
type numbering struct {
	db       *sql.DB
	revision string
}

func (n numbering) allocate(ctx context.Context) (int64, error) {
	var id int64
	if err := n.db.QueryRowContext(ctx, `SELECT nextval('`+metaSchema+`.order_no_seq')`).Scan(&id); err != nil {
		return 0, err
	}
	return id, n.event(ctx, id, "allocated", "", nil)
}

func (n numbering) event(ctx context.Context, id int64, event, reason string, ext *string) error {
	var r any
	if reason != "" {
		r = reason
	}
	_, err := n.db.ExecContext(ctx, `INSERT INTO `+metaSchema+`.numbering_event (order_id, event, reason, external_id, model_revision) VALUES ($1, $2, $3, $4, $5)`, id, event, r, ptrValue(ext), n.revision)
	return err
}

// GapReport explains the order numbers that were allocated but not used.
type GapReport struct {
	Allocated      int              `json:"allocated"`
	Committed      int              `json:"committed"`
	Gaps           []int64          `json:"gaps"`
	Explained      map[int64]string `json:"explained"`
	Unexplained    []int64          `json:"unexplained"`
	SequenceLast   int64            `json:"sequenceLastValue"`
	UnrecordedUsed []int64          `json:"numbersWithoutAllocationRecord"`
}

func (n numbering) gaps(ctx context.Context) (GapReport, error) {
	g := GapReport{Explained: map[int64]string{}}
	if err := n.db.QueryRowContext(ctx, `SELECT last_value FROM `+metaSchema+`.order_no_seq`).Scan(&g.SequenceLast); err != nil {
		return g, err
	}
	rows, err := n.db.QueryContext(ctx, `
SELECT a.order_id,
       (SELECT count(*) FROM public.orders o WHERE o.order_id = a.order_id),
       (SELECT string_agg(e.reason, ',' ORDER BY e.seq) FROM `+metaSchema+`.numbering_event e WHERE e.order_id = a.order_id AND e.event = 'cancelled' AND e.reason IS NOT NULL)
FROM `+metaSchema+`.numbering_event a WHERE a.event = 'allocated' ORDER BY a.order_id`)
	if err != nil {
		return g, err
	}
	defer rows.Close()
	seen := map[int64]bool{}
	for rows.Next() {
		var id int64
		var present int
		var reason sql.NullString
		if err := rows.Scan(&id, &present, &reason); err != nil {
			return g, err
		}
		seen[id] = true
		g.Allocated++
		if present == 1 {
			g.Committed++
			continue
		}
		g.Gaps = append(g.Gaps, id)
		if reason.Valid && reason.String != "" {
			g.Explained[id] = reason.String
		} else {
			g.Unexplained = append(g.Unexplained, id)
		}
	}
	if err := rows.Err(); err != nil {
		return g, err
	}
	for id := int64(5001); id <= g.SequenceLast; id++ {
		if !seen[id] {
			g.UnrecordedUsed = append(g.UnrecordedUsed, id)
		}
	}
	return g, nil
}
