//go:build tei_api_only && tei_order_only && !tei_repair_only

package main

import (
	"os"
	"strings"
	"testing"

	"tei/internal/ir"
)

func TestAPIOnlyListenDeniesNonLoopbackAndAmbiguousAddress(t *testing.T) {
	for _, tc := range []struct {
		addr string
		allow bool
	}{
		{"127.0.0.1:8080", true},
		{"[::1]:8443", true},
		{"127.0.0.1:65535", true},
		{"0.0.0.0:8080", false},
		{"[::]:8080", false},
		{":8080", false},
		{"localhost:8080", false},
		{"example.com:8080", false},
		{"192.168.1.20:8080", false},
		{"[fe80::1]:8080", false},
		{"127.0.0.2:8080", false},
		{"[::ffff:127.0.0.1]:8080", false},
		{"127.0.0.1", false},
		{"127.0.0.1:", false},
		{"127.0.0.1:0", false},
		{"127.0.0.1:-1", false},
		{"127.0.0.1:65536", false},
		{"127.0.0.1:abc", false},
	} {
		t.Run(strings.ReplaceAll(tc.addr, "/", "-"), func(t *testing.T) {
			err := validateReferenceListenAddress(tc.addr)
			if (err == nil) != tc.allow {
				t.Fatalf("diagnostic API -listen %q: err=%v, expected allow=%t", tc.addr, err, tc.allow)
			}
		})
	}
}

func TestAPIOnlyRejectsPublicListenBeforeLoadingWorldOrExecutingRunners(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{
		"order-entry", "-workspace", "this-path-does-not-exist",
		"-baseline-id", "B",
		"-baseline-revision", strings.Repeat("a", 40),
		"-target-revision", strings.Repeat("b", 40),
		"-listen", "0.0.0.0:7777",
	})
	called := false
	err := runSingleApplicationTarget("order_entry", func(ir.World, singleApplicationConfig) (
		readScenarioRunner, commandScenarioRunner, []webTargetOption, error,
	) {
		called = true
		return nil, nil, nil, nil
	})
	if err == nil || !strings.Contains(err.Error(), "explicit loopback") || called {
		t.Fatalf("public listen was not rejected before loading the World (error=%v, factory called=%t)", err, called)
	}
}
