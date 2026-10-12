//go:build tei_api_only && tei_order_only && !tei_repair_only

package main

import (
	"fmt"
	"net"
	"strconv"
)

// The API-only first-Web reference uses synthetic authentication fixtures.
// Neither a matching Handover B/R nor Target build T grants a network-facing
// deployment. Until real Host/TLS/IdP authority exists, expose this diagnostic
// API only over an explicitly numerical loopback listen address.
//
// This verifies the program's chosen bind address, NOT host isolation:
// local forwarders, containers and reverse proxies may still expose it.
func validateReferenceListenAddress(address string) error {
	host, portText, err := net.SplitHostPort(address)
	if err != nil || (host != "127.0.0.1" && host != "::1") {
		return fmt.Errorf("diagnostic API-only Target requires explicit loopback -listen address")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("diagnostic API-only Target requires a TCP port in 1..65535")
	}
	return nil
}
