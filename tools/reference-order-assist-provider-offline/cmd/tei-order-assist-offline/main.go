// Command tei-order-assist-offline runs the external credit provider process
// and the saga worker of the order-assist-provider steps 7 and 8 sample.
package main

import (
	"os"

	orderassistoffline "tei/tools/reference-order-assist-provider-offline"
)

func main() { os.Exit(orderassistoffline.Main(os.Args[1:], os.Stdout, os.Stderr)) }
