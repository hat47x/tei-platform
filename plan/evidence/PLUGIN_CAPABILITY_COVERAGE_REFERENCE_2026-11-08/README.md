# 宣言されたプラグイン能力は、環境から選ばれているか（2026-11-08）

- 対象: [Plugin / Bundle](../../../docs/plugins.md)と、[構成の選択と実行許可を分ける原則](../../design/PROJECT_PLUGIN_COMPOSITION_DESIGN.md)
- 位置づけ: **参照実装と実行記録**。選ばれていない宣言を欠陥と決めるものではない。
- 計測した版: `raw/head.txt`（`origin/main` `fb85a191` の作業ツリー）。

プラグインは `provides` で能力を宣言し、`Environment` がその能力を選びます。しかし**宣言された能力のうち、どの環境からも選ばれていないもの**を一覧できる経路は、これまでありませんでした。この記録は、その一覧を出す参照実装を作り、参照Workspaceで観測したものです。

## 1. 何を作ったか

`tools/reference-plugin-capability-coverage`（新規・読み取り専用）。

- **宣言**: `World.Plugins` の `Provides[]`（Kind + ID）。
- **選択**: 環境の `deployment-executor`・`observation-provider`・`evidence-provider` の Bindingが、**同じ Kind + 同じ ID** を指しているか。
- 出力: 宣言ごとに `selected`／`unselected` と、選んだ環境名。
- 書込みなし。Provider解決・実装選択・権限発行はしません。

**`database-provider` は数えない**ことにしました。環境の `database-instance` が指すのは `postgres`／`oracle` という**提供技術名**で、能力ID `database.postgres` ではありません。この2つを同一視すると「選ばれた」と誤って報告します。この判断は規則の指紋に含めています。

## 2. 観測

`reference/workspace` の結果です。

```text
environments: ['development', 'production']
declared capabilities: 12
unselected: 6

selected   tei.database.observer.reference  observation-provider   database.observer      development,production
unselected tei.database.oracle              contract-query-executor database.query.oracle  -
unselected tei.database.oracle              database-provider      database.oracle        -
unselected tei.database.postgres            contract-query-executor database.query.postgres -
unselected tei.database.postgres            database-provider      database.postgres      -
unselected tei.editor.intelligence.sql      editor-intelligence    editor.sqlglot         -
selected   tei.health.reference            evidence-provider      health.reference       development,production
selected   tei.migration.atlas              deployment-executor    migration.atlas        development,production
unselected tei.migration.atlas              migration-provider     migration.atlas        -
selected   tei.runtime.reference            deployment-executor    runtime.reference      development,production
selected   tei.runtime.reference            observation-provider   runtime.observer       development,production
selected   tei.traffic.reference            deployment-executor    traffic.reference      development,production
```

`plugin-lock` も同じ2環境を持ち（`development` 8プラグイン、`production` 7プラグイン）、環境の存在自体は一致しています。

## 3. この観測が示すこと

- **12の宣言のうち6つが、どの環境からも選ばれていません。** 内訳は、`database-provider`2件（上述のとおり数え方の問題）、`contract-query-executor`2件、`migration-provider`1件、`editor-intelligence`1件です。
- `database-provider` の2件は**数え方の限界**であり、実際には Postgres/Oracle の database instance が存在します。この一覧だけを見て「使われていない」と読むのは誤りです。
- `editor-intelligence`（`tei.editor.intelligence.sql`）は `development` の環境が `plugin tei.editor.intelligence.sql ^1` として**明示的に選んで**いますが、**能力の Binding は無い**ため `unselected` です。プラグインの選択と能力の選択は別であることが、この一覧で見えます。
- `migration-provider`（`migration.atlas`）は `deployment-executor` としては選ばれていますが、`migration-provider` としては選ばれていません。**同じプラグインの複数能力のうち、一部だけが選ばれている**例です。

## 4. この記録が示さないこと

- **`unselected` が欠陥であること**。Bundleが別環境向けの候補を同梱する場合、選ばれないのが正常です。
- **`database-provider` の実際の利用**。数え方を変えれば結果は変わります。
- **Provider の適合・Artifact の信頼・Host の権限**。選ばれたことは宣言であり、実装の存在や適合の証明ではありません。
- **実DB・実Provider**。合成の Workspace と読み取りのみです。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順 |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（新規ツールと`aw/`のみ） |
| `raw/reference-coverage.json` | 宣言と選択の一覧 |
| `raw/reference-plugin-lock.json` | `tei plugin-lock` の出力（環境の存在の照合） |
| `raw/summary.txt` | 上の表を作る元の一覧 |

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `41224f0dcb7635272f3d87835e930a77c1eeaa3b56a9b33ae8bac4081fa9b2b7` | 1778 |
| `raw/head.txt` | `f893af02b5a4e93b1eb4f59530e36394441c1476e1442522d0df543cbd673362` | 73 |
| `raw/pre-status.txt` | `1384f484b117e1d1205566be7582a77cf4b6784edce1e9e071245e1866623214` | 289 |
| `raw/reference-coverage.json` | `85879902e5d55deb049ba49ef7170dae4c4462aa498e90fb00d370eecde3660a` | 3473 |
| `raw/reference-plugin-lock.json` | `bfd0c518c5de49757d7c982029aa50def567a686996aee1fefa246c1a16a7d4d` | 3542 |
| `raw/summary.txt` | `5964b21cf745db8ecfa952e09c50b627f44c6b4b41d78f2bd6ab9793c0a7bd36` | 1416 |

`raw/pre-status.txt` が0行でないのは、計測時に**新規ツール本体が未追跡**だったためです（`tools/reference-plugin-capability-coverage/`の3ファイルと`aw/`）。
