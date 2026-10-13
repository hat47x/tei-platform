//go:build !tei_repair_only

package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"sync"

	"tei/internal/composition"
	renderer "tei/plugins/report-renderer"
)

var reportContract = scenarioContract("tei.reference.order-entry.report", "read selection -> file receipt; selected renderer; exclusive host storage; current database grant")
var batchContract = scenarioContract("tei.reference.order-entry.batch", "bounded frozen selections -> per-item command and observation; stop on first failure; no automatic retry; independent transactions")

type nonScreenInput struct {
	From       string
	Rows       []int
	LoginID    string
	Credential string
	Journal    batchJournal
}
type reportReceipt struct {
	Path        string            `json:"path"`
	Digest      string            `json:"digest"`
	MediaType   string            `json:"mediaType"`
	Rows        int               `json:"rows"`
	Renderer    *compositionTrace `json:"renderer"`
	Composition *compositionTrace `json:"composition"`
	Limits      renderer.Limits   `json:"limits"`
}
type batchItem struct {
	OrderID int64                 `json:"orderId"`
	Result  commandScenarioResult `json:"result"`
	Error   string                `json:"error,omitempty"`
}
type batchReceipt struct {
	Stage       string            `json:"stage"`
	Planned     []int64           `json:"planned"`
	Items       []batchItem       `json:"items"`
	Composition *compositionTrace `json:"composition"`
}
type nonScreenServices struct {
	report    *composition.Service[nonScreenInput, reportReceipt]
	batch     *composition.Service[nonScreenInput, batchReceipt]
	render    *composition.Service[[]renderer.Row, renderer.Document]
	closeOnce sync.Once
	closeErr  error
}

func (s *nonScreenServices) Close() error {
	s.closeOnce.Do(func() {
		var failures []error
		if s.report != nil {
			if err := s.report.Shutdown(context.Background(), 1); err != nil {
				failures = append(failures, fmt.Errorf("report service cleanup: %w", err))
			}
		}
		if s.batch != nil {
			if err := s.batch.Shutdown(context.Background(), 1); err != nil {
				failures = append(failures, fmt.Errorf("batch service cleanup: %w", err))
			}
		}
		if s.render != nil {
			if err := s.render.Shutdown(context.Background(), 1); err != nil {
				failures = append(failures, fmt.Errorf("renderer service cleanup: %w", err))
			}
		}
		s.closeErr = errors.Join(failures...)
	})
	return s.closeErr
}

type frozenRead struct{ value readScenarioResult }

func (*frozenRead) ApplicationID() string { return "order_entry" }
func (r *frozenRead) Run(ctx context.Context, _ readScenarioInput) (readScenarioResult, error) {
	return r.value, ctx.Err()
}

// Freeze identities and validation inputs before any effect. Each database
// transaction still checks the captured version and line digest independently.
func nonScreenPlan(ctx context.Context, read readScenarioRunner, input nonScreenInput) ([]readScenarioResult, []int64, error) {
	if len(input.Rows) == 0 || len(input.Rows) > 100 {
		return nil, nil, fmt.Errorf("select between 1 and 100 rows")
	}
	var snapshots []readScenarioResult
	var ids []int64
	seen := map[int64]bool{}
	for _, row := range input.Rows {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		if row < 0 {
			return nil, nil, fmt.Errorf("negative selected row")
		}
		r, err := read.Run(ctx, readScenarioInput{From: input.From, RowIndex: row})
		if err != nil {
			return nil, nil, err
		}
		v, ok := semanticRowValueByAttribute(r.SelectedOrder, "sales.order.order_id")
		id, typed := v.(int64)
		if !ok || !typed || seen[id] {
			return nil, nil, fmt.Errorf("missing or duplicate selected order identity")
		}
		seen[id] = true
		snapshots = append(snapshots, r)
		ids = append(ids, id)
	}
	return snapshots, ids, nil
}

// Physical destination selection is host-owned. Existing files are never
// replaced, including symlink aliases. This is a reference checkpoint, not an
// atomic transaction spanning database revocation and the filesystem.
func publishReport(ctx context.Context, root, path string, doc renderer.Document) (reportReceipt, error) {
	if err := ctx.Err(); err != nil {
		return reportReceipt{}, err
	}
	path, err := artifactOutsideModel(root, path)
	if err != nil {
		return reportReceipt{}, err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return reportReceipt{}, fmt.Errorf("cannot exclusively create report")
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	if _, err := f.Write(doc.Bytes); err != nil {
		return reportReceipt{}, fmt.Errorf("report write failed")
	}
	if err := ctx.Err(); err != nil {
		return reportReceipt{}, err
	}
	if err := f.Sync(); err != nil {
		return reportReceipt{}, fmt.Errorf("report sync failed")
	}
	if err := f.Close(); err != nil {
		return reportReceipt{}, fmt.Errorf("report close failed")
	}
	ok = true
	return reportReceipt{Path: path, Digest: fmt.Sprintf("sha256:%x", sha256.Sum256(doc.Bytes)), MediaType: doc.MediaType}, nil
}

func wireNonScreen(ctx context.Context, registry *composition.Registry, read readScenarioRunner, command *composedCommandScenario, state *postgresScenarioState, workspace, destination, format string, progress func(batchItem), limits ...renderer.Limits) (*nonScreenServices, error) {
	if state == nil || command == nil || command.service == nil || registry == nil || read == nil {
		return nil, fmt.Errorf("non-screen reference requires PostgreSQL and composed read/submit")
	}
	budget := renderer.DefaultLimits()
	if len(limits) > 1 {
		return nil, fmt.Errorf("one report budget is required")
	}
	if len(limits) == 1 {
		budget = limits[0]
	}
	provider, render, err := renderer.SelectBounded(format, budget)
	if err != nil {
		return nil, err
	}
	scope := "reference/order_entry/development"
	_, err = registry.Apply(registry.Snapshot().Revision, []composition.Provider{
		{ID: provider, Scope: scope, Offers: renderer.Contract},
		{ID: "order-report", Scope: scope, Offers: reportContract, Requires: []composition.Dependency{{Contract: readServiceContract, ProviderID: "order-read"}, {Contract: renderer.Contract, ProviderID: provider}}},
		{ID: "order-batch", Scope: scope, Offers: batchContract, Requires: []composition.Dependency{{Contract: readServiceContract, ProviderID: "order-read"}, {Contract: postgresSubmitServiceContract, ProviderID: "order-submit"}}},
	}, nil)
	if err != nil {
		return nil, err
	}
	s := &nonScreenServices{}
	lifecycle := composition.Lifecycle{Prepare: func(context.Context, *composition.Scope) error { return nil }, Ready: func(context.Context, composition.Binding) error { return nil }}
	guard := func(b composition.Binding) error {
		contracts := map[string]composition.Contract{provider: renderer.Contract, "order-report": reportContract, "order-batch": batchContract}
		c, ok := contracts[b.ProviderID]
		if !ok || b.ContractID != c.ID || b.ContractVersion != c.Version || b.ArtifactDigest != state.digest || b.Generation != 1 {
			return fmt.Errorf("host rejects non-screen instance")
		}
		return nil
	}
	selection, err := registry.Select(provider, scope)
	if err != nil {
		return nil, err
	}
	binding := func(id string, c composition.Contract) composition.Binding {
		return composition.Binding{ContractID: c.ID, ContractVersion: c.Version, ProviderID: id, ArtifactDigest: state.digest, Generation: 1}
	}
	s.render, err = composition.NewService(ctx, registry, selection, binding(provider, renderer.Contract), guard, render, lifecycle)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*nonScreenServices, error) { return nil, errors.Join(err, s.Close()) }
	selection, err = registry.Select("order-report", scope)
	if err != nil {
		return fail(err)
	}
	s.report, err = composition.NewService(ctx, registry, selection, binding("order-report", reportContract), guard, func(ctx context.Context, input nonScreenInput) (reportReceipt, error) {
		if err := state.checkGrant(ctx, state.db, false); err != nil {
			return reportReceipt{}, err
		}
		snapshots, ids, err := nonScreenPlan(ctx, read, input)
		if err != nil {
			return reportReceipt{}, err
		}
		var rows []renderer.Row
		for i, snapshot := range snapshots {
			for _, line := range snapshot.OrderLineRows {
				values := make([]string, 4)
				for j, attribute := range []string{"line_number", "item_code", "quantity", "unit_price"} {
					v, ok := semanticRowValueByAttribute(line, "sales.order_line."+attribute)
					if !ok || v == nil {
						return reportReceipt{}, fmt.Errorf("missing report attribute")
					}
					values[j] = fmt.Sprint(v)
				}
				rows = append(rows, renderer.Row{OrderID: fmt.Sprint(ids[i]), LineNumber: values[0], ItemCode: values[1], Quantity: values[2], UnitPrice: values[3]})
			}
		}
		doc, renderBinding, err := s.render.Invoke(ctx, rows)
		if err != nil {
			return reportReceipt{}, err
		}
		if err := state.checkGrant(ctx, state.db, false); err != nil {
			return reportReceipt{}, err
		}
		receipt, err := publishReport(ctx, workspace, destination, doc)
		receipt.Rows = len(rows)
		receipt.Limits = budget
		receipt.Renderer = projectCompositionBinding(renderBinding)
		return receipt, err
	}, lifecycle)
	if err != nil {
		return fail(err)
	}
	selection, err = registry.Select("order-batch", scope)
	if err != nil {
		return fail(err)
	}
	s.batch, err = composition.NewService(ctx, registry, selection, binding("order-batch", batchContract), guard, func(ctx context.Context, input nonScreenInput) (batchReceipt, error) {
		result := batchReceipt{Stage: "planning", Items: []batchItem{}}
		if err := state.checkGrant(ctx, state.db, false); err != nil {
			return result, err
		}
		snapshots, ids, err := nonScreenPlan(ctx, read, input)
		result.Planned = ids
		if err != nil {
			return result, err
		}
		invocation, _ := composition.InvocationBinding(ctx)
		plannedBinding := projectCompositionBinding(invocation)
		if state.model != nil {
			plannedBinding.ModelRevision = state.model.revision
		}
		if err := appendBatchJournal(input.Journal, batchJournalEvent{Kind: "planned", Planned: ids, Composition: plannedBinding}); err != nil {
			result.Stage = "journal-failed"
			return result, err
		}
		for i, snapshot := range snapshots {
			if err := ctx.Err(); err != nil {
				result.Stage = "cancelled"
				return result, err
			}
			if err := appendBatchJournal(input.Journal, batchJournalEvent{Kind: "starting", OrderID: ids[i]}); err != nil {
				result.Stage = "journal-failed"
				return result, err
			}
			r, err := command.RunCommand(ctx, commandScenarioInput{LoginID: input.LoginID, Credential: input.Credential, From: input.From, RowIndex: input.Rows[i], plannedRead: &snapshot})
			item := batchItem{OrderID: ids[i], Result: r}
			if err != nil {
				item.Error = "item execution failed; inspect command and observation receipts before retry"
			}
			result.Items = append(result.Items, item)
			if journalErr := appendBatchJournal(input.Journal, batchJournalEvent{Kind: "result", Item: &item}); journalErr != nil {
				result.Stage = "journal-failed"
				return result, journalErr
			}
			if progress != nil {
				progress(item)
			}
			if err != nil {
				result.Stage = "execution-failed"
				return result, err
			}
			if r.Stage != "completed" {
				result.Stage = "business-stopped"
				return result, nil
			}
		}
		result.Stage = "completed"
		return result, nil
	}, lifecycle)
	if err != nil {
		return fail(err)
	}
	return s, nil
}
