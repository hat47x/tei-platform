package main

import (
	"fmt"
	"testing"

	"tei/tools/internal/repositoryevidence"
)

func TestLoadPostgresDSNAfterCleanRepositoryPreflightLoadsOnlyAfterCleanSource(t *testing.T) {
	called := false
	got, err := loadPostgresDSNAfterCleanRepositoryPreflight(
		"/repo",
		func() (repositoryevidence.Source, error) {
			return repositoryevidence.Source{RepositoryRevision: "R", WorkingTreeClean: true}, nil
		},
		func(repositoryRoot string) (string, error) {
			called = true
			if repositoryRoot != "/repo" {
				t.Fatalf("repositoryRoot=%q", repositoryRoot)
			}
			return "postgres://safe", nil
		},
	)
	if err != nil {
		t.Fatalf("clean sourceでerror=%v", err)
	}
	if !called {
		t.Fatal("clean source後にsecret loaderが呼ばれていません")
	}
	if got != "postgres://safe" {
		t.Fatalf("DSN=%q", got)
	}
}

func TestLoadPostgresDSNAfterCleanRepositoryPreflightDoesNotLoadOnDirtySource(t *testing.T) {
	called := false
	_, err := loadPostgresDSNAfterCleanRepositoryPreflight(
		"/repo",
		func() (repositoryevidence.Source, error) {
			return repositoryevidence.Source{RepositoryRevision: "R", WorkingTreeClean: false}, nil
		},
		func(string) (string, error) {
			called = true
			return "postgres://must-not-read", nil
		},
	)
	if err == nil {
		t.Fatal("dirty sourceを受理しています")
	}
	if called {
		t.Fatal("dirty sourceでsecret loaderを呼び出しています")
	}
}

func TestLoadPostgresDSNAfterCleanRepositoryPreflightDoesNotLoadWhenSourceObservationFails(t *testing.T) {
	called := false
	_, err := loadPostgresDSNAfterCleanRepositoryPreflight(
		"/repo",
		func() (repositoryevidence.Source, error) {
			return repositoryevidence.Source{}, fmt.Errorf("git failed")
		},
		func(string) (string, error) {
			called = true
			return "postgres://must-not-read", nil
		},
	)
	if err == nil {
		t.Fatal("source観測失敗を受理しています")
	}
	if called {
		t.Fatal("source観測失敗時にsecret loaderを呼び出しています")
	}
}
