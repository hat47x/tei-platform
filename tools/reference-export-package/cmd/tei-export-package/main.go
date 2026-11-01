package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	exportpackage "tei/tools/reference-export-package"
)

// tei-export-package builds or verifies a Reference delivery package. It does
// not sign the package or install it on a customer host.
func main() {
	if len(os.Args) < 2 || (os.Args[1] != "export" && os.Args[1] != "verify") {
		fmt.Fprintln(os.Stderr, "usage: tei-export-package <export|verify> [-root DIR] [-out FILE] [-package FILE] [-include LIST] [-exclude LIST]")
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	root := fs.String("root", "", "package root")
	out := fs.String("out", "", "output zip")
	packagePath := fs.String("package", "", "package zip to verify")
	include := fs.String("include", "", "comma-separated categories to carry: meaning,data,authority,evidence")
	exclude := fs.String("exclude", "", "comma-separated categories to leave out")
	_ = fs.Parse(os.Args[2:])
	switch command {
	case "export":
		if *root == "" || *out == "" {
			fail(fmt.Errorf("-root and -out are required for export"))
		}
		var manifest exportpackage.Manifest
		var err error
		if *include == "" && *exclude == "" {
			manifest, err = exportpackage.Export(*root, *out)
		} else {
			manifest, err = exportpackage.ExportWithPolicy(*root, *out, exportpackage.Policy{
				Include: splitList(*include),
				Exclude: splitList(*exclude),
			})
		}
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

func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
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
