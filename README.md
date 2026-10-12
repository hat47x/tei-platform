# TEI（鼎／テイ）

![TEI — Traceable Execution of Intent](docs/assets/tei-branding-image.jpg)

TEIは、生成AI時代のローコード開発基盤を目指す実験的なプロジェクトです。業務システムに限らず、分野固有の知識と技術を分けて扱うことが役立つ領域を対象とします。人と生成AIが協調して設計を進めるための足場を提供します。

TEIでは、案件固有のデータモデルを含む分野固有の知識（ドメイン知識）を、実装技術と分けて管理します。TEIのコアは、プラグインを登録・選択・組み合わせ、実行へつなぐ共通機構を担います。プラグインの仕様と実装を生成AIが体系的に読み取れる形にすることで、案件ごとの技術拡張を個別に用意できることを目指します。

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

## まず動かす

Goの開発環境で、試験と参照用作業領域の確認を行えます。

```bash
go test ./...
go run ./cmd/tei validate -workspace reference/workspace
```

詳しくは[使い始める](docs/getting-started.md)をご覧ください。
