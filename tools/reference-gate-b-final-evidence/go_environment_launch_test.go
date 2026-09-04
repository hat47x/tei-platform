package main

import (
	"strings"
	"testing"
)

func TestInspectGoEnvironmentForLaunchRequiresExplicitGoEnvOff(t *testing.T) {
	setEnvCalled := false
	setEnv := func(name, value string) error {
		setEnvCalled = true
		return nil
	}

	for _, tc := range []struct {
		name   string
		value  string
		set    bool
	}{
		{name: "unset", set: false},
		{name: "empty", value: "", set: true},
		{name: "path", value: "/private/build/goenv", set: true},
		{name: "other", value: "on", set: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setEnvCalled = false
			environment, err := inspectGoEnvironmentForLaunch(
				fakeGoEnvironmentOutput("", ""),
				setEnv,
				func(string) (string, bool) { return tc.value, tc.set },
			)
			if err == nil {
				t.Fatal("GOENV=offでない起動を受理しました")
			}
			if environment.GoEnvDisabled {
				t.Fatalf("goEnvDisabled=true: %#v", environment)
			}
			if setEnvCalled {
				t.Fatal("GOENV launch guardでrejectする前にprocess environmentを書き換えました")
			}
			if strings.Contains(err.Error(), tc.value) && tc.value != "" {
				t.Fatalf("GOENV値をerrorへ露出しています: %q", err)
			}
		})
	}
}

func TestInspectGoEnvironmentForLaunchAcceptsExplicitGoEnvOff(t *testing.T) {
	var fixed int
	environment, err := inspectGoEnvironmentForLaunch(
		fakeGoEnvironmentOutput("off\n", ""),
		func(name, value string) error {
			fixed++
			return nil
		},
		func(string) (string, bool) { return "off", true },
	)
	if err != nil {
		t.Fatal(err)
	}
	if !environment.GoEnvDisabled {
		t.Fatalf("goEnvDisabled=false: %#v", environment)
	}
	if environment.ExternalWorkspaceActive || !environment.GoFlagsEmpty {
		t.Fatalf("Go environment Evidenceが不正です: %#v", environment)
	}
	if fixed != 3 {
		t.Fatalf("child environment固定回数=%d want=3", fixed)
	}
}
