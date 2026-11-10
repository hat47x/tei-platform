# realize による実装束縛が参照グラフに現れない（2026-11-10、第41ラウンド）

- 対象: [#3983](https://github.com/hat47x/tei-platform/issues/3983)、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md)、[#3419](https://github.com/hat47x/tei-platform/issues/3419)、[#3884](https://github.com/hat47x/tei-platform/issues/3884)。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **read-only の実測。新しい欠陥を検出して起票。コードは変更しない。**

## 1. 目的

「この capability は何で実装されているか」を CLI だけで答えられるかを試した。Binding/Realization 層は ADR-0048 の4層のうち3層目にあたるので、設計者が最も参照する情報の1つである。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `1b48671e8`（`origin/main`） |
| CLI実行物 | `sha256:66ac2a9fb0def7c9189e670669eaf4a0f18cc6995fc96a39cdc1a40ef1eb2b0c` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 補助として確かめた正常な挙動

先に、`trace` のページングと指紋の扱いが正しいことを確認した。これらは**問題なかった**。

| 確認 | 結果 |
| --- | --- |
| `-limit 2` でのページ送り | 2+2+1=5 で `nextOffset` が 2→4→`null` と正しく終端 |
| `continuation.fingerprint` | `result.snapshotFingerprint`（外側）を指す |
| 継続トークンの失効 | 無関係な名前空間を追加しても `STALE_SNAPSHOT`（World 全体を含むため） |
| `graphFingerprint` と `snapshotFingerprint` | **常に同値**（後者は `0b85c031a` 系の改名で残る deprecated alias、コメントに明記） |
| ファイル削除の影響 | `validate` が33 error を返す（`UNKNOWN_PERSISTENCE_ATTRIBUTE` 等） |

`graphFingerprint` の alias は `internal/modelref/trace.go` に「deprecated alias、v0 の1周期だけ同値で出す、継続トークンに使ってはならない」と明記され、`docs/getting-started.md:228` にも説明がある。**設計どおり**である。

## 4. 観測: 実装束縛がグラフに無い

### 4.1 workspace には4つの `realize` 宣言がある

```text
applications/order-entry/realizations/default.tei      : realize order_entry.compose_order_summary with custom go
applications/order-entry/realizations/authenticate.tei : realize security.authenticate with custom go
applications/order-entry/realizations/authorize.tei    : realize security.authorize with custom go
applications/order-entry/realizations/submit-order.tei : realize sales.order.submit with custom go
```

### 4.2 どの trace にも実装束縛の関係が出ない

全 subject（scope `c` と `s`、`-direction both`）を走査して観測された関係は15種。**実装束縛に相当する関係は0種**である。

```text
type 80 / underlying 36 / owns-pattern 24 / input 24 / uses-semantic 20
relation 16 / outcome 16 / uses-validation 12 / attribute 12 / targets 12
contract-query-input 8 / data 8 / query 4 / capability 4 / output 4
```

```bash
tei trace -scope c -subject-kind semantic-capability \
  -subject-id security.authenticate -direction incoming -mode full
# total: 0
```

### 4.3 私はこれで誤読した

`security.authenticate` と `security.authorize` の incoming が 0 件だったため、最初は「この2つは未結合」と結論した。実際には `realize` 宣言があり、`grep` して初めて見つかった。

## 5. 原因

| 層 | 状態 |
| --- | --- |
| `ir.World` | `Bindings []realization.CustomGoBinding` を持つ（`internal/ir/world.go:35`） |
| 派生索引 | `d.CustomGoBindings` に写している（`internal/index/index.go:478`）。`Application`・`Capability`・`Adapter`・`Hook` を保持 |
| **参照グラフ** | **`CustomGoBindings` を1度も参照しない**（`grep -c` で 0） |

`derivedbridge.FromIndex` は `d.SemanticDependencies` と `d.DeclaredDependencies` だけを辺にする。束縛は `d` に入っているのに、どちらにも写されていない。

## 6. 影響の実測

`authenticate.tei` を削除して前後を比べた。

| 欄 | 変化 |
| --- | --- |
| `worldFingerprint` | `c92de09f…` → `b465bf81…`（変化する） |
| `sourceIndexFingerprint` | `1827f4b4…` → `4a559093…`（変化する） |
| **`graphFingerprint`** | **`a1b016fc…` → `a1b016fc…`（変化しない）** |
| `trace.rows` | 4 → 4（変化しない） |
| `trace.total` | 4 → 4（変化しない） |

**意味層の変更が参照グラフに現れない。** そのため `trace` を使った設計検証では、実装束縛の欠落も誤りも検出できない。`validate` も束縛について何も言わない。

## 7. 起票

[#3983](https://github.com/hat47x/tei-platform/issues/3983)。3案を示した。

| 案 | 内容 | 影響 |
| --- | --- | --- |
| **A（推奨）** | `d.DeclaredDependencies` に「pattern → capability を束縛する」辺を追加 | 束縛が見える。`graphFingerprint` が変わる（正しい方向） |
| B | `AuthorityLimits` に「実装束縛は参照グラフに含まない」と明記 | 実装変更なし。設計質問には答えられない |
| C | 束縛の無い capability を `validate` で警告 | 「未結合」を検出できる。全 capability に要求するかは設計判断 |

索引が既に束縛を保持しているので、A は `DeclaredDependencies` への追加だけで済む。ただし `graphFingerprint` が変わるため保存済み継続トークンは失効する。

## 8. 読み取れないこと（NOT RUN）

- 案 A を実装した場合の `graphFingerprint` の変化が既存の Evidence と整合するか。
- `realize` 以外の束縛（`HeaderDetailContractQueryBinding`・`ContractQueryExecutorBinding`・`ContractQuerySQLRealization`）も同様に辺になっていないか（`ir.World` に4種あるが、今回は `Bindings` だけを確認）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

**「モデルにあるが CLI から見えない」型**である。第28ラウンドで `capability.outcome` の辺を追加したのと同じ構造で、今回は `realize` 束縛が該当した。

ただし第28ラウンドと違い、**索引は既に束縛を保持している**。欠けているのは索引からグラフへの写像1箇所だけである。第40ラウンドで扱った「正本はあるが入口が1つ」と近い形でもある。

補助として確かめた `trace` のページング・指紋・alias は**すべて設計どおり**で、問題は見つからなかった。この領域は丁寧に作られている。
