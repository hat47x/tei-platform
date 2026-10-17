# Data Derivation D3：導出値の鮮度・キャッシュ・確定値 Reference（#1641）

本ディレクトリは **導出値の鮮度契約とキャッシュ再利用条件を読取り専用で検査する小さな参照** であり、TEIの正式な分散キャッシュ製品、履歴保存基盤、認可エンジン、永続化実行器、受入済みAPIではありません。実装主体は `derivationcache.go`、受入は `derivationcache_test.go`、起動点は `cmd/tei-derivation-cache/` です。`正本`を書き換える経路はありません。

## 区別する対象

| 種別 | 例 | 本参照での扱い |
| --- | --- | --- |
| 計算可能な導出値 | 数量2×単価300 → 600 | キャッシュ契約を必須にしない（`DIRECT_EVALUATION`） |
| キャッシュ／事前計算結果 | 引当可能20 の事前計算 | 規則版・入力版・評価文脈・情報源時点で再利用可否を判定 |
| 業務上の確定保存値 | 受注確定で保存した600 | キャッシュ経路・再描画で上書きしない |
| 外部情報を含む計算 | 複数提供元の輸入 | 時点一致を保証できないなら保証不可 |

## 判定する意味（採用前）

| 条件 | 期待 | 結果コード |
| --- | --- | --- |
| キャッシュを要求しない純粋計算 | 鮮度宣言を必須にしない | `DIRECT_EVALUATION` |
| 全版・文脈・時点が一致 | 再利用 | `CACHE_REUSED` |
| 入力版が変わった | 旧値を現在値として返さない（値は不明） | `STALE_INPUT` |
| 規則版が変わった | 根拠なく旧キャッシュを流用しない | `STALE_RULE` |
| 権限・tenant・情報源時点が違う | 同一キャッシュとして共有しない | `CONTEXT_MISMATCH` |
| 確定保存値への再評価 | 上書きしない | `CONFIRMED_SAVED_IMMUTABLE` |
| 複数情報源で時点不一致 | 強い整合性を偽装しない | `SNAPSHOT_INTEGRITY_UNPROVEN` |

値が不明な再利用拒否では `valueNull=true` を返し、古い値を「現在値」として表示させません。

## 合成Fixture

在庫120・引当済100 → 引当可能20。明細 数量2×単価300 → 600。

| 操作 | 本参照の結果 |
| --- | --- |
| 単純導出（キャッシュ要求なし） | `DIRECT_EVALUATION`、`value=600` |
| 引当済が90に変わった後の旧20 | `STALE_INPUT`、値は不明 |
| 商品単価300→350後の確定保存600 | `CONFIRMED_SAVED_IMMUTABLE`、`value=600`（新規未確定計算は700） |
| 規則v1→v2後の旧キャッシュ | `STALE_RULE`、値は不明 |
| tenant・権限・時点が違う | `CONTEXT_MISMATCH`、値は不明 |
| 複数情報源・時点不一致 | `SNAPSHOT_INTEGRITY_UNPROVEN` |

## 実行方法

```sh
go run ./tools/reference-derivation-cache/cmd/tei-derivation-cache -input request.json
```

`-input -`（既定）は標準入力から読みます。入力例：

```json
{
  "cacheRequested": true,
  "entry": { "ruleVersion": "1", "inputVersion": "R0", "contextClass": "tenantA/admin", "sourceTimePoint": "t0", "value": "20" },
  "current": { "ruleVersion": "1", "inputVersion": "R1", "contextClass": "tenantA/admin", "sourceTimePoint": "t0" }
}
```

## 責務と分界

- 規則の意味契約は #1637（D1）、Query計画は #1640（D2）、Lowering・版・失効は #1121、情報鮮度の文脈は #958、副作用は #1174 が所有します。
- 強い整合性の保証判断は #1708（D11）へ委譲します。確定値のOwnerは #1129／#1641 です。
- 汎用分散キャッシュ、全情報源横断の原子的トランザクション、全案件への履歴保存義務、認可エンジン、確定値の暗黙再計算はしません。

## 限界

- 合成入力に対する静的な判定です。既存Provider・Application・Caching実装への適合や、実DB・実キャッシュでの振る舞いの証拠ではありません。
- 合成Fixture成功と実Providerの実証を区別しており、後者は本PRでは未実施です。
- `-race` は cgo を伴わない可搬Goでは実行していません。
- D3-M2（具体案件が発生した場合の最小実行Reference追加）は本PRの範囲外であり、未実証として残します。
