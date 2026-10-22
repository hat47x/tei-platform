// Command tei-order-data-realization runs the order-data-realization chain.
package main

import (
	"os"

	orderdatarealization "tei/tools/reference-order-data-realization"
)

func main() {
	os.Exit(orderdatarealization.Main(os.Args[1:], os.Stdout, os.Stderr))
}
