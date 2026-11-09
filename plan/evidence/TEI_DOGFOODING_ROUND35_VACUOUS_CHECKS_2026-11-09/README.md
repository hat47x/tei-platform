# 2つの試験が対象0件のまま常に成功していた（2026-11-09、第35ラウンド）

- 対象: [#3931](https://github.com/hat47x/tei-platform/issues/3931)。
- 前段: [第33ラウンド](../TEI_DOGFOODING_ROUND33_PLAN_HEADING_DUPLICATE_2026-11-09/README.md)（walk が0ファイル）、[第34ラウンド](../TEI_DOGFOODING_ROUND34_ITOA_DIGITS_2026-11-09/README.md)（正規表現が0一致）。**同じ型の3例目。**
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **検査の空回りを検出し、空入力を拒否する形へ修正。**

## 1. 目的

第33・34ラウンドで「検査の入力が空になると、検査は成功を返す」型の欠陥を2回踏んだ。同じ型が `tools/repository-plan-check` 全体に残っていないかを確かめる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `bacd0686f`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 観測

### 3.1 `plan/issue` には番号付き文書が1件も無い

```text
$ ls plan/issue
README.md
```

`plan/issue/README.md`:

```text
今後の作業は GitHub Issues で管理します。過去のローカルIssue文書は、
一般公開の対象とせずPrivateリポジトリへ退避しました。
```

したがって `numberedFiles(t, plan/issue, 6)`（`^([0-9]{6})-.+\.md$`）は **0件**。

```text
issue: 0 numbered files
adr:   47 numbered files
```

### 3.2 空集合に対して2つの試験が「成功」する

実測:

```text
=== RUN TestVacuumAssertUnique
    assertUnique on empty returned normally (no assertion)
--- PASS

=== RUN TestVacuumIssueStates
    six-state test would iterate over 0 files
--- PASS
```

| 試験 | 0件のときの挙動 |
| --- | --- |
| `TestIssueNumbersAreUnique` | `assertUnique` が空集合を走査して成功 |
| `TestIssueStatesUseTheSixAllowedValues` | ループが0回で成功 |

**`TestIssueStatesUseTheSixAllowedValues` は「AGENTS.mdの6値（Proposed / Ready / In Progress / Blocked / Done / Superseded）の外です」と検査しているのに、Issue が0件なので何も検査していない。**

### 3.3 対照: ADR 側は動いている

`plan/adr` には47件の番号付きファイルがあり、`TestADRNumbersAreUnique` は実際に重複を検査している。同じ `assertUnique` を使っているのに片方だけ無効。

### 3.4 なぜ気づかれないか

`assertUnique` は「重複が無いこと」を検査する。空集合は重複が無いので**論理的に正しく成功**する。**入力が空になったことは、成功と区別がつかない。**

## 4. 修正

| # | 変更 | 理由 |
| --- | --- | --- |
| 1 | `assertUnique` が `len(byNumber) == 0` で `t.Fatalf` | 空集合の一意性は無意味。0件を成功にしない |
| 2 | `TestIssueNumbersAreUnique` を削除 | `plan/issue` に文書が無いので検査できない |
| 3 | `TestIssueNumbersAreLocalDocumentsAbsent` を追加 | 「番号付き文書が0件である」ことを明示。将来 1件でも現れたら落ちて、一意性検査の復活を促す |
| 4 | `TestIssueStatesUseTheSixAllowedValues` は0件なら `t.Skip` | 成功ではなく **SKIP** として可視化。`-v` で理由が出る |

### 4.1 検証

```text
=== RUN   TestIssueNumbersAreLocalDocumentsAbsent
--- PASS
=== RUN   TestADRNumbersAreUnique
--- PASS
=== RUN   TestIssueStatesUseTheSixAllowedValues
    plan/issue に番号付き文書が無い（GitHub Issues へ移行済み）。6値規則はローカル文書に適用されない (#3931)
--- SKIP
```

空集合を渡すと落ちることも確認した。

```text
Issue: 対象が0件です。一意性を検査できません
```

## 5. 残る穴（起票して明示）

**GitHub Issues の状態が6値に従っているかは、どこでも検査されていません。**

`AGENTS.md` の6値規則は、`plan/issue` のローカル文書に対してのみ実装されていました。文書が GitHub へ移った時点で、規則を検査する対象が消えています。`TestIssueStatesUseTheSixAllowedValues` の SKIP はそれを可視化しただけで、解決はしていません。

解決するには `gh issue list` を呼ぶ検査が要りますが、CI にネットワーク依存とトークンを要求します。**運用の判断なので #3931 に残す。**

## 6. 読み取れないこと（NOT RUN）

- GitHub Issues の状態が6値に従っているか（未検査）。
- `plan/issue` 以外に「対象が空になりうる」検査が残っていないか（今回は `numberedFiles` の2つの利用者だけを確認）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 7. 位置付け

**同じ型の欠陥に3ラウンド連続で遭遇した。**

| ラウンド | 空回りの形 |
| --- | --- |
| 33 | `filepath.Walk("..")` が0ファイル |
| 34 | `itoa(10)` で正規表現が0一致 |
| 35 | `plan/issue` に文書が無く、2試験が0件を走査 |

共通するのは「**入力が空のとき、検査は成功を返す**」。第22〜24ラウンドで扱った「手書きの一覧が漏れる」型は漏れが可視だが、こちらは検査の不在に気づけない。より危険な型である。

今回は `assertUnique` に空入力の拒否を入れ、SKIP を可視化した。ただし全検査に同じ守りを入れたわけではなく、`numberedFiles` の利用者2件を見たに留まる。
