# E34：案件Go＋Litの契約Bridge追加を、実PR改訂差分で再資格確認する

- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)。[E32](E32_REAL_PROJECT_PLUGIN_FOOTPRINT.md)は初期の実Draft、[E33](E33_PROJECT_PLUGIN_WIRING_BOUNDARY.md)は当時のSourceに正式Providerの裏付けがないことを検査した。E34では**同じPR #1567の後続実改訂**を別の観測として扱う。
- **固定したGitHub改訂**：[PR #1567](https://github.com/hat47x/tei-platform/pull/1567)の `be193266f1556e090b59cf2c237a23c0ca60236e`（E32） → `596c522489dde0ad5184eba6e0220cd758f16636`（E34）。GitHubの`compare_commits`と`pulls/1567/files`からファイルパス、変更種別、前後Blob SHA、増減行数を記録した。両HEAD間は**8コミット**。
- **機械可読証拠**：[固定差分](e34_project_bridge_pr1567_delta.json)／[差分読取り器](e34_bridge_revision_review.mjs)／[Node.js回帰試験](e34_bridge_revision_review.test.mjs)。差分の旧Blobは[E32の20ファイルの出典](e32_project_plugin_pr1567_snapshot.json)と一致するかを照合する。これは**参照試験の入力**であり、製品の第二のSource Index／Lock／Resolverではない。

## 当時のPRから増えた実装範囲

| 元のE32固定HEADからの変更 | 件数 | 差分行数 | 内容 |
| --- | ---: | ---: | --- |
| 新規Go Bridge | 1 | +93 | `contract_bridge.go`。既存`composition.Registry.Apply/Select`と`NewSelectedCoordinator/Acquire`を診断用に使用 |
| 新規Go Bridgeテスト | 1 | +58 | `contract_bridge_test.go`。選択したProviderでの助言と、削除・案件Scope違いの拒否 |
| 既存Host改修 | 1 | +5 / −1 | `main.go`。案件Goの直接呼出しをBridge経由へ変更 |
| 既存README改修 | 1 | +5 / −1 | `README.md`。Bridgeの実験と制約を説明 |
| **合計** | **4変更パス（新規2・変更2）** | **+161 / −2（純増159行）** | 元の20ファイルのうち他18ファイルは、この二改訂のGit差分上は未変更 |

初期PRの差分は**20新規ファイル・1,267行追加**だった。その後の純増159行により、後続固定HEADのPR全体では**22新規ファイル・1,426行**の差分となった。これはGit差分の量であり、工数・生成AIトークン・質問回数ではない。今回のGo実装は**PR #1567の別担当による変更**で、このE34はBridgeソースを移植・二重実装していない。

## 実Bridgeを読んで確認したこと

固定HEADの`tools/reference-project-advice-lit/contract_bridge.go`を読んだ結果、案件Go関数に到達する前に以下の**診断用の構成経路**が追加されていた。

`Registry.Apply`（案件ScopeにContractとProviderを宣言）→ `Registry.Select`（宣言上のProvider選択）→ `NewSelectedCoordinator`（参照BindingとGuardを受ける）→ `Acquire`／`Release`（利用中Leaseを操作）→ 案件Go `Evaluate`。

この実変更により、E33に記録した「**E32時点のPRにProvider登録の証拠がない**」という観測は**過去の固定改訂に限る**。E34の後続改訂では、**診断Host内のテスト用Contract/Provider宣言と構造上の選択が追加された**。

ただし、次の事項を区別する。

- Contract `tei.reference.project.customer-advice/v0`・Shape `customer-code-note-to-advisory-v0`は**参照プログラムで宣言**されたもので、製品Runtimeが正規の#951 Contractとして発行・適合検査した証拠ではない。
- Bridgeの`ArtifactDigest`は`sha256:`＋**64個の0**という合成値で、実バイナリのDigestではない。
- Host Guardは構造体`Binding`の一致を検査する。現在の本人・権限・Artifact・fencing状態を照会する製品Guardではない。
- 受注明細は別プロセスの`GET /scenario/read`を読み取るだけで、正式受注Actionの変更やAccepted業務意味の変更は確認できない。

既存Compositionの再利用により、**独立Profileや第二Resolverを追加せずに案件用の候補選択は試作できた**。Product Releaseで必要な証拠とPermissionの未成立は残る。

## 実行時証拠の現状

固定HEAD `596c522...` の[案件Go/Node専用CI #38372666380](https://github.com/hat47x/tei-platform/actions/runs/38372666380)は、Goの整形検査・両Goパッケージの単体試験までは成功し、その後の`smoke_order_entry.sh`で**failure**となった。ログには`reference process exited`しか出ず、API／案件Hostのどちらが終了したかは判定できない。上流・Hostのログを一時ファイルに書いた後に削除する仕組みなので、この記録だけから原因を断定しない。

この失敗と、同HEADにおける実Chromeの進行・結果は**別のCIの証拠**であり、構成機構の受理資格を互いに代用しない。PR #1567は当該観測時点でDraft・未マージ。後続更新が入れば、正確なHEADを固定して再検証する。CI実行先では全てのネットワーク・権限・Artifact状態を再現できたと主張しない。

## 再現と負例

```sh
node --test reference/application-composition/e34_bridge_revision_review.test.mjs
node reference/application-composition/e34_bridge_revision_review.mjs
```

E34は、GitHubで観測した**4件の改訂ファイル**とE32の前後Blobを照合し、差分・変更箇所を再計算する。偽ったSHA、前版に存在したファイルを「新規」と申告すること、パスの重複、`internal/composition`を実PR変更と偽装することなどを拒否する。テストはGitHubへ再接続せず**固定したSourceメタデータの内部整合**だけを検証する。

`bridgeSourcePresent=true`はBridgeファイルの出現を示す**メタデータ**であり、実装適合・Provider Authorityを与える状態ではない。契約正本、Artifact真正性、Host Permission、正式受注Actionの業務受理、本番配備、実作業工数、AI質問回数はすべて`null`のまま保持する。

## E34の判断

追加のComposition Profile採用は**保留**する。今回、案件Plugin内で既存Compositionを再利用してProvider候補の選択とLease取得まで進められたのは有用な実証だが、`#951`側から信頼できる契約・成果物・適合証拠と、Host側の現在権限が揃ったわけではない。次の実験ではPR #1567の診断Hostの起動失敗を局所的に切り分け、正規のArtifact/Provider証拠とHost Guardを通す対象・方法を、#951／#997と重複しない境界で確かめる。
