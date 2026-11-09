# 突き合わせ規則の反例 — 宣言を一つ足すと、どう変わるか（2026-11-07）

- 対象: [操作とPatternの突き合わせの参照実装](../../../tools/reference-operation-pattern-crosswalk/README.md)と、[操作面の設計候補](../../design/OPERATION_SURFACE_DESIGN_2026-11.md)の「追記（2026-11-07）」
- 位置づけ: **実行して観測した記録**。規則の採用判断・Owner受理・Gate合否ではない。
- 計測した版: `raw/head.txt`（`origin/main` `88ab1561` の作業ツリー。追加は`plan/evidence/`配下のみで、Goの入力に影響しません）。

前の記録（[操作とPatternの突き合わせを参照実装で試す](../OPERATION_PATTERN_CROSSWALK_REFERENCE_2026-11-07/README.md)）は、受理済みWorkspaceで**一致4件・不一致2件**を観測しました。この記録は、**宣言を一つだけ足すと規則の結果がどう変わるか**を、5つの場合に分けて確かめます。

受理済みの`reference/workspace`は編集しません。各場合とも複製し、**宣言を一つだけ**加えています。

## 1. 何を足したか

| 場合 | 足した宣言 | 予想 |
| --- | --- | --- |
| `base` | なし（比較の基準） | 一致4・不一致2 |
| `effect-cmd` | 効果を宣言する能力 + それに束縛された`command` | 新しい一致が1件増える |
| `readpath` | `master_data`の`customer.customer`へのPersistence | `customer_reference`が一致に変わる |
| `conflict` | 同じDataを指す2つ目の`browse` | 衝突が1件出て、その操作は一致から外れる |
| `no-persist` | 効果を宣言する能力（ただしPersistenceの無いData） | `missing-input`になる（一致しない） |

## 2. 観測

```text
base          一致4 不一致2 衝突0
effect-cmd    一致5 不一致2 衝突0   ← create.sales.order ← register_order [command]
readpath      一致5 不一致1 衝突0   ← read.customer.customer ← customer_reference [reference]
conflict      一致3 不一致2 衝突1   ← list.sales.order を2つのbrowseが取る
no-persist    一致4 不一致3 衝突0   ← rename_customer [command] missing-input
```

それぞれの意味:

- **`effect-cmd`**: 能力が`effect create sales.order`を宣言し、`command`がその能力を指すと、`create.sales.order`に**能力IDで**一致します。効果の宣言が無い`submit_order`は、同じWorkspaceでも`out-of-scope`のままです。**効果の有無だけが差**です。
- **`readpath`**: `master`の参照先DataにPersistenceを足すと、`read.customer.customer`が導出され、`customer_reference`が**Relation targetで**一致します。`datasource master`は一致の判定に使っていません。**参照先Dataを読む標準操作の有無**が効いています。
- **`conflict`**: 同じ`data sales.order`を指す2つ目の`browse`を足すと、`list.sales.order`を2つの宣言が取り合います。規則は**どちらも選ばず**、`conflicts`に両方の名前を出し、その操作の一致を消します。**黙って後勝ちにしないこと**が観測できました。
- **`no-persist`**: 効果はあるがPersistenceが無いDataへの`command`は、`update.customer.customer`が導出されないため`missing-input`になります。**効果の宣言だけでは一致しない**（標準操作の導出が別に要る）ことが分かります。

## 3. この記録が示すこと

- 規則は、**足した宣言の効果だけ**を結果に反映します。他の宣言の一致は変わりません（`base`との差分が1件ずつ）。
- 一致が増える条件は2つあり、**別々**です。
  - 能力の効果宣言（`command`の能力ID一致）
  - 参照先Dataの標準操作の導出（`reference`のRelation target一致）
- 同じ操作を2つが取る場合は、**衝突として返し、選ばない**。既定値・後勝ち・ロード順での解決はしていません。
- 効果を宣言しても、標準操作が導出されなければ一致しません。**「宣言がある」と「操作がある」は別**であることが、`no-persist`で見えます。

## 4. この記録が示さないこと

- **規則を採用すべきという判断**。W3は未着手のままです。これらの反例は、案が何を読み、何を読まないかを示すだけです。
- **`missing-input`が欠陥であること**。標準操作の導出にはPersistenceの対応が要るという、既存の導出規則の結果です。
- **衝突の解決方法**。2つの宣言が同じ操作を取ったときに、どちらを残すべきかは決めていません。
- **実DB・実Provider・権限・実Consumer**。合成のWorkspaceと読み取りのみです。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（5つの複製を作り、それぞれに宣言を一つ足してCLIを実行） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（このディレクトリを除く） |
| `raw/*-crosswalk.json` | 5つの場合の突き合わせ結果 |
| `raw/summary.txt` | 上の一覧 |

`*.err.txt`はすべて0バイトです。

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `8665218479991c5e72019997bc26578d90c57176a2afbad29498b462256ff934` | 5005 |
| `raw/head.txt` | `cec46f5fa3282d977f7380f7df85b6eb2d8a438442cb7fb617d92e087e327ca7` | 73 |
| `raw/pre-status.txt` | `c7e82e15951ef0fd8196ebf5efa61400274bb0e48b7d1b2d215440f1a055c556` | 41 |
| `raw/base-crosswalk.json` | `5e0d1452ad989d9882b63cd272ddc33d8821e08c98a604d3adabceb2f2efc0b3` | 3396 |
| `raw/effect-cmd-crosswalk.json` | `9f4fed220e15b14e23a5b1cb990e9f0ae64d662ccf93f06bea66001081e60899` | 3702 |
| `raw/readpath-crosswalk.json` | `5102657f292503c33195c849e820ff06ab62118b5839d2ef85928c736b318b45` | 3730 |
| `raw/conflict-crosswalk.json` | `e95871fb055b2dbc4dc8d597c7ebd00cbb96704edae2664f04f360d3a6cd980f` | 3480 |
| `raw/no-persist-crosswalk.json` | `13a0875f993ad955da71afacf51f0ad4dd6d19684db6cb9ef92e65fe71b509fd` | 3643 |
| `raw/summary.txt` | `bb764cd3bbe5714e15c2761d5816c3c0952438aa5c5225c43d246bdfbaddc9c0` | 3435 |

`base-crosswalk.json`のSHA-256は、前の記録の`raw/accepted-crosswalk.json`と同じ値です。同じHEAD・同じ入力から同じ出力が出たことを、ハッシュの一致で確かめられます。

`raw/pre-status.txt`が空でないのは、**計測前にこのワークツリーで削除した`tmpw/`の削除が未コミット**だったためです（`tmpw/`は別のPRでリポジトリから外します）。計測対象のGoコードと入力には影響しません。
