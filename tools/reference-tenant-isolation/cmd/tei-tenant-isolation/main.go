package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	tenantisolation "tei/tools/reference-tenant-isolation"
)

// tei-tenant-isolation checks one request against one tenant scope. The library
// is the foundation an application uses; this CLI only demonstrates the
// boundary.
func main() {
	command := "check"
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "check" {
		args = args[1:]
	}
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	tenant := fs.String("tenant", "", "current tenant id")
	root := fs.String("root", "", "tenant root directory")
	requested := fs.String("request", "", "tenant named by the request")
	_ = fs.Parse(args)
	if *tenant == "" || *root == "" || *requested == "" {
		fail(fmt.Errorf("-tenant, -root, and -request are required"))
	}
	scope, err := tenantisolation.Open(*tenant, *root)
	if err != nil {
		fail(err)
	}
	defer scope.Close()
	denial := scope.RequireTenant(*requested)
	result := map[string]any{
		"tenant":    *tenant,
		"requested": *requested,
		"allowed":   denial == nil,
	}
	if denial != nil {
		result["denial"] = denial
	}
	emit(result)
	if denial != nil {
		os.Exit(1)
	}
}

func emit(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
