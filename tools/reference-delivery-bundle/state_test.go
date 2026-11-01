package deliverybundle

import (
	"os"
	"path/filepath"
	"testing"
)

func catalogWith(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	source := filepath.Join(repoRoot(t), "tools", "reference-delivery-profile", "testdata", "plugins")
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(source, name+".json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name+".json"), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestActivateAndRollback(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	first := catalogWith(t, "audit")
	second := catalogWith(t, "audit", "authentication")
	activated, err := Activate(first, statePath)
	if err != nil {
		t.Fatal(err)
	}
	if activated.Active == nil || activated.Previous != nil {
		t.Fatalf("unexpected first state: %+v", activated)
	}
	activated, err = Activate(second, statePath)
	if err != nil {
		t.Fatal(err)
	}
	if activated.Active.Digest == activated.Previous.Digest {
		t.Fatal("activation did not record a distinct previous deployment")
	}
	rolledBack, err := Rollback(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if rolledBack.Active.Digest != activated.Previous.Digest {
		t.Fatalf("rollback did not restore the previous deployment: %+v", rolledBack)
	}
	if _, err := Status(statePath); err != nil {
		t.Fatal(err)
	}
}

func TestActivateRejectsEmptyOrInvalidCatalog(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	if _, err := Activate(t.TempDir(), statePath); err == nil {
		t.Fatal("empty catalog was activated")
	}
	bad := t.TempDir()
	if err := os.WriteFile(filepath.Join(bad, "bad.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Activate(bad, statePath); err == nil {
		t.Fatal("invalid catalog was activated")
	}
}

func TestRollbackWithoutPreviousIsRejected(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	if _, err := Activate(catalogWith(t, "audit"), statePath); err != nil {
		t.Fatal(err)
	}
	if _, err := Rollback(statePath); err == nil {
		t.Fatal("rollback without a previous deployment was accepted")
	}
}

func TestStatusRejectsChangedCatalog(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	catalog := catalogWith(t, "audit")
	if _, err := Activate(catalog, statePath); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(catalog, "audit.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n', '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(statePath); err == nil {
		t.Fatal("status accepted a changed catalog")
	}
}

func TestMultiStepRollbackUsesHistory(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	first := catalogWith(t, "audit")
	second := catalogWith(t, "audit", "authentication")
	third := catalogWith(t, "audit", "authentication", "offline-storage")
	firstState, err := Activate(first, statePath)
	if err != nil {
		t.Fatal(err)
	}
	secondState, err := Activate(second, statePath)
	if err != nil {
		t.Fatal(err)
	}
	thirdState, err := Activate(third, statePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(thirdState.History) != 2 {
		t.Fatalf("history=%+v", thirdState.History)
	}
	rolledBack, err := Rollback(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if rolledBack.Active.Digest != secondState.Active.Digest {
		t.Fatalf("first rollback did not restore second: %+v", rolledBack)
	}
	rolledBack, err = Rollback(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if rolledBack.Active.Digest != firstState.Active.Digest {
		t.Fatalf("second rollback did not restore first: %+v", rolledBack)
	}
}

func TestHistoryIsCapped(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	catalog := catalogWith(t, "audit")
	var state State
	var err error
	for index := 0; index < maxHistory+1; index++ {
		state, err = Activate(catalog, statePath)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(state.History) != maxHistory {
		t.Fatalf("history=%d want %d", len(state.History), maxHistory)
	}
}

func TestStatusRejectsChangedHistoryCatalog(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	first := catalogWith(t, "audit")
	second := catalogWith(t, "audit", "authentication")
	third := catalogWith(t, "audit", "authentication", "offline-storage")
	if _, err := Activate(first, statePath); err != nil {
		t.Fatal(err)
	}
	if _, err := Activate(second, statePath); err != nil {
		t.Fatal(err)
	}
	if _, err := Activate(third, statePath); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(first, "audit.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(statePath); err == nil {
		t.Fatal("status accepted a changed history catalog")
	}
}
