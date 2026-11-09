package runtime_test

import (
	"context"
	"testing"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/permission"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

type recordingJournal struct {
	records []access.Record
	err     error
}

func (j *recordingJournal) Append(record access.Record) error {
	if j.err != nil {
		return j.err
	}
	j.records = append(j.records, record)
	return nil
}

func policyProfile() pruntime.SandboxProfile {
	return pruntime.SandboxProfile{DenyNetworkByDefault: true, DenyFilesystemByDefault: true}
}

func TestSandboxRecordsAnAllowedDecision(t *testing.T) {
	journal := &recordingJournal{}
	launcher := pruntime.HostSandboxLauncher{Network: &netEnforcer{}, Filesystem: &fsEnforcer{}, Secrets: &secretInjector{}, Runner: &sandboxRunner{}, Access: journal}
	spec := pruntime.SandboxSpec{
		Command:     pruntime.ProcessCommand{Executable: "worker"},
		Plugin:      plugin.Identity{ID: core.MustCanonicalID("tei.demo"), Version: "1.0.0"},
		Profile:     policyProfile(),
		Permissions: []permission.Spec{{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"}},
	}
	if _, err := launcher.Invoke(context.Background(), spec, pruntime.Request{}); err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if len(journal.records) != 1 || !journal.records[0].Allowed {
		t.Fatalf("the allowed decision must be recorded: %+v", journal.records)
	}
	if journal.records[0].Capability != "tei.demo" {
		t.Fatalf("the record must name the capability: %+v", journal.records[0])
	}
}

func TestSandboxRecordsARefusalBeforeRefusing(t *testing.T) {
	journal := &recordingJournal{}
	launcher := pruntime.HostSandboxLauncher{Runner: &sandboxRunner{}, Access: journal}
	spec := pruntime.SandboxSpec{
		Command: pruntime.ProcessCommand{Executable: "worker"},
		Plugin:  plugin.Identity{ID: core.MustCanonicalID("tei.demo"), Version: "1.0.0"},
		Profile: pruntime.SandboxProfile{NetworkAllowlist: []string{"evil:6379"}},
	}
	if _, err := launcher.Invoke(context.Background(), spec, pruntime.Request{}); err == nil {
		t.Fatal("a profile without deny-by-default must be refused")
	}
	if len(journal.records) != 1 || journal.records[0].Allowed || len(journal.records[0].Missing) == 0 {
		t.Fatalf("the refused decision must be recorded with its reasons: %+v", journal.records)
	}
}
