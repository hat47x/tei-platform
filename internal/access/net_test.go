package access_test

import (
	"context"
	"net"
	"testing"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/permission"
)

type countingDialer struct {
	calls int
	conn  net.Conn
	err   error
}

func (d *countingDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	d.calls++
	return d.conn, d.err
}

func TestNetworkConnectsWhenTheGrantCoversTheAddress(t *testing.T) {
	journal := &memoryJournal{}
	dialer := &countingDialer{}
	grant := permission.Spec{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"}
	network := access.Network{Capability: core.MustCanonicalID("sales.close"), Guard: access.Guard{Granted: []permission.Spec{grant}, Journal: journal}, Dialer: dialer}

	if _, err := network.DialContext(context.Background(), "tcp", "db:5432"); err != nil {
		t.Fatalf("dial: %v", err)
	}
	if dialer.calls != 1 {
		t.Fatalf("the dialer must be used once: %d", dialer.calls)
	}
	if len(journal.records) != 1 || !journal.records[0].Allowed {
		t.Fatalf("the decision must be recorded: %+v", journal.records)
	}
}

func TestNetworkRefusesAnUngrantedAddressWithoutDialing(t *testing.T) {
	dialer := &countingDialer{}
	network := access.Network{Capability: core.MustCanonicalID("sales.close"), Guard: access.Guard{Granted: []permission.Spec{{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"}}}, Dialer: dialer}

	if _, err := network.DialContext(context.Background(), "tcp", "evil:6379"); err == nil {
		t.Fatal("an ungranted address must be refused")
	}
	if dialer.calls != 0 {
		t.Fatal("a refused connection must not open a socket")
	}
}

func TestNetworkWithoutADialerFails(t *testing.T) {
	grant := permission.Spec{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"}
	network := access.Network{Capability: core.MustCanonicalID("sales.close"), Guard: access.Guard{Granted: []permission.Spec{grant}}}
	if _, err := network.DialContext(context.Background(), "tcp", "db:5432"); err == nil {
		t.Fatal("no dialer means the connection is not attempted")
	}
}

func TestNetworkRecordsAJournalFailure(t *testing.T) {
	grant := permission.Spec{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"}
	network := access.Network{Capability: core.MustCanonicalID("sales.close"), Guard: access.Guard{Granted: []permission.Spec{grant}, Journal: &memoryJournal{err: net.ErrClosed}}, Dialer: &countingDialer{}}
	if _, err := network.DialContext(context.Background(), "tcp", "db:5432"); err == nil {
		t.Fatal("an unrecorded decision must be reported")
	}
}
