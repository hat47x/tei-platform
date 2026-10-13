# E24：機能宣言の型と、実際の操作入力を検証する責務

- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)。
- **対象**：SUI Nativeの確定操作 `sui.move`。実際の入力検査・保存・認証は[Issue #993](https://github.com/hat47x/tei-platform/issues/993)が所有する。
- **確認時点**：2026-10-12。TEI側は試験の基点main `aa24a97a4473e8a2c9b82cd7bde63e6c5602498c`、SUI側はmain `4bd493f9942890d52117461886207d628f3554e2` の読み取り結果。
- **実装**：[E24のGo試験](e24_native_input_contract_boundary_test.go)。一時領域の宣言だけを変更し、新しい共通解析器、独立した入力スキーマ、保存先は追加しない。

## 現在の二つの契約が表すもの

[E22](E22_NATIVE_APPLICATION_CANDIDATE.md)で追加した空の `capability sui.move {}` は、TEIの既存Driverが操作識別子を解決できることしか示さない。進行中の[E23（PR #1398）](https://github.com/hat47x/tei-platform/pull/1398)は、構成の選択から共通Actionと模擬HTTPS保存先への接続を担当する。どちらもSUIの入力形式をTEIの型だけで検証したという証拠ではない。

SUIの実ソースを確認すると、操作の確定時に外側の共通Action v1形式を使い、内側の入力には `cardId`、`x`、`y` の三項目を送る。

| 情報 | 正本と責務 | TEIの現行機構で言えること | 言えないこと |
| --- | --- | --- | --- |
| `applicationID`・`actionID` | TEI共通ActionとApplicationの操作定義 | 既存Driverは`sui`と`sui.move`を照合できる | それだけで実行を認可しない |
| `resourceID`・`expectedRevision` | 共通Actionの外側、実際の保存先が版を照合 | 汎用の要求形式を再利用できる | SUIが要求する64桁の版識別と原子的書込みを意味定義だけで保証しない |
| `payload.cardId` | SUI固有の入力、空文字不可 | 仮の`std.text`属性で文字列という概念を表せる | 空文字禁止やCard存在・識別一意性までは証明しない |
| `payload.x`・`payload.y` | SUI固有の数値。有限値と移動先の判定はSUIが検証 | 仮の`std.decimal`属性で数値の意味的分類を示せる | JSON数値、有限性、Canvasの座標系や範囲を確認したことにならない |
| `payload`全体 | [SUIのTypeScriptアダプター](https://github.com/hat47x/sui-sensemaking/blob/4bd493f9942890d52117461886207d628f3554e2/03_Implement/frontend/src/api/tei_card_move_action.ts)と[Python受信口](https://github.com/hat47x/sui-sensemaking/blob/4bd493f9942890d52117461886207d628f3554e2/03_Implement/backend/src/sui_sensemaking_api/routes/docs.py) | 型付き意味定義への参照を保持できる | Payloadのキー名、厳密な型、未知項目、重複キー、制限値を自動検証したことにはならない |
| 利用者と権限 | SUIおよびTEI Hostの現在の認証済み文脈 | Capabilityの所在を解決できる | 宣言を追加しても操作権限は得られない |

特に、TEIの`Capability.Inputs`は意味上の入力名と型への参照であり、共通Action v1のJSON入力を直接復号する実行機構ではない。[CanonicalCommandGate](../../tools/reference-sui-action-boundary/actionboundary/canonical_gate.go)はCommandの対応を調べ、[JSONCommitHandler](../../tools/reference-sui-action-boundary/actionboundary/http_commit.go)はApplication固有の`PayloadDecoder`を要求する。責務を重ねない。

## 最小の型付き候補

試験では[E22の隔離作業領域](e22_native_application_candidate_test.go)の`native-action.tei`だけを置き換えて、次の**仮の入力モデル**が既存の構文解析、出典索引、Driver、意味検証にどう映るかを確認する。

```tei
namespace sui

data CardMoveInput {
    attribute cardId type std.text
    attribute x type std.decimal
    attribute y type std.decimal
}

capability sui.move {
    input move sui.card_move_input
}
```

これはSUIのCardやDocumentをTEIへ複写するものではない。操作入力を意味モデルに投影できるかを調べる試験専用の宣言である。SUIの正本は引き続きSUIに置く。`DataEffect update`も付けない。仮の入力モデルの更新と、SUIが所有するDocumentの実変更を同一とみなせないためである。

## 回帰試験で区別すること

1. 型付きCapabilityの`move`入力が`sui.card_move_input`へ解決され、`cardId`・`x`・`y`が既存の型と出典を持つことを確認する。
2. 入力が存在しない型`sui.absent_input`を参照した場合、Driverの操作識別子解決に成功しても、既存の意味検証で`UNKNOWN_SEMANTIC_CAPABILITY_INPUT`が返ることを確認する。**識別子解決と意味検証を一つの合格にしない。**
3. 型付き宣言が存在しても、共通の操作判定器はJSONを検証しない。アプリケーション固有の復号処理を登録しなければ、共通HTTP入口が`503 payload_decoder_unavailable`で拒否し、模擬保存先への書込みがゼロであることを確認する。

これらは型付きモデルを宣言した後の、**検証を要する境界の明確化**である。SUIのTypeScript・Python本体をもう一つのGo版として実装する試験ではない。既存アプリ側の入力検証をTEIの静的な型情報で置き換えない。

## 構成採用時に残る選択

正式な構成採用に必要なのは、正本を増やすことではなく、採用した操作入力契約の版・出典、TEIの意味定義との対応、実際のアプリケーション側復号処理、対象実行先、現在の資格情報と認可の所在を**別々に明示**すること。既存の契約索引・選択機構へ参照で接続できる場合は新しい定義を作らない。

今回の試験で数えられるものは、既存のE22試験用作業領域の入力定義1ファイルを差し替えたことと、既存の構文解析・解決・検証・HTTP入口を再利用したことだけである。実SUIへの導入に必要な変更ファイル数、生成AIへの追加質問数、実行先選択、別リポジトリ更新時の再検証量は未計測であり、ゼロとは扱わない。

```sh
go test -count=1 ./reference/application-composition -run '^TestE24' -v
```

**判定**：既存TEIは入力の意味上の構造を記述できるが、そこからアプリケーション固有の実際のJSON入力検証を自動生成して実行しているわけではない。この差を新しいCore種別や第二のスキーマで直ちに埋めるのではなく、まず共通Actionとアプリケーション所有の復号処理の既存境界を維持する。[E21の比較](E21_DUAL_CONSUMER_COMPOSITION_GAP.md)に必要な実利用先での採用費用と証拠が得られるまで、独立した構成プロファイルの正式採用は保留する。
