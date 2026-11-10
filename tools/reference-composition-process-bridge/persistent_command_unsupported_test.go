//go:build !linux

package compositionprocessbridge

import (
	"context"
	"testing"
)

func TestPersistentCommandRejectsUnverifiedPlatform(t *testing.T) {
	cmd, closeImage, err := preparePersistentCommand(context.Background(), "untrusted-path", "untrusted-digest", nil)
	if err == nil || cmd != nil || closeImage != nil {
		t.Fatal("unverified platform admitted a path-based fallback")
	}
}
