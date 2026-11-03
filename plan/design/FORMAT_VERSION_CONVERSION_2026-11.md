# 定義形式の版と変換の境界（#1920）

- 状態: 決定の記録（Owner判断 2026-11-02）と、実装済み範囲の整理
- 対象: 形式の版の記録、版間変換の方式、変換結果の分類、属性の三状態
- 非目標: 業務データの値の移行、新→旧の変換、外部形式（X-TEA等）の取込写像

## 1. 決定

| 論点 | 決定 | 理由 |
| --- | --- | --- |
| 版間変換の既定方式 | **中心表現経由** | 各版を一度中心表現へ落とし、中心表現から各版へ戻す。往復と属性の取りこぼしを一か所で検査できる |
| 変換を実行する部品 | **プラグインが変換を実行**し、コアは呼び出しと結果の型だけを定める | 形式ごとの知識をプラグインに閉じ込めつつ、入口・結果分類・記録はコアで統一する（ADR-0033の境界と整合） |
| 属性の「無い／空／値あり」 | **`.tei`文法も拡張して区別する** | 空値を「未指定」と同じに扱うと、版上げ時の暗黙既定の明示化ができない |

## 2. 実装済み: 形式の版の記録

### 作業領域

`tei.workspace.yaml` の `formatVersion` が作業領域全体の版を定める。

- 未指定は拒否（`missing formatVersion (supported: 1)`）
- 未知の将来版は拒否（`unsupported workspace formatVersion N`）
- 凍結artifactだけが `TEI_WORKSPACE_ALLOW_UNSPECIFIED_FORMAT=1` で明示的に旧形式を読む

### `.tei`本体

`.tei`は任意の先頭ヘッダで自分の版を宣言できる。

```text
formatVersion 1
namespace sales
```

- 未宣言（`FormatVersion == 0`）は**作業領域の版を継承**する。
- 宣言した場合は作業領域の版と一致しなければならない。
- 未知の版は拒否（`unsupported semantic formatVersion N`）。
- これは文法の世代であり、`contract <id> <version>`（内容の版）とは別の値である。
- ヘッダは`namespace`より前に置く。位置がずれた場合は構文エラーになる。

## 3. 実装した変換の形（決定に沿う設計）

`internal/formatversion` が単一の入口と結果分類を持つ。プラグインは`Codec`（`Decode`/`Encode`）だけを提供する。

- `Registry`: 1つの形式内の版ごとのcodecと変換入口
- `Formats`: **プラグインの登録口**。形式ID（例 `tei.semantic`）ごとに`Registry`を持ち、`Formats.Convert(format, source, from, to)`で振り分ける。同じ版番号でも形式が違えば衝突しない。同じ形式・版の二重登録は拒否する。未登録形式は`UNKNOWN_FORMAT`で失敗し、登録済み形式をメッセージに含める

### 中心表現

```text
[各版のSource] --decode--> [Central] --encode--> [各版のSource]
```

`Central`は次だけを保持する。

- 要素の種別と識別子
- 属性の三状態（`Absent` / `Empty` / `Value`）と値
- 元の位置（警告・失敗の対象を示すため）
- 未知の要素（そのまま保持し、黙って捨てない）

### 呼び出しと結果

コアが呼び出しと結果の型を定め、プラグインが`decode`/`encode`を提供する。

```text
Convert(source, from Version, to Version) Result
```

`Result`の分類:

| 分類 | 意味 |
| --- | --- |
| `Success` | 変換成功。失われた属性・要素は無い |
| `SuccessWithWarnings` | 成功したが、廃止属性・既定値の明示化など対象要素付きの警告がある |
| `Failure` | 変換できない。対象要素の識別子と理由を必ず付ける |

`SuccessWithWarnings`と`Failure`の診断は、要素識別子（例: `sales.order_line.quantity_positive`）と属性名を持つ。識別子を持たない診断は許可しない。

### 不変条件（試験で固定する）

1. **冪等**: 同じ版への変換は内容を変えない（`to == from`は恒等）。
2. **属性消失0件**: 入力の全属性が、出力の属性か`SuccessWithWarnings`/`Failure`の診断のどちらかに対応する。
3. **版だけ上げない**: 変換に成功したときだけ版の値を上げる。内容が変わらないのに版だけ上がる経路を作らない。
4. **版と内容の不一致**: 宣言版と、その版で解釈できない要素の組み合わせを検出して失敗にする。
5. **未知版は処理単位の失敗**: 必要な道具の版をメッセージに含める。

## 4. `.tei`属性の三状態（実装済み）

宣言は`attribute NAME type TYPE`で、任意の`value "..."`節を持つ。

```text
attribute quantity type std.decimal
attribute note type std.text value ""
attribute label type std.text value "受注"
```

- 節なし → `Absent`（`semantic.Attribute.Value == nil`）
- `value ""` → `Empty`（空文字へのポインタ）
- `value "..."` → `Value`

`value`は文字列リテラルだけを受け付ける。anchored parserも`description`/`display`/`required`/`immutable`/`role`/`value`を読み飛ばし、anchor位置を変えない。既存ファイルは節を持たないため影響しない。

版上げ時は、旧版の暗黙の既定値を`Empty`または`Value`として明示し、`Absent`のまま残さない（変換側の責務）。

## 5. 実装済みの不変条件

`internal/formatversion`の試験で固定している。

1. **冪等**: 同じ版への変換はバイト列を書き換えない。
2. **属性消失0件**: 出力に現れない属性は、要素識別子と属性名を持つ診断が必要。無ければ`ATTRIBUTE_LOST`で失敗する。
3. **版だけ上げない**: 内容が変わらず警告も無い変換は`VERSION_ONLY_BUMP`で失敗する。
4. **版の不一致**: 宣言版と変換元版が違えば`VERSION_MISMATCH`で失敗する。
5. **未知版**: 処理単位の失敗とし、対応版をメッセージに含める。
6. **要素識別子の必須**: プラグインが出す診断に要素識別子が無ければ、変換全体をエラーにする。

## 6. 実装済みの`.tei` Codec

`internal/formatversion/teicodec` が実際の`.tei`意味定義をCentralへdecodeし、Centralから`.tei`へencodeする。

- 対象は`namespace`、`data`、`attribute`宣言の範囲。`type`・`capability`・`validation`・`period`・`relation`・candidate-key・dataのdescription等は**要素識別子付きで拒否**し、黙って落とさない。
- `attribute`は`type`/`required`/`immutable`/`role`/`description`/`display`/`value`をCentralの属性として保持する。
- v1 codecはヘッダ任意（未宣言はWorkspaceから継承）。v2 codecはヘッダ必須で、継承された版を`formatVersion 2`として**明示化**し、`IMPLICIT_DEFAULT_MATERIALIZED`警告を出す。
- v1→v2は、継承版の明示化があるため`SuccessWithWarnings`。ヘッダを既に持つv1→v2は内容が変わらないため`VERSION_ONLY_BUMP`で失敗する。
- 出力は要素・属性をソートした正規形で、v2→v2はバイト同一（冪等）。

### v2の製品採用（Owner判断 2026-11-02）

v2を製品の受理版として採用する。`workspace.Load`は次を満たす。

- 作業領域manifestの`formatVersion`は**1または2**を受理し、3以上と未指定は拒否する。
- 作業領域が2のとき、配下の`.tei`は**明示の`formatVersion 2`ヘッダを必須**とする（継承の暗黙既定を認めない）。
- `.tei`が宣言した版は作業領域の版と一致しなければならない（1の作業領域に2の`.tei`は不一致）。
- 作業領域が1のときは従来どおり、ヘッダ未宣言の`.tei`は1を継承する。

## 7. 残る作業

- 対象外の宣言（type/validation/relation等）をCentralで表現する拡張
- 隣接版の連鎖を併用するかの再判断

（プラグインcodecの登録口は`Formats`として実装済み。`.tei`codecは`tei.semantic`として登録できる。v2の製品採用も完了。）

## 8. 主張しないこと

- `.tei` codecが扱うのは意味定義の一部（namespace/data/attribute）で、型・検証・関係は要素識別子付きで拒否する。
- 新→旧の変換、業務データの移行、外部形式の取込写像は扱わない。
- v2のmanifest schemaは変えていない。変わったのは受理する版の集合と、v2でのヘッダ必須規則である。
