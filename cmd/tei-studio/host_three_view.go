package main

import (
	"tei/internal/studiohost"
	"tei/internal/threeviewadapter"
	xteathreeview "tei/plugins/xtea-three-view"
)

// studioThreeViewInventory is the three-view Provider inventory compiled into
// the normal Studio: the read-only X-TEA three-view plugin and nothing else.
// The X-TEA reading lives in that plugin; this Host only selects and calls it.
func studioThreeViewInventory() []threeviewadapter.Provider {
	return []threeviewadapter.Provider{{
		ID:       xteathreeview.ProviderID,
		Entry:    xteathreeview.Entry,
		Navigate: xteathreeview.Navigate,
	}}
}

// newStudioThreeViewHost selects the plugin through the composition Registry.
// This is a static local composition, not dynamic Plugin deployment.
func newStudioThreeViewHost() (*threeviewadapter.Host, error) {
	return threeviewadapter.NewHostSelecting(xteathreeview.ProviderID, studioThreeViewInventory())
}

// composeStudioThreeView gives the read-only host the three-view Provider the
// composing Host already selected. The renderer supplies a workspace ID and a
// source path only; it cannot name a Provider, and every root reaches the
// Provider through the host's startup allowlist.
func composeStudioThreeView(host *studiohost.ReadOnly, threeView *threeviewadapter.Host) *studiohost.ReadOnly {
	host.ThreeView = func(trustedRoot string) (studiohost.ThreeViewService, error) {
		return threeView.Bind(trustedRoot)
	}
	return host
}

// withStudioThreeView is the step both Studio host variants share.
func withStudioThreeView(host *studiohost.ReadOnly, err error) (*studiohost.ReadOnly, error) {
	if err != nil {
		return nil, err
	}
	threeView, err := newStudioThreeViewHost()
	if err != nil {
		return nil, err
	}
	return composeStudioThreeView(host, threeView), nil
}
