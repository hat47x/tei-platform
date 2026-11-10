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

`tei self-build`は`-artifact-out`と`-state`をchildへ渡すため、そのままchildとして起動できます。`-plan`・`-plugin`・`-intent`は`-child-arg`で渡します。

`-intent`は省略できます。渡すと、このchildは**宣言も共有readerで読み**、自分が読んだ値を申告します。渡さない場合は宣言の欄を出しません（欄が無いことを合格と読ませないため）。

## 出力

stdoutへ1件のJSONを出します。`tei self-build`が読む欄は`artifactDigest`・`planFingerprint`・`stateDir`です。**`stateDir`は省略できません**（下の表を参照）。

| 欄 | 意味 |
| --- | --- |
| `planBytesDigest` | **このchildが保持した**planファイルのbyte digest |
| `planFingerprint` | `internal/pluginのBuildPlanFingerprint`が**正規化したplanから導いた**照合値 |
| `artifactDigest` | 書いた配布物のbyte digest |
| `definitionBytesDigest` | `-plugin`で読んだ定義ファイルのbyte digest。省略すると`definitionBinding`が`unperformed`になるが、**試行は失敗せず`l2b.claimed`も変わらない**（実測）。値が計画の名指す定義と違えば`mismatched`になり、**`ok:false`・終了コード1**になる（実測） |
| `stateDir` | `-state`で渡されたディレクトリをそのまま返す。**省略すると再現の観測が`unperformed`になり、`l2b.claimed`は`false`・終了コード1になる**（実測） |
| `claimBoundary` | 何を主張しないかの文 |
| `intentBytesDigest` / `intentFingerprint` | **このchildが読んだ宣言**のbyte digestと照合値（`-intent`指定時のみ） |
| `intentBoundToPlan` | 宣言のsubjectがplanのPluginと一致することをchild自身が確かめたか |
| `intentBindingNote` | 何を主張しないかの文（宣言側） |

**`planBytesDigest`と`planFingerprint`は別物です。** 前者は「読んだバイト」、後者は「正規化した意味」であり、同じ値になるとは限りません。両方を出すのは、呼び手が「同じ計画を渡したか」と「同じ計画を名乗ったか」を別々に確かめられるようにするためです。

### `definitionBinding`と`l2b.claimed`は別の問いに答える

実測した三つの状態です。

| childが返す`definitionBytesDigest` | `definitionBinding.state` | `ok` | 終了コード | `l2b.claimed` |
| --- | --- | --- | --- | --- |
| 正しい値 | `bound` | `true` | 0 | `true` |
| **省略** | `unperformed` | `true` | 0 | `true` |
| **誤った値** | `mismatched` | **`false`** | **1** | **`true`** |

**`l2b.claimed`は試行全体の成否ではありません。** 誤った定義digestでは**試行が失敗しているのに`l2b.claimed`は`true`のまま**です。`claimed`は五条件だけを述べ、定義の束縛は別の欄だからです。

**`l2b.claimed`だけを読んで成功と判断しないでください。** 終了コードと`ok`、そして`definitionBinding.state`を併せて読みます。

## 何を保証し、何を保証しないか

保証する。

- planが`Validate()`を通らなければ**artifactを書かない**（失敗が「buildできた」と読まれない）
- Plugin定義の同一性がplanと一致しなければ停止する
- `-intent`を渡した場合、宣言のsubjectがplanのPluginと一致しなければ**reportを出さずに停止する**（一致し得ない二つの値を「読んだ値」として返さないため）
- `artifactDigest`はディスクへ書いたバイトと一致する（自己申告をディスクと独立に検算できる）

保証しない。

- **意味の受理・L3・Runtime Authority**。`claimBoundary`が明記する
- **その計画が特定の宣言から導かれたこと**。`-intent`を渡すと、このchildは**宣言を読んでidentityを照合する**が、planを宣言から導いてはいない。`-plan`は入力として受け取る
- **宣言とplanの結び付け**。childは両方を読み、subject identityの一致を確かめるが、それは「同じsubjectを指す」ことであって「その宣言がそのplanを生んだ」ことではない
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
| **宣言の申告** | `-intent`指定時に`intentBytesDigest`・`intentFingerprint`が入り、digestがディスクのバイトと一致する |
| 別subjectの宣言 | artifactを書かず、reportを出さず、exit 1 |
| `-intent`無し | 宣言の欄を出さない（沈黙を合格と読ませない） |
| **座標の不一致** | `outputKind`・`target`・`profile`のいずれかが違えばartifactを書かず、reportを出さず、exit 1。違った座標を名指しする |
| `profile`の省略 | planの値と一致しないものとして拒否する |
| **builder要求の不一致** | `contract`・`compatibleMajor`・`provider`・`pin`のいずれかが解決結果と違えばreportを出さずexit 1。欄を名指しする |
| builder要求の一致 | `provider`と`pin`が一致すれば`intentBoundToPlan=true`で通る（常に失敗する比較ではないことの確認） |
| **その他の欄** | `recipe`・`sources`・`dependencies`の不一致を名指しで拒否し、**導出値**（sourceの`locator`/`digest`、dependencyの`version`/`artifactDigest`）の違いは不一致と読まない |
| **`parameters`** | 名前で突き合わせ、逆順でも一致し、値が違えば名前と両方の値を名指しする |
| 正規化の性質 | 両側とも名前順ソートのみで**値を変えない**ことを直接固定する（照合がこの性質に依存しているため） |

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

## 宣言とplanの座標

`-intent`を渡すと、このchildは**subject identity**に加えて**build座標**もplanと比べる。

| 座標 | 何を確かめるか |
| --- | --- |
| `outputKind` | 宣言とplanが同じ出力種を指すか |
| `target` | 同じ対象環境を指すか |
| `profile` | 同じ構成を指すか |

違えば**artifactを書いた後にreportを出さずexit 1**にし、**どの座標が違ったかを名指しする**。identityが一致しても別のbuildを指す場合があるためである。

**`profile`の省略は一致と読まない。** 宣言が`profile`を書かないことは「planの`profile`と同じ」ではなく「profileを宣言していない」である。

座標の一致が言えるのは「**宣言とplanが同じbuildを指す**」ことまでで、「その宣言がそのplanを生んだ」ことではない。

## 宣言のその他の欄

identity・座標・`builder`要求に加えて、このchildは**宣言とplanが同じ形で持つ欄**も比べる。違えばreportを出さずexit 1にし、**どの欄が違ったかを名指しする**。

| 欄 | 比べ方 |
| --- | --- |
| `recipe.kind`・`recipe.entrypoint` | 完全一致 |
| `sources` | **kindとid**を集合として（件数も） |
| `dependencies` | **kindとid**を集合として（件数も） |
| `parameters` | **名前と値**を対応として（件数も） |

### 比べない欄と、その理由

| 欄 | 理由 |
| --- | --- |
| sourceの`locator`・`digest` | **plan側は導出値**。宣言は参照を書き、planは解決済みの値を持つ。比べると**正しいbuildをすべて不一致と呼ぶ** |
| dependencyの`version`・`artifactDigest` | 同じく導出値 |
| `builder.provider`・`builder.pin` | **方針**であり、解決結果ではない |

**「比べられる欄」と「比べるべき欄」は違う。** 導出値を比べると常に不一致になるので、比べないことを明示している。

### `parameters`を比べる根拠

`parameters`は一度「正規化の意味が未決」として除外していたが、**測るとどちらの側も名前順にソートするだけで値を変えない**（`buildintent.NormalizeIntent`と`plugin.NormalizeBuildPlan`）。したがって値の違いは本物の違いであり、名前で突き合わせれば順序に依存しない。**この性質自体を試験で固定している**（`TestNormalisersOnlyReorderParameters`）。

### この照合が効く範囲

**同じ宣言からresolverが導いたplanに対しては決して失敗しない。** 要求を満たさない宣言からはresolverがplanを作らないためである。効くのは**planがその宣言から導かれたものではない**場合だけであり、そのときにどこが食い違うかを名指しで示す。**保証の強さを増やすものではなく、原因究明を助けるものである。**

## 宣言の`builder`要求とplanの解決結果

宣言は`builder`を**要求**として書き（`contract`と`compatibleMajor`、任意で`provider`・`pin`）、planは**解決結果**として`BuilderIdentity`を持つ。このchildは両者を照合する。

| 宣言 | plan | 規則 |
| --- | --- | --- |
| `contract` + `compatibleMajor` | `builder.contractVersion` | `internal/buildintent`の`BuilderContractVersion`で組み立てて完全一致 |
| `provider.plugin`（指定時） | `builder.id` | 完全一致 |
| `pin.artifactDigest`（指定時） | `builder.artifactDigest` | 完全一致 |

**対応規則はこのchildが決めたものではない。** resolverの`ResolveBuilder`が候補を`Contract`と`ContractMajor`で絞り、`BuilderContractVersion`と同じ書式で`ContractVersion`を組み立てている。両側が同じ関数を通るので、照合は**書式ではなく要求**を試す。

`provider.implementation`は**照合しない**。`plugin.BuilderIdentity`に対応する欄が無いためである。比べないことをコードにも書いてある。

**この照合は、同じ宣言からresolverが導いたplanに対しては決して失敗しない。** 要求を満たさない宣言からはresolverがplanを作らないためである。したがってこの照合が効くのは、**planがその宣言から導かれたものではない**場合だけである（宣言が解決後に変わった、別の宣言から導かれた、resolver以外で作られた）。

## 実行した確認

`tei self-build`を解決child（`tools/reference-build-authoring-intent`）とこのchildで起動し、**実プロセスで`planCrossCheck.state=matched`を観測**しました。記録は[`TEI_GO_SELF_BUILD_PLAN_MATCHED_2026-11-09.md`](../../plan/evidence/TEI_GO_SELF_BUILD_PLAN_MATCHED_2026-11-09.md)にあります。

child自身のlockを足した後の実測では、**entryが導いたlockとchildのlockが同じfingerprint**になりました。記録は[`TEI_GO_SELF_BUILD_CHILD_LOCK_2026-11-09.md`](../../plan/evidence/TEI_GO_SELF_BUILD_CHILD_LOCK_2026-11-09.md)にあります。**`l2bClaimed`はどちらも`false`のままです。**
