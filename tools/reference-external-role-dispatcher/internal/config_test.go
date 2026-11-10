package dispatch

import (
	"testing"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid",
			cfg: Config{
				Schema: Schema,
				Models: []Model{
					{ID: "m1", Harness: "dsh", Roles: []string{"implementer"}, MaxConcurrent: 1, Enabled: true, Priority: 1},
				},
				Roles: []RoleConfig{
					{Name: "implementer", ModelIDs: []string{"m1"}},
				},
				GitHub: GitHubConfig{
					Repo:         "owner/repo",
					AllowedRepos: []string{"owner/repo"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty models",
			cfg: Config{
				Schema: Schema,
				Roles:  []RoleConfig{{Name: "implementer", ModelIDs: []string{"m1"}}},
				GitHub: GitHubConfig{Repo: "owner/repo", AllowedRepos: []string{"owner/repo"}},
			},
			wantErr: true,
		},
		{
			name: "repo not allowed",
			cfg: Config{
				Schema: Schema,
				Models: []Model{{ID: "m1", Harness: "dsh", Roles: []string{"implementer"}, MaxConcurrent: 1, Enabled: true, Priority: 1}},
				Roles:  []RoleConfig{{Name: "implementer", ModelIDs: []string{"m1"}}},
				GitHub: GitHubConfig{Repo: "owner/repo", AllowedRepos: []string{"other/repo"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolveRoleModel(t *testing.T) {
	cfg := Config{
		Schema: Schema,
		Models: []Model{
			{ID: "low", Harness: "dsh", Roles: []string{"implementer"}, MaxConcurrent: 1, Enabled: true, Priority: 10, CostTier: "low"},
			{ID: "high", Harness: "dsh", Roles: []string{"implementer"}, MaxConcurrent: 1, Enabled: true, Priority: 20, CostTier: "high"},
			{ID: "disabled", Harness: "dsh", Roles: []string{"implementer"}, MaxConcurrent: 1, Enabled: false, Priority: 5, CostTier: "low"},
		},
		Roles: []RoleConfig{
			{Name: "implementer", ModelIDs: []string{"disabled", "high", "low"}},
		},
		GitHub: GitHubConfig{Repo: "owner/repo", AllowedRepos: []string{"owner/repo"}},
	}

	model, err := ResolveRoleModel(cfg, "implementer")
	if err != nil {
		t.Fatalf("ResolveRoleModel() error = %v", err)
	}
	if model.ID != "low" {
		t.Errorf("ResolveRoleModel() = %v, want low", model.ID)
	}
}

func TestResolveRoleModelFallback(t *testing.T) {
	cfg := Config{
		Schema: Schema,
		Models: []Model{
			{ID: "primary", Harness: "dsh", Roles: []string{"implementer"}, MaxConcurrent: 1, Enabled: true, Priority: 10, CostTier: "low"},
			{ID: "fallback", Harness: "dsh", Roles: []string{"implementer"}, MaxConcurrent: 1, Enabled: true, Priority: 20, CostTier: "medium"},
		},
		Roles: []RoleConfig{
			{Name: "implementer", ModelIDs: []string{"primary", "fallback"}},
		},
		GitHub: GitHubConfig{Repo: "owner/repo", AllowedRepos: []string{"owner/repo"}},
	}

	model, err := ResolveRoleModel(cfg, "implementer")
	if err != nil {
		t.Fatalf("ResolveRoleModel() error = %v", err)
	}
	if model.ID != "primary" {
		t.Errorf("ResolveRoleModel() = %v, want primary", model.ID)
	}

	cfg.Models[0].Enabled = false
	model, err = ResolveRoleModel(cfg, "implementer")
	if err != nil {
		t.Fatalf("ResolveRoleModel() error = %v", err)
	}
	if model.ID != "fallback" {
		t.Errorf("ResolveRoleModel() after disable = %v, want fallback", model.ID)
	}
}
