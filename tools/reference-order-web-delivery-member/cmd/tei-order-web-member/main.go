// Command tei-order-web-member runs one deployment unit (member or staff) of
// the order-web-delivery step-6 sample.
package main

import (
	"os"

	orderwebmember "tei/tools/reference-order-web-delivery-member"
)

func main() { os.Exit(orderwebmember.Main(os.Args[1:], os.Stdout, os.Stderr)) }
