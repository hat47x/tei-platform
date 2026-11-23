# reference-go-build-from-plan

`BuildPlan` を読み、**計画が名指す source を自分で解釈し直して突き合わせ**、一致したときだけ Go の command をビルドする参照実装です（[#2831](https://github.com/hat47x/tei-platform/issues/2831) SB-C0 §4.39 段1）。

## 何をするか

| 段 | 内容 |
| --- | --- |
| 1 | `-plan` を**厳格に**読む（未知キー・case違い・後続値は拒否）。`Validate()`と`BuildPlanFingerprint`を通す |
| 2 | `-intent` があれば読み、その**byte digest**を申告する |
| 3 | 計画の`Sources`を`DirSourceResolver`で**同じrootから解釈し直し**、計画のdigestと比べる。**違えばビルドしない** |
| 4 | `gobuilder.BuildPackage`で`go build`する（隔離したstate、`-trimpath`・`-buildvcs=false`・`-mod=readonly`） |
| 5 | 成果物のdigestを計算し、**自分でlockを作って検証**する |
| 6 | 上の全部を1つのJSONで出す |

## 使い方

```bash
go run ./tools/reference-go-build-from-plan \
  -plan plan.json \
  -intent intent.json \
  -repo /path/to/checkout \
  -artifact-out /tmp/artifact \
  -state /tmp/state
```

| flag | 必須 | 内容 |
| --- | --- | --- |
| `-plan` | ○ | ビルドする計画 |
| `-artifact-out` | ○ | 成果物の出力先 |
| `-state` | ○ | 分離したBuild state |
| `-repo` | | sourceを解決するroot（既定は現在地） |
| `-intent` | | 計画の元になった宣言（あればbyte digestを申告） |
| `-go` | | exact Go実行ファイル（既定はPATHの`go`） |

## 出力（例）

```json
{"schemaVersion":"tei.reference.go-build-from-plan/v0",
 "planFingerprint":"...","intentBytesDigest":"sha256:...",
 "sources":[{"kind":"go-package","locator":"tools/...","plannedDigest":"sha256:...","actualDigest":"sha256:...","matched":true}],
 "artifactDigest":"sha256:...","stateDir":"...",
 "toolchain":{"goExecutable":"/.../bin/go","goExecutableDigest":"sha256:...","goRoot":"/...","closureDigest":"sha256:...","plannedClosureDigest":"sha256:...","matchesPlan":true},
 "buildArtifactLock":{"planFingerprint":"...","artifactDigest":"sha256:...","lockFingerprint":"sha256:...","validated":true},
 "buildSucceeded":true}
```

## 実装済みの検査と、証明できない範囲

Go commandのBuildPlanでは、`go-package`と`go-module`を実際に読むSourceとして扱います。計画に列挙したSourceのDigestを実ファイルと照合し、`checkSourcesAreRead`は**ビルドが読まない種類のSourceを計画に混ぜることを拒否**します。依存パッケージは`go list`による`DependencyCoverage`の検査対象です。計画で`go.mod`等のモジュール文脈を明示した場合、隔離したSourceツリーによるBuildが既定で有効になります。`-isolate`を明示すれば、その経路を指定できます。

このため、従来の「隔離ビルドは案のまま」「依存パッケージの照合は未実装」という説明は現在の実装に当てはまりません。ただし隔離を**無効にした**Buildでは、照合とコンパイルの間にSourceが変わり得ます。外部モジュール供給の完全性、Toolchainの同一性、実受領先、権限は別の検収を要します。

### Rule Markdownから生成したGoをビルドする場合

[製品Ruleを使った既存Go回帰](validationgen_build_binding_test.go)では、選択済みMarkdown Rule→型付き意味→`goemit.Emit`で生成したGo package（照合用のmain関数は`goemittest`がTraceの対応づけから書き足す）→`DirSourceResolver`で固定したBuildPlan→隔離Build→実バイナリの独立Expectedまでのコードがmainに収録されています。このファイルがmainにあることと、提案元PRのマージ状態、対象HEADの`go test`成功は区別します。

このBuildPlanが検査するのは**生成済みGoとGo依存Source**です。元の`*.tei.md`や型宣言`*.tei`はGoコマンドの直接入力ではないため、BuildPlanの`Sources`へ見かけ上だけ追加してはいけません。そうすると`checkSourcesAreRead`が拒否するか、入力の由来とコンパイル対象を混同します。

**残る接続点は製品GeneratorとBuilderの間です。** 現在選択されたRule MarkdownのSource Revision・型・Language・Generator版から、どのbyteのGoが生成されたかを記録し、そのGo byteがBuilderの実入力と一致することを**同じAttempt**で検証する必要があります。GoのDigestだけ合っていても、Rule原文を変更後に古い生成Goを使っていないことは証明できません。説明文だけの変更では意味と生成Goが同じ可能性があり、Source-backed Evidenceの再確認とArtifact再利用の可否を混同してはいけません。

製品側の引渡し条件は[Rule縦断計画 §14.5〜14.6](../../plan/design/PLUGIN_RULE_PRODUCT_VERTICAL_SLICE_2026-11.md)に置きます。参照Builderの入口・Resultに新たな業務MeaningやAuthorityを追加しない方針です。

- **検収対象外**：Ownerの業務意味受理、現在のRuntime Authority・Grant、実Providerの呼出し、供給元の信頼性、実配備、独立Consumer、StudioのStage1→Stage2 Windows実再入。

## 試験

```bash
go test ./tools/reference-go-build-from-plan -count=1
```
