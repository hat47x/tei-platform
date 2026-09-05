package main

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func TestEnvironmentWithoutPostgresSecretsRemovesDSNAndFilePath(t *testing.T) {
	base := []string{
		"PATH=/usr/bin",
		"TEI_TEST_POSTGRES_DSN=postgres://direct-secret",
		"tei_test_postgres_dsn=postgres://duplicate-secret",
		"TEI_TEST_POSTGRES_DSN_FILE=/private/dsn",
		"tei_test_postgres_dsn_file=/private/duplicate-dsn",
		"OTHER=value",
	}
	before := append([]string(nil), base...)

	got := environmentWithoutPostgresSecrets(base)
	want := []string{"PATH=/usr/bin", "OTHER=value"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("environmentWithoutPostgresSecrets()=%#v want=%#v", got, want)
	}
	if !reflect.DeepEqual(base, before) {
		t.Fatalf("base environmentを破壊しました: %#v", base)
	}
}

func TestEnvironmentForLaneWithPostgresDSNInjectsSecretOnlyIntoPostgresLane(t *testing.T) {
	const secret = "postgres://user:password@example.invalid/evidence"
	base := []string{
		"PATH=/usr/bin",
		"TEI_TEST_POSTGRES_DSN_FILE=/private/dsn",
		"OTHER=value",
	}

	for _, spec := range laneSpecs(false)[:2] {
		got := environmentForLaneWithPostgresDSN(spec, base, secret)
		if !reflect.DeepEqual(got, []string{"PATH=/usr/bin", "OTHER=value"}) {
			t.Fatalf("%s environment=%#v", spec.Name, got)
		}
	}

	postgres := laneSpecs(false)[2]
	got := environmentForLaneWithPostgresDSN(postgres, base, secret)
	want := []string{"PATH=/usr/bin", "OTHER=value", postgresDSNEnv + "=" + secret}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PostgreSQL lane environment=%#v want=%#v", got, want)
	}
}

func TestPostgresSecretEnvironmentIsolatedInActualChildProcess(t *testing.T) {
	const secret = "postgres://user:password@example.invalid/evidence"
	base := append([]string(nil), os.Environ()...)
	base = append(base,
		postgresDSNFileEnv+"=/private/secret-path-must-not-leak",
		"TEI_GATE_B_SECRET_ENV_HELPER=1",
	)

	for _, spec := range laneSpecs(false) {
		env := environmentForLaneWithPostgresDSN(spec, base, secret)
		cmd := exec.Command(os.Args[0], "-test.run=^TestPostgresSecretEnvironmentHelper$")
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s child error=%v: %s", spec.Name, err, out)
		}
		got := strings.TrimSpace(string(out))
		if spec.Name == "real-postgres-ephemeral-fixture" {
			if got != "postgres-secret-only" {
				t.Fatalf("%s child result=%q", spec.Name, got)
			}
			continue
		}
		if got != "no-postgres-secret" {
			t.Fatalf("%s child result=%q", spec.Name, got)
		}
	}
}

func TestPostgresSecretEnvironmentHelper(t *testing.T) {
	if os.Getenv("TEI_GATE_B_SECRET_ENV_HELPER") != "1" {
		return
	}
	if _, exists := os.LookupEnv(postgresDSNFileEnv); exists {
		fmt.Fprint(os.Stdout, "file-path-present")
		os.Exit(17)
	}
	secret, hasSecret := os.LookupEnv(postgresDSNEnv)
	if hasSecret {
		if secret == "postgres://user:password@example.invalid/evidence" {
			fmt.Fprint(os.Stdout, "postgres-secret-only")
			os.Exit(0)
		}
		fmt.Fprint(os.Stdout, "unexpected-secret")
		os.Exit(18)
	}
	fmt.Fprint(os.Stdout, "no-postgres-secret")
	os.Exit(0)
}
