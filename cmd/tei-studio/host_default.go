//go:build !studio_generated

package main

import "tei/internal/studiohost"

// Ordinary builds keep the existing hand-written, read-only Operation route.
func newStudioHost(paths []string) (*studiohost.ReadOnly, error) {
	return studiohost.NewReadOnly(paths)
}
