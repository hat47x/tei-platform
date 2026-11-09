# reference-observation-router-connect

`ObservationProviders` の宣言から provider を解決し、**既存の `observation.Router` で実際に入る**参照です。調査で見つかった「解決はあるが、製品の入口から `Observe` が走らない」断絶を、**新しい解決器を作らずに**閉じられるかを試します。

~~~text
go run ./tools/reference-observation-router-connect/cmd/tei-observation-router-connect \
  -workspace reference/workspace -environment development
~~~

## 何をするか

| 段 | 使う既存の関数 | 意味 |
| --- | --- | --- |
| 解決 | `observation.Binding` | 宣言から provider の ID を選ぶ（`cmd/tei -plan` が既に使う関数） |
| 選択 | （Host の registry を引く） | その ID が Host に登録されているか |
| 入場 | `observation.Router.Observe` | provider の `Observe` が実際に呼ばれたか |

**Router も Provider もこの参照は組み立てません。** どちらも呼出し側（Host）が渡します。

## 数え方

| 数 | 意味 |
| --- | --- |
| `observedCount` | `OBSERVED` になった件数 |
| `refusedCount` | それ以外の件数（解決不能・未登録・不適合・入場後失敗） |
| `invokedCount` | **provider の `Observe` に入った件数**。応答が使えなくても数えます |

各結果は `resolved`（宣言から解決した）／`selected`（Host が登録していた）／`invoked`（実際に入った）を**別々**に持ちます。この3つを混ぜないことが、この参照の目的です。

## 状態

| 状態 | 意味 |
| --- | --- |
| `OBSERVED` | 解決・選択・入場がすべて成立し、断片の同一性も合った |
| `BINDING_MISSING` | その kind の宣言が無い、または重複している |
| `NOT_REGISTERED` | 宣言が指す provider を Host が登録していない（**入場前**） |
| `NOT_CONFORMING` | provider が kind を支えない、または ID が合わない（**入場前**） |
| `OBSERVE_FAILED` | **入場後**の失敗。断片の同一性不一致もここに入る（既存 Router が拒否する） |
| `FRAGMENT_INVALID` | Router が provider に入らずに応答した（参照は観測として扱わない） |

## 部分成功をどう表すか（CON-12d）

`attemptedCount` は試した件数、`complete` は**すべての試行が成立したか**です。
件数だけでは 1/1 と 1/2 を区別できないため、`attemptedCount` を併記します。
**一部だけ成功した場合は `complete=false`** で、拒否はそのまま残ります。
何も試さなかった場合も `complete=false` です。

**`complete` は「世界がそうなっている」証明ではありません。** 試した全部に答えが返ったという意味だけです。

## unknown を refusal と分ける（CON-12d 追補）

| 数 | 意味 |
| --- | --- |
| `refusedCount` | **入る前**に止めた回数。何も呼ばれていない |
| `unknownCount` | **入った後**に失敗した回数。`invokedCount` の内数 |

**入った後の失敗を `refusedCount` に数えると「実行していない」と読めます。** 実際には呼出しは起きており、世界を読んだかどうかは分かりません。そのため `unknownCount` として別に数えます。

**`unknownCount` は `invokedCount` の内数**です。3つの数を足し合わせないでください。

## この参照が示さないこと

- **観測が外部状態を反映していること**。付属 CLI の Provider は固定の断片を返し、何も読みません。
- **製品 CLI へ `Observe` を足すべきか**。C-4（外部に委ねる）も候補のままです。
- Artifact の信頼、現在の Grant、開示 Scope。
- 認証・権限・永続的な receipt。

記録は [`plan/evidence/OBSERVATION_ROUTER_CONNECT_REFERENCE_2026-11-09`](../../plan/evidence/OBSERVATION_ROUTER_CONNECT_REFERENCE_2026-11-09/README.md) にあります。
