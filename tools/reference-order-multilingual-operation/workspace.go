package ordermultilingualoperation

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/plugin"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// CopyWorkspace copies reference/workspace into dst. The fixed baseline is
// only read.
func CopyWorkspace(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// PluginSource renders an ordinary builtin plugin declaration. The plugins
// added by the sample declare nothing about localization.
func PluginSource(id plugin.Identity, requires ...string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "plugin %s v%s {\n    runtime builtin\n", id.ID, id.Version)
	for _, r := range requires {
		fmt.Fprintf(&b, "    requires plugin %s ^1\n", r)
	}
	b.WriteString("}\n")
	return b.String()
}

func pluginPath(root string, id plugin.Identity) string {
	dir := strings.NewReplacer(".", "-", "_", "-").Replace(string(id.ID))
	return filepath.Join(root, "plugins", dir, id.Version, "plugin.tei")
}

// WritePlugin adds or replaces one plugin declaration in the copy.
func WritePlugin(root string, id plugin.Identity, source string) error {
	path := pluginPath(root, id)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(source), 0o644)
}

// RemovePlugin removes a plugin declaration from the copy.
func RemovePlugin(root string, id plugin.Identity) error {
	return os.RemoveAll(filepath.Dir(pluginPath(root, id)))
}

// SetEnvironmentPlugin adds (present=true) or removes one `plugin X ^1`
// requirement in environments/<env>/environment.tei of the copy.
func SetEnvironmentPlugin(root string, env core.CanonicalID, id core.CanonicalID, present bool) error {
	path := filepath.Join(root, "environments", string(env), "environment.tei")
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	line := fmt.Sprintf("    plugin %s ^1", id)
	lines := strings.Split(string(raw), "\n")
	var out []string
	found := false
	for _, l := range lines {
		if strings.TrimRight(l, " ") == line {
			found = true
			if !present {
				continue
			}
		}
		out = append(out, l)
	}
	if present && !found {
		head := fmt.Sprintf("environment %s {", env)
		inserted := false
		var withLine []string
		for _, l := range out {
			withLine = append(withLine, l)
			if !inserted && strings.TrimSpace(l) == head {
				withLine = append(withLine, line)
				inserted = true
			}
		}
		if !inserted {
			return fmt.Errorf("environment header %q not found", head)
		}
		out = withLine
	}
	if !present && !found {
		return fmt.Errorf("environment %s has no %s", env, id)
	}
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644)
}

// Workspace is one load of the copy with the lock built from that same load.
type Workspace struct {
	Root          string
	Loaded        workspace.Result
	Catalog       plugin.Catalog
	Env           environment.Environment
	Lock          plugin.Lock
	LockFP        string
	Index         sourceindex.Index
	ModelRevision string
	Graph         modelref.Graph
}

// Open loads the copy, validates the World, builds a fresh lock over every
// environment and the typed graph from the Derived Index.
func Open(root string, u Unified) (Workspace, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return Workspace{}, fmt.Errorf("load: %w", err)
	}
	if validation.HasErrors(validation.Validate(loaded.World)) {
		return Workspace{}, fmt.Errorf("invalid world")
	}
	catalog := plugin.Catalog{Plugins: loaded.World.Plugins, Bundles: loaded.World.Bundles}
	lock, err := plugin.BuildLock(catalog, loaded.World.Environments)
	if err != nil {
		return Workspace{}, fmt.Errorf("lock: %w", err)
	}
	return openWithLock(root, loaded, lock, u)
}

// Reopen loads the copy again but keeps the given (possibly stale) lock.
func Reopen(root string, lock plugin.Lock, u Unified) (Workspace, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return Workspace{}, fmt.Errorf("load: %w", err)
	}
	return openWithLock(root, loaded, lock, u)
}

func openWithLock(root string, loaded workspace.Result, lock plugin.Lock, u Unified) (Workspace, error) {
	ws := Workspace{Root: root, Loaded: loaded, Lock: lock,
		Catalog: plugin.Catalog{Plugins: loaded.World.Plugins, Bundles: loaded.World.Bundles}}
	found := 0
	for _, e := range loaded.World.Environments {
		if e.ID == u.Environment {
			ws.Env = e
			found++
		}
	}
	if found != 1 {
		return Workspace{}, fmt.Errorf("environment %s found %d times", u.Environment, found)
	}
	fp, err := plugin.EnvironmentLockFingerprint(lock, u.Environment)
	if err != nil {
		return Workspace{}, fmt.Errorf("lock fingerprint: %w", err)
	}
	ws.LockFP = fp
	// IndexedLoadedSnapshot refuses this copy: reference/workspace has
	// databases/sales-core/README.md, which is managed but not parsed by Load
	// (PlatformGaps). BuildSourceIndex reads that file after Load.
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return Workspace{}, fmt.Errorf("source index: %w", err)
	}
	ws.Index = idx
	ws.ModelRevision = sourceindex.ModelRevision(idx)
	ws.Graph = derivedbridge.FromIndex(index.Build(loaded.World), u.Scope).Graph
	return ws, nil
}

// HasNode reports whether the Derived Index has the typed key.
func (w Workspace) HasNode(k modelref.Key) bool {
	for _, n := range w.Graph.Nodes {
		if n.Key == k {
			return true
		}
	}
	return false
}

// Resolved returns the plugin identities pinned for the environment.
func (w Workspace) Resolved() ([]plugin.Identity, error) {
	r, err := w.Catalog.ResolveEnvironmentWithLock(w.Env, w.Lock)
	if err != nil {
		return nil, err
	}
	return r.Plugins, nil
}

// LockedEntries returns "id@version=fingerprint" for the environment lock.
func LockedEntries(lock plugin.Lock, env core.CanonicalID) map[string]string {
	out := map[string]string{}
	for _, e := range lock.Environments {
		if e.Environment != env {
			continue
		}
		for _, p := range e.Plugins {
			out[string(p.ID)+"@"+p.Version] = p.Fingerprint
		}
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
