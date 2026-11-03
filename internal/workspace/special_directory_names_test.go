package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// specialDirectoryNames are valid on Windows and POSIX and contain characters
// that are glob metacharacters ([ ] ^ and so on) or otherwise unusual. A unit
// directory with such a name must be listed exactly like an ordinary one.
var specialDirectoryNames = []string{
	"u[1]", "u{a,b}", "u^caret", "u!bang", "u space", "uéè", "u[", "u]", "u[a-z]", "u{", "u}",
}

// renameUnitDirectories renames every unit directory below each parent
// (relative, slash separated; a missing parent is skipped) to
// "<NN>-<special>" so the sort order of the units is unchanged. It returns
// new relative path -> old relative path.
func renameUnitDirectories(t *testing.T, root string, parents []string) map[string]string {
	t.Helper()
	renamed := map[string]string{}
	counter := 0
	for _, parent := range parents {
		dir := filepath.Join(root, filepath.FromSlash(parent))
		entries, err := os.ReadDir(dir)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			newName := fmt.Sprintf("%02d-%s", counter, specialDirectoryNames[counter%len(specialDirectoryNames)])
			counter++
			if err := os.Rename(filepath.Join(dir, e.Name()), filepath.Join(dir, newName)); err != nil {
				t.Fatalf("rename %s/%s: %v", parent, e.Name(), err)
			}
			renamed[parent+"/"+newName] = parent + "/" + e.Name()
		}
	}
	return renamed
}

func restoreRenamedPath(path string, renamed map[string]string) string {
	for newPath, oldPath := range renamed {
		if path == newPath || strings.HasPrefix(path, newPath+"/") {
			return oldPath + strings.TrimPrefix(path, newPath)
		}
	}
	return path
}

func compareWithOrdinaryNames(t *testing.T, ordinary, special Result, renamed map[string]string) {
	t.Helper()
	if len(ordinary.Sources) == 0 {
		t.Fatal("ordinary Workspace has no Sources; the comparison proves nothing")
	}
	if len(special.Sources) != len(ordinary.Sources) {
		t.Errorf("Sources = %d, want %d (units with special names were silently omitted)", len(special.Sources), len(ordinary.Sources))
	}
	if len(special.ManagedFiles) != len(ordinary.ManagedFiles) {
		t.Errorf("ManagedFiles = %d, want %d", len(special.ManagedFiles), len(ordinary.ManagedFiles))
	}
	ordinaryKeys, specialKeys := sourceKeys(ordinary.Sources), sourceKeys(special.Sources)
	if !reflect.DeepEqual(specialKeys, ordinaryKeys) {
		t.Errorf("Source keys differ: only in ordinary %v; only in special %v",
			difference(ordinaryKeys, specialKeys), difference(specialKeys, ordinaryKeys))
	}
	for key, location := range special.Sources {
		want, ok := ordinary.Sources[key]
		if !ok {
			continue
		}
		if got := restoreRenamedPath(location.File, renamed); got != want.File {
			t.Errorf("Source %s file = %s (restored %s), want %s", key, location.File, got, want.File)
		}
	}
	worldSpecial, err := json.Marshal(special.World)
	if err != nil {
		t.Fatal(err)
	}
	worldOrdinary, err := json.Marshal(ordinary.World)
	if err != nil {
		t.Fatal(err)
	}
	if string(worldSpecial) != string(worldOrdinary) {
		t.Errorf("World differs from the same content under ordinary names (JSON lengths %d vs %d)", len(worldSpecial), len(worldOrdinary))
	}
}

// Unit directories whose name contains glob metacharacters must be loaded.
// The expected result is the same content under ordinary names (compared by
// semantic content; paths are mapped back to the ordinary names first).
func TestLoadListsUnitDirectoriesWithGlobMetacharactersInTheirNames(t *testing.T) {
	ordinary, err := Load(copyReferenceWorkspace(t))
	if err != nil {
		t.Fatal(err)
	}
	specialRoot := copyReferenceWorkspace(t)
	renamed := renameUnitDirectories(t, specialRoot, []string{
		"applications", "databases", "environments",
		"releases/applications", "releases/databases", "releases/solutions",
	})
	if len(renamed) == 0 {
		t.Fatal("no unit directory was renamed; the test proves nothing")
	}
	special, err := Load(specialRoot)
	if err != nil {
		t.Fatalf("Load of a Workspace with special unit directory names: %v", err)
	}
	compareWithOrdinaryNames(t, ordinary, special, renamed)
}

// The directories one level below a unit (migrations/<id>/migration.tei) are
// unit directories too and are matched by name, never by a pattern.
func TestLoadListsNestedUnitDirectoriesWithGlobMetacharactersInTheirNames(t *testing.T) {
	ordinary, err := Load(copyReferenceWorkspace(t))
	if err != nil {
		t.Fatal(err)
	}
	specialRoot := copyReferenceWorkspace(t)
	renamed := renameUnitDirectories(t, specialRoot, []string{"databases/master-data/migrations", "databases/sales-core/migrations"})
	if len(renamed) == 0 {
		t.Fatal("no migration directory was renamed; the test proves nothing")
	}
	special, err := Load(specialRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(ordinary.World.Migrations) == 0 {
		t.Fatal("reference Workspace has no Migrations; the test proves nothing")
	}
	compareWithOrdinaryNames(t, ordinary, special, renamed)
}

func sourceKeys(m map[string]SourceLocation) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func difference(a, b []string) []string {
	in := map[string]bool{}
	for _, x := range b {
		in[x] = true
	}
	var out []string
	for _, x := range a {
		if !in[x] {
			out = append(out, x)
		}
	}
	return out
}
