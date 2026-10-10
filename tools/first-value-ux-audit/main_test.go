package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const exampleSession = `{"schema":"tei.first-value-ux-session/v1","sessionId":"fixture-01","taskId":"E0-model-change","phase":"E0","actor":"new-agent","provenance":"synthetic","repositorySHA":"0000000000000000000000000000000000000000","startedAt":"2026-10-10T08:00:00Z","finishedAt":"2026-10-10T08:00:40Z","events":[{"at":"2026-10-10T08:00:01Z","kind":"command","exitCode":0,"evidenceRef":"log/validate-1.txt"},{"at":"2026-10-10T08:00:02Z","kind":"source-edit"},{"at":"2026-10-10T08:00:03Z","kind":"manual-fields","count":1},{"at":"2026-10-10T08:00:05Z","kind":"command","exitCode":1,"evidenceRef":"log/validate-2.txt"},{"at":"2026-10-10T08:00:10Z","kind":"recovery"},{"at":"2026-10-10T08:00:20Z","kind":"source-edit"},{"at":"2026-10-10T08:00:39Z","kind":"command","exitCode":0,"evidenceRef":"log/validate-3.txt"}]}`

func TestReduceDerivedMetricsNeverQualifiesSynthetic(t *testing.T) {
	var b bytes.Buffer
	if err := execute(strings.NewReader(exampleSession), &b); err != nil {
		t.Fatal(err)
	}
	var m Metrics
	if err := json.Unmarshal(b.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m.DurationSeconds == nil || *m.DurationSeconds != 40 || m.Commands != 3 || m.FailedCommands != 1 || m.SourceEdits != 2 || m.ManualFields != 1 || m.RecoveryActions != 1 || m.Qualification != "unverified" {
		t.Fatalf("incorrect metrics: %+v", m)
	}
	if len(m.EvidenceReferences) != 3 || len(m.Limitations) != 2 {
		t.Fatalf("missing provenance boundaries: %+v", m)
	}
	if !strings.Contains(b.String(), "synthetic-session-not-user-evidence") {
		t.Fatal("lost synthetic limitation")
	}
}

func TestValidOpenSessionDoesNotInventDuration(t *testing.T) {
	s := strings.Replace(exampleSession, `,"finishedAt":"2026-10-10T08:00:40Z"`, "", 1)
	var b bytes.Buffer
	if err := execute(strings.NewReader(s), &b); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "durationSeconds") || !strings.Contains(b.String(), "duration-not-recorded") {
		t.Fatal(b.String())
	}
}

func TestE1CommandSuccessWithoutIndependentReadIsNotVerified(t *testing.T) {
	s := strings.Replace(exampleSession, `"phase":"E0"`, `"phase":"E1"`, 1)
	var b bytes.Buffer
	if err := execute(strings.NewReader(s), &b); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "independent-read-not-recorded") || !strings.Contains(b.String(), `"qualification": "unverified"`) {
		t.Fatal(b.String())
	}
}

func TestE1ReadIsClaimOnly(t *testing.T) {
	s := strings.Replace(exampleSession, `"phase":"E0"`, `"phase":"E1"`, 1)
	s = strings.Replace(s, `{"at":"2026-10-10T08:00:39Z","kind":"command","exitCode":0,`, `{"at":"2026-10-10T08:00:39Z","kind":"independent-read","evidenceRef":"logs/independent-read.json"},{"at":"2026-10-10T08:00:39Z","kind":"command","exitCode":0,`, 1)
	var b bytes.Buffer
	if err := execute(strings.NewReader(s), &b); err != nil {
		t.Fatal(err)
	}
	var m Metrics
	if err := json.Unmarshal(b.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m.IndependentReadClaims != 1 || m.Qualification != "unverified" {
		t.Fatal(m)
	}
}

func TestRejectInvalidAndMisleadingRecords(t *testing.T) {
	tests := map[string]string{
		"invalid-revision": strings.Replace(exampleSession, `0000000000000000000000000000000000000000`, `not-a-git-commit`, 1),
		"too-large": strings.Repeat(" ", 1<<20) + exampleSession,
		"unknown-schema":  strings.Replace(exampleSession, `tei.first-value-ux-session/v1`, `unknown`, 1),
		"unknown-actor":   strings.Replace(exampleSession, `"new-agent"`, `"expert"`, 1),
		"unknown-field":   strings.Replace(exampleSession, `"phase":"E0"`, `"phase":"E0","succeeded":true`, 1),
		"exit-missing":    strings.Replace(exampleSession, `"kind":"command","exitCode":0`, `"kind":"command"`, 1),
		"negative-count":  strings.Replace(exampleSession, `"count":1`, `"count":-1`, 1),
		"backwards-clock": strings.Replace(exampleSession, `2026-10-10T08:00:39Z`, `2026-10-10T07:00:39Z`, 1),
		"out-of-order":    strings.Replace(exampleSession, `2026-10-10T08:00:39Z`, `2026-10-10T08:00:01Z`, 1),
		"trailing-json":   exampleSession + ` {}`,
		"e0-read":         strings.Replace(exampleSession, `"kind":"recovery"`, `"kind":"independent-read","evidenceRef":"read.json"`, 1),
	}
	for name, src := range tests {
		t.Run(name, func(t *testing.T) {
			var b bytes.Buffer
			if err := execute(strings.NewReader(src), &b); err == nil {
				t.Fatalf("invalid evidence accepted: %s", name)
			}
		})
	}
}

func TestDeterministicOutput(t *testing.T) {
	var a, b bytes.Buffer
	for _, dst := range []*bytes.Buffer{&a, &b} {
		if err := execute(strings.NewReader(exampleSession), dst); err != nil {
			t.Fatal(err)
		}
	}
	if a.String() != b.String() {
		t.Fatal("output varied")
	}
}
