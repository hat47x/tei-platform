# reference-format-version-migration（#1920）

定義（`.tei`、作業領域の宣言、プラグインが持ち込む定義形式）の**形式の版**を読み、版の間の変換を一か所の規則表で行う読取り専用の参照です。版の無い定義を暗黙の既定として読まず、内容の版を形式の版の代わりにせず、変換に成功したときだけ版の値を上げます。Sourceを書き換えません。

この参照は定義を解析しません。形式の版・属性・変換規則を入力として受け取り、変換の境界だけを扱います。

~~~text
go run ./tools/reference-format-version-migration/cmd/tei-format-version-migration -input request.json
~~~

`-input`は変換の依頼を書いたJSONです。`-`で標準入力から読みます（既定）。

## 入力

- `current`: コアが扱う現行の形式の版
- `rules`: 変換規則。`from` / `to`、`renames`（旧名→新名）、`setDefaults`（旧版の暗黙の既定を明示値として書き出す）、`removed`（定義上はあるが実行時に効かない属性）、`dropBySilently`（黙って捨てる不正な規則の負例）
- `needRoundTrip`: 古い版へ戻す往復が必要か
- `documents`: 定義。`id`、`version`（空は未記録）、`kind`（`format` / `content`）、`contentVersion`、`attrs`（`value`が`null`なら無い、`""`なら空、値があれば有る）

## 出力

- `strategy`: 一段ずつの連鎖（`chain`）と中心の表現（`hub`）のどちらを既定にするかと、その理由
- `versions`: 扱う版の一覧
- `outcomes`: 定義ごとの`status`（`success` / `success-with-warning` / `failure`）、`from` / `to`、`versionRaised`、`path`、変換後の`attrs`、`warnings` / `failures`
- `summary` / 指紋

## 守っている境界

- 形式の版が記録されていない定義は、暗黙の既定として読まず`missing-format-version`で失敗します。
- 内容の版を形式の版として扱う定義（`content-version-not-format`）と、版の値と内容の版が食い違う定義（`version-content-mismatch`）を拒否します。
- 知らない新しい版は処理単位で失敗とし、必要な道具の版を示します（`unsupported-newer-version`）。
- 属性の「無い」「空」「値がある」を区別し、変換後も保ちます。
- 廃止した属性は黙って読み飛ばさず、`attribute-removed`として要素と属性を示します。黙って捨てる規則は`attribute-silently-dropped`で失敗とします。
- 旧版の暗黙の既定は、版を上げるときに明示の値として書き出します。
- 現行の版は変換せずそのまま通し、同じ入力を繰り返しても結果が変わりません（冪等）。
- 変換に失敗した場合は元の定義を変えず、版の値も上げません（`versionRaised=false`）。
- 一段ずつの連鎖と中心の表現のどちらを既定にするかを、版の数と往復の要否から決め、理由を残します。

## 未実装

- `.tei`・`tei.workspace.yaml`の解析と、版の記録そのもの
- プラグイン固有の定義形式の変換規則の登録と呼び出しの入口
- 実際の編集環境・実行系・生成器・検証器・取込処理への接続
- 業務データの値の移行
