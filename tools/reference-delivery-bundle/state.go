package deliverybundle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	deliveryprofile "tei/tools/reference-delivery-profile"
)

const StateSchema = "tei.reference.delivery-state/v0"

const maxHistory = 20

type Deployment struct {
	Catalog string `json:"catalog"`
	Digest  string `json:"digest"`
}

type State struct {
	Schema   string       `json:"schema"`
	Active   *Deployment  `json:"active"`
	Previous *Deployment  `json:"previous,omitempty"`
	History  []Deployment `json:"history,omitempty"`
}

func catalogEntriesDigest(catalog deliveryprofile.Catalog) string {
	type item struct{ id, version, digest string }
	var items []item
	for id, versions := range catalog {
		for version, entry := range versions {
			items = append(items, item{id: id, version: version, digest: entry.Digest})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].id != items[j].id {
			return items[i].id < items[j].id
		}
		return items[i].version < items[j].version
	})
	hash := sha256.New()
	for _, item := range items {
		fmt.Fprintf(hash, "%s@%s\x00%s\n", item.id, item.version, item.digest)
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func verifyDeployment(deployment Deployment) error {
	if deployment.Catalog == "" || deployment.Digest == "" {
		return fmt.Errorf("deployment catalog and digest are required")
	}
	catalog, err := deliveryprofile.LoadCatalog(deployment.Catalog)
	if err != nil {
		return err
	}
	if actual := catalogEntriesDigest(catalog); actual != deployment.Digest {
		return fmt.Errorf("catalog digest mismatch: declared %s, actual %s", deployment.Digest, actual)
	}
	return nil
}

func readState(path string) (State, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return State{}, fmt.Errorf("%s: %w", path, err)
	}
	if state.Schema != StateSchema {
		return State{}, fmt.Errorf("state schema %q is not %q", state.Schema, StateSchema)
	}
	if state.Active == nil {
		return State{}, fmt.Errorf("state has no active deployment")
	}
	return state, nil
}

func writeState(path string, state State) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(absolute), ".tei-state-*")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer os.Remove(temp)
	if _, err := file.Write(append(raw, '\n')); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Chmod(temp, 0o600); err != nil {
		return err
	}
	return os.Rename(temp, absolute)
}

// Activate records a validated catalog as the active deployment. The previous
// active deployment is kept for one rollback step.
func Activate(catalogDir, statePath string) (State, error) {
	absolute, err := filepath.Abs(catalogDir)
	if err != nil {
		return State{}, err
	}
	catalog, err := deliveryprofile.LoadCatalog(absolute)
	if err != nil {
		return State{}, err
	}
	if len(catalog) == 0 {
		return State{}, fmt.Errorf("catalog is empty: %s", absolute)
	}
	next := State{
		Schema: StateSchema,
		Active: &Deployment{Catalog: absolute, Digest: catalogEntriesDigest(catalog)},
	}
	if previous, err := readState(statePath); err == nil {
		next.Previous = previous.Active
		next.History = append([]Deployment(nil), previous.History...)
		next.History = append(next.History, *previous.Active)
		if len(next.History) > maxHistory {
			next.History = next.History[len(next.History)-maxHistory:]
		}
	} else if !os.IsNotExist(err) {
		return State{}, err
	}
	if err := writeState(statePath, next); err != nil {
		return State{}, err
	}
	return next, nil
}

// Rollback restores the newest history entry after verifying its catalog. If
// history is empty, it falls back to the previous deployment for compatibility
// with state written before history existed. It does not start or stop a
// service.
func Rollback(statePath string) (State, error) {
	state, err := readState(statePath)
	if err != nil {
		return State{}, err
	}
	var target Deployment
	if len(state.History) > 0 {
		target = state.History[len(state.History)-1]
	} else if state.Previous != nil {
		target = *state.Previous
	} else {
		return State{}, fmt.Errorf("state has no previous deployment")
	}
	if err := verifyDeployment(target); err != nil {
		return State{}, fmt.Errorf("previous deployment is not usable: %w", err)
	}
	next := State{Schema: StateSchema, Active: &target, Previous: state.Active}
	if len(state.History) > 0 {
		next.History = append([]Deployment(nil), state.History[:len(state.History)-1]...)
	}
	if err := writeState(statePath, next); err != nil {
		return State{}, err
	}
	return next, nil
}

func Status(statePath string) (State, error) {
	state, err := readState(statePath)
	if err != nil {
		return State{}, err
	}
	if err := verifyDeployment(*state.Active); err != nil {
		return State{}, fmt.Errorf("active deployment is not usable: %w", err)
	}
	if state.Previous != nil {
		if err := verifyDeployment(*state.Previous); err != nil {
			return State{}, fmt.Errorf("previous deployment is not usable: %w", err)
		}
	}
	for index := range state.History {
		if err := verifyDeployment(state.History[index]); err != nil {
			return State{}, fmt.Errorf("history[%d] is not usable: %w", index, err)
		}
	}
	return state, nil
}
