package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestEnvironmentWithoutPostgresDSNRemovesAllCaseVariants(t *testing.T) {
	base := []string{
		"PATH=/usr/bin",
		"TEI_TEST_POSTGRES_DSN=postgres://first-secret",
		"tei_test_postgres_dsn=postgres://second-secret",
		"OTHER=value",
	}

	got := environmentWithoutPostgresDSN(base)
	want := []string{"PATH=/usr/bin", "OTHER=value"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("environmentWithoutPostgresDSN()=%#v want=%#v", got, want)
	}

	if base[1] != "TEI_TEST_POSTGRES_DSN=postgres://first-secret" || base[2] != "tei_test_postgres_dsn=postgres://second-secret" {
		t.Fatalf("base environmentを破壊しました: %#v", base)
	}
}

func TestCommandOutputDoesNotExposePostgresDSNToUtilitySubprocess(t *testing.T) {
	t.Setenv(postgresDSNEnv, "postgres://user:secret@example.invalid/evidence")
	t.Setenv("TEI_GATE_B_PROBE_HELPER", "1")

	out, err := commandOutput(os.Args[0], "-test.run=^TestCommandOutputProbeHelper$")
	if err != nil {
		t.Fatalf("commandOutput() error=%v", err)
	}
	if got := strings.TrimSpace(out); got != "dsn-absent" {
		t.Fatalf("utility subprocess result=%q want=dsn-absent", got)
	}
}

func TestCommandOutputProbeHelper(t *testing.T) {
	if os.Getenv("TEI_GATE_B_PROBE_HELPER") != "1" {
		return
	}
	if _, exists := os.LookupEnv(postgresDSNEnv); exists {
		fmt.Fprint(os.Stdout, "dsn-present")
		os.Exit(17)
	}
	fmt.Fprint(os.Stdout, "dsn-absent")
	os.Exit(0)
}
