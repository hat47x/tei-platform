# reference-build-plan-authoring

ある package と、その **module 内の閉包**を名指す `BuildPlan` を書く参照実装です（[#2831](https://github.com/hat47x/tei-platform/issues/2831) SB-C0 §4.49）。

## なぜ要るか

§4.46 は「計画は、ビルドが読む source だけを名指す」と決めました。`./cmd/tei` のような対象は **module 内だけでも73の package** を読みます。手で並べるのは現実的でなく、**抜ければ §4.41 の被覆検査で止まります**（正しく止まるが、計画が書けない）。

このツールは `go list -deps` から閉包を取り、**各 source の digest を resolver で計算して**計画にします。**digest を宣言することはできません**（実ファイルから計算する）。

## 使い方

```bash
go run ./tools/reference-build-plan-authoring \
  -repo . \
  -entry ./cmd/tei \
  -out /tmp/plan.json
```

| flag | 内容 |
| --- | --- |
| `-repo` | source を解決する root（既定は現在地） |
| `-entry` | ビルドする package |
| `-out` | 書く計画のパス |

出力は「`sources: 75 (go-package 73, go-module 2)`」のような1行です。

## 何を名指すか

| kind | 対象 |
| --- | --- |
| `go-package` | `go list -deps` が返す **module 内**の package（標準ライブラリと外部 module は含めない） |
| `go-module` | `go.mod`・`go.sum` |

## 何を主張しないか

- **計画の正しさ**。閉包が「読むもの全部」であることは、ビルド時の被覆検査（§4.41）が確かめます
- ビルド・受理・Runtime Authority。**計画を書くだけ**です
- 外部 module の完全性。供給の Trust の範囲です

## 試験

```bash
go test ./tools/reference-build-plan-authoring -count=1
```
