# reference-studio-plugin-disable（#2046）

Studio のプラグインの**有効化・無効化・削除**について、必須依存の候補展開、適用直前の再検査、設定更新と拡張起動の分離、設計ソースの保持を扱う読取り専用の参照です。

Studio の実画面、プラグインの実行、Resolver、権限機構ではありません。設計ソースを書き換えず、プラグインを導入・起動・削除しません。実 Studio 管理操作と Host への接続は [#2550](https://github.com/hat47x/tei-platform/issues/2550) へ移しています。

## 実行

```bash
# 候補の提示（設定は変えない）
go run ./tools/reference-studio-plugin-disable/cmd/tei-studio-plugin-disable \
  -input tools/reference-studio-plugin-disable/testdata/disable-b.json

# 明示的に候補を確定して適用（設定の写しを返す。ソースは書かない）
go run ./tools/reference-studio-plugin-disable/cmd/tei-studio-plugin-disable \
  -mode apply -confirm \
  -input tools/reference-studio-plugin-disable/testdata/disable-b.json

# 拡張の起動失敗を模擬する
go run ./tools/reference-studio-plugin-disable/cmd/tei-studio-plugin-disable \
  -mode apply -confirm -fail-extension \
  -input tools/reference-studio-plugin-disable/testdata/disable-b.json
```

## 入力

```json
{
  "state": {
    "settingsVersion": "s3",
    "gitHead": "head-abc",
    "permission": "allowed",
    "standardEditorAvailable": true,
    "plugins": [
      {"id": "a", "installed": true, "enabled": true, "requiredDeps": ["b"]},
      {"id": "b", "installed": true, "enabled": true, "requiredDeps": ["c"]},
      {"id": "c", "installed": true, "enabled": true},
      {"id": "d", "installed": true, "enabled": true, "optionalDeps": ["b"]}
    ],
    "sources": [
      {"path": "semantic/order.tei", "fingerprint": "sha256:aaa", "unknownAttributes": true, "unknownElements": true}
    ]
  },
  "request": {"action": "disable", "pluginId": "b"}
}
```

- `action`は`disable`・`remove`・`enable`
- `requiredDeps`は、そのプラグインが動くために必要なプラグインです。`optionalDeps`は機能を追加するだけです
- `sources`は設計ソースの path と指紋です。`unknownAttributes`・`unknownElements`は、未知の属性・子要素を含むソースであることを示します
- `retainedData`はデータが残っていることを示します。自動の再導入・自動起動の理由にはしません

## 無効化・削除の計画

`b`を無効にすると、`b`を必須依存とする`a`も候補になります。

| 出力 | 意味 |
| --- | --- |
| `requested` | 利用者が選んだ対象（`b`） |
| `candidates` | 実際に無効化する集合（`a`・`b`） |
| `additional` | 依存のために追加された候補（`a`）。`requested`と別に示す |
| `unchanged` | 変更しない行（`c`・`d`・…）。行選択は変えない |
| `optionalFeaturesLost` | 任意依存の機能が失われる組（`d:b`） |
| `confirmationRequired` | 追加候補がある場合に`true`。明示の確定が無いと適用しない |
| `restartRequired` | 再起動・再生成が必要な対象。静的構成では設定変更だけでは反映されない |
| `sourcesPreserved` | この計画がソースを書かないこと |
| `selectionChanged` | 常に`false`。行選択を勝手に変えない |

- 必須依存の循環は`dependency-cycle`として拒否し、候補を提示しません。任意依存と区別します
- 存在しない依存は`unresolved-dependency`として、`<plugin>:<dep>`を`unresolved`に残して拒否します
- 未導入のプラグインは`not-installed`、すでに無効なら`already-disabled`として拒否します
- 権限がない場合は`permission-denied`で拒否します

## 適用直前の再検査

`Apply`は、計画時のスナップショットと現在の状態を比べます。1つでも違えば設定を変えず、理由を返します。

| 変化 | 理由コード |
| --- | --- |
| 設定版が変わった | `stale-settings:<旧>-><新>` |
| 宣言（依存・版・導入状態）が変わった | `stale-declaration` |
| Git HEADが変わった | `stale-git-head:<旧>-><新>` |
| ソースの指紋が変わった | `stale-source:<path>` |
| 権限が失効した | `permission-revoked:<現在>` |
| 計画の内容が改変された | `plan-tampered` |
| 追加候補を確定していない | `confirmation-required` |

**Git HEADが同じでも、未コミットの編集でソース指紋が変われば`stale-source`で止まります。** HEADの一致だけを根拠にしません。

## 設定更新と拡張起動

`ApplyResult`は、設定の更新と拡張の起動を別の値で返します。

- `settingsApplied`：設定の写しを適用できたか。`settingsAtomic=true`
- `extensionsStarted`：拡張の起動・再生成に成功したか
- `allSucceeded`：両方が成功した場合だけ`true`
- `extensionErrors`：起動の失敗
- `externalEffectAtomic=false`：設定の原子性と、拡張の外部作用を同じ保証にしない

起動に失敗しても`settingsApplied=true`のままにし、`allSucceeded=false`とします。

## ソースの保持

`Apply`はソースを書きません。`sourcesAfter`は入力と同じで、`sourcePreserved=true`です。未知の属性・子要素を含むソースも、バイト列の指紋が変わらないことを確認します。専用機能が失われても`standardEditorAvailable=true`（標準の編集入口）を維持し、`pluginSpecificFeature=false`として区別します。

## 再有効化

`enable`は明示の操作です。必須依存がすべて導入済み・有効でなければ`dependency-not-enabled:<dep>`で拒否します。不足する依存を自動導入しません。保持データがあっても`autoInstalled=false`・`autoStarted=false`で、自動導入・自動起動をしません。任意依存が使えない場合は`optional-not-available:<dep>`を返します。

## 試験

```bash
go test ./tools/reference-studio-plugin-disable/... -count=1
```

- `b`の無効化で`a`が追加候補になり、`c`は変わらない
- 追加候補がある計画は、確定するまで適用しない
- 適用後も無関係な行の有効設定が変わらない
- 必須依存の循環と、存在しない依存を区別して拒否する
- 設定版・宣言・ソース指紋・権限の変化を、適用前に検出する
- Git HEADが同じでも未コミットのソース編集を検出する
- 設定更新と拡張起動を分け、起動失敗を全成功にしない
- 無効化・削除の前後でソースがバイト同一で、標準編集入口が残る
- 再有効化は明示操作で、依存の自動導入・自動起動をしない
- 同じ入力から同じ計画・結果を得て、入力を変更しない

## 範囲と非主張

- 合成入力に対する静的な計画・判定です。実 Studio・実 Host・実プラグインの動作ではありません
- 正式な Studio 管理操作、Host への適用、再起動・再生成の実測は未実施です（[#2550](https://github.com/hat47x/tei-platform/issues/2550)）
- プラグインの動的ロード、悪意あるプラグインの隔離、外部DBの保持、本番無停止は扱いません
- 新しい Resolver・権限機構・正本を作りません
- ソースを書き換えず、データを削除しません
