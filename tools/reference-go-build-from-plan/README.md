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
 "buildArtifactLock":{"planFingerprint":"...","artifactDigest":"sha256:...","lockFingerprint":"sha256:...","validated":true},
 "buildSucceeded":true}
```

## 何を主張しないか

- **sourceの照合は、照合した時点の話**です。**照合とcompileの間にファイルが変わり得ます**（TOCTOU）。閉じるには、閉包を隔離した木へ写してビルドする形（§4.39の案B）が要ります
- **計画に載っていないsourceを読んでいないこと**は示していません（`go list -deps`との突き合わせは案のまま）
- 受理・Runtime Authority・供給のTrust・成果物の配備。**いずれも行いません**

## 試験

```bash
go test ./tools/reference-go-build-from-plan -count=1
```
