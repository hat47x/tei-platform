# 三要素CRUD：SnapshotからStudio Inspectorまでの実装・検収計画（2026-11-22）

> **位置づけ：実装計画。** #5310、#5323、#5356で実装済みのread-only Producerを基線とする。以下のCLI・Studio・Source Reader接続、Windows Stage2、Owner受理は、この文書の作成時点では**未検証**。

## 目的と既存成果

一つのFunction×DataのCRUDセルから、その表示に寄与した個々の宣言を確かめ、必要に応じて正しいSourceへ移動できるようにする。三要素Methodのレビュー体験を、Studio Desktopのセルフビルドへつながる小さな縦断として実装する。

| 既存の成果 | 役割 | まだ保証しないこと |
| --- | --- | --- |
| #5310 | 同一Function×Dataの複数操作を一セルに集約 | 実DBの更新回数、Action単位のEffect |
| #5323 | 元のFunctionOperation／CRUDDefinitionを個別に列挙 | Source Span、永続的な宣言ID |
| #5356 | 一つのモデル入力からCRUD表・Origins・ModelFingerprintを生成 | Source Revision、別要求間のSnapshot一致、編集資格 |

これらは既存の`internal/threeelement`の補助Projectionであり、第二の意味正本ではない。

## DT-02B：最小のConsumer縦断

1. **CLI Producer**：既存の三要素モデル読込経路で`GenerateCRUDSnapshotProjection`を利用する。既存`crud`コマンドの互換出力を変更せず、読み取り専用のJSON出力として追加する。新しいReaderやSource Registryは作らない。
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
| 1 | #1124／#4472 | Focused Go testと通常CLI JSONの実出力を確認 |
| 2 | #2514 | E1〜E6を独立Expectedと比較し、失敗・未取得を隠さない |
| 3 | #3788 | qualified Source解決の正例・曖昧・stale・unavailableを確認 |
| 4 | #1824／#2831 | 実Wails Inspectorでセル→Originsを操作し、Windows Stage2で再現 |
| 5 | #3250 | qualifiedな受理済み関係だけをSubsystem評価に渡し、宣言の重複を更新責任の重複と数えない |

DesktopセルフビルドのP0を維持する。CLIだけの成功でStage2の実証を代替しない。StudioがSourceを編集する機能やSHOへの保存は、別の受入条件に置く。

## 実施状態

この文書は実装済みのProducerと未実装・未検証のConsumerを接続するための作業計画である。新たなGoテスト、CLI実行、Windows Studio実行、DB Effect、Owner意味受理を実施した記録ではない。
