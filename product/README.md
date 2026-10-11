# TEI 上位文書

このディレクトリには、TEI（鼎／テイ）が守る価値と目指す方向を定めた上位文書を置きます。

実装や個別の設計判断を読む前に、これらの文書を基準として確認します。

## 文書の役割

- [`value/TEI_PRODUCT_VALUE.md`](value/TEI_PRODUCT_VALUE.md): TEIが利用者へ提供し、開発中も失ってはならない価値。その価値が個人・組織・社会のそれぞれに何をもたらし、どう確かめるかもここで定める
- [`vision/TEI_PRODUCT_VISION.md`](vision/TEI_PRODUCT_VISION.md): その価値が十分に実現されたとき、TEIがどのような開発基盤になっているか
- [`principles/TEI_PRODUCT_PRINCIPLES.md`](principles/TEI_PRODUCT_PRINCIPLES.md): 設計や実装で迷ったときに用いる判断原則

具体的な要求は[`../requirements/TEI_PRODUCT_REQUIREMENTS.md`](../requirements/TEI_PRODUCT_REQUIREMENTS.md)、着手順や並行する作業の関係は[`../plan/`](../plan/)で管理します。

## 文書の優先関係

文書同士が矛盾した場合は、原則として次の順で上位の意図を優先します。

```text
Product Value
  ↓
Product Vision
  ↓
Product Principles
  ↓
Product Requirements
  ↓
ADR / Issue
  ↓
Architecture / Detailed Design
  ↓
Implementation / Test / Evidence
```

マイルストーンや作業計画は、何を先に進めるかを示します。上位の価値や要求を書き換える根拠にはしません。

実装から得た知見によって上位文書を見直す場合は、実装に合わせて意味を暗黙に変えず、上位文書の変更として明示します。

## 文書作成の基準

TEIの文書では、仕様上の用語を正確に保ち、自然な日本語で説明することを重視します。

文書を作成・更新するときは、次の順で確認します。

1. まず内容、境界、条件、例外を整理する
2. 他文書との意味上の矛盾がないか確認する
3. 意味を変えず、自然な日本語として全文を読み直す
4. 箇条書きだけで意味が途切れていないか確認する
5. 最後に、実装やテストが主張を過大に見せていないか確認する

技術用語は必要に応じてそのまま用いますが、英語の抽象語だけで説明を済ませず、日本語で何を意味するかを先に伝えます。
