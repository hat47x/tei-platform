# 三要素CRUD：SnapshotからStudio Inspectorまでの実装・検収計画（2026-11-22）

> **位置づけ：実装計画。** #5310、#5323、#5356で実装済みのread-only Producerを基線とする。通常CLIの`crud-snapshot`公開は#5698で**実装済み**。Studio・Source Reader接続、Windows Stage2、Owner受理は**未検証**。

## 目的と既存成果

一つのFunction×DataのCRUDセルから、その表示に寄与した個々の宣言を確かめ、必要に応じて正しいSourceへ移動できるようにする。三要素Methodのレビュー体験を、Studio Desktopのセルフビルドへつながる小さな縦断として実装する。

| 既存の成果 | 役割 | まだ保証しないこと |
| --- | --- | --- |
| #5310 | 同一Function×Dataの複数操作を一セルに集約 | 実DBの更新回数、Action単位のEffect |
| #5323 | 元のFunctionOperation／CRUDDefinitionを個別に列挙 | Source Span、永続的な宣言ID |
| #5356 | 一つのモデル入力からCRUD表・Origins・ModelFingerprintを生成 | Source Revision、別要求間のSnapshot一致、編集資格 |\n| #5698 | `three-element crud-snapshot -model`を既存CLIに追加。JSONでRows・Origins・ModelFingerprint・診断を返す | 実Studio Consumer、Source Navigate、Owner受理 |

これらは既存の`internal/threeelement`の補助Projectionであり、第二の意味正本ではない。

## DT-02B：最小のConsumer縦断

1. **CLI Producer（#5698で実装済み）**：既存の`runThreeElementCommand`に`crud-snapshot`を登録し、`loadThreeElementModel`から`GenerateCRUDSnapshotProjection`を呼ぶ。`tei.cli.three-element.crud-snapshot/v0`のJSON応答に`Rows`、`Origins`、`ModelFingerprint`、`structuralOk`、`crudConflictStatus`を含む。`ok: true`は`okScope: projection-produced-only`の範囲に限る。従来の`crud`応答は別に維持する。**通常製品CLIの実行結果はこの文書では確認していない**。
2. **Studio Inspector**：一つの応答に含まれる`Rows`と`Origins`を同じSnapshotの情報として扱う。セルの局所キーはFunction ID×Data ID。再ロード後は旧Snapshotの選択・Originsを無効にし、同じ表示名だけで再結合しない。
3. **Source Navigate**：#3788の既存qualified Selection契約を通し、Source identity・Revision・対象Kind・一意な出現位置が確認できた場合に限り移動する。モデル内の配列位置を原文の行番号として使わない。
4. **編集境界**：この縦断はread-only。Sourceへの変更は別のPreview→Save→Re-read→Re-projectの検収を通す。非一意、Source未取得、古い版、部分開示のときは編集を許可しない。

## 独立Expectedと負例

| ケース | 独立した期待 |
| --- | --- |
| E1：同一モデル内のRとUの別宣言 | 一つのRUセルから二つの元宣言を確認できる |
| E2：同じIDのまま操作内容が変わる | 別SnapshotのOriginsを再利用しない |
| E3：同じIDを異なるScopeで使用 | Sourceを誤って開かない |
| E4：矛盾する明示CRUDが二件 | 両方を保持し、Conflictを消さない |
| E5：Source unavailable／partial | ローカルのInspect成功をSource Locate成功と扱わない |
| E6：Functionを使わないBusiness Action | Function-level CRUDをActionの実Effectとして表示しない |

Expectedは実装出力から逆算せず、Sourceの固定版と対象範囲を明記する。宣言されたCRUD、派生したCRUD、Providerで観測されたEffectは別々の資格として扱う。

## 実装順と担当境界

| 順序 | 既存の担当 | 完了条件 |
| --- | --- | --- |
| 1 | #5698／#1124／#4472 | CLI配線・Focused Testのコードは実装済み。通常製品CLIの実行結果と出力互換性を改めて確認 |
| 2 | #2514 | E1〜E6を独立Expectedと比較し、失敗・未取得を隠さない |
| 3 | #3788 | qualified Source解決の正例・曖昧・stale・unavailableを確認 |
| 4 | #1824／#2831 | 実Wails Inspectorでセル→Originsを操作し、Windows Stage2で再現 |
| 5 | #3250 | qualifiedな受理済み関係だけをSubsystem評価に渡し、宣言の重複を更新責任の重複と数えない |

DesktopセルフビルドのP0を維持する。CLIだけの成功でStage2の実証を代替しない。StudioがSourceを編集する機能やSHOへの保存は、別の受入条件に置く。

## 実施状態

この文書は実装済みのProducerと未実装・未検証のConsumerを接続するための作業計画である。#5698のCLI実装とテストコードはmainに存在する。ただし、この文書更新に伴ってGoテストや通常CLIを新たに実行した記録ではない。Windows Studio実行、DB Effect、Owner意味受理も未検証。

## DT-02C：一セルを再照合するCLI Consumer（2026-11-22）

`crud-snapshot`が同一モデルの`Rows`・`Origins`・`ModelFingerprint`を返すようになったため、次は**以前表示したセルを、再ロードしたモデルの中で読み取り直す**小さなConsumerを用意する。

```sh
tei three-element crud-snapshot -model order-model.json
tei three-element crud-cell -model order-model.json \
  -expected-fingerprint sha256:<先ほど取得したモデル指紋> \
  -function submit -data order
```

新しい`crud-cell`は既存のJSON読込を利用し、入力ファイルを再読込してから現在の`ModelFingerprint`を照合する。応答は対象セル一件と元宣言すべてを含む。`okScope`は`model-snapshot-cell-inspected-only`、出典の資格は`model-local-only`と明示する。Sourceの移動、Source Revision、World Fingerprint、Provider Effect、Owner受理を主張しない。

**失敗時のふるまい**：指紋の省略は拒否、異なる指紋は`STALE_SNAPSHOT`、存在しない組は`CRUD_CELL_NOT_FOUND`、同一Kind内の重複IDは`CRUD_CELL_AMBIGUOUS`として拒否する。異Kind間の同名IDは正当なため混同しない。対立する明示CRUDは消さずに閲覧へ残す。

### Studioへ引き渡す二段階の現在性

Studioに接続する際は、モデルの`ModelFingerprint`でセルと元宣言を再照合した後、既存の`InspectSubjectAtSnapshot`によって**World＋Source Indexの現在性**を別に確認する。これらが一致しても、qualifiedなSource位置の一意性や書込資格は得られない。#5749／#5764のSource移動後Snapshot失効の負例を参照し、古い位置へ自動的に付け替えない。

CLIの一セル取得はStudioの実Consumerと同一ではない。Studio UI→Host選択→実Invocation→実応答→Windows WebView2／Stage2の検証、および正式Source保存／Review／Accept／結果照会は引き続き未検証の工程とする。新規CLIコード・テストの実行結果は対象PRのCIで別途確認する。
