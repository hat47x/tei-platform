# TEI Product Vision

## 目指す姿

TEI（鼎／テイ）が十分に成熟したとき、業務システム開発は「設計書を書く工程」「コードへ翻訳する工程」「実装後に設計書を追従させる工程」に分断されません。

業務の意味を表す設計内容が基準として存在し、その同じ内容を人が読みやすい詳細設計書として見たり、生成AIが検査したり、実装へRealizeしたり、Driverで動かして確かめたりできます。

**設計内容が、そのまま検証・実装・実行へつながり、しかも特定の実装技術には閉じないこと。**

これがTEIの長期的なプロダクト像です。

## 開発者から見たTEI

開発者は、低密度なウィザードを延々と進めるのではなく、業務システム全体を高い情報密度で見渡せるEditorを使います。

同じCanonicalな意味を、用途に応じて次のように表示できます。

- 詳細設計書に近い高密度な表形式
- データ間の関係を追うグラフ
- 状態や処理を追うFlow
- 実際の画面に近いUI Projection
- Database Workbench
- Natural Language Projection
- Source / Diagnostic / Evidenceの追跡ビュー

表示形式は正本ではありません。編集しやすいProjectionとして存在し、同じ意味を別の見方へ切り替えられます。

## AI・認知機能から見たTEI

AIや認知系は、いきなりコードを書く前に、TEI内の既存知識とCanonicalな意味を探索します。

```text
既存の業務部品
  ↓
既存の意味パターン
  ↓
既存のPolicy / Variation
  ↓
組み合わせ
  ↓
Capability
  ↓
不足部分だけCustom Implementation
```

また、TEIへ入ってくる情報が最初からCanonicalな形であるとは限りません。既存設計書の項目対応、命名揺れ、自然言語とモデルの対応、例外らしい箇所などでは、規則、embedding、分類器、ローカル軽量モデル、外部の認知基盤を適材適所で使えることを目指します。

認知系は、データモデル上の不自然な1:N関係、関数従属性、履歴管理、認証・認可、監査、状態遷移など、業務システムで典型的に見落とされやすい論点を検査できます。能力が高まれば、候補提示だけでなく、一定範囲の探索・変換・検証をまとめて担うこともできます。

ただし、認知結果はCanonicalな意味そのものではありません。どの入力と根拠から何が提案され、どの検証を経て正式なモデルへ採用されたかを追跡できる必要があります。AIの能力が高まっても、会話状態やProvider固有の内部表現を暗黙の正本にはしません。

何でも人に質問するのではなく、既に回答済みの問い、過去の選択、類似システムの知識、決定論的に導出できる情報を再利用し、必要な場面だけ確認することも重要です。

## 隣接Productとの協調

TEIは、意味探索を担うSUI Sensemaking、認知・判断の継続性を担うSEI Cognition、分散実行を担うEKI Fabricと連携できます。

この連携は一つの必須スタックを作ることが目的ではありません。SUIからformalization candidateを受け取る場合も、SEIの認知Capabilityを使う場合も、EKIへ計算を委ねる場合も、TEI自身が守るCanonical meaning、Binding、Validation、Evidenceの境界は維持します。

同様に、各ProductがTEIを利用するときも、TEIの内部モデルを相手側のUniversal Coreへ複製することを前提にしません。交換に必要な意味だけを明示的なContractやAdapterで接続できる状態を目指します。

## 自作OSS群から見たTEI

TEIは、個別の業務システムだけでなく、UI・Interaction・domain logic・Validation・Application lifecycleを持つ自作OSSの**Application Implementation Layer**として利用できることを長期的な実証対象とします。

これは、他のProductをTEIの下位moduleへすることを意味しません。Product Value、Domain、Evidence claimは各Repositoryが正本を持ちます。TEIが担うのは、その意味を実装・Projection・Target・Capabilityへ接続するApplication側の共通基盤です。

最初のReference AdoptionとしてSUI Sensemakingを扱います。SUIはspatial canvas、複雑Interaction、review / approval、安全境界、AI / Cognitive Capability等を持つため、TEIが単純なCRUD以外でも開発負荷を下げられるかをPressureできます。

不足が見つかった場合は、次の順で対応します。

1. 既存のTEI model / Pattern / Projection / Capabilityで表現する
2. reusable Plugin / Adapter / Capabilityとして切り出す
3. native asset / custom implementationとして接続する
4. 複数Applicationに共通する意味上の欠落がReferenceで確認できた場合だけCore / Product Requirement候補にする

一方、scheduler、worker、protocol、sandbox、model runtime、認知アルゴリズム本体等のKernel / Infrastructure PlaneまでTEIで実装することは既定としません。SEI CognitionやEKI Fabric等では、管理UI・設定・Projection・業務surfaceはTEI-firstの候補になりますが、専門kernelは各Productのnative implementationを維持できます。

SUIでのReference Adoptionは、TEI利用率ではなく、**同じ意味変更に必要な手書き実装・検証・同期負荷が減るか**で評価します。これにより、TEIが自身のOSS群でも実際に使える開発基盤であるかをdogfoodします。

## 既存システムから見たTEI

TEIは、新規開発だけを前提にしません。

既存システムには、固有の認証テーブル、独特な状態管理、長年使われたSQL、複数Database、外部バッチ、帳票、運用手順などがあります。

移行時にそれらを一度「TEIらしい形」へ作り替えなければならないのでは、現実の移行基盤にはなりません。

既存仕様を変えずに移行し、その後必要な部分から段階的に標準部品やCapabilityへ置き換えられることを目指します。

## 実行面から見たTEI

TEIは、Web画面だけを出力する製品にはしません。

Presentation / Interactionの意味をTarget-neutralに保ち、少なくとも次のTargetを同じ上位設計から扱える構造を目指します。

- Web
- Desktop
- Mobile
- API
- MCP
- VUI

新しいTargetを追加するとき、TEI Coreそのものを書き換えるのではなく、独立したAdapter / PackageとConformanceによって拡張できることを基本とします。

同様に、Databaseや外部Runtimeについても、Coreへ製品固有知識を増殖させるのではなく、Provider境界から接続します。

## Driverの役割

Driverは、単に完成した本番アプリケーションを動かすだけのRuntimeではありません。

設計途中でも、データ、業務ルール、画面、処理、権限、Database Contractなどの整合性を、閉じた環境で実際に動かして確かめるシミュレーション基盤として使えることを重視します。

人が設計書を読んで確認するだけでなく、「この設計を動かすと何が起きるか」を早い段階で検証できる状態を目指します。

## 厳格だが、使うときは軽い

TEIの内部では、意味、型、依存、権限、実行境界を厳格に扱います。

しかし利用者に見える体験は、その厳格さをそのまま設定項目の数へ変換したものにはしません。

- 自動で導出できるものは自動で導出する
- 一度確認した内容は履歴として再利用する
- 変更したいときは、後から選択経緯まで追って変更できる
- 問題の可能性が高いところだけを提示する
- 詳細を見たい人には深く掘れるが、毎回すべてを見せない

「規則が厳しいから使いにくい」のではなく、**規則が厳しいからこそ、普段は安心して軽く使える**状態を目指します。

## 成熟したTEIで可能になること

最終的には、次のような開発の流れを一続きにしたいと考えます。

1. 業務上の要求や既存システムを読み解く
2. 既存の業務部品やパターンを再利用する
3. Editorで意味と関係を整理する
4. AIが抜けや矛盾を検査する
5. Driverで設計を動かして確認する
6. WebやDesktop等のTargetへProjection / Realizeする
7. Databaseや外部CapabilityへBindingする
8. 実行・配備する
9. 実際の状態をObservation / Evidenceで確かめる
10. 実装方式を変更しても、業務上の意味と履歴を保つ

この循環を、独自製品の中だけで閉じず、既存技術やOSSと組み合わせながら成立させることがTEIのVisionです。
