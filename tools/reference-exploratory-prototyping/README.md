# reference-exploratory-prototyping（#1441 B5）

探索的な試作の反復と、その成果の部分採用を確認する読取り専用の参照です。仮説を受理済みの仕様へ昇格させず、AIによる候補と実際の観測を区別し、試作の一部だけを正式な設計へ取り込みます。Sourceを変更しません。

この参照はコードを生成・実行しません。試作の反復記録と採用の選択を入力として受け取り、境界だけを扱います。

~~~text
go run ./tools/reference-exploratory-prototyping/cmd/tei-exploratory-prototyping -input request.json
~~~

`-input`は検査の依頼を書いたJSONです。`-`で標準入力から読みます（既定）。`-effort`に`{rigorous, exploratory}`の作業量を渡すと、比較も出力します。

## 入力

- `profile`: `exploratory` / `rigorous`
- `pluginAvailable`: 試作工程プラグインが導入されているか
- `iterations`: 試作の反復。`id`、`hypothesis`、`inputFixture`、`revision`、`expected`、`observed`、`observation`（`execution` / `ai-candidate` / `none`）、`nextDecision`（`continue` / `adopt` / `discard` / `unresolved`）、`selfDerived`
- `selections`: 採用の単位。`iteration`、`kind`（`data` / `rule` / `function` / `adapter` / `scenario` / `artifact` / `discard` / `hold`）、`ref`

## 出力

- `applicable`: 試作工程を使えるか
- `iterations`: 反復ごとの`measured`（実行から観測したか）と`reproducible`
- `adoptions`: 選択ごとの`accepted`と、拒否した理由
- `adopted` / `discarded` / `held`: 採用する部分・破棄する部分・引き続き不明な部分
- `summary` / 診断 / 指紋
- `effortComparison`: `-effort`指定時の比較

## 守っている境界

- 試作工程プラグインが無い場合は`applicable=false`とし、理由を示します（`plugin-unavailable`）。通常の堅牢工程とコア契約・権限を共用します。
- 反復は二つ以上を求め、仮説・入力fixture・Git revision・観測・次の判断を反復ごとに区別します（`insufficient-iterations`）。AIによる候補は観測として数えません（`observation-not-measured`）。
- 期待値を試作品自身から得た場合は採用を拒否します（`self-derived-expectation`）。
- 意味を持つ`data` / `rule` / `function`は、`adopt`とされた反復からだけ採用します（`iteration-not-accepted`）。`scenario` / `artifact` / `adapter`は候補として保持できます。
- 採用は試作全体ではなく一部とし、破棄・保留を同時に表せます（`no-discard-or-hold`）。
- 作業量の比較は、両方を実測した場合だけ数を使います。未実測は`measured=false`と`note`で示し、推測しません。
- 過去のfixture PASSや画面描画の成功を、実Providerや本番運用の保証にしません。

## 未実装

- 試作コードの生成・実行と、観測の自動収集
- 正式なSource / Plugin / Scenarioへの反映と再読込（#267、#1131、#1180）
- 通常の業務向け工程（#1445）との切替時の再入力・二重管理の計測
- 実際の案件での作業量の実測
