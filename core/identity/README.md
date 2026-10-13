# Canonical Identity（M0互換）

`core/identity/`は、従来`internal/core/`にあった最小の論理識別子、参照、数値版、既定の名称導出について、**唯一のGo実装**を置く場所です。既存の製品用ソース`id.go`と`version.go`を移し、`internal/core`側は型エイリアスと委譲のみを保持します。

- `CanonicalID`、`LocalName`、`Kind`、`Ref`と、既存の六つの`Kind`定数。
- M0の`ParseCanonicalID`・`ParseLocalName`・`SymbolToSegment`・`ValidNumericVersion`。
- `Must...`系のpanic契約、および`Ref.String()`の文字列表現。

## 独立利用

リポジトリのルートで`bash tools/check-core-identity-isolation.sh`を実行すると、`core/identity`に加えて、先行移管した`core/subject`・`core/effect`の実ソースだけを別の一時Goモジュールへ複製し、ネットワーク取得を禁止した状態でテスト・Vetを実行します。独立した消費者モジュールでも、この三領域をimportして使います。常設の子`go.mod`は作りません。

## 境界と今後の課題

本移管は**現行M0の識別子規則を変更しません**。特に現行`CanonicalID`はハイフンを受け付けず、`SymbolToSegment`は名称をsnake_caseへ整える既定規則です。ユーザー向けTEI識別子でハイフンを認める将来要件や、新しいService Contractの識別子モデルとは別途設計します。型の再配置を根拠に、旧規則を恒久的な製品仕様として固定しません。

このパッケージはPlugin identity・製品Host authority・契約の発行者・Build Artifactのdigestや実行権限を定めません。`Ref.Kind`に既存の業務分類が含まれていても、固定CapabilityKindを新しいPlugin Service Contractへ転用するものではありません。#951・#1223・ADR-0032の契約移行の正本を追加していません。
