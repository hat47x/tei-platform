package workspace

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const raceTestPlugin = `plugin tei.example v1.0.0 {
    runtime builtin
    provides deployment-executor runtime.expected {
        action application-start
    }
}
`

const raceTestImplementation = `plugin-implementation tei.example v1.0.0 {
    profile static-response-v1
    capability deployment-executor runtime.expected
    operation execute json { {"attemptId":"a","outcome":"accepted"} }
}
`

func writePluginImplementationRaceWorkspace(t *testing.T, root, implementationDir string) {
	t.Helper()
	files := map[string]string{
		"tei.workspace.yaml":               "workspace: impl_race\nformatVersion: 1\nsources:\n  - plugins\n  - plugin-implementations\n",
		"plugins/example/1.0.0/plugin.tei": raceTestPlugin,
		"plugin-implementations/" + implementationDir + "/implementation.tei": raceTestImplementation,
	}
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// The plugin-implementations listing must come from the Root pinned by Load,
// not from the Workspace pathname. After the pin, the alias is retargeted to a
// directory whose implementation lives under another name: an unpinned listing
// would see that name and fail to read it through the pinned Root.
func TestLoadListsPluginImplementationsFromPinnedRootWhenAliasIsRetargeted(t *testing.T) {
	base := t.TempDir()
	original := filepath.Join(base, "original")
	replacement := filepath.Join(base, "replacement")
	writePluginImplementationRaceWorkspace(t, original, "pinned-name/1.0.0")
	writePluginImplementationRaceWorkspace(t, replacement, "swapped-name/1.0.0")
	alias := filepath.Join(base, "alias")
	makeDirAlias(t, alias, original)

	retargeted := false
	result, err := loadWithStackHooks(alias, map[string]bool{}, loadHooks{afterPin: func() {
		if err := os.Remove(alias); err != nil {
			t.Fatalf("remove alias: %v", err)
		}
		makeDirAlias(t, alias, replacement)
		retargeted = true
	}})
	if !retargeted {
		t.Fatal("the alias was not retargeted; the test proves nothing")
	}
	if err != nil {
		t.Fatalf("Load must list plugin implementations from the pinned original directory, got error: %v", err)
	}
	var files []string
	for _, location := range result.PluginImplementationSources {
		files = append(files, location.File)
	}
	sort.Strings(files)
	want := []string{"plugin-implementations/pinned-name/1.0.0/implementation.tei"}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("plugin implementation sources = %v, want %v", files, want)
	}
	for _, f := range files {
		if strings.Contains(f, "swapped-name") {
			t.Fatalf("listing followed the retargeted alias: %s", f)
		}
	}
}
