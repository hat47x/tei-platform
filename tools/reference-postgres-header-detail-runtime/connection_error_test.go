package main

import (
	"errors"
	"strings"
	"testing"
)

func TestPostgresConnectionErrorDoesNotExposeCredentialBearingDriverError(t *testing.T) {
	tests := []struct {
		name    string
		stage   string
		raw     string
		secrets []string
	}{
		{
			name:  "URL DSN",
			stage: "PostgreSQL接続設定の初期化",
			raw:   `parse "postgres://evidence-user:super-secret@example.invalid/evidence%zz": invalid URL escape "%zz"`,
			secrets: []string{
				"postgres://evidence-user:super-secret@example.invalid/evidence%zz",
				"super-secret",
			},
		},
		{
			name:  "keyword DSN",
			stage: "PostgreSQL接続確認",
			raw:   `connection failed for host=example.invalid user=evidence-user password=super-secret dbname=evidence`,
			secrets: []string{
				"password=super-secret",
				"super-secret",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := postgresConnectionError(tt.stage, errors.New(tt.raw)).Error()
			if !strings.Contains(got, tt.stage) {
				t.Fatalf("error=%q want stage %q", got, tt.stage)
			}
			if strings.Contains(got, tt.raw) {
				t.Fatalf("driver error本文を露出しています: %q", got)
			}
			for _, secret := range tt.secrets {
				if strings.Contains(got, secret) {
					t.Fatalf("credential候補 %q を露出しています: %q", secret, got)
				}
			}
		})
	}
}
