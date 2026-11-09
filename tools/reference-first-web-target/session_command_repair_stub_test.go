//go:build tei_repair_only && !tei_order_only

package main

import "testing"

// A repair-only process must never turn an attempted order_entry session
// Command configuration into an unprotected handler or successful startup.
func TestRepairOnlySessionCommandConfigurationFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		count  int
		config *protectedSessionCommandConfig
		wantOK bool
	}{
		{name: "no session route", wantOK: true},
		{name: "attempted session route", count: 1},
		{name: "duplicate session route", count: 2},
		{name: "config without route counter", config: &protectedSessionCommandConfig{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := &webTargetServer{
				protectedSessionOptionCount: tc.count,
				protectedSessionConfig:      tc.config,
			}
			err := server.configureProtectedSessionCommand()
			if (err == nil) != tc.wantOK {
				t.Fatalf("session route configuration: got error %v, success wanted %t", err, tc.wantOK)
			}
			if server.protectedSessionHandler != nil {
				t.Fatal("repair-only target mounted an order session handler")
			}
		})
	}
}
