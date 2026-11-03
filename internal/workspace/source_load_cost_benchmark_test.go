package workspace_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

// BenchmarkLoadManagedWorkspaceFiles measures Load over synthetic Workspaces
// with many managed semantic files. It is opt-in (benchmarks do not run in a
// normal `go test`), and it is the measurement behind
// plan/evidence/WORKSPACE_SOURCE_READER_ENV_EVIDENCE_2026-11.md.
func BenchmarkLoadManagedWorkspaceFiles(b *testing.B) {
	for _, files := range []int{50, 200, 800, 3200} {
		b.Run(fmt.Sprintf("files=%d", files), func(b *testing.B) {
			root := b.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "semantic"), 0o755); err != nil {
				b.Fatal(err)
			}
			manifest := "workspace: load_cost\nformatVersion: 1\nsources:\n  - semantic\n"
			if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o644); err != nil {
				b.Fatal(err)
			}
			body := make([]byte, 0, 48)
			for i := 0; i < files; i++ {
				body = body[:0]
				body = fmt.Appendf(body, "namespace bench%d\n", i)
				path := filepath.Join(root, "semantic", fmt.Sprintf("f%05d.tei", i))
				if err := os.WriteFile(path, body, 0o644); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := workspace.Load(root); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
