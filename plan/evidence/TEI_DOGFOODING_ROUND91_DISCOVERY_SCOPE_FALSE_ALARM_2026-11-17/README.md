# 「38コマンドが発見面から欠落」は誤報だった — 範囲は明示されていた（2026-11-17、第91ラウンド）

- 起票: **なし**（欠陥ではなかった）
- 前段: [第90ラウンド](../TEI_DOGFOODING_ROUND90_LEGACY_SURFACE_DEAD_END_2026-11-17/README.md)
- 状態: **欠陥を1件も作らなかった回。** 代わりに、自分の誤読を記録する。

## 1. 目的

前ラウンドで残した設計判断（`.tei` は `no-overlap` を表現できない）を踏まえ、**その回避経路が本当に成立しているか**を確かめる。

## 2. 確かめたこと（成立していた）

| 段階 | 結果 |
| --- | --- |
| `.tei.md` で `compared`・`group by`・`or open` を宣言 | **`ok`** |
| 読み返し | `kind=no-overlap / compared / groupBy / openEnded=true / relation` が**すべて入る** |
| 評価器 | `validationeval` に **35 subtests PASS**（実 overlap、部分読み、open period、group 付き） |
| 製品からの呼出し | `tools/reference-first-web-target`・`cmd/tei/scenario.go` ほか**実在** |

**`no-overlap` は宣言も評価もできる。** 前ラウンドの「`.tei` では無理」は正しく、回避経路も機能している。

## 3. 誤報: 「38コマンドが発見面から欠落」

`tei commands` の出力と `cli_commands.go` の正準一覧を機械的に突き合わせた:

```text
canonical: 56
discovered: 25
MISSING from discovery: ['scenario', 'artifacts', 'grants', 'periods', ...]  (38件)
```

バイナリで確認すると、これらの**すべてが実際に受理される**:

```text
$ tei scenario   -> scenario requires verify, generate, translate, run, or check
$ tei artifacts  -> exactly one of -workspace or -world is required
$ tei grants     -> exactly one of -workspace or -world is required
```

**私はこれを「発見可能性の欠陥」として起票しようとした。**

## 4. しかし、範囲は明示されていた

```go
Scope: "selected-read-only-commands",
```

設計文書も明記している:

> `schemaVersion=tei.cli.discovery/v0`、`scope=selected-read-only-commands` は、**一覧が全コマンドやPlugin実装の完全な登録簿ではない**ことを示す。

さらに `plan/design/TEI_MEANING_TO_CAPABILITY_SKILL_WITNESS_2026-11.md` は、**まさにこの状況**（`operations`/`artifacts` が dispatch に存在するが `commands` に無い）を扱い、**「不掲載を未実装に昇格させない」**と書いている。

**「欠落」ではなく「宣言された範囲」。** 起票は誤報になるため、**しなかった。**

## 5. 発見面は既に守られている

| テスト | 内容 |
| --- | --- |
| `catalogue_accepts_test.go` | カタログが名指す flag を、そのコマンドが実際に受理するか（#3956 の再発防止） |
| `cli_discovery_process_test.go` | カタログの全コマンドを実プロセスで起動 |
| `cli_discovery_write_flags_test.go` | 書き込み flag の整合 |

**この領域は既に厚く守られていた。**

## 6. 読み取れないこと（NOT RUN）

- 範囲外38コマンドの**個別の契約**（本ラウンドは発見面の話だけを見た）。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 7. 位置付け

**「計測は正しかったが、解釈が間違っていた」回**である。56 と 25 という数字は事実だが、**その差は欠陥ではなく、`Scope` が宣言した範囲**だった。**起票しなかったことが成果**であり、次に同じ数字を見た人が同じ誤りをしないよう記録する。
