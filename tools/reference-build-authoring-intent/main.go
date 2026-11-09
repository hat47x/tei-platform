// reference-build-authoring-intent は、人・生成AIが書くBuild Intentから
// exact BuildPlanを導出する境界のRead-only Reference（GitHub #1109）である。
//
// Source、Catalog、Intentを読むだけで、ファイルの作成・変更やネットワーク
// 接続は行わない。導出結果（Receipt）は標準出力へ出す。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reference-build-authoring-intent", flag.ContinueOnError)
	fs.SetOutput(stderr)
	intentPath := fs.String("intent", "", "Build Intent JSON")
	catalogPath := fs.String("catalog", "", "Builder Provider / Build依存 / Target CatalogのJSON")
	sourceRoot := fs.String("source-root", "", "Source参照を解決するroot directory")
	verifyPath := fs.String("verify", "", "保存済みの導出Receiptを再導出結果と照合する（指定時は照合だけ行う）")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *intentPath == "" || *catalogPath == "" || *sourceRoot == "" {
		fmt.Fprintln(stderr, "-intent, -catalog, -source-root are required")
		return 2
	}
	intentRaw, err := os.ReadFile(*intentPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	intent, err := ParseIntent(intentRaw)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	catalogRaw, err := os.ReadFile(*catalogPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	catalog, err := ParseFixtureCatalog(catalogRaw)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	ctx := NewFixtureContext(*sourceRoot, catalog)
	if *verifyPath != "" {
		stored, err := os.ReadFile(*verifyPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := VerifyDerivedReceipt(intent, ctx, stored); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(stdout, "derived receipt matches the intent resolution")
		return 0
	}
	receipt, err := Resolve(intent, ctx)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	raw, err := MarshalReceipt(receipt)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, string(raw))
	return 0
}
