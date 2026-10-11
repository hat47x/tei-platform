# TEI（鼎／テイ）

TEIは、業務上の意味を長く保ち、実装・データベース・実行先・実行環境を結び付けて扱うソフトウェア開発基盤を目指す実験的なプロジェクトです。

画面や表を素早く作ることだけでなく、技術を変更しても次の点を追跡・検証できる状態を重視します。

- この業務概念は何を意味するか
- どの設計元情報から来たか
- どの実装・提供元・実行先へ対応付けられたか
- 実行した操作が何を返したか
- 外部世界で実際に何が観測されたか
- 何が未検証・未解決か

## どこから読むか

### 製品の価値と開発計画

- **最上位の価値:** [TEIが大切にする価値](product/value/TEI_PRODUCT_VALUE.md)
- **目指す姿:** [TEIが目指す姿](product/vision/TEI_PRODUCT_VISION.md)
- **判断原則:** [TEIの設計原則](product/principles/TEI_PRODUCT_PRINCIPLES.md)
- **製品要求:** [TEIの製品要求](requirements/TEI_PRODUCT_REQUIREMENTS.md)
- **要求から受入条件までの追跡:** [上流の受入シナリオ追跡](plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md)
- **開発全体:** [TEI全体計画](plan/MASTER_PLAN.md)
- **v1主線:** [v1 Reference Application 実証計画](plan/V1_REFERENCE_APPLICATION_PLAN.md)
- **要求の不足:** [製品要求の不足に関する調査](plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md)
- **節目:** [TEIの開発段階](plan/MILESTONES.md)

### 技術を知る・動かす

- [設計文書の案内](docs/README.md)
- [TEIを初めて知る方へ](docs/for-readers.md)
- [TEIの考え方](docs/overview.md)
- [使い始める](docs/getting-started.md)
- [全体構成](docs/architecture.md)
- [現在の実装状況](docs/status.md)

## 設計上の主要原則

### 意味と実装を分ける

```text
設計元の記述
      ↓
正本となる意味
      ↓
対応付け・実現方法
      ↓
提供元・実行先・実行基盤
      ↓
実行
      ↓
観測・検証根拠
```

「何を意味するか」と「何で実装するか」を区別します。

### ファイルの場所を意味の識別子にしない

設計元の記述から正本となる意味までをたどれるようにします。ファイルの場所は意味を識別するものではありません。

ファイルの移動だけで論理上の識別子が変わらない場合、意味の変更はありません。論理名や正本となる意味の識別子を変える場合は、意味の変更として扱います。

### SQLやGoをそのまま生かす

`.tei`の定義に加え、既存のSQLやGoなども、それぞれの形式のまま利用します。

```text
*.tei   意味・アプリケーション・ライフサイクルの定義
*.sql   SQL
*.go    個別の実装
*.json  観測・導出した記録
*.yaml  作業領域・外部設定
*.md    文書
```

既存形式の資産を正本として扱いながら、設計元の記述から正本となる意味までをたどれるようにします。

### 命令と観測を分ける

```text
命令に対する業務上の結果
        ↓ 別の操作
明示した読み取り境界
        ↓
観測結果・実行時のエラー
```

命令を受け付けたという応答だけで、外部の状態が変わったとは判断しません。

業務上の結果、実行時のエラー、観測結果、開発者向け診断を区別します。

### 設計情報を共通の受入境界へ集約する

Gate C後の設計と編集は、次の形にまとめていきます。

```text
ファイルを直接扱う方法・Studio・コマンドラインツール・AI
          ↓ 候補
共通の受入境界
          ↓
正本となる記述
          ↓ reload
正本となる意味
```

プレビューや却下では元の記述を変えず、古い`baseModelRevision`は明示的に扱います。

利用者が安定識別子やUUIDを記述する方法も、差分から編集する方法も選べます。

### Studioの実行環境とアプリケーションの実行先を分ける

Studioが開発者向けのデスクトップ環境としてWailsを使う場合も、生成するWailsデスクトップアプリケーションの実行先とは別のものです。

Studioの画面が動いたことを、二つ目のアプリケーション実行先における意味・機能・表示の検証根拠として数えません。

## まず動かす

clone後、基本的なRepository確認はGo toolchainだけで行えます。

```bash
git clone https://github.com/hat47x/tei-platform.git
cd tei-platform

go test ./...
go run ./cmd/tei validate -workspace reference/workspace
```

Gate B Product Evidenceの正規取得手順は通常の開発確認とは別です。actual Gate B Evidenceを取得するときは必ず [Gate B Evidence Runbook](plan/GATE_B_EVIDENCE_RUNBOOK.md) を参照してください。

Source Index例:

```bash
go run ./cmd/tei source-index \
  -workspace reference/workspace \
  -subject-kind query \
  -subject-id order_entry.recent_orders
```

## リポジトリの構成

```text
product/                 製品の価値・目指す姿・判断原則
requirements/            製品要求
plan/                    全体計画・開発段階・受入条件・判定・設計記録・作業課題
cmd/tei/                 コマンドラインツール
internal/                実行構成・検証・実行基盤
reference/workspace/     一連の流れを確認する参照用作業領域
reference/observed/      観測状態を模した試験用データ
tools/                   検証根拠・参照用の補助ツール
examples/                個別の実行例
docs/                    利用者向け・構成・準拠範囲の文書
```

