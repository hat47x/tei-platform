//go:build tei_repair_only && !tei_order_only

package main

import "errors"

// The shared diagnostic server needs a typed configuration slot, but a
// repair-only executable must not import or link the order_entry session
// Command runner or the session HTTP/authorization host.
type protectedSessionCommandConfig struct{}

// Fail closed if an unsupported session Command is ever wired in this
// repair-only target. The type intentionally exposes no order-only features.
func (s *webTargetServer) configureProtectedSessionCommand() error {
	if s.protectedSessionOptionCount != 0 || s.protectedSessionConfig != nil {
		return errors.New("order_entry session command cannot be installed in repair_intake target")
	}
	return nil
}
