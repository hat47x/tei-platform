# Repositoryの依存方向を守る最小ガード（2026-10）

- 状態: 実装候補。正式なRepository Health Evidenceは未取得
- 対象: RQ-023（継続開発・保守性）。Plugin拡張性はRQ-031〜033とも関連
- 作業: GitHub #980
- 観測基線: main `f27c45f718e288b2ee99f1e09eab88f946ca77d4`（2026-10-08）
- 正本との関係: 技術構造の正本は `docs/architecture.md`、開発・統合手順の正本は `AGENTS.md` / `CONTRIBUTING.md`。この文書は依存検査の対象と限界を説明するもので、新しいProduct RequirementやPlugin Contractを定義しない。

## 背景

TEIでは業務上の意味、実装、実行権限、観測結果を区別する。標準機能を含めたPlugin化や、Web・Desktop等への展開を進めても、その区別がソースコードの参照関係で崩れてはならない。

一方、現在は多数の作業レーンが進行している。リポジトリを一括分割したり、全Goパッケージを移動したりすると、機能改善とは無関係な差分によって競合やqualificationの再取得を増やしてしまう。まずは既に成立している小さな依存境界を検査し、後続変更が不用意に逆流させないことを優先する。

## 観測と判断

2026-10-08時点のmain treeには、`internal/` 397 files（production Go 172、test Go 225）、`tools/` 368 files、`plan/` 338 filesが存在した。`go.mod`は単一の `module tei` である。これらはディレクトリ規模の観測であって、コード量や文書量だけから構成不良と判定するものではない。

代表的なGo source importを確認したところ、`internal/core`と`internal/subject`は他のTEI内部packageに依存せず、`internal/permission`はcoreだけに依存する。`internal/semantic`配下ではcore / semanticへの依存に収まり、`internal/stdlib`はcore / semanticだけを参照する。

対照的に、`internal/plugin/model.go`はdatabase/migration、deployment、environment、permission等の型を参照している。これを現時点で単純に「違反」と定義すると、既存のCapability表現を壊す。汎用Capability Contractへの移行判断は #951 とDraft PR #963の担当範囲であり、本ガードには先取りしない。

全パッケージの依存グラフは未取得であり、循環やすべての望ましくない依存が解消されたと主張しない。Go自体が検出するimport cycleと、アーキテクチャ上好ましくないがコンパイル可能な依存は別問題である。

## 防ぎたい失敗

- 共通ID等の最下層へPluginやRuntimeの型が流入し、将来のTarget実装まで引きずる。
- Semantic modelがDatabase / Deployment等の実現方式へ依存し、意味の定義だけを独立に読み込めなくなる。
- `internal/`から実行用`cmd/`や検証用`tools/`をimportして、ReferenceやCLIの実装が製品の依存となる。
- 実装変更に伴う依存の逆流が、レビューでは見落とされ、後の分割・self-build段階まで蓄積する。

## 最小の検査契約

`tools/repository-architecture-check/dependency_test.go`はGo標準ライブラリのparserを使い、`internal/`配下のproduction `.go`を再帰的に調べる。`_test.go`の依存は製品依存と区別する。

| Source | 現段階で許容するTEI内部の直接import |
| --- | --- |
| `internal/core/**` | なし |
| `internal/subject/**` | なし |
| `internal/permission/**` | `internal/core` |
| `internal/semantic/**` | `internal/core` と `internal/semantic/**` |
| `internal/stdlib/**` | `internal/core` と `internal/semantic` |

さらに、すべての`internal/**`から `tei/cmd/**`、`tei/tools/**`、`tei/examples/**`、`tei/reference/**` へのGo package importを禁止する。Go標準ライブラリや外部ライブラリについては、この検査で新しい禁止規則を設けない。

拒否したimportはsource package、import path、理由を報告する。parseまたはfilesystemの失敗も検査成功として扱わない。検査するローカルimportのprefixは現在の`go.mod`の`module`宣言から取得する。将来module pathを変更しても、以前の`tei/`という文字列のままにして依存検査が無効化されることを防ぐ。

通常の `go test -count=1 ./...` に含まれるため、従来の `tools/repository-health-check.sh` の仕様は変更しない。CLIの成否だけでなく、正しいrepository revisionに対する全検査PASSが必要であるという既存Gateも変更しない。

## 検証の範囲

実装候補はGo 1.23.2の隔離した模擬moduleで単体テストを行った。

- 許容した依存でPASS
- `internal/core`から`internal/plugin`へ不正importを注入するとFAIL
- 不正importを戻すとPASS

これは分類器とsource走査の局所検証に限る。**TEIのactual main / PR head全体のGo test、vet、Workspace validationを通過したEvidenceではない。** PRのmerge前には、対象exact headのtrusted full checkoutでRepository Healthの全項目を実行する必要がある。

## 段階的な次の判断

1. この狭い依存ガードをfull Repository Healthへ組み込み、既存mainや並行PRに不意のfailを起こさないことを確認する。
2. #951のCapability Contract移行が成立した後、Plugin descriptorとimplementationの参照方向について別の機械検査を検討する。
3. 全packageのimport graph、変更の集中箇所、Referenceからproductionへの移行例を取得し、必要な制約だけ増やす。
4. 外部利用が生じるまでは`module tei`の置換、多module化、リポジトリ分割は保留する。

## 非目標

本ガードはRuntime Permission、Plugin Authority、Provider Conformance、外部実装のsource identityを検査しない。また、Plugin packageとphysical process / Wasm artifactが一対一であることを要求しない。Source importがきれいでもRuntimeの振る舞いまで安全・適合済みとはいえない。
