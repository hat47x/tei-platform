//go:build !tei_repair_only

package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"tei/internal/composition"
	"tei/internal/driver"
)

var batchObservationContract = scenarioContract("tei.reference.order-entry.batch-observe", "bounded explicit order identities -> current external states in one read-only snapshot; current host grant; no command or resume decision")

type batchStateObservation struct {
	OrderID     int64                 `json:"orderId"`
	Observation observationProjection `json:"observation"`
}

type batchObservationReceipt struct {
	Items           []batchStateObservation `json:"items"`
	Composition     *compositionTrace       `json:"composition"`
	AuthorityLimits []string                `json:"authorityLimits"`
}

func validateObservationIDs(ids []int64) error {
	if len(ids) == 0 || len(ids) > 100 {
		return fmt.Errorf("select between 1 and 100 order identities")
	}
	seen := map[int64]bool{}
	for _, id := range ids {
		if id <= 0 || seen[id] {
			return fmt.Errorf("order identities must be positive and unique")
		}
		seen[id] = true
	}
	return nil
}

func parseObservationIDs(raw string) ([]int64, error) {
	parts := strings.Split(raw, ",")
	if len(parts) > 100 {
		return nil, fmt.Errorf("select at most 100 order identities")
	}
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid order identity")
		}
		ids = append(ids, id)
	}
	return ids, validateObservationIDs(ids)
}

// This observes present state only. Neither a submitted state nor a caller's
// receipt proves which operation caused it, or authorizes a resumed command.
func (s *postgresScenarioState) observeBatch(ctx context.Context, ids []int64) (batchObservationReceipt, error) {
	if err := validateObservationIDs(ids); err != nil {
		return batchObservationReceipt{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return batchObservationReceipt{}, fmt.Errorf("cannot start PostgreSQL batch observation")
	}
	defer tx.Rollback()
	if err := s.checkGrant(ctx, tx, false); err != nil {
		return batchObservationReceipt{}, err
	}
	result := batchObservationReceipt{Items: []batchStateObservation{}, AuthorityLimits: []string{
		"diagnostic host observation; no business-user authorization",
		"current state does not identify the operation that caused it",
		"no durable request receipt, automatic retry or resume authorization",
	}}
	for _, id := range ids {
		observation, err := driver.ExecuteObservationRead(ctx, "postgres.order_state.read", driver.ObservationEvidenceExternalObserved, func(ctx context.Context) (any, error) {
			var status string
			if err := tx.QueryRowContext(ctx, "SELECT status FROM public.orders WHERE order_id = $1", id).Scan(&status); err != nil {
				return nil, fmt.Errorf("PostgreSQL batch observation failed")
			}
			return status, nil
		})
		if err != nil {
			return batchObservationReceipt{}, err
		}
		result.Items = append(result.Items, batchStateObservation{OrderID: id, Observation: observationProjection{
			ReadIdentity: observation.ReadIdentity, Nature: string(observation.Nature), Result: observation.Result,
		}})
	}
	if err := tx.Commit(); err != nil {
		return batchObservationReceipt{}, fmt.Errorf("PostgreSQL batch observation snapshot failed")
	}
	// Recheck outside the snapshot so a revocation or Source change seen after
	// reading suppresses output. This is not atomic publication with revocation.
	if err := s.checkGrant(ctx, s.db, false); err != nil {
		return batchObservationReceipt{}, err
	}
	return result, nil
}

func wireBatchObservation(ctx context.Context, registry *composition.Registry, state *postgresScenarioState) (*composition.Service[[]int64, batchObservationReceipt], error) {
	if registry == nil || state == nil {
		return nil, fmt.Errorf("batch observation requires PostgreSQL and a host registry")
	}
	const provider = "order-batch-observe"
	const scope = "reference/order_entry/development"
	_, err := registry.Apply(registry.Snapshot().Revision, []composition.Provider{{ID: provider, Scope: scope, Offers: batchObservationContract}}, nil)
	if err != nil {
		return nil, err
	}
	selection, err := registry.Select(provider, scope)
	if err != nil {
		return nil, err
	}
	binding := composition.Binding{ContractID: batchObservationContract.ID, ContractVersion: batchObservationContract.Version,
		ProviderID: provider, ArtifactDigest: state.digest, Generation: 1}
	return composition.NewService(ctx, registry, selection, binding,
		referenceInstanceGuard(state.digest, map[string]composition.Contract{provider: batchObservationContract}), state.observeBatch,
		composition.Lifecycle{Prepare: func(context.Context, *composition.Scope) error { return nil }, Ready: func(context.Context, composition.Binding) error { return nil }})
}
