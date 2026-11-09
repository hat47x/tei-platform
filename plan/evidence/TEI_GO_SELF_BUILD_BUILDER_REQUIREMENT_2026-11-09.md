# builder childが宣言の`builder`要求をplanの解決結果と照合する（§4.19）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.18末尾が「規則そのものを決める必要がある」として残した`builder`の照合を実装し、実測した記録です。

**規則を決める必要はなかった。規則は既にresolverの中にあった。**

## 何を測ったか

§4.18は、宣言の`builder.contract`・`compatibleMajor`とplanの`BuilderIdentity`は「形が違う」ため対応規則が要ると書きました。実際に`ResolveBuilder`を読むと、**対応は既に一箇所で決まっていました。**

| 段 | 実装 |
| --- | --- |
| 候補の絞り込み | `p.Contract == req.Contract && p.ContractMajor == req.CompatibleMajor` |
| 解決結果の生成 | `ContractVersion: fmt.Sprintf("%s/v%d", p.Contract, p.ContractMajor)` |

つまり`契約名 + "/v" + major`という対応は**推測ではなく、resolverが既に適用している規則**です。したがって照合は「測れば決まる」種類でした。§4.18の分類が誤っていたことを、測って訂正しました。

## 何をしたか

対応規則を**一箇所に定義**しました。`internal/buildintent.BuilderContractVersion(contract, major)`がそれで、resolverの`ResolveBuilder`もchildの照合もこの関数を通ります。**二箇所で書式を書き直すと、照合が「要求」ではなく「書式」を試すことになる**ためです。

childは宣言の要求を次の三点で比べます。

| 宣言 | plan | 規則 |
| --- | --- | --- |
| `contract` + `compatibleMajor` | `builder.contractVersion` | `BuilderContractVersion`で組み立てて完全一致 |
| `provider.plugin`（指定時） | `builder.id` | 完全一致 |
| `pin.artifactDigest`（指定時） | `builder.artifactDigest` | 完全一致 |

`provider.implementation`は**照合しません**。`plugin.BuilderIdentity`に対応する欄が無いためです。**比べないことをコードのコメントに残し**、後から読む人が「確かめた」と誤解しないようにしました。

不一致は欄を名指しして拒否し、**reportを出しません。**

## 観測結果

| 項目 | 値 |
| --- | --- |
| 作業領域 | 隔離worktree（`origin/main` = `5d5d5de6b`） |
| OS / Go | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |
| `tei` 実行物 | `sha256:4a7d95ed726744ec456cec41fe85d5f575bb50cb7b289673fdeabcc6b000dbc7` |
| 解決child | `sha256:f32f050a28b878cc7ee83739dcf57fcda35802bf3b74e49a60b90311c4f87cbc` |
| builder child | `sha256:b6e1b9814f6c67faf1523a23055720e8275d722f787bd10b64de173390685cb8` |
| planのbuilder | `tei.builder.go@1.2.0`、`contractVersion` = `tei.plugin-build/v1` |

### 宣言とplanが一致する場合

| 欄 | 値 |
| --- | --- |
| `ok` / `qualified` | `true` / **`false`** |
| `intentByteCheckPerformed` / `Matched` | `true` / `true` |
| `planArtifactBinding.state` | `bound` |
| `l2bClaimed` | **`false`** |

### 一致しない場合（5ケース）

| ケース | 何を変えたか | 結果 |
| --- | --- | --- |
| 2 | 宣言の`compatibleMajor`を2に | **resolverが先に停止**：`no compatible builder provider for contract tei.plugin-build/v2` |
| 3 | 上記の宣言を、major 1で解いたplanと組にする | **child exit 1**・stdout 0バイト：`the declaration requires builder contract "tei.plugin-build/v2" but the plan resolved "tei.plugin-build/v1"` |
| 4 | 宣言が別のbuilder artifactをpin | **child exit 1**・stdout 0バイト：`the declaration pins builder artifact sha256:dddd… but the plan resolved sha256:1111…` |
| 5 | 宣言が別のproviderを指定 | **child exit 1**・stdout 0バイト：`the declaration's builder provider is tei.builder.other.seed but the plan resolved tei.builder.go` |

## この実測から分かった重要な性質

**同じ宣言からresolverが導いたplanに対して、builderの照合は決して失敗しません。** ケース2が示すとおり、要求を満たさない宣言からはresolverがplanを作らないためです。

したがってこの照合が効くのは、**planがその宣言から導かれたものではない**場合だけです。具体的には次の三つです。

- 宣言がplanの解決後に変わった
- planが別の宣言から導かれた（§4.8のSC0-13が問う状況）
- planがresolver以外（手書き・別ツール）で作られた

**これは弱い検査ではなく、狙いどおりの検査です。** childは「渡されたplanが渡された宣言と対応しているか」を確かめており、対応していなければ止まります。ケース3〜5がその動作を示しています。

## この実測が言えないこと

- **L2-b（解釈の成立）。** `l2bClaimed`は`false`のまま
  - builder照合は「宣言の**要求**がplanの**解決結果**と一致する」ことを示す。**「その計画がそのartifactを生んだ」ことではない**（(c)は変わらない）
- **`provider.implementation`** は照合していない（planに対応欄が無い）
- childの申告が真である保証はない
- 意味の受理・Runtime Authority・供給のTrust・Seed Trust・Windows・別Go版

## 残る作業

| 欄 | 状態 |
| --- | --- |
| `secretRequirements` | **未着手。** plan（`plugin.BuildPlan`）に対応欄が無く、`SecretRequirementNames`は`ResolvedBuildReceipt`にしかない。childはplanを消費するので、比べる相手が入力に無い |
| entryの既定経路（in-process vs child） | **未決。** §4.16のR-2据え置きが現時点の既定 |
| D-1（catalogの`scope`拡張） | **未決。** 契約判断 |

`secretRequirements`は`builder`と違い、**対応規則を決めれば済む話ではありません。** planという入力に相手が存在しないので、次のいずれかを選ぶ必要があります。

1. 照合しないと明示する（現状の延長。§4.17の表に理由付きで残す）
2. childがreceiptも受け取る（childの入力契約が変わり、「planを消費するchild」ではなくなる）
3. `plugin.BuildPlan`に秘密名を持たせる（**製品の型の意味変更**であり、契約判断）

**本節は1を現時点の既定とし、2と3を未決として名指しします。** 3は「planが秘密の名前を持つ」という意味の追加であり、PoCの範囲で決めてよいものではありません。

## 検証

- `gofmt -l internal/buildintent tools/reference-build-authoring-intent tools/reference-plugin-package-build-child`：差分なし
- `go vet`（3パッケージ）：PASS
- `go test ./internal/buildintent -count=1`：PASS（`BuilderContractVersion`の対応を固定する試験を追加）
- `go test ./tools/reference-build-authoring-intent -count=1`：PASS（`ResolveBuilder`が同じ関数を通る形へ変更）
- `go test ./tools/reference-plugin-package-build-child -count=1`：PASS（**36ケース**。contract名・major・provider・pinの不一致と、provider+pinが一致する場合を追加）
- 実行5回：一致（`ok=true`、`bound`、exit 0）、resolver拒否1件、child拒否3件（いずれもstdout 0バイト）
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：Windows、別Go版、二回目の独立試行、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.18
- 対応規則：[`internal/buildintent/intent.go`](../../internal/buildintent/intent.go)の`BuilderContractVersion`
- 解決規則：[`tools/reference-build-authoring-intent/fixture.go`](../../tools/reference-build-authoring-intent/fixture.go)の`ResolveBuilder`
