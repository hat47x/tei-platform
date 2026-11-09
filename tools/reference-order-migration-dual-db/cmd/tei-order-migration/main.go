// Command tei-order-migration runs steps 1-3 of the order-migration-dual-db
// technical validation sample against three dedicated PostgreSQL databases.
package main

import (
	"os"

	ordermigrationdualdb "tei/tools/reference-order-migration-dual-db"
)

func main() {
	os.Exit(ordermigrationdualdb.Main(os.Args[1:], os.Stdout, os.Stderr))
}
