package dispatch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const Schema = "tei.reference.external-role-dispatcher/v0"

// Model represents an executable model/harness binding.
type Model struct {
	ID            string   `json:"id"`
	Harness       string   `json:"harness"`
	Roles         []string `json:"roles"`
	MaxConcurrent int      `json:"maxConcurrent"`
	Enabled       bool     `json:"enabled"`
	Priority      int      `json:"priority"` // lower = preferred
	CostTier      string   `json:"costTier,omitempty"`
}

// RoleConfig maps a logical role to eligible model IDs.
type RoleConfig struct {
	Name     string   `json:"name"`
	ModelIDs []string `json:"modelIds"`
}

// GitHubConfig holds GitHub adapter knobs.
type GitHubConfig struct {
	Repo              string   `json:"repo"`
	AllowedRepos      []string `json:"allowedRepos"`
	RouteLabelPrefix  string   `json:"routeLabelPrefix"`
	StateLabelPrefix  string   `json:"stateLabelPrefix"`
	ReviewLabelPrefix string   `json:"reviewLabelPrefix"`
	ReadyState        string   `json:"readyState"`
	WorkingState      string   `json:"workingState"`
	ReviewState       string   `json:"reviewState"`
	BlockedState      string   `json:"blockedState"`
}

// HarnessConfig holds harness adapter knobs.
type HarnessConfig struct {
	DefaultHarness string    `json:"defaultHarness"`
	DSH            DSHConfig `json:"dsh"`
}

// DSHConfig holds DSH-specific adapter knobs.
type DSHConfig struct {
	Profile       string `json:"profile"`
	CLI           string `json:"cli"`
	DefaultDryRun bool   `json:"defaultDryRun"`
	TaskTimeout   string `json:"taskTimeout"`
}

// LeaseConfig holds local lease store knobs.
type LeaseConfig struct {
	DefaultTTL        string `json:"defaultTTL"`
	HeartbeatInterval string `json:"heartbeatInterval"`
	SweepInterval     string `json:"sweepInterval"`
	DBPath            string `json:"dbPath"`
}

// Config is the top-level dispatcher configuration.
type Config struct {
	Schema  string        `json:"schema"`
	Models  []Model       `json:"models"`
	Roles   []RoleConfig  `json:"roles"`
	GitHub  GitHubConfig  `json:"github"`
	Harness HarnessConfig `json:"harness"`
	Lease   LeaseConfig   `json:"lease"`
}

// Validate checks that the config is internally consistent.
func Validate(cfg Config) error {
	if cfg.Schema != Schema {
		return fmt.Errorf("schema %q must be %q", cfg.Schema, Schema)
	}
	if len(cfg.Models) == 0 {
		return errors.New("models must not be empty")
	}
	if len(cfg.Roles) == 0 {
		return errors.New("roles must not be empty")
	}

	modelByID := make(map[string]Model, len(cfg.Models))
	for _, m := range cfg.Models {
		if m.ID == "" {
			return errors.New("model id must not be empty")
		}
		if _, ok := modelByID[m.ID]; ok {
			return fmt.Errorf("duplicate model id %q", m.ID)
		}
		modelByID[m.ID] = m
	}

	roleNames := make(map[string]bool, len(cfg.Roles))
	for _, r := range cfg.Roles {
		if r.Name == "" {
			return errors.New("role name must not be empty")
		}
		if roleNames[r.Name] {
			return fmt.Errorf("duplicate role %q", r.Name)
		}
		roleNames[r.Name] = true
		if len(r.ModelIDs) == 0 {
			return fmt.Errorf("role %q must reference at least one model", r.Name)
		}
		for _, id := range r.ModelIDs {
			if _, ok := modelByID[id]; !ok {
				return fmt.Errorf("role %q references unknown model %q", r.Name, id)
			}
		}
	}

	if cfg.GitHub.Repo == "" {
		return errors.New("github.repo must not be empty")
	}
	if len(cfg.GitHub.AllowedRepos) == 0 {
		return errors.New("github.allowedRepos must not be empty")
	}
	for _, r := range cfg.GitHub.AllowedRepos {
		if !strings.Contains(r, "/") {
			return fmt.Errorf("github.allowedRepos entry %q must be owner/repo", r)
		}
	}
	allowed := false
	for _, r := range cfg.GitHub.AllowedRepos {
		if r == cfg.GitHub.Repo {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("github.repo %q is not in github.allowedRepos", cfg.GitHub.Repo)
	}

	return nil
}

// ResolveRoleModel selects the best available model for the given role.
// It prefers enabled models with lower priority number, then lower cost tier.
func ResolveRoleModel(cfg Config, role string) (*Model, error) {
	type scored struct {
		model Model
		score int
	}
	var candidates []scored
	for _, r := range cfg.Roles {
		if r.Name != role {
			continue
		}
		for _, id := range r.ModelIDs {
			for _, m := range cfg.Models {
				if m.ID == id && m.Enabled {
					candidates = append(candidates, scored{model: m, score: m.Priority})
				}
			}
		}
		break
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no enabled model found for role %q", role)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score < candidates[j].score
		}
		return candidates[i].model.CostTier < candidates[j].model.CostTier
	})
	return &candidates[0].model, nil
}

// LoadConfig reads and validates a dispatcher config file.
func LoadConfig(path string) (Config, error) {
	var cfg Config
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, err
	}
	return cfg, Validate(cfg)
}
