// tei-executor-id-resolution prints whether the Environment's binding selects a
// deployment executor that the plugin lock actually admits. It reads only.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
	"tei/internal/workspace"
	executoridresolution "tei/tools/reference-executor-id-resolution"
)

func main() {
	root := flag.String("workspace", "", "Source Workspace root to read")
	environmentID := flag.String("environment", "", "Environment whose binding is resolved")
	lockPath := flag.String("plugin-lock", "", "plugin lock JSON for this environment")
	action := flag.String("action", "", "deployment action kind, e.g. database-release")
	database := flag.String("database", "", "target database, when the binding is scoped")
	application := flag.String("application", "", "target application, when the binding is scoped")
	hardCoded := flag.String("hard-coded-id", "", "id the reference implementation currently embeds, for comparison")
	flag.Parse()
	if *root == "" || *environmentID == "" || *action == "" {
		fail(fmt.Errorf("-workspace, -environment and -action are required"))
	}

	loaded, err := workspace.Load(*root)
	if err != nil {
		fail(fmt.Errorf("load Workspace: %w", err))
	}
	env, ok := findEnvironment(loaded.World.Environments, *environmentID)
	if !ok {
		fail(fmt.Errorf("environment %s not found", *environmentID))
	}

	// An absent -plugin-lock is reported honestly: the plan cannot admit
	// anything without a lock, and this reference does not invent one.
	var lock plugin.Lock
	if *lockPath != "" {
		raw, readErr := os.ReadFile(*lockPath)
		if readErr != nil {
			fail(fmt.Errorf("read plugin lock: %w", readErr))
		}
		if decodeErr := json.Unmarshal(raw, &lock); decodeErr != nil {
			fail(fmt.Errorf("decode plugin lock: %w", decodeErr))
		}
	}

	report, err := executoridresolution.Resolve(executoridresolution.Input{
		WorkspaceRoot: *root,
		Environment:   env,
		Action:        *action,
		Lock:          lock,
		HardCodedID:   *hardCoded,
		Database:      coreID(*database),
		Application:   coreID(*application),
	})
	if err != nil {
		fail(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fail(err)
	}
}

func findEnvironment(envs []environment.Environment, id string) (environment.Environment, bool) {
	for _, env := range envs {
		if string(env.ID) == id {
			return env, true
		}
	}
	return environment.Environment{}, false
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(2) }

func coreID(s string) core.CanonicalID { return core.CanonicalID(s) }
