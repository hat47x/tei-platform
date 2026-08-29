# はじめてTEIを知る方へ

TEI（鼎／テイ）は、業務システムを「画面とデータベースを素早く作る道具」だけとして捉えず、**業務上の意味を長く保ちながら、実装技術を入れ替えやすくするための基盤**を目指している実験的なプロジェクトです。

まだ完成した製品ではありません。現在は、後から変更しにくい設計境界をGoのコードとテストで確かめています。

## 何が問題なのか

業務システムには、画面に見えている項目より多くの意味があります。

たとえば「受注」には、単に受注テーブルがあるだけでなく、次のような事情があります。

- どの番号を業務上の識別子とするか
- どの状態からどの状態へ変わってよいか
- いつの時点の情報を履歴として残すか
- 顧客マスタや在庫など、別のDatabaseをどう参照するか
- 古いApplicationと新しいDatabase Schemaを一時的に共存させられるか
- 本番へ変更命令を出したあと、実際に変更されたことをどう確認するか

従来は、こうした意味が画面定義、SQL、Java/Go等のコード、DB制約、運用手順書、担当者の知識へ分散しがちでした。

TEIは、その分散を一つの巨大な独自言語へ押し込めるのではなく、**意味・実装・配備・観測を分離したまま相互に追跡できる状態**を目指します。

## 具体的にはどう違うのか

### ApplicationとDatabaseを別々に考える

一つのApplicationが複数Databaseを使うことを普通に扱います。

```text
受注Application
├─ 販売DB      読み書き
└─ 顧客マスタDB 読み取り
```

Applicationは「どのDatabase Contractが必要か」を宣言し、環境ごとのBindingが実際のDatabaseへ結びつけます。

そのため、ApplicationのReleaseとDatabaseのReleaseを一対一に固定しません。

### SQLや既存コードを捨てない

TEIには独自の記述形式がありますが、SQLをすべて独自DSLへ変換したり、Goの実装を隠したりすることを目的にしていません。

SQLがもっとも自然ならSQLをsource of truthとして扱い、その意味を必要な範囲で解析します。既存技術を残したまま、上位の意味との接続を明示する方向です。

### 「命令した」と「起きた」を分ける

たとえばDatabase migrationの実行コマンドが正常終了しても、それだけで「本番DBは期待した状態になった」とは限りません。

TEIのDeploymentモデルでは、おおむね次のように分けます。

```text
変更を要求する
    ↓
Executorが受理する
    ↓
外部状態をObservationする
    ↓
必要なEvidenceを確認する
    ↓
次のActionへ進む
```

これにより、再試行や障害復旧で「同じ副作用を二重実行してよいか」を曖昧にしにくくします。

## 生成AIとはどう関係するのか

生成AIがコードを書けるようになるほど、「毎回大量のコードを生成する」だけでは、長期保守で同じ意味が何度も別実装される可能性があります。

TEIでは、AIがまず次を探せる状態を目指します。

1. 既存の業務概念やDomain Component
2. 既存のSemantic Pattern
3. 既存のPolicyやVariation
4. 既存Capabilityの組み合わせ
5. それでも足りない部分だけのCustom implementation

つまり、AIに渡す再利用単位をコード片だけでなく、**意味と契約の単位まで引き上げる**ことを狙っています。

## TEIは何ではないのか

現時点のTEIは、次のものではありません。

- 今すぐ業務アプリをGUIだけで完成できる市販ローコード製品
- すべてのSQLや既存コードを独自形式へ置換するプラットフォーム
- すべての業務を一つのWorkflow Engineへ従属させる仕組み
- KubernetesやVault等を内製し直す基盤
- AIへ自動的に本番変更権限を与える仕組み

外部製品や既存実装はCapability / Providerとして利用でき、必要なら交換できることを重視しています。

## なぜ最初からEditorを作らないのか

見栄えのよいEditorを先に作ると、そのUIで表現しやすい概念がそのままCore設計になりがちです。

TEIでは先に、Application / Database / Query / Release / Deployment / Plugin / Resource authorityといった境界を実行可能なコードで検証しています。

Editorはその上に載るauthoring/projectionの一つと考え、意味の唯一の所有者にはしません。

## 現在どこまで来ているか

初回公開チェックポイントでは、次を一本の縦切りとして接続しています。

```text
Semantic definition
  ↓
Application / Database contract
  ↓
Query / Persistence / Migration
  ↓
Application / Database / Solution Release
  ↓
Deployment Run
  ↓
Executor / Observation / Evidence
  ↓
Plugin permission / Resource lease
  ↓
Host takeover / fencing / runtime reattach
```

完成したGUIやproduction adapterより前に、この縦方向の意味が途中で崩れないことを確かめている段階です。

## 次に読むもの

- 実際に動かしたい: [Getting Started](getting-started.md)
- 設計をもう少し知りたい: [TEIの考え方](overview.md)
- 実装範囲を確認したい: [Current implementation status](status.md)
- 技術境界を追いたい: [Architecture](architecture.md)
