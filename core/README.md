# コア

このディレクトリは、TEIのコアを独立したサブプロジェクトとして管理するための入口です。**Subject値型と一時Effect Scopeを先行移管した段階であり、コア全体をここだけで構築できる状態ではありません。**

コアは、サービス契約に基づく合成、依存関係の解決、実行世代の切り替え、信頼境界など、個別の実装に左右されない最小限の仕組みを担います。具体的な認証方式、データベース接続、画面・実行先固有の機構を抱え込まない方針です。

## 最初の実ソース移管

[Subject値型](subject/README.md)は、論理対象をソース位置から切り離して識別する小さな値型です。実装の正本を`core/subject/`へ移し、旧`internal/subject/`は型エイリアスの互換入口としました。旧型と新型はGo言語上同一で、値判定とJSON形式をテストしています。

`bash tools/check-core-subject-isolation.sh`は正本のソースだけを一時Goモジュールへ複製し、Go標準ライブラリ以外への依存を持たず、別モジュールの利用者から使えることをオフラインで検証します。この一例から、新Service Contractや合成基盤が製品コアへ移管済みとは判断しません。

[一時Effect Scope](effect/README.md)は、`internal/composition/scope.go`にあった、逆順の一時資源解放と失敗集約を担当する実装を移したものです。旧`composition.Scope`は`effect.Scope`への型エイリアスとなり、旧APIの利用者は同じ状態管理とエラーを引き続き使用します。合成基盤の契約選択・Host権限・世代管理は旧パッケージ側に残したままです。

`bash tools/check-core-isolation.sh`では、CompositionとEffectの実ソースを別々の一時Goモジュールへ取り込み、隔離した外部Consumerからもこの型の同一性を確認します。

## 現在の実装

現在の主な関連コードは、リポジトリ直下の次の場所にあります。

- [基本要素](../internal/core/)
- [合成基盤の試作](../internal/composition/)
- [従来のプラグイン管理機構](../internal/plugin/)

これらを機械的にまとめたものが、将来のコアになるわけではありません。既存実装の成立範囲は[実装状況](../docs/status.md)を参照してください。

## 独立構築の確認

合成基盤が他のTEIパッケージに依存せず試験できるかを調べるため、[隔離試験の手順](../plan/design/CORE_SUBPROJECT_ISOLATION_PROBE_2026-10.md)を用意しています。リポジトリのルートから`tools/check-core-isolation.sh`を実行します。これは現在の`internal/composition/`を一時的なモジュールへ複製して試すもので、`core/`への移管完了を意味しません。

## 構成の判断

- [サブプロジェクト化の設計判断](../plan/adr/0033-core-and-plugin-subprojects.md)
- [新しい合成基盤の設計判断](../plan/adr/0032-composition-kernel-replacement.md)
- [全体構成](../docs/architecture.md)

今後、公開契約、内部実装、ビルド入口、検証方法を確定してから段階的に移管します。移管が成立するまでは、従来の`go.mod`とテスト方法が有効です。
