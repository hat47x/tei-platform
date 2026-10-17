# 多言語化：表示・メッセージ・業務データの差分影響を比較するPoC

関連：[#1551](https://github.com/hat47x/tei-platform/issues/1551)（表示オーバーレイ）、[#1649](https://github.com/hat47x/tei-platform/issues/1649)（標準メッセージ）、[#1453](https://github.com/hat47x/tei-platform/issues/1453)（多言語業務データ）

## 位置付け

このディレクトリは**複数の既存参照実装を一つのテストシナリオから呼ぶだけ**の横断PoCである。TEI Coreの新しい概念、統一翻訳DB、合否Gate、Provider、Host権限機構を追加しない。実際のSaaS接続・AI呼出し・データベース・Windows GUIも使用しない。

合成した「受注」シナリオには、次の三種類のテキストが存在する。**これらの意味が同一という主張ではなく、一つの業務画面に共存する独立した資源**として扱う。KouseiKanriの原典Function等と`order_entry.submit_order`のmappingが確定したことは示さない。

| 表現領域 | 実際に使用する既存実装 | 識別と失効条件 | 判断・公開の境界 |
| --- | --- | --- | --- |
| Data/Action等の表示名 | `multilingual-overlays.BuildLockedDisplay` | `modelref.Key`・表示面・通常Plugin版／原文表示名の指紋 | 環境のSurface/Grant/Override。試験値は**Host受理ではない** |
| 業務メッセージ | `message-localization.Resolve`と`std.message` | Plugin版とMessage ID／原文・型付き引数の契約版 | Store/Binding/Grant/ReviewGate。**実Host検証は未接続** |
| 業務レコードの文章 | `multilingual-business-data.Select` | テナント・レコード・意味属性・言語・用途別版・文章版 | 正式版Publicationと業務上の承認。試験用Snapshotは**実運用の公開証拠でない** |

## 何を実証するか

`impact_test.go`は既存の実際のResolverを組み合わせ、次の差分を確認する。

| 一つだけ変えた対象 | その領域への影響 | 他領域への影響 |
| --- | --- | --- |
| 表示名「受注を確定」の原文改訂 | 旧訳を拒否する | メッセージ・公開業務文章は従来どおり |
| メッセージ「Order {orderId} confirmed」の原文改訂 | 旧訳を拒否する | 表示名・公開業務文章は従来どおり |
| 同メッセージの`orderId`引数型改訂 | 旧訳を拒否する | 表示名・公開業務文章は従来どおり |
| 業務文章の日本語正式版差替え | 古い原文に依存する英訳を拒否する | 表示名・メッセージは従来どおり |
| 翻訳を「候補」に変更 | 表示名・メッセージは選択拒否、業務文章は候補を公開しない | 他領域の判断状態を変更しない |
| 英語を独立した業務正式版として公開 | 日本語原文改訂で自動失効**させない** | UIの翻訳失効規則を持ち込まない |

**検証する横断共通項は「意味対象に結び付いた表現の来歴・改訂・使用可否を照合できること」**であり、採用判断や保存構造の共通化ではない。いずれの翻訳も実際の権限者が受理したことにはならない。テスト結果は限られたGo参照実装の実行観測であり、製品接続・GUI受入ではない。

## 外部製品からの学びとの対照（一次資料）

- **Salesforce**: Translation Workbenchはメタデータラベルと、対象を限定したデータ翻訳を扱い、言語別翻訳担当者・ファイル入出力を提供する。TEIでは「翻訳の管理」と「意味対象の採用」「業務データ正式版」を区別する参考となる。[公式説明](https://help.salesforce.com/s/articleView?id=workbench.htm&language=en_US&type=5)
- **Mendix**: Maia Translation Generatorは未翻訳のアプリ／システム文字列の一括生成を支援し、利用者の確認・保存操作を必要とする。TEIでもAI出力は候補に留める。[公式説明](https://docs.mendix.com/refguide/translation-generator/)
- **SAP CAP**: 静的ラベル用i18n bundleと、業務ペイロード用localized dataを分ける。TEIの表示面と業務属性の責務分離との比較に用いる。[i18n](https://cap.cloud.sap/docs/guides/uis/i18n) / [localized data](https://cap.cloud.sap/docs/guides/uis/localized-data)

これらは**外部製品における参考事実**であり、TEIの設計決定または実装受入へ自動的に昇格しない。各社のフォールバック、承認、データ公開規則をそのままTEIへ移植しない。

## 追試

```sh
go test -count=1 ./reference/multilingual-integration
go test -race -count=3 ./reference/multilingual-integration
go vet ./reference/multilingual-integration
```

表示名・メッセージ・業務文章が同じStorage/Grant/Policyを用いるとは仮定しない。特に、別言語で独立して作成・承認された契約や文章の改訂を、原文改訂に従属させない。

**残る実務課題**：正式なHostの現在受理済みSource/Store/Grant/権限、業務Recordの正本とPublication、本番利用時の再検証、モデルからの翻訳対象自動抽出、MF2/複数形・数値/日付等の書式、認証されたWeb/Desktop配布、生成AIによる候補作成/機密送信承認。横断PoCだけで解消したとは扱わない。

## Plugin世代と現在権限を分離する回帰検査（#1551 / #1223）

`generation_test.go`は既存のO3a言語別バンドル、Plugin Lock、O2表示、`std.message`、業務データPublicationをそのまま呼び出す。新たなResolver・Authority・Storage・Gate・公開APIは設けない。

| 比較する変化 | 参照実装で確認する境界 | この検査で証明しないこと |
| --- | --- | --- |
| 通常Pluginをv1.2.0からv1.3.0へ切り替えた合成世代 | 旧バンドルは新Lockと一致しない。旧版の表示対象は新世代へ暗黙移行しない | 二世代の実Host同時稼働、drain、切替の原子性 |
| 合成環境から表示Grantを取り除く | 新しい表示合成は拒否される。一方、発行済みバンドルの整合性とLockだけの照合では現在のGrant失効を検出できない | 発行済みバンドルの利用許可、Hostの真正なGrant失効 |
| 元の表示名を改訂する | 新しい表示合成では旧訳を失効と判定する。一方、発行済みバンドルとPlugin Lockだけでは現在のSource改訂を検出できない | 現在のAccepted Source確認、認証された配布やキャッシュ無効化 |
| メッセージの試験用Reviewが拒否する | 当該メッセージは拒否され、表示名と独立した業務正式文章は変わらない | 実Host権限、業務文章の正式なPublication受理 |

ここで重要なのは、**「バンドルが自己整合している」ことと「現在の環境で配布・表示してよい」ことは別**という観測である。古いバンドルが静的検査を通ることを、継続利用可能という保証に読み替えない。製品化時には既存のCurrent Accepted Source・Host Authority・配布経路から、利用時の判断根拠を取得する必要がある（#1551）。世代切替と旧呼出しの保持・解放はComposition Kernel側（#1223）の責務であり、ここに第二の世代管理器を設けない。

この追加は**合成fixtureのGoテスト**である。実際の更新・再配布、Web/Desktop/Windows GUI、DB、実人間受理、AI翻訳、KouseiKanriと`order_entry.submit_order`の対応は検証しない。実行結果は対象commitとCI記録を確認するまで未確認として扱う。

試験対象：

```sh
go test -count=1 ./reference/multilingual-integration
go test -race -count=3 ./reference/multilingual-integration
```
