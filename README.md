# TEI（鼎／テイ）

> 本プロジェクトは生成AIを用いて実験的・模索的な開発を行っている段階にあり、**人的レビュー・利用を伴っていません**。
> また実際の利用において必要な機能が大幅に不足しています。現時点での利用は、**TEIの設計思想や実装の理解を目的とした試験的な利用に限られます**。
> 破壊的変更が頻繁に発生する可能性があります。利用は自己責任でお願いします。

![TEI — Traceable Execution of Intent](docs/assets/tei-branding-image.jpg)

TEIは、生成AI時代のローコード開発基盤を目指す実験的なプロジェクトです。業務システムに限らず、分野固有の知識と技術を分けて扱うことが役立つ領域を対象とします。人と生成AIが協調して設計を進めるための足場を提供します。

TEIでは、案件固有のデータモデルを含む分野固有の知識（ドメイン知識）を、実装技術と分けて管理します。プラグインの契約や依存関係を照合し、既存の.NET・Java等のアプリケーションへ組み込める成果物や、必要な機能を含む単独のネイティブ実行ファイルを構成することを目指します。TEI独自の共通Hostは前提としません。プラグインの仕様と実装を生成AIが体系的に読み取れる形にすることで、案件ごとの技術拡張を個別に用意できることを目指します。

製品の価値と目指す姿は、[TEIが大切にする価値](product/value/TEI_PRODUCT_VALUE.md)と[TEIの将来像](product/vision/TEI_PRODUCT_VISION.md)をご覧ください。

## どこから読むか

- [設計文書の案内](docs/README.md)
- [TEIを初めて知る方へ](docs/for-readers.md)
- [TEIの考え方](docs/overview.md)
- [使い始める](docs/getting-started.md)
- [全体構成](docs/architecture.md)
- [現在の実装状況](docs/status.md)
- [設計原則](product/principles/TEI_PRODUCT_PRINCIPLES.md)
- [製品要求](requirements/TEI_PRODUCT_REQUIREMENTS.md)
- [開発全体の計画](plan/MASTER_PLAN.md)

## ソース構成と開発

コアとプラグインを段階的に独立したサブプロジェクトへ整理します。現在の実装と、これからの管理方針を区別して参照してください。

- [コアの管理領域](core/README.md)
- [プラグインの管理領域](plugins/README.md)
- [サブプロジェクト化の設計判断](plan/adr/0033-core-and-plugin-subprojects.md)

## まず動かす

Goの開発環境で、試験と参照用作業領域の確認を行えます。

```bash
go test ./...
go run ./cmd/tei validate -workspace reference/workspace
```

詳しくは[使い始める](docs/getting-started.md)をご覧ください。

## Studio PoC の読み取り機能

Studio Host は、許可された Workspace 内の `.xead` ファイルを読み取り、テーブルと項目の概要を返す `ObserveXead` を提供します。入力は4 MiB、XML深度は128、要素数は100,000までに制限されます。返却値は読み取り専用の観察結果であり、Canonical World、編集権限、実行権限を与えません。未対応の構造は未解決のまま扱います。

応答の代表的な失敗コードは `WORKSPACE_ROOT_CHANGED`、`XEAD_SOURCE_READ_FAILED`、`XEAD_SOURCE_TOO_LARGE`、`XEAD_UNSUPPORTED_FORMAT`、`XEAD_PARSE_FAILED`、`XEAD_PROJECTION_FAILED` です。エラー応答には入力パスやXML本文を含めません。
