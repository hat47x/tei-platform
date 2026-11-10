# Plugin基盤の4つの課題は、いまどこまで実装されているか（2026-11-11）

- 対象: [`internal/plugin/`](../../../internal/plugin/)、[`internal/formatversion/`](../../../internal/formatversion/)
- 位置づけ: **読み取り専用の調査**。何も変更しません。
- 計測した版: `raw/head.txt`（`origin/main` `72d934e1`）。

ゴールは「Everything is Plugin を実現する最小コア、拡張ポイント、依存関係解決、ロード時診断」の4つを挙げています。**それぞれが実装としてどこまで在るか**を確かめました。

## 1. 依存関係解決 —— 在ります

| 関数 | 役割 |
| --- | --- |
| `Catalog.ResolveEnvironment` | 環境の選択から再帰的に解決する |
| `Catalog.resolvePlugin` | 版の適合・重複・要件の検証 |
| `Catalog.resolveBundle` | Bundle の展開 |

**巡回検出も推移的な不整合検査もあります。**

```text
plugin dependency cycle at %s
plugin %s already resolved to %s, incompatible with required major %d
```

## 2. ロード時診断 —— 値ではなく文字列です

**`internal/plugin` に `Diagnostic` 型はありません。** 解決は `fmt.Errorf` の文字列を返します。

```text
plain fmt.Errorf returns: 2
a Diagnostic type in internal/plugin: (none)
```

**近い型は別パッケージに在ります**（`internal/formatversion`）。

```text
type Diagnostic struct {
    Severity  Severity
    Code      string
    ElementID string
    Attribute string
    Message   string
}
```

**この形はロード時診断にそのままは使えません。** `ElementID`・`Attribute` は意味要素向けの欄で、Plugin の解決には**対象の Plugin ID・要件・版**が要ります。

### 呼出し側は文字列に依存しています

```text
test assertions matching substrings: 26
internal/plugin/bundle_requirements_test.go:62:  err.Error() != firstError
```

**26箇所が `err.Error()` の部分一致で判定しています。** メッセージを変えると試験が壊れます。**診断を構造化するなら、この26箇所も移行対象**です。

## 3. 拡張ポイント —— 能力種別は閉じた7値です

```text
type CapabilityKind string   (internal/plugin/model.go:16)
```

`CapabilityKind` は**閉じた文字列型**で、7つの定数を持ちます（`contract-query-executor` など）。**新しい能力種別を Plugin 側から足せません。**

これが [#951](https://github.com/hat47x/tei-platform/issues/951)（「Plugin Capability contract を Core 固定 enum から拡張可能な Interface へ分離する」）の対象です。

**能力種別以外の拡張面**は既にあります。

```text
internal/plugin/implementation/  (builtin)
internal/plugin/inputprofile/    (admission, current_host, definition)
```

## 4. 最小コア —— 判定していません

**「何が最小か」はこの調査では判定できません。** コアが `internal/plugin` の外へどれだけ漏れているかを測っていないためです。次の段階の対象として残します。

## 5. この記録が示すこと

- **4つのうち「依存関係解決」は既に在ります。** 新規に作る必要はありません。
- **「ロード時診断」は文字列で、構造化されていません。** 既存の `Diagnostic` 型は意味要素向けで、**Plugin 向けの欄（対象 ID・要件・版）を持ちません**。26箇所が文字列に依存しているので、構造化はその移行と一体です。
- **「拡張ポイント」は能力種別が閉じています。** 能力種別以外の拡張面は既にあります。

## 6. この記録が示さないこと

- **どれを先にやるべきか。** 優先順位は決めていません。
- **最小コアの範囲。**
- **`Diagnostic` 型を Plugin 向けに拡張すべきか、別の型を作るべきか。**
- 実 Provider の実行、配備。

## 7. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（4課題の実測） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（0バイト＝クリーン） |
| `raw/anchors.txt` | 上の1〜4節の出力 |

## 8. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `e72a34c36e22892d9f241341034c7829a73c6f5a2fe86599804110da9cfcff5e` | 2862 |
| `raw/head.txt` | `56904077bca2ea88a89f91b98b688c3bc28866ce2beaa5b8d4a20874f5ff3f30` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 |
| `raw/anchors.txt` | `33b6419e4b1ebd025f0b2e1bd8ddf53528bce3d81a5f8ab49151fec829eb0a48` | 1585 |
