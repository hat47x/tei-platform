# reference-service-carrier-connect

判断材料の **C-2**（`servicehost` を実行の入口へ入れる）を、**試験の外で組み立てる**参照です。C-1（[`reference-binding-to-execution`](../reference-binding-to-execution/README.md)）と比べられるようにするためだけに置いています。

## C-1 との違い

| | C-1 | C-2（この参照） |
| --- | --- | --- |
| 実行する対象 | 配備の動作（`executor.Command`） | Service の要求（`requestID` + payload） |
| 世代のピン留め | なし | **あり**（`Coordinator.Acquire`） |
| Host が渡すもの | executor の registry | Coordinator・Runtime registry・RouteSource |

## 何を組み立てるか

Host が渡した入力から、以下の3つの既存関数を順に呼んで Invoker を作ります。

1. `servicehost.BindHostApplicationEnvironmentRoute`（Application/Environment を固定した RouteSource）
2. `composition.NewSelectedCoordinator`（Host の世代ガード）
3. `servicehost.NewSelectedServiceCarrierInvoker`（ピン留めした Invoker）

**参照は Runtime registry も Workspace も組み立てません。** すべて呼出し側（Host）が渡します。

## 拒否する場合

| 理由 | 扱い |
| --- | --- |
| Coordinator・RouteSource・Registry が無い | 実行しない（`CONNECT_INPUT_INVALID`） |
| Route が取得できない | 実行しない（`ROUTE_UNAVAILABLE`）。**Invoker を呼ぶ前に確かめます** |
| Route が別の Binding 用 | 実行しない（`ROUTE_BINDING_MISMATCH`） |
| Carrierへ入った後の失敗 | `ranCount=1`／`unverifiedCount=1`として`error`へ残す（`SERVICE_INVOCATION_UNVERIFIED`）。**事前拒否ではない** |
| Carrier応答が別Generation | `SERVICE_RESULT_GENERATION_MISMATCH`。先にCarrierを呼んだ事実を消さない |

拒否は値として返し、エラーで落ちません。

## 3つの数（Report v1）

| 数 | 意味 |
| --- | --- |
| `ranCount` | **carrier へ入った回数**。失敗しても、応答が使えなくても数えます |
| `refusedCount` | **carrier へ入る前に止めた回数**（Route 不在・入力不足・Route の Binding 違い・**Host guard の拒否**） |
| `unverifiedCount` | `ranCount` の内数。carrier に入ったが、エラーまたは別世代の応答で確定できなかった回数 |

**初期のv0実測**はCarrierへ入った後の失敗も`refusedCount`へ数える可能性があり、「未実行」と読めてしまいます。現行の[参照レポートv1](connect.go)は`Schema`・`RuleVersion`・`RuleFingerprint`を別版にして、呼出し前の拒否と呼出し後の不確定を分けます。過去の[実行記録](../../plan/evidence/SERVICE_CARRIER_CONNECT_REFERENCE_2026-11-09/README.md)のv0 JSON・SHA-256は変更せず保持します。

**参照が「入る前に止めた」と言えるのは、自分の事前検査だけで止めた場合だけです。** carrier の内部で
artifact 検証が拒否した場合は、carrier には入っているので `unverifiedCount` になります。

## Host guard の拒否は、実行ではない（2026-11-09追補）

`SelectedServiceCarrierInvoker.Invoke` は、**Host guard を通ってから** carrier に入ります。
したがって guard の拒否は **carrier へ入る前**に起きており、`refusedCount` です。

**初期の実装は `Invoke` を呼ぶ前に `ranCount` を上げていたため、guard の拒否が「実行した」と読めました。** 現行は入場を観測してから数えます。

| 事象 | 数え方 |
| --- | --- |
| Host guard が拒否 | `refusedCount++`（`carrierInvoked=false`） |
| Route 取得失敗 | `refusedCount++`（`carrierInvoked=false`） |
| carrier に入った後の失敗 | `ranCount++` / `unverifiedCount++`（`carrierInvoked=true`） |

## この参照が示さないこと

- 製品 CLI へ繋ぐべきか（判断材料の C-1〜C-4 は未決）。
- 実配備・実 Service の成立（記録用の Process は外部に触れません）。
- 認証・権限・永続的な receipt・外部状態の確認。
- `carrierInvoked`は**Carrierの入口**であり、Artifact検証で止まればProvider Processは実際には呼ばれません。`ranCount=1`からProvider実呼出しを推論しません。
- 引き渡したJSONと外部Providerの現在資格・結果の完全受領の検証。

実行記録は [`plan/evidence/SERVICE_CARRIER_CONNECT_REFERENCE_2026-11-09`](../../plan/evidence/SERVICE_CARRIER_CONNECT_REFERENCE_2026-11-09/README.md) にあります。
