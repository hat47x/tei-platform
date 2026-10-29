package externaladoptionauthority

// The through test builds the two asset CLIs with the Go executable named by
// TEI_EAA_GO, normally provided by tools/verify-external-adoption-authority.py.
// Without it the test skips; a skip is not evidence that anything passed.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestThroughExternalAdoptionAuthority(t *testing.T) {
	goExe := os.Getenv(GoEnv)
	if goExe == "" {
		t.Skip("requires TEI_EAA_GO to build the asset CLIs; skip is not a pass")
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo, err := FindRepo(wd)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	work := t.TempDir()
	tools, err := BuildTools(ctx, repo, goExe, filepath.Join(work, "bin"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSample(repo, filepath.Join(work, "run"), tools)
	if err != nil {
		t.Fatal(err)
	}
	runErr := s.Run(ctx)
	raw, err := json.MarshalIndent(s.Rec, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv(OutDirEnv)
	if dir == "" {
		dir = t.TempDir()
	}
	if err := os.WriteFile(filepath.Join(dir, "chain-record.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if runErr != nil {
		t.Fatalf("chain stopped: %v", runErr)
	}
	for _, list := range [][]Check{s.Rec.Faults, s.Rec.Completion} {
		for _, c := range list {
			if !c.Passed {
				obs, _ := json.Marshal(c.Observed)
				t.Errorf("%s did not pass: %s", c.ID, obs)
			}
		}
	}
	if !s.Rec.Through.Complete {
		t.Errorf("through: %+v", s.Rec.Through)
	}
	if !s.Rec.Passed {
		t.Fatal("chain record is not passed")
	}
}
