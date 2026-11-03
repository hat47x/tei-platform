//go:build !js && !wasip1 && !plan9

package operations

import "os"

func openWorkspaceSourceRoot(root string) (*os.Root, error) { return os.OpenRoot(root) }
