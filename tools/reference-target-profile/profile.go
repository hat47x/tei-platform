// Package targetprofile applies one declared Target Profile to the generator
// selection path. A profile describes execution constraints; it does not grant
// runtime authority or prove that a target enforces them.
package targetprofile

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const Schema = "tei.reference.target-profile/v0"

const (
	ProcessOneShot     = "oneshot"
	ProcessLongRunning = "long-running"
	ProcessBatch       = "batch"

	StorageNone       = "none"
	StorageFilesystem = "filesystem"
	StorageSQLite     = "sqlite"
	StorageRemote     = "remote"

	NetworkNone     = "none"
	NetworkLoopback = "loopback"
	NetworkAllowed  = "allowed"

	UpdateReplace = "replace"
	UpdateInPlace = "in-place"
	UpdateRolling = "rolling"
)

type Profile struct {
	Schema           string   `json:"schema"`
	ID               string   `json:"id"`
	Version          string   `json:"version"`
	Description      string   `json:"description,omitempty"`
	ProcessModel     string   `json:"processModel"`
	Storage          string   `json:"storage"`
	Network          string   `json:"network"`
	Update           string   `json:"update"`
	MemoryLimitBytes int64    `json:"memoryLimitBytes"`
	Observability    []string `json:"observability"`
	Unsupported      []string `json:"unsupported,omitempty"`
	GeneratorID      string   `json:"generatorId,omitempty"`
}

func Load(path string) (Profile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var profile Profile
	if err := decoder.Decode(&profile); err != nil {
		return Profile{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := Validate(profile); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

func Validate(profile Profile) error {
	if profile.Schema != Schema {
		return fmt.Errorf("profile schema %q is not %q", profile.Schema, Schema)
	}
	if strings.TrimSpace(profile.ID) == "" || strings.TrimSpace(profile.Version) == "" {
		return fmt.Errorf("profile id and version are required")
	}
	switch profile.ProcessModel {
	case ProcessOneShot, ProcessLongRunning, ProcessBatch:
	default:
		return fmt.Errorf("unknown process model %q", profile.ProcessModel)
	}
	switch profile.Storage {
	case StorageNone, StorageFilesystem, StorageSQLite, StorageRemote:
	default:
		return fmt.Errorf("unknown storage %q", profile.Storage)
	}
	switch profile.Network {
	case NetworkNone, NetworkLoopback, NetworkAllowed:
	default:
		return fmt.Errorf("unknown network mode %q", profile.Network)
	}
	switch profile.Update {
	case UpdateReplace, UpdateInPlace, UpdateRolling:
	default:
		return fmt.Errorf("unknown update mode %q", profile.Update)
	}
	if profile.MemoryLimitBytes <= 0 {
		return fmt.Errorf("memoryLimitBytes must be positive")
	}
	if len(profile.Observability) == 0 {
		return fmt.Errorf("observability must not be empty")
	}
	return nil
}
