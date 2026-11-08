# 計画を消費してPlugin配布物を作るchild（SB-C0 §4.10 P-1）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.10でP-1として選んだ参照childです。`tei self-build`のchild文法を満たし、**入力として受け取った`BuildPlan`の照合値を自分の申告として返します**。

`tools/reference-go-build-with-supply`はchild文法を満たしますが計画を名乗りません。逆に計画を名乗るtoolはchild文法を持ちません（[設計 §4.10.1](../../plan/design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)）。この非対称のため、計画突合は`unperformed`のままになっていました。このchildはその非対称を埋めます。

## 文法

```text
reference-plugin-package-build-child
  -plan <exact BuildPlan JSON>      必須
  -plugin <Plugin定義 .tei>          必須
  -artifact-out <出力パス>           必須
  -state <分離したstateディレクトリ> 任意
```

`tei self-build`は`-artifact-out`と`-state`をchildへ渡すため、そのままchildとして起動できます。`-plan`と`-plugin`は`-child-arg`で渡します。

## 出力

stdoutへ1件のJSONを出します。`tei self-build`が読む欄は`artifactDigest`と`planFingerprint`です。

| 欄 | 意味 |
| --- | --- |
| `planBytesDigest` | **このchildが保持した**planファイルのbyte digest |
| `planFingerprint` | `internal/pluginのBuildPlanFingerprint`が**正規化したplanから導いた**照合値 |
| `artifactDigest` | 書いた配布物のbyte digest |
| `claimBoundary` | 何を主張しないかの文 |

**`planBytesDigest`と`planFingerprint`は別物です。** 前者は「読んだバイト」、後者は「正規化した意味」であり、同じ値になるとは限りません。両方を出すのは、呼び手が「同じ計画を渡したか」と「同じ計画を名乗ったか」を別々に確かめられるようにするためです。

## 何を保証し、何を保証しないか

保証する。

- planが`Validate()`を通らなければ**artifactを書かない**（失敗が「buildできた」と読まれない）
- Plugin定義の同一性がplanと一致しなければ停止する
- `artifactDigest`はディスクへ書いたバイトと一致する（自己申告をディスクと独立に検算できる）

保証しない。

- **意味の受理・L3・Runtime Authority**。`claimBoundary`が明記する
- **その計画が特定の宣言から導かれたこと**。childが読むのは`-plan`のファイルであり、宣言との結び付けは`tei self-build`側の`intentBinding`が担う
- **artifactがその計画の出力であること**（SB-C0 §4.8の(c)）。`plugin.LockedBuild`のような束ね型での別観測が要る
- 供給のTrust・署名、`go.sum`のTrust根拠、Seed Trust

## 試験

| 試験 | 確かめていること |
| --- | --- |
| 報告 | plan byte digestとplan fingerprintが別値で、`internal/plugin`の導出値と一致する |
| **自分のlock** | `buildArtifactLock`が`validated: true`で、fingerprintが非空、plan/artifactがreportと一致する |
| artifact照合 | 申告digestがディスクのバイトと一致する |
| 不正plan | artifactを書かず、stdoutへJSONを出さず、exit 1 |
| 定義不一致 | artifactを書かず、exit 1 |
| 文法 | 必須flag欠落はexit 2と理由 |

## child自身のlock

reportは`buildArtifactLock`を持つ。**このchildが自分で組み立て、`internal/plugin`の`Validate`を通した束縛**である。

| 欄 | 意味 |
| --- | --- |
| `validated` | `Validate`を通したか |
| `lockFingerprint` | childが導いたlockの照合値 |
| `planFingerprint` / `artifactDigest` | lockに載せた値（childが実際に保持したもの） |
| `boundary` | 「lockが整っていることは、その計画がそのartifactを生んだ証明ではない」 |

lockが`Validate`を通らなければ、**artifactを書いた後にreportを出さずexit 1**にする。reportを返すと、検証できなかった束縛が「返ってきた」と読まれるためである。

`tei self-build`はこのlockを、自分が独立に導いたlockと比べる。**経路の違う二つの組み立てが同じfingerprintになる**ことが、ここで増える観測である。それでも、両方がchildの申告に依存している点は変わらない。

## 実行した確認

`tei self-build`を解決child（`tools/reference-build-authoring-intent`）とこのchildで起動し、**実プロセスで`planCrossCheck.state=matched`を観測**しました。記録は[`TEI_GO_SELF_BUILD_PLAN_MATCHED_2026-11-09.md`](../../plan/evidence/TEI_GO_SELF_BUILD_PLAN_MATCHED_2026-11-09.md)にあります。

child自身のlockを足した後の実測では、**entryが導いたlockとchildのlockが同じfingerprint**になりました。記録は[`TEI_GO_SELF_BUILD_CHILD_LOCK_2026-11-09.md`](../../plan/evidence/TEI_GO_SELF_BUILD_CHILD_LOCK_2026-11-09.md)にあります。**`l2bClaimed`はどちらも`false`のままです。**
