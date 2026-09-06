package main

import (
	"errors"
	"reflect"
	"testing"

	"tei/tools/internal/repositoryevidence"
)

func TestCheckOnlyCommandsNeverUseWriteMode(t *testing.T) {
	got := checkOnlyCommands()
	want := []commandSpec{
		{
			Name: "world-plugin-lock-golden",
			Args: []string{"run", "./tools/reference-golden"},
		},
		{
			Name: "durable-run-golden",
			Args: []string{"run", "./tools/reference-durable-run-golden"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("check-only command=%#v want=%#v", got, want)
	}
	for _, spec := range got {
		for _, arg := range spec.Args {
			if arg == "-write" {
				t.Fatalf("完了Evidence runnerが修復操作-writeを含んでいます: %#v", spec)
			}
		}
	}
}

func TestCommandDisplayKeepsArgumentBoundaries(t *testing.T) {
	spec := commandSpec{Name: "example", Args: []string{"run", "./tools/reference-golden"}}
	got := commandDisplay(spec)
	want := []string{"go", "run", "./tools/reference-golden"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command display=%#v want=%#v", got, want)
	}
}

func TestRunWithRechecksSourceAndKeepsReproducibleFalseWhenCheckFails(t *testing.T) {
	postCalls := 0
	checkCalls := 0

	result, err := runWith(
		cleanStart,
		func(initial string) (repositoryevidence.PostRun, error) {
			postCalls++
			if initial != "R" {
				t.Fatalf("InspectPost initial=%q want R", initial)
			}
			return cleanPostRun("R"), nil
		},
		func(commandSpec) bool {
			checkCalls++
			return checkCalls != 1
		},
		fakeGoVersion,
	)
	if err == nil {
		t.Fatal("runWith() error=nil, want golden check failure")
	}
	if checkCalls != 2 {
		t.Fatalf("check calls=%d want 2", checkCalls)
	}
	if postCalls != 3 {
		t.Fatalf("post-run source checks=%d want 3 (each check + final)", postCalls)
	}
	if !result.PostRunRepository.Checked {
		t.Fatal("postRunRepository.checked=false after child check failure")
	}
	if result.ReproducibleRevision {
		t.Fatal("reproducibleRevision=true on failed golden check")
	}
}

func TestRunWithStopsBeforeNextCheckWhenFirstCheckLeavesDirtyTree(t *testing.T) {
	postCalls := 0
	checkCalls := 0

	result, err := runWith(
		cleanStart,
		func(initial string) (repositoryevidence.PostRun, error) {
			postCalls++
			return repositoryevidence.PostRun{
				Checked:            true,
				RepositoryRevision: initial,
				RevisionStable:     true,
				WorkingTreeClean:   false,
			}, nil
		},
		func(commandSpec) bool {
			checkCalls++
			return true
		},
		fakeGoVersion,
	)
	if err == nil {
		t.Fatal("runWith() error=nil, want dirty working tree failure")
	}
	if checkCalls != 1 {
		t.Fatalf("check calls=%d want 1", checkCalls)
	}
	if postCalls != 1 {
		t.Fatalf("post-run source checks=%d want 1", postCalls)
	}
	if result.PostRunRepository.WorkingTreeClean {
		t.Fatal("postRunRepository.workingTreeClean=true, want false")
	}
	if result.ReproducibleRevision {
		t.Fatal("reproducibleRevision=true after check-only tool changed working tree")
	}
}

func TestRunWithRejectsUncheckedPostRun(t *testing.T) {
	result, err := runWith(
		cleanStart,
		func(string) (repositoryevidence.PostRun, error) {
			return repositoryevidence.PostRun{
				Checked:            false,
				RepositoryRevision: "R",
				RevisionStable:     true,
				WorkingTreeClean:   true,
			}, nil
		},
		func(commandSpec) bool { return true },
		fakeGoVersion,
	)
	if err == nil {
		t.Fatal("runWith() error=nil, want unchecked post-run failure")
	}
	if result.ReproducibleRevision {
		t.Fatal("reproducibleRevision=true with unchecked post-run")
	}
}

func TestRunWithRejectsSourceDrift(t *testing.T) {
	result, err := runWith(
		cleanStart,
		func(string) (repositoryevidence.PostRun, error) {
			return repositoryevidence.PostRun{
				Checked:            true,
				RepositoryRevision: "R2",
				RevisionStable:     false,
				WorkingTreeClean:   true,
			}, nil
		},
		func(commandSpec) bool { return true },
		fakeGoVersion,
	)
	if err == nil {
		t.Fatal("runWith() error=nil, want source drift failure")
	}
	if result.ReproducibleRevision {
		t.Fatal("reproducibleRevision=true after source drift")
	}
}

func TestRunWithMarksReproducibleOnlyAfterChecksAndPostRun(t *testing.T) {
	postCalls := 0
	checkCalls := 0

	result, err := runWith(
		cleanStart,
		func(string) (repositoryevidence.PostRun, error) {
			postCalls++
			return cleanPostRun("R"), nil
		},
		func(commandSpec) bool {
			checkCalls++
			return true
		},
		fakeGoVersion,
	)
	if err != nil {
		t.Fatalf("runWith() error=%v", err)
	}
	if checkCalls != 2 {
		t.Fatalf("check calls=%d want 2", checkCalls)
	}
	if postCalls != 3 {
		t.Fatalf("post-run source checks=%d want 3 (each check + final)", postCalls)
	}
	if !result.ReproducibleRevision {
		t.Fatal("reproducibleRevision=false after all checks and clean post-run")
	}
}

func cleanStart() (repositoryevidence.Source, error) {
	return repositoryevidence.Source{
		RepositoryRevision: "R",
		WorkingTreeClean:   true,
	}, nil
}

func cleanPostRun(revision string) repositoryevidence.PostRun {
	return repositoryevidence.PostRun{
		Checked:            true,
		RepositoryRevision: revision,
		RevisionStable:     true,
		WorkingTreeClean:   true,
	}
}

func fakeGoVersion(name string, args ...string) (string, error) {
	if name != "go" || len(args) != 1 || args[0] != "version" {
		return "", errors.New("unexpected command")
	}
	return "go version go1.23.2 linux/amd64", nil
}
