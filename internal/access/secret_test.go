package access_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/permission"
)

type fakeSecrets struct {
	reads int
	value string
	err   error
}

func (f *fakeSecrets) ReadSecret(context.Context, string) (string, error) {
	f.reads++
	return f.value, f.err
}

func TestSecretsReadsWhenTheGrantCoversIt(t *testing.T) {
	journal := &memoryJournal{}
	reader := &fakeSecrets{value: "hunter2"}
	secrets := access.Secrets{
		Capability: core.MustCanonicalID("sales.close"),
		Guard: access.Guard{
			Granted: []permission.Spec{{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "vault:db"}},
			Journal: journal,
		},
		Reader: reader,
	}
	value, err := secrets.Read(context.Background(), "vault:db")
	if err != nil || value != "hunter2" {
		t.Fatalf("read: %q err=%v", value, err)
	}
	if reader.reads != 1 || len(journal.records) != 1 || !journal.records[0].Allowed {
		t.Fatalf("the read and its decision must be recorded: reads=%d records=%+v", reader.reads, journal.records)
	}
}

func TestSecretsRefusesAnUngrantedNameWithoutReading(t *testing.T) {
	reader := &fakeSecrets{value: "hunter2"}
	secrets := access.Secrets{Capability: core.MustCanonicalID("sales.close"), Reader: reader}
	if _, err := secrets.Read(context.Background(), "vault:db"); err == nil {
		t.Fatal("an ungranted secret must be refused")
	}
	if reader.reads != 0 {
		t.Fatal("a refused secret must not be read")
	}
}

func TestSecretsWithoutAReaderFails(t *testing.T) {
	secrets := access.Secrets{
		Capability: core.MustCanonicalID("sales.close"),
		Guard:      access.Guard{Granted: []permission.Spec{{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "vault:db"}}},
	}
	if _, err := secrets.Read(context.Background(), "vault:db"); err == nil {
		t.Fatal("no reader means the secret is not read")
	}
}

func TestSecretsReportsAJournalFailure(t *testing.T) {
	secrets := access.Secrets{
		Capability: core.MustCanonicalID("sales.close"),
		Guard:      access.Guard{Granted: []permission.Spec{{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "vault:db"}}, Journal: &memoryJournal{err: errors.New("disk full")}},
		Reader:     &fakeSecrets{value: "hunter2"},
	}
	if _, err := secrets.Read(context.Background(), "vault:db"); err == nil {
		t.Fatal("an unrecorded decision must be reported")
	}
}
