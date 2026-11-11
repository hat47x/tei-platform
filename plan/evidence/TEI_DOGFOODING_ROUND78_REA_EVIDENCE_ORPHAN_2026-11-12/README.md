# rea-evidence を要求するテストが main に在り、実装は未統合ブランチにのみ存在する（2026-11-12、第78ラウンド）

- 起票: [#4455](https://github.com/hat47x/tei-platform/issues/4455)
- 前段: [第77ラウンド](../TEI_DOGFOODING_ROUND77_REA_INTAKE_ROUTE_2026-11-11/README.md)
- 状態: **起票のみ。私は修正しない（理由は本文）。**

## 1. 目的

第77ラウンドで「`rea-evidence` は実装自体が無い」と記録した。**その実装が本当にどこにも無いのか、どこに在るのかを確定させ、赤の正体を明文化する。**

## 2. 実測

```text
$ go test ./cmd/tei -run 'TestDiscoveryAdvertisesEveryModeAResponseCanCarry|TestDiscoveryWriteFlagsMatchObservedFileEffects'
--- FAIL: TestDiscoveryAdvertisesEveryModeAResponseCanCarry
--- FAIL: TestDiscoveryWriteFlagsMatchObservedFileEffects
    --- FAIL: TestDiscoveryWriteFlagsMatchObservedFileEffects/-request-out
stderr: unknown command "rea-evidence"
```

## 3. 実装の所在

```text
$ git log --all --diff-filter=A -- cmd/tei/rea_evidence.go
716fb3612 feat(rea): receive one external analysis Evidence and inspect its delivery

$ git merge-base --is-ancestor 716fb3612 origin/main   -> NO
$ git branch -r --contains 716fb3612
  origin/feat/rea-external-evidence-intake-20261111-r2
$ git rev-list --count 716fb3612..origin/main           -> 157
```

**実装はブランチに在り、157コミット分 main から遅れている。**

依存パッケージ `internal/externalanalysis` も **main に無い**（`internal/jsonl` は在る）。

| 対象 | 行数 |
| --- | --- |
| `cmd/tei/rea_evidence.go` | 229 |
| `internal/externalanalysis`（receive 505・view 405・request 248・pins 44） | 1,202 |
| testdata | バイナリ4＋jsonl5 |

## 4. なぜこの状態になったか

`7b0ab6b63` が `cli_discovery_modes_test.go` と `cli_discovery_write_flags_test.go` に **`rea-evidence` を前提とする2試験を追加**したが、`rea_evidence.go` を**含んでいなかった**（`--name-only` で0件）。**テストだけが先に main に入り、実装は入らなかった。**

```text
$ git show 7b0ab6b63 --name-only | grep -c rea_evidence
0
```

## 5. なぜ私が直さないか

| 案 | 内容 | 採否 |
| --- | --- | --- |
| A | 未統合ブランチを main へ取り込む | **所有者の判断**（157コミット分の解消が前提。機能の意味を私が決めてしまう） |
| B | テストを実装が入るまで外す | **所有者の判断**（契約を一時的に外す） |
| C | スタブで `unknown command` を返す | **採らない**（動かないコマンドを動くように見せる） |

**C はドッグフーディングの目的に反する。A は機能全体の再統合で、私が独断で流し込むべきではない。**

## 6. 読み取れないこと（NOT RUN）

- `rea-evidence` の動作（実装が main に無いため）。
- 機能全体の再統合（A 案）。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 7. 位置付け

**「赤が常態化する」危険を名指しした回**である。第76ラウンドで私はこの2件を「main でも同じ」として切り捨てた。**切り捨てが続くと赤が常態化する。** その正体（未統合機能の孤児テスト）を確定させ、判断を所有者に返した。
