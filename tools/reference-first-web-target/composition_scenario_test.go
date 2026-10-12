//go:build !tei_repair_only

package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/workspace"
)

func TestCompositionScenariosKeepDriverResultsAndBusinessDenials(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	read := newReferenceReadScenarioRunner(loaded.World)
	command, err := newReferenceCommandScenarioRunner(loaded.World, legacyFixturePath(t, "users.json"), legacyFixturePath(t, "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	digest, err := runningExecutableDigest()
	if err != nil {
		t.Fatal(err)
	}
	registry := composition.NewRegistry()
	var denied atomic.Bool
	guard := func(b composition.Binding) error {
		if denied.Load() || b.ArtifactDigest != digest {
			return errors.New("host denied")
		}
		return nil
	}
	cr, cc, err := wireComposedScenarios(context.Background(), registry, read, command, digest, guard)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cc.service.Withdraw(1)
		_ = cc.service.Reap(context.Background())
		_ = cr.service.Withdraw(1)
		_ = cr.service.Reap(context.Background())
	})
	input := readScenarioInput{From: "2026-09-18T08:30:00Z", RowIndex: 0}
	expected, err := read.Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := cr.Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if actual.Composition == nil || actual.Composition.ArtifactDigest != digest {
		t.Fatal("missing executable provenance")
	}
	actual.Composition = nil
	if !reflect.DeepEqual(actual, expected) {
		t.Fatal("composed read changed business results")
	}
	for _, credential := range []string{"editor-secret", "incorrect"} {
		in := commandScenarioInput{LoginID: "legacy-order-editor", Credential: credential, From: input.From, RowIndex: 0}
		want, err := command.RunCommand(context.Background(), in)
		if err != nil {
			t.Fatal(err)
		}
		got, err := cc.RunCommand(context.Background(), in)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(got)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), credential) {
			t.Fatal("credential leaked into result")
		}
		if got.Composition == nil || got.Composition.ProviderID != "order-submit" {
			t.Fatal("submit bypassed composition")
		}
		got.Composition = nil
		if got.Read != nil {
			got.Read.Composition = nil
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("changed command semantics: %s", got.Stage)
		}
	}
	projection, err := buildWebTargetProjection(loaded.World, projectionConfig{HandoverBaselineID: "B", BaselineRepositoryRevision: "reference-base", TargetBuildRevision: "reference-target", ApplicationID: core.MustCanonicalID("order_entry")})
	if err != nil {
		t.Fatal(err)
	}
	server, err := newWebTargetServer(projection, cr, cc)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest("GET", "/scenario/read?from=2026-09-18T08%3A30%3A00Z&row=0", nil))
	if response.Code != 200 || !strings.Contains(response.Body.String(), "order-read") {
		t.Fatalf("HTTP consumer: %d %s", response.Code, response.Body.String())
	}
	denied.Store(true)
	if _, err := cr.Run(context.Background(), input); err == nil {
		t.Fatal("host denial bypassed")
	}
	if _, err := cc.RunCommand(context.Background(), commandScenarioInput{}); err == nil {
		t.Fatal("host denial bypassed by command")
	}
	denied.Store(false)
	if _, err := registry.Apply(1, nil, []string{"order-read"}); err != nil {
		t.Fatal(err)
	}
	if _, err := cr.Run(context.Background(), input); err == nil {
		t.Fatal("removed read invoked")
	}
	if _, err := cc.RunCommand(context.Background(), commandScenarioInput{}); err == nil {
		t.Fatal("submit ran without read coeffect")
	}
}
