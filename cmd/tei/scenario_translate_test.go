package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/scenariotest"
)

func TestScenarioTranslateFromMarkdownProducesValidScenario(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "driver-scenarios", "from-md", "order-entry-from-md.tei.md")
	workspace := filepath.Join("..", "..", "reference", "workspace")
	var out bytes.Buffer
	err := runScenarioCommand([]string{"translate", "-scenario-md", reference, "-scenario-id", "order-entry-from-md", "-workspace", workspace}, &out)
	if err != nil {
		t.Fatalf("translate error: %v\noutput: %s", err, out.String())
	}
	var got scenariotest.Scenario
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("translate output is not JSON Scenario: %v\n%s", err, out.String())
	}
	if got.ID != "order-entry-from-md" || got.Application != "order_entry" || len(got.Steps) != 3 {
		t.Fatalf("unexpected translated Scenario: %#v", got)
	}
	for _, id := range []string{"browse-pattern", "validate-positive", "observe-synthetic"} {
		found := false
		for _, step := range got.Steps {
			if step.ID == id {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing translated step %q", id)
		}
	}
}

func TestScenarioTranslatePreservesExecutableSemantics(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "driver-scenarios", "from-md", "order-entry-from-md.tei.md")
	workspace := filepath.Join("..", "..", "reference", "workspace")
	var translated bytes.Buffer
	err := runScenarioCommand([]string{"translate", "-scenario-md", reference, "-scenario-id", "order-entry-from-md", "-workspace", workspace}, &translated)
	if err != nil {
		t.Fatalf("translate error: %v\noutput: %s", err, translated.String())
	}
	var s scenariotest.Scenario
	if err := json.Unmarshal(translated.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	scenarioPath := filepath.Join(tmp, "scenario.json")
	if err := os.WriteFile(scenarioPath, b, 0600); err != nil {
		t.Fatal(err)
	}
	fixtureDir := filepath.Join("..", "..", "reference", "driver-scenarios", "from-md", "fixtures")
	if err := os.Mkdir(filepath.Join(tmp, "fixtures"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"order_valid.json", "order_state_pending.json"} {
		src, err := os.ReadFile(filepath.Join(fixtureDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmp, "fixtures", name), src, 0600); err != nil {
			t.Fatal(err)
		}
	}
	var out bytes.Buffer
	if err := runScenarioCommand([]string{"generate", "-scenario", scenarioPath, "-workspace", workspace}, &out); err != nil {
		t.Fatalf("translated Scenario was not executable: %v\n%s", err, out.String())
	}
	var plan scenariotest.Plan
	if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	wantIDs := []string{"browse-pattern", "validate-positive", "observe-synthetic"}
	if len(plan.Commands) != len(wantIDs) {
		t.Fatalf("plan command count mismatch: got=%d want=%d", len(plan.Commands), len(wantIDs))
	}
	for i, want := range wantIDs {
		if plan.Commands[i].StepID != want {
			t.Fatalf("plan step %d mismatch: got=%s want=%s", i, plan.Commands[i].StepID, want)
		}
	}
}

func TestScenarioTranslateFailsClosedOnInvalidMarkdown(t *testing.T) {
	md := filepath.Join("..", "..", "reference", "driver-scenarios", "fixtures", "recent_orders.json")
	var out bytes.Buffer
	err := runScenarioCommand([]string{"translate", "-scenario-md", md, "-scenario-id", "x", "-workspace", filepath.Join("..", "..", "reference", "workspace")}, &out)
	if err == nil {
		t.Fatalf("invalid markdown source was accepted: %s", out.String())
	}
}

func TestStoredFromMdScenarioMatchesMarkdownTranslation(t *testing.T) {
	markdown := filepath.Join("..", "..", "reference", "driver-scenarios", "from-md", "order-entry-from-md.tei.md")
	stored := filepath.Join("..", "..", "reference", "driver-scenarios", "from-md", "order-entry-from-md.json")
	workspace := filepath.Join("..", "..", "reference", "workspace")
	var translated bytes.Buffer
	if err := runScenarioCommand([]string{"translate", "-scenario-md", markdown, "-scenario-id", "order-entry-from-md", "-workspace", workspace}, &translated); err != nil {
		t.Fatal(err)
	}
	storedSource, err := os.ReadFile(stored)
	if err != nil {
		t.Fatal(err)
	}
	var fromMarkdown, fromStored scenariotest.Scenario
	if err := json.Unmarshal(translated.Bytes(), &fromMarkdown); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(storedSource, &fromStored); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromMarkdown, fromStored) {
		t.Fatalf("stored from-md JSON differs from Markdown translation\nmarkdown: %#v\nstored:   %#v", fromMarkdown, fromStored)
	}
}
