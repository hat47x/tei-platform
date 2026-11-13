package main

import (
	"encoding/json"
	"io"
)

// headlessThreeViewWorkspace is the only Workspace a headless read addresses:
// the first -workspace of this process.
const headlessThreeViewWorkspace = "workspace-1"

// runHeadlessThreeView answers one three-view navigation on w and returns the
// process exit code, without starting a window or a server.
//
// It calls the same StudioBridge method the renderer calls, on the same Host
// composition, so a command line or an AI tool reads through the Provider the
// Studio selected and gets the envelope the screen gets. It is the Studio
// binary that offers this, not the `tei` command: the `tei` command is built
// from the root module, which must build without the X-TEA source reader.
//
// Exit code 0 means the Operation answered. It does not mean the source is
// complete or accepted; the envelope's coverage and layers say that.
func runHeadlessThreeView(w io.Writer, bridge *StudioBridge, sourcePath, pinOccurrence, pinSourceDigest string) int {
	response := bridge.NavigateThreeView(headlessThreeViewWorkspace, sourcePath, pinOccurrence, pinSourceDigest)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(response); err != nil {
		return 2
	}
	if !response.OK {
		return 1
	}
	return 0
}
