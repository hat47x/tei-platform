//go:build !tei_repair_only

package main

import (
	"strings"
	"testing"
)

func TestBatchObservationRejectsAmbiguousOrUnboundedIdentities(t *testing.T) {
	for _, raw := range []string{"", "0", "-1", "1001,1001", "1001,", "1001,invalid", "9223372036854775808", strings.Repeat("1,", 100) + "2"} {
		if _, err := parseObservationIDs(raw); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
	ids, err := parseObservationIDs("1003,1001")
	if err != nil || len(ids) != 2 || ids[0] != 1003 || ids[1] != 1001 {
		t.Fatalf("identities changed: %v %v", ids, err)
	}
	if err := validateObservationIDs(make([]int64, 101)); err == nil {
		t.Fatal("unbounded typed input")
	}
}
