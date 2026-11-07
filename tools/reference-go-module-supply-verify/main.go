// reference-go-module-supply-verify verifies an offline Go module supply
// against the repository's go.sum and re-resolves the target from it (#2831
// SB-0b). It does not build, publish or accept anything.
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
	fs := flag.NewFlagSet("reference-go-module-supply-verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	repo := fs.String("repo", ".", "対象リポジトリのcheckout（go.sumとgo.modを読む）")
	supply := fs.String("supply", "", "GOPROXY=fileで指定するoffline供給ディレクトリ（必須）")
	pkg := fs.String("package", "", "再解決するpackageのimport path（必須）")
	goExecutable := fs.String("go", "", "exact Go実行ファイル（省略時は現在のruntime GOROOT）")
	stateDir := fs.String("state", "", "分離した観測用state（省略時は一時ディレクトリ）")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *supply == "" || *pkg == "" {
		fmt.Fprintln(stderr, "-supply and -package are required")
		return 2
	}
	goPath, err := gobuilder.SelectExecutable(*goExecutable, runtime.GOROOT(), runtime.GOOS)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	state := *stateDir
	if state == "" {
		dir, err := os.MkdirTemp("", "tei-go-module-supply-*")
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		defer os.RemoveAll(dir)
		state = dir
	}

	verification, err := VerifySupply(*repo, *supply, *pkg, goPath, state, nil)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(verification); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	// 終了コードは観測が行えたかだけを示す。verified=falseは「供給が違う」
	// という証拠であり、対象の受理可否ではない。
	return 0
}
