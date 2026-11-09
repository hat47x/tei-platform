# child自身が計画とartifactのlockを作って返す（§4.11の次段）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.11が次に詰めるとした「**builder自身が`BuildArtifactLock`を作って返す経路**」を実装し、実測した記録です。

**L2-bは依然として成立しません。** `l2bClaimed`は`false`のままです。

## 何が変わったか

これまでentryが**二つの自己申告**（builderの`planFingerprint`と`artifactDigest`）からlockを組み立てていました。builderが申告を偽れば`bound`が成立してしまう構造です。

今回、**child自身が自分のlockを作り、`internal/plugin`の検証を通してから返す**ようにしました。entryはそれを**独立に導いた自分のlockと比べます**。

| 段 | 変更前 | 変更後 |
| --- | --- | --- |
| lockを作る主体 | entryのみ | **childも作る**（childのreportへ`buildArtifactLock`を追加） |
| lockの検証 | entryが`Validate` | **childも`Validate`し、通らなければreportを書かずに停止** |
| 突合 | なし | **entryが導いたlock fingerprintとchildのlock fingerprintを比較** |

**二つの独立した組み立てが同じfingerprintになる**ことが、ここで増えた観測です。childは自分が読み込んだplanと自分が書いたバイトからlockを作り、entryはreceiptのplan identityとchildの申告からlockを作ります。経路が違うのに同じ値になるなら、両方の読みが同じ束縛を指しています。

## child側の変更

`tools/reference-plugin-package-build-child`のreportへ`buildArtifactLock`を追加しました。

| 欄 | 意味 |
| --- | --- |
| `validated` | childが`Validate`を通したか |
| `lockFingerprint` | childが導いたlockの照合値 |
| `planFingerprint` / `artifactDigest` | lockに載せた値（childが実際に保持した値） |
| `boundary` | 「lockが整っていることは、その計画がそのartifactを生んだ証明ではない」 |

lockが`Validate`を通らなければ**artifactを書いた後にreportを出さずexit 1**にします。reportを返してしまうと、検証できなかった束縛が「返ってきた」と読まれるためです。

## entry側の変更

`planArtifactBinding`の欄を追加しました。

| 欄 | 意味 |
| --- | --- |
| `childLockChecked` | childのlockを比べたか |
| `childLockValidated` | childが検証済みと申告したか |
| `childLockMatched` | entryのlock fingerprintと一致したか |
| `childLockFingerprint` | childのlock fingerprint |

childのlockが無い場合は`childLockChecked=false`のままです（**欄が無いことを合格と読ませない**）。childのlockがあるのに一致しない場合は`mismatched`として`ok=false`・`failureCause`になります。

## 観測結果

| 項目 | 値 |
| --- | --- |
| 作業領域 | 隔離worktree（`origin/main` = `99b220ae0`） |
| OS / Go | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |
| `tei` 実行物 | `sha256:39b8169799f1c75d94af63dd9c1ba8bb3a48785d50f46a05af7ac32d9508493a` |
| 解決child | `tools/reference-build-authoring-intent`、`sha256:abb0f12b…` |
| builder child | `tools/reference-plugin-package-build-child`、`sha256:26c8ed5b6378a26b1396e32fd33d42d2f4651fcc4f7d284eeab2123c25430676` |
| 解決計画fingerprint | `9aeb4616b02d6ec899a421af8445a0242426b01fa9fa3a9fc3d57a034866b951` |

| 欄 | 値 |
| --- | --- |
| `ok` / `qualified` | `true` / **`false`** |
| `planCrossCheck.state` | `matched` |
| **`planArtifactBinding.state`** | **`bound`** |
| `planArtifactBinding.lockFingerprint` | `322ea86b3ce8d1fd7aec7f78c04e4cafd277db334cef47507753f960a79dbe52` |
| `planArtifactBinding.childLockChecked` | `true` |
| `planArtifactBinding.childLockValidated` | `true` |
| **`planArtifactBinding.childLockMatched`** | **`true`** |
| `planArtifactBinding.childLockFingerprint` | `322ea86b…`（**entryのlockと同一**） |
| `planArtifactBinding.l2bClaimed` | **`false`** |
| `childRun.childLock.validated` | `true` |
| `childRun.artifactCrossCheckMatched` | `true` |

**entryが導いたlockとchildが導いたlockが同じfingerprintになりました。** 経路の違う二つの組み立てが一致しています。

## この実測が言えること

- childが自分でlockを作り、`internal/plugin`の検証を通して返した
- entryが独立に導いたlockと、childのlockが**同じfingerprint**になった
- artifact digestはディスクのバイトと一致した

## この実測が言えないこと

- **L2-b（解釈の成立）。** `l2bClaimed`は`false`のまま
  - builder childが読んだのは`-plan`のファイルであり、**entryが保持した宣言そのものではない**（`intentByteCheckPerformed=false`）
  - (c)が本来問う「**その計画がそのartifactを生んだか**」は、lockが一致しても出ない。lockが示すのは「計画identity・planFingerprint・artifactDigestの三つ組が、二つの独立した組み立てで同じになった」ことである
- **childの申告が真であること。** childのlockはchildのプロセス内での整合を示すが、childが嘘をつかない保証ではない。entryが比べられるのは**entry自身が導いた値との一致**までである
- 意味の受理・Runtime Authority・供給のTrust・Seed Trust・Windows・別Go版

## 残る限界と次の一歩

`childLockMatched=true`は「**二つの独立した組み立てが一致した**」ことを示しますが、**両方が同じ入力（childの申告）に依存している**点は変わりません。entryのlockもchildの`planFingerprint`と`artifactDigest`から作られるためです。

依存をさらに減らすには、**entryがchildのartifactを自分で読み直してlockを作る**（今も`artifactDigestOnDisk`で一部やっている）だけでなく、**計画側もentryが独立に持つ**必要があります。それは`-intent-resolver`のreceiptではなく、entry自身が宣言から計画を導く経路であり、§4.7でI-1（`internal/`への切り出し）を採らなかった制約に戻ります。

## 検証

- `gofmt -l cmd/tei tools/reference-plugin-package-build-child`：差分なし
- `go vet ./cmd/tei` / `go vet ./tools/reference-plugin-package-build-child`：PASS
- `go test ./tools/reference-plugin-package-build-child -count=1`：PASS（lockの検証済み・fingerprint非空・plan/artifact一致の主張を追加）
- `go test ./cmd/tei -count=1 -run 'PlanArtifactBinding|SelfBuild'`：PASS
- 実行1回：`ok=true`、`planArtifactBinding.state=bound`、`childLockMatched=true`、exit 0

**NOT RUN**：Windows、別Go版、二回目の独立試行、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.11
- child：[`tools/reference-plugin-package-build-child`](../../tools/reference-plugin-package-build-child/README.md)
- 同時実測：[`TEI_GO_SELF_BUILD_MATCHED_AND_BOUND_2026-11-09.md`](TEI_GO_SELF_BUILD_MATCHED_AND_BOUND_2026-11-09.md)
