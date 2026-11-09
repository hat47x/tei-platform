package main

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestDBCheckWithoutADSNReportsMissing(t *testing.T) {
	var out bytes.Buffer
	err := runDBCheck(context.Background(), "  ", &out)
	if !errors.Is(err, errDSNMissing) {
		t.Fatalf("a missing DSN must be reported as such: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("nothing must be printed when nothing is attempted: %q", out.String())
	}
}

func TestDBCheckDSNPrefersTheFlag(t *testing.T) {
	t.Setenv("TEI_TEST_POSTGRES_DSN", "postgres://from-env")
	if got := dbCheckDSN("postgres://from-flag"); got != "postgres://from-flag" {
		t.Fatalf("the flag must win: %q", got)
	}
	if got := dbCheckDSN(""); got != "postgres://from-env" {
		t.Fatalf("the environment is the fallback: %q", got)
	}
}

func TestFirstLineKeepsOneLine(t *testing.T) {
	if got := firstLine("PostgreSQL 16.2 on x86_64\nmore"); got != "PostgreSQL 16.2 on x86_64" {
		t.Fatalf("unexpected line: %q", got)
	}
}
