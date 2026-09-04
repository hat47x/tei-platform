package main

import (
	"errors"
	"reflect"
	"testing"
)

func TestInspectGoEnvironmentFixesChildProcessEnvironmentAfterValidation(t *testing.T) {
	var got [][2]string
	setEnv := func(name, value string) error {
		got = append(got, [2]string{name, value})
		return nil
	}

	evidence, err := inspectGoEnvironmentWith(fakeGoEnvironmentOutput("off\n", ""), setEnv)
	if err != nil {
		t.Fatalf("inspectGoEnvironmentWith() error=%v", err)
	}
	if evidence.ExternalWorkspaceActive || !evidence.GoFlagsEmpty {
		t.Fatalf("Go environment Evidenceが不正です: %#v", evidence)
	}

	want := [][2]string{
		{"GOWORK", "off"},
		{"GOFLAGS", ""},
		{"GOENV", "off"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("set environment=%#v want=%#v", got, want)
	}
}

func TestInspectGoEnvironmentDoesNotNormalizeRejectedAmbientEnvironment(t *testing.T) {
	called := false
	setEnv := func(name, value string) error {
		called = true
		return nil
	}

	if _, err := inspectGoEnvironmentWith(fakeGoEnvironmentOutput("/outside/go.work\n", ""), setEnv); err == nil {
		t.Fatal("外部Go workspaceを受理しました")
	}
	if called {
		t.Fatal("ambient環境のreject前にprocess environmentを書き換えました")
	}
}

func TestInspectGoEnvironmentFailsClosedWhenChildEnvironmentCannotBeFixed(t *testing.T) {
	calls := 0
	setEnv := func(name, value string) error {
		calls++
		if name == "GOENV" {
			return errors.New("setenv failed")
		}
		return nil
	}

	if _, err := inspectGoEnvironmentWith(fakeGoEnvironmentOutput("", ""), setEnv); err == nil {
		t.Fatal("child Go environment固定失敗をerrorにしていません")
	}
	if calls != 3 {
		t.Fatalf("setenv calls=%d want=3", calls)
	}
}
