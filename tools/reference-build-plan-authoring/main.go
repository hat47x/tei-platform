// reference-build-plan-authoring は、あるpackageとmodule内の閉包を名指す
// BuildPlanを書きます（GitHub #2831 SB-C0 §4.49）。
//
// 計画は「ビルドが読むsourceだけ」を名指す必要がある（§4.46）。手で73件を
// 並べるのは現実的でないので、`go list -deps` から閉包を取り、各sourceの
// digestをresolverで計算して計画にする。digestを宣言することはできない。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"tei/internal/buildintent"
	"tei/internal/plugin"
)

func main() {
	code, err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if code != 0 {
		os.Exit(code)
	}
}

func run(args []string) (int, error) {
	fs := flag.NewFlagSet("reference-build-plan-authoring", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository root")
	entry := fs.String("entry", "./cmd/tei", "package to build")
	out := fs.String("out", "", "plan path")
	if err := fs.Parse(args); err != nil {
		return 2, err
	}
	if *out == "" {
		return 2, fmt.Errorf("-out is required")
	}
	root, err := filepath.Abs(*repo)
	if err != nil {
		panic(err)
	}

	cmd := exec.Command("go", "list", "-deps", "-f", "{{if not .Standard}}{{.Dir}}{{end}}", *entry)
	cmd.Dir = root
	raw, err := cmd.Output()
	if err != nil {
		return 1, fmt.Errorf("go list: %w", err)
	}
	dirs := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		dir := strings.TrimSpace(line)
		if dir == "" || !strings.HasPrefix(dir, root) {
			continue
		}
		dirs[dir] = true
	}
	ordered := make([]string, 0, len(dirs))
	for dir := range dirs {
		ordered = append(ordered, dir)
	}
	sort.Strings(ordered)

	resolver := buildintent.DirSourceResolver{Root: root}
	sources := make([]plugin.BuildSource, 0, len(ordered)+2)
	index := 0
	for _, dir := range ordered {
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			panic(err)
		}
		ref := filepath.ToSlash(rel)
		resolved, err := resolver.Resolve(buildintent.IntentSource{Kind: "go-package", ID: fmt.Sprintf("pkg%d", index), Ref: ref})
		if err != nil {
			return 1, fmt.Errorf("resolve %s: %w", ref, err)
		}
		sources = append(sources, plugin.BuildSource{Kind: "go-package", ID: fmt.Sprintf("pkg%d", index), Locator: ref, Digest: resolved.Source.Digest})
		index++
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		resolved, err := resolver.Resolve(buildintent.IntentSource{Kind: "go-module", ID: name, Ref: name})
		if err != nil {
			return 1, fmt.Errorf("resolve %s: %w", name, err)
		}
		sources = append(sources, plugin.BuildSource{Kind: "go-module", ID: name, Locator: name, Digest: resolved.Source.Digest})
	}

	plan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: "tei.command.tei", Version: "0.0.0"},
		PluginDefinitionFingerprint: strings.Repeat("a", 64),
		Sources:                     sources,
		Builder: plugin.BuilderIdentity{
			ID: "tei.builder.go", Version: "1.2.0", ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest: "sha256:" + strings.Repeat("b", 64),
		},
		Recipe:     plugin.BuildRecipe{Kind: "go-command", Entrypoint: *entry},
		OutputKind: plugin.BuildOutputRuntimeArtifact,
		Target:     "host/linux/amd64",
	}
	if err := plan.Validate(); err != nil {
		return 1, fmt.Errorf("plan does not validate: %w", err)
	}
	body, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*out, body, 0o600); err != nil {
		panic(err)
	}
	fmt.Printf("sources: %d (go-package %d, go-module 2)\n", len(sources), index)
	return 0, nil
}
