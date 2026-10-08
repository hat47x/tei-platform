# reference/ — 再現可能な入力・fixture・golden

このルールは`reference/`以下に適用する。ルート`../AGENTS.md`と`README.md`を先に読む。

## 階層別の管理

- `workspace/`は実際に`tei`が読み込むReference Workspaceのauthoring source。`workspace/.tei/plugin-lock.json`などのLockや`reference/world.json`は、そのsourceから導出されるgoldenであり、独立の編集正本ではない。
- `observed/`などは外部状態の観測・固定fixture。`legacy-auth/`、`business-system-design/`、`reference-model-transfer/`、`sei-adoption/`、`sui-adoption/`、`ui-customization/`、`plugin-self-build/`等は**用途別の検証素材**。別アプリ・別Provider・本番環境への一般的な成功証明ではない。
- 意図した失敗・警告、未解決状態、反例も検証契約の一部。データを見栄えのために補正して成功例に変えない。秘密情報や実利用者の機密をfixtureに含めない。

## 変更時の規則

1. 入力source、期待出力、生成手順、依存artifact（Lock / World / Durable Runなど）を特定する。生成物だけの手動更新で差分を隠さない。
2. `go run ./cmd/tei validate -workspace reference/workspace`や`go run ./tools/reference-golden`など、対象に対応する再現手順で比較する。Goldenを更新する場合は正規生成経路を使い、差分の意味と依存関係をレビューする。
3. Fixtureの種別、由来、観測時点、環境、検証した境界を明示する。synthetic / mocked / historical / actual providerの違いを保ち、未実行の検証をPASSと書かない。
4. Referenceの変更でGo codeも変わる場合は`../internal/AGENTS.md`とルートの統合Gateを適用する。文書・fixtureだけの変更でも参照リンクと正本性・Gate影響を確認する。
