package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

// The new runtime path is opt-in: an ordinary view keeps its projection
// contract, and an explicit database view cannot quietly discard the input.
func TestDestinationRuntimeViewRequiresExplicitInputsAndConnection(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	t.Setenv("TEI_VIEW_PG_DSN", "")
	base := []string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_lines",
		"-environment", "development",
		"-format", "json",
	}
	for _, tc := range []struct {
		name string
		args []string
		code string
	}{
		{name: "no input", code: "VIEW_EXECUTION_FAILED"},
		{name: "malformed input", args: []string{"-input", "1001"}, code: "INVALID_ARGUMENT"},
		{name: "wrong input", args: []string{"-input", "line_number=1"}, code: "VIEW_EXECUTION_FAILED"},
		{name: "missing database connection", args: []string{"-input", "order_id=1001"}, code: "VIEW_EXECUTION_FAILED"},
		{name: "duplicate input", args: []string{"-input", "order_id=1001", "-input", "order_id=1002"}, code: "INVALID_ARGUMENT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			args := append(append([]string(nil), base...), tc.args...)
			exit, err := runViewCommand(args, &out)
			if err != nil || exit == 0 {
				t.Fatalf("exit=%d err=%v json=%s", exit, err, out.String())
			}
			var result struct {
				OK    bool `json:"ok"`
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatalf("invalid JSON report: %v (%s)", err, out.String())
			}
			if result.OK || result.Error.Code != tc.code {
				t.Fatalf("report=%+v, want %s", result, tc.code)
			}
		})
	}
}

func TestDestinationRuntimeViewDoesNotSilentlyChangeStaticOrCommandViews(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "runtime without a pattern", args: []string{"-environment", "development"}},
		{name: "runtime with a command", args: []string{"-pattern", "order_entry.order_lines", "-environment", "development", "-command", "sales.order.submit"}},
		{name: "runtime with text format", args: []string{"-pattern", "order_entry.order_lines", "-environment", "development", "-format", "text"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			args := append([]string{"-workspace", root, "-application", "order_entry"}, tc.args...)
			exit, err := runViewCommand(args, &out)
			if err != nil || exit == 0 {
				t.Fatalf("exit=%d err=%v report=%s", exit, err, out.String())
			}
			var report struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(out.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			if report.Error.Code != "INVALID_ARGUMENT" {
				t.Fatalf("code=%s, want INVALID_ARGUMENT", report.Error.Code)
			}
		})
	}
}
