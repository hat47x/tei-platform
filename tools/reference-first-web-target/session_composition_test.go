//go:build !tei_repair_only

package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"tei/internal/composition"
	webcomposition "tei/reference/web-composition/go-host"
)

// This uses the real order_entry World/Driver/Canonical authorization and
// Action Runner, but synthetic session/CSRF and a synthetic artifact identity.
// A selected logical service is not an actual replaceable process artifact.
func TestSelectedCompositionRunsCanonicalProtectedOrderAction(t *testing.T) {
	projection, runner, read := sessionWebProjectionAndRunner(t)
	config := syntheticProtectedSessionConfiguration(runner)
	contract := composition.Contract{
		ID: "order_entry.submit_order", Version: "1.0.0", Shape: "reference-canonical-command",
	}
	registry := composition.NewRegistry()
	snapshot, err := registry.Apply(0, []composition.Provider{{
		ID: "order-command-v1", Scope: "reference/order_entry", Offers: contract,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := registry.Select("order-command-v1", "reference/order_entry")
	if err != nil {
		t.Fatal(err)
	}
	digest := "sha256:" + strings.Repeat("a", 64) // synthetic declaration, never byte attestation
	binding := composition.Binding{
		ContractID: contract.ID, ContractVersion: contract.Version,
		ProviderID: selected.ProviderID, ArtifactDigest: digest, Generation: 1,
	}
	currentPermission := true
	guardCalls := 0
	guard := func(actual composition.Binding) error {
		guardCalls++
		if !currentPermission || actual != binding {
			return errors.New("current host authority unavailable")
		}
		return nil
	}
	coordinator, err := composition.NewSelectedCoordinator(registry, selected, binding, composition.NewScope(), guard)
	if err != nil {
		t.Fatal(err)
	}
	config.SelectedAction = &selectedSessionActionConfig{
		Coordinator:     coordinator,
		ContractVersion: contract.Version,
		ProviderDigests: map[string]string{"order-command-v1": digest},
	}
	var records []webcomposition.AuditRecord
	config.Audit = func(record webcomposition.AuditRecord) { records = append(records, record) }
	server, err := newWebTargetServer(projection, read, runner, withProtectedSessionCommand(config))
	if err != nil {
		t.Fatal(err)
	}
	handler := server.Handler()

	// The Host must freeze the selected adapter rather than reading a mutable
	// map on each request. This is not a new provider resolver.
	config.SelectedAction.ProviderDigests["order-command-v1"] = "sha256:" + strings.Repeat("b", 64)
	for _, identity := range []string{"corporate-editor", "legacy-editor"} {
		t.Run(identity, func(t *testing.T) {
			beforeAudit, beforeGuard := len(records), guardCalls
			response := postSyntheticSession(handler, "/scenario/session-submit", identity, "matching",
				`{"from":"2026-09-18T08:30:00Z","row":0}`)
			if response.Code != http.StatusOK {
				t.Fatalf("canonical Action: status=%d body=%s", response.Code, response.Body.String())
			}
			var body webCommandScenarioResponse
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Result.Stage != "completed" || body.Result.Command == nil ||
				body.Result.Observation == nil || len(body.Result.Validations) != 3 ||
				body.Result.Authorization == nil || body.Result.Authorization.Outcome != "allowed" ||
				body.Result.EffectAuthorization == nil || body.Result.EffectAuthorization.Outcome != "allowed" {
				t.Fatalf("selected Action skipped canonical business path: %+v", body.Result)
			}
			if guardCalls != beforeGuard+1 || len(records) != beforeAudit+1 ||
				records[beforeAudit].Stage != "completed" {
				t.Fatalf("selected Action did not acquire one generation or audit: guard=%d records=%+v",
					guardCalls-beforeGuard, records)
			}
		})
	}
	t.Run("business authorization denies viewer before provider selection", func(t *testing.T) {
		beforeGuard := guardCalls
		response := postSyntheticSession(handler, "/scenario/session-submit",
			"corporate-viewer", "matching", `{"row":0}`)
		if response.Code != http.StatusForbidden || guardCalls != beforeGuard {
			t.Fatalf("business denial reached composition: status=%d guards=%d", response.Code, guardCalls-beforeGuard)
		}
	})
	t.Run("no authenticated session never selects provider", func(t *testing.T) {
		beforeGuard := guardCalls
		response := postSyntheticSession(handler, "/scenario/session-submit",
			"", "matching", `{"row":0}`)
		if response.Code != http.StatusUnauthorized || guardCalls != beforeGuard {
			t.Fatalf("unauthenticated request reached composition: status=%d guards=%d", response.Code, guardCalls-beforeGuard)
		}
	})
	t.Run("revoked host authority prevents the actual business Action", func(t *testing.T) {
		currentPermission = false
		defer func() { currentPermission = true }()
		beforeAudit := len(records)
		response := postSyntheticSession(handler, "/scenario/session-submit",
			"corporate-editor", "matching", `{"row":0}`)
		if response.Code != http.StatusServiceUnavailable || len(records) != beforeAudit+1 ||
			records[beforeAudit].Stage != "error" {
			t.Fatalf("revoked selection was not rejected and audited: status=%d records=%+v", response.Code, records)
		}
		if strings.Contains(response.Body.String(), "current host authority") {
			t.Fatal("host authority details leaked to browser")
		}
	})
	t.Run("provider withdrawn rejects selected canonical Action", func(t *testing.T) {
		if _, err := registry.Apply(snapshot.Revision, nil, []string{"order-command-v1"}); err != nil {
			t.Fatal(err)
		}
		beforeAudit := len(records)
		response := postSyntheticSession(handler, "/scenario/session-submit",
			"corporate-editor", "matching", `{"row":0}`)
		if response.Code != http.StatusServiceUnavailable || len(records) != beforeAudit+1 ||
			records[beforeAudit].Stage != "error" {
			t.Fatalf("withdrawn provider dispatched business action: status=%d records=%+v", response.Code, records)
		}
	})
}

func TestSelectedOrderActionRejectsIncompleteHostConfiguration(t *testing.T) {
	projection, runner, read := sessionWebProjectionAndRunner(t)
	for _, tc := range []struct {
		name     string
		selected *selectedSessionActionConfig
	}{
		{"missing coordinator", &selectedSessionActionConfig{
			ContractVersion: "1.0.0", ProviderDigests: map[string]string{"provider": "sha256:" + strings.Repeat("a", 64)},
		}},
		{"missing contract version", &selectedSessionActionConfig{
			ProviderDigests: map[string]string{"provider": "sha256:" + strings.Repeat("a", 64)},
		}},
		{"missing provider mapping", &selectedSessionActionConfig{ContractVersion: "1.0.0"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := syntheticProtectedSessionConfiguration(runner)
			config.SelectedAction = tc.selected
			if _, err := newWebTargetServer(projection, read, runner, withProtectedSessionCommand(config)); err == nil {
				t.Fatal("incomplete composition binding unexpectedly enabled a protected Action")
			}
		})
	}
}
