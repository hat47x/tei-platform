// reference-go-module-closure-preflight は、Go対象の選択Module閉包を
// オフラインで分類する読取り専用の入口です（GitHub #2831 SB-0a）。
// Build を実行せず、artifact を作らず、権限も与えません。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"

	"tei/tools/internal/gobuilder"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reference-go-module-closure-preflight", flag.ContinueOnError)
	fs.SetOutput(stderr)
	repo := fs.String("repo", ".", "対象リポジトリのcheckout")
	pkg := fs.String("package", "", "対象packageのimport path（必須）")
	goExecutable := fs.String("go", "", "exact Go実行ファイル（省略時は現在のruntime GOROOT）")
	stateDir := fs.String("state", "", "分離した観測用stateディレクトリ（省略時は一時ディレクトリ）")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *pkg == "" {
		fmt.Fprintln(stderr, "-package is required")
		return 2
	}

	goPath, err := gobuilder.SelectExecutable(*goExecutable, runtime.GOROOT(), runtime.GOOS)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	state := *stateDir
	if state == "" {
		dir, err := os.MkdirTemp("", "tei-go-module-preflight-*")
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		defer os.RemoveAll(dir)
		state = dir
	}

	preflight, err := gobuilder.ResolveModuleClosurePreflight(*repo, goPath, *pkg, state)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(preflight); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	// 分類は観測記録であり、対象の受理可否ではない。分類がsupportedでも
	// artifact成立や権限成立を意味しないため、終了コードは観測結果に
	// 依存させず0を返す。
	return 0
}
