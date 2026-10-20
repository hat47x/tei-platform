// Command tei-month-end-close runs steps 1-3 of the month-end-close-correction
// technical validation sample against a dedicated PostgreSQL database.
package main

import (
	"os"

	monthendclose "tei/tools/reference-month-end-close-correction"
)

func main() {
	os.Exit(monthendclose.Main(os.Args[1:], os.Stdout, os.Stderr))
}
