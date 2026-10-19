package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	exportpackage "tei/tools/reference-export-package"
)

// tei-export-package builds or verifies a Reference delivery package. It does
// not sign the package or install it on a customer host.
func main() {
	if len(os.Args) < 2 || (os.Args[1] != "export" && os.Args[1] != "verify") {
		fmt.Fprintln(os.Stderr, "usage: tei-export-package <export|verify> [-root DIR] [-out FILE] [-package FILE]")
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	root := fs.String("root", "", "package root")
	out := fs.String("out", "", "output zip")
	packagePath := fs.String("package", "", "package zip to verify")
	_ = fs.Parse(os.Args[2:])
	switch command {
	case "export":
		if *root == "" || *out == "" {
			fail(fmt.Errorf("-root and -out are required for export"))
		}
		manifest, err := exportpackage.Export(*root, *out)
		if err != nil {
			fail(err)
		}
		emit(manifest)
	case "verify":
		if *packagePath == "" {
			fail(fmt.Errorf("-package is required for verify"))
		}
		manifest, err := exportpackage.Verify(*packagePath)
		if err != nil {
			fail(err)
		}
		emit(manifest)
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
