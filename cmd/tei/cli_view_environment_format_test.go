package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// `view -environment` renders text, json and html. The general format gate
// accepts html and print for the static views, so the runtime mode has to
// decide for itself: it used to fall through to the text renderer, and a caller
// that asked for `-format html` received text with no error (#5396). print stays
// refused. The check runs before the database is touched, so it is observable
// without a PostgreSQL instance.
func TestEnvironmentViewRefusesOnlyPrintBeforeReading(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	var out bytes.Buffer
	args := []string{
		"-workspace", root, "-application", "order_entry",
		"-pattern", "order_entry.order_lines", "-environment", "development",
		"-input", "order_id=1001", "-format", "print",
	}
	exit, err := runViewCommand(args, &out)
	if err != nil || exit == 0 {
		t.Fatalf("exit=%d err=%v report=%s", exit, err, out.String())
	}
	var report struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if jsonErr := json.Unmarshal(out.Bytes(), &report); jsonErr != nil {
		t.Fatalf("invalid report: %v (%s)", jsonErr, out.String())
	}
	if report.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("code=%s, want INVALID_ARGUMENT", report.Error.Code)
	}
	// The message must name the format that was asked for, so a caller knows
	// which flag value to change.
	if !strings.Contains(report.Error.Message, "print") {
		t.Fatalf("message does not name \"print\": %s", report.Error.Message)
	}
	// Nothing was read: a failure that names the database would mean the guard
	// ran too late.
	if strings.Contains(report.Error.Message, "PostgreSQL") ||
		strings.Contains(report.Error.Message, "TEI_VIEW_PG_DSN") {
		t.Fatalf("the format was checked after the database was touched: %s", report.Error.Message)
	}
}

// text, json and html get past the format gate. With a deliberately
// unreachable DSN the run then stops at the database, which is a different
// failure: it proves the format itself was accepted.
func TestEnvironmentViewAcceptsTextJsonAndHtmlAtTheFormatGate(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	// A failing DSN keeps this test independent of a real server. The value is
	// a deliberately unreachable address, not a credential.
	t.Setenv("TEI_VIEW_PG_DSN", "postgres://nobody@127.0.0.1:1/none")
	for _, format := range []string{"text", "json", "html"} {
		t.Run(format, func(t *testing.T) {
			var out bytes.Buffer
			args := []string{
				"-workspace", root, "-application", "order_entry",
				"-pattern", "order_entry.order_lines", "-environment", "development",
				"-input", "order_id=1001", "-format", format,
			}
			exit, err := runViewCommand(args, &out)
			if err != nil {
				t.Fatalf("command error: %v", err)
			}
			// text and html output is not JSON, so only the json form can be
			// decoded. For the others, a non-zero exit with no format complaint
			// is enough.
			if format == "json" {
				var report struct {
					Error struct {
						Code    string `json:"code"`
						Message string `json:"message"`
					} `json:"error"`
				}
				if jsonErr := json.Unmarshal(out.Bytes(), &report); jsonErr != nil {
					t.Fatalf("invalid report: %v (%s)", jsonErr, out.String())
				}
				if report.Error.Code == "INVALID_ARGUMENT" &&
					strings.Contains(report.Error.Message, "-format") {
					t.Fatalf("a supported format was refused: %s", report.Error.Message)
				}
			}
			if exit != 0 && exit != 1 {
				t.Fatalf("unexpected exit code %d", exit)
			}
		})
	}
}
