# 設計演習の第2回: 縦の一連を CLI だけで設計し、application-release の欠落を検出（2026-11-10、第60ラウンド）

- 対象: [#4128](https://github.com/hat47x/tei-platform/issues/4128)（C 案の続き）。
- 前段: [第57ラウンド](../TEI_DOGFOODING_ROUND57_DESIGN_REEXERCISE_2026-11-10/README.md)（設計演習1回目）、[第59ラウンド](../TEI_DOGFOODING_ROUND59_OVERVIEW_SCOPE_2026-11-10/README.md)（E 案）。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **縦の一連を設計し、全7成果物を CLI で読み返せた。`application-release` の欠落を検出して修正。**

## 1. 目的

目的文は「再びドッグフーディングに戻る」ことを求めている。第58・59ラウンドは道具側の作業だったので、本ラウンドは**設計演習に戻り、改善が効いているかを測る**。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `278f499d6`（`origin/main`） |
| CLI実行物（設計時） | `sha256:efa4bc0176e75cfdb2fa1b43ab800c20eecaaab4afaf0f1ad794033607581485` |
| CLI実行物（修正後） | `sha256:a509093385377b2e721830d40a23389ebf6c7b4809f3fa98aaa0354e66c37856` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 設計課題（着手前に宣言）

> `returns_intake` という新しいアプリケーションを、既存の意味データの上に browse パターンで作り、索引を追加する migration、その database release、それを配備する solution release を加える。最後に CLI で読み返す。

## 4. 経過（往復と、何が教えたか）

| Attempt | 応答 | 学んだこと | 情報源 |
| --- | --- | --- | --- |
| 1 | `expected database, uses, or binds` | `semantic` ではなく `uses` | CLI |
| 2 | `expected semantic or validation after uses` | `uses semantic <ns>` | CLI |
| 3 | `expected database, uses, or binds` | パターンは application 本体に書かない | CLI（限界） |
| — | — | **パターンの置き場所は CLI では分からず、モデルの `.tei` を読んだ** | モデル |
| 4 | `browse pattern ... requires query` | browse には query が要る | CLI |
| 5 | `ok`（警告1件） | — | CLI |
| 6 | `SOLUTION_RELEASE_NOT_EXECUTABLE ... application release not found` | application release が要る | CLI |
| 7 | `expected source-revision or database` | `datasource` ではなく `database` | CLI |
| 8 | `ok`（警告1件） | — | CLI |

**アプリ層3往復、配備層2往復。** 8回中7回は CLI の診断だけで進めた。

### 4.1 CLI で詰まった1点

Attempt 3 の `expected database, uses, or binds` は「パターンを application 本体に書くな」とは言うが、**どこに書くかは言わない**。`world` も `inspect` も「既に在るもの」しか返さないので、無い構文の置き場所は分からない。

これは**構文の全体像を引くコマンドが無い**ことの現れである。語彙（値の集合）は診断が教えるが、**構文の配置（どのブロックに何が入るか）は教えない**。

## 5. 読み返し（第58・59ラウンドの改善の検証）

```text
model overview: sectionTotal 8 -> 10
  my application returns_intake in sections? True
  my query returns_intake.returns_list in sections? True

inspect（設計した7成果物）:
  application          ok
  application-pattern  ok
  query                ok
  migration            ok  sourceStatus=not-indexed
  application-release  ERR UNSUPPORTED_SUBJECT_KIND   <-- 欠落
  database-release     ok  versions 4.0.0,4.1.0,4.2.0,5.0.0
  solution-release     ok  versions 5.0.0
```

**第58ラウンドの C 案で追加した6種は機能した。** 第57ラウンドでは `world` の JSON を手で解析する必要があったが、今回は `inspect` で読めた。

**ただし `application-release` だけ未対応だった。** 今回まさに自分で作った種であり、第58ラウンドで「データ層と solution release」に絞った際の取りこぼしである。

## 6. 修正: `application-release` を inspect に追加

`database-release`・`solution-release` と同じ形（subject は owner、射影は全バージョン）。

```go
case "application-release":
	var versions []apprelease.Definition
	for _, item := range world.ApplicationReleases {
		if string(item.Application) == subject.ID {
			versions = append(versions, item)
		}
	}
	if len(versions) > 0 {
		matches = append(matches, applicationReleaseSet{ID: subject.ID, Releases: versions})
	}
```

受理種は 15 → **16**。MCP アダプタの enum ピンも追随。

### 6.1 修正後

```text
application-release returns_intake -> ok, versions=['1.0.0']
```

**設計した7成果物すべてが CLI で読み返せるようになった。**

### 6.2 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | `application-release` の dispatch を外す | FAIL `application-release:order_entry did not resolve` **と** `TestEveryAdvertisedKindWithADeclarationResolves` の両方が落ちる |

第58ラウンドで足した「広告と実宣言の突き合わせ」試験が、**今回の追加でもそのまま効いた**。試験を足さずに変異が捕まったのは、その試験が種に依存しない作りだったため。

## 7. 残った摩擦（未起票）

`UNBOUND_DATASOURCE` 警告:

```text
warning UNBOUND_DATASOURCE query returns_intake.returns_list datasource sales
        is not bound in any environment
```

**何が足りないかは言うが、何を書けばよいかは言わない。** 環境のバインディングは `environments/**` に分散し、`contract-query-executor-binding` のような別の宣言と絡む。今回は `ok`（警告のみ）で先に進んだ。

**起票はしない。** 一度しか踏んでおらず、正しい書き方を私が特定できていない。次に踏んだら起票する。

## 8. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/operations` | ok |
| `./tools/reference-mcp-readonly-adapter` | ok |
| `./cmd/tei` | ok |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

## 9. 読み取れないこと（NOT RUN）

- `UNBOUND_DATASOURCE` の正しい解消手順（特定できていない）。
- 構文の配置を引くコマンド（存在しない）。
- `#4128` の A・B・D。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 10. 位置付け

**目的文の「再びドッグフーディングに戻る」を実行した回**である。道具を2ラウンド整えた後に設計へ戻り、その道具が実際に効いたこと（`inspect` で6/7が読めた、`overview` に自分の設計が現れた）と、効かなかったこと（`application-release` の欠落、構文配置が引けない）の両方を測った。

**8往復のうち7往復は CLI の診断だけで進んだ。** 残る1往復は「置き場所」の質問で、値の集合ではないため既存の語彙診断では埋まらない種類の穴である。
