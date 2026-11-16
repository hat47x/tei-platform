# `-state-boundary` を実装し、書込境界が実際に効くことを確かめた（S2）

[#4486](https://github.com/hat47x/tei-platform/issues/4486) の S2 を実装しました。**`tei self-build -state-boundary` を付けると、2回目のbuildが「root tree が読み取り専用で、渡したstateと成果物ディレクトリだけ書ける」mount namespace の中で走ります。**

## 実装

| 部分 | 内容 |
| --- | --- |
| `stateBoundaryLauncher` | `unshare -rm sh -c <script> sh` を組む。root を `remount,bind,ro` し、`ALLOW_STATE`・`ALLOW_ART` だけ `remount,bind,rw` で戻し、`ALLOW_FIRST_STATE` を空のtmpfsで覆い、`exec "$@"` |
| 起動 | 既存の `driveSelfBuildIn`（記録するbuilderは**`-child`のまま**。launcher は記録しない） |
| 入口 | `-state-boundary`（opt-in）。**`-rebuild-check` を暗黙に有効にする** |
| 報告 | `stateSeparation.basis` が **`host-boundary`**、`scope` は「したこと」だけ |

**allow path は namespace に入る前に作ります。** script は最初にrootをroにするので、その後では作れません。

## 実測1：境界が書込を実際に止める

同じ script 形を、allow path を正しく渡して駆動しました。

| 試したこと | 結果 |
| --- | --- |
| 許可外のパスへの書込 | **`Read-only file system` で失敗**（エラー出力へのリダイレクト自体が失敗した） |
| `ALLOW_STATE` への書込 | **成功** |
| `ALLOW_ART` への書込 | **成功** |
| 1回目のstate の中身 | **`FIRST_STATE_ENTRIES=0`**（元は1件。tmpfsで覆われ、**読取も見えない**） |
| namespace の後、許可外ファイル | **存在しない** |

**書込は閉じ、許可した2つは開き、1回目のstateは読めなくなっています。**

## 実測2：product の flag 経由で動く

| 実行 | 終了 | `claimed` | `repro` | `basis` |
| --- | --- | --- | --- | --- |
| 既定（境界なし） | 0 | `true` | `reproduced` | `entry-post-hoc` |
| **`-state-boundary`** | **0** | **`true`** | **`reproduced`** | **`host-boundary`** |
| **`-state-boundary` 単独** | **0** | **`true`** | **`reproduced`** | **`host-boundary`** |

**3行目が重要です。** `-rebuild-check` を書かなくても再現検査が走り、境界が使われています（flag のヘルプと実装が一致）。

## 独立レビューの指摘と対応（前段の実装から継続）

| 指摘 | 判定 | 対応 |
| --- | --- | --- |
| `-state-boundary` が `-rebuild-check` を有効にせず、単独で黙って無視される | **実在** | 修正。上の3行目で確認 |
| `scope` が機構より強く、`/proc`・`/sys`・`/dev` は書き込み可能なまま | **実在** | 文言を「rootと別マウントは `not covered`」へ変更。**試験で固定** |
| 引数名の綴り誤り | **誤り** | 実物は正しい。**直していない** |

## 実装担当の分担（実測の記録）

`stateBoundaryLauncher` の初稿は Step Code（`step-5-preview`）に `implementer` として書かせました。**その出力には `exec.LookPath(unshare)` という未定義変数のバグがありました**（Go では識別子 `unshare` は存在しない）。**そのままではコンパイルが通らないため、修正して採用しています。** 生成物をそのまま信じない手順を踏みました。

## 限界（明示）

- **読取の共有は閉じていません。** 境界が閉じるのは**書込による共有**だけです
- **root と別にマウントされたファイルシステム（`/proc`・`/sys`・`/dev`）は対象外**。`scope` に `not covered` と書きました
- ネットワーク・常駐Process・環境変数は境界外
- **opt-in** で、既定の挙動は変えていません
- `linux/amd64`（WSL2）での観測。**`darwin`・Windows は対象外**（mount namespace は Linux 限定）
- **`qualified` は `false`**、G4 は `UNQUALIFIED` のまま

## 検証

`gofmt`（変更3ファイルは整形済み）、`go vet ./cmd/tei` 通過、`TestStateBoundaryLauncherShape` **PASS（skipではない）**、`TestStateBoundaryScopeNamesWhatIsNotCovered` PASS、既存の state 分離・再現性試験 PASS。

## 追記：境界の外側を実測し、`scope` を実態に合わせ、伝播の危険を塞いだ

**`scope` の文言を「実測に基づく」形へ直しました。** 前版は「root と別マウントは対象外」と一般的に書いていましたが、**どのパスが実際に書けるかを測っていませんでした。**

### 実測：境界内で何が書けるか

`unshare -rm` の中で `mount -o remount,bind,ro /` を実行した後、各パスへ書いてみました。

| パス | 境界内 |
| --- | --- |
| `/tmp` | **拒否** |
| `/proc` | **拒否** |
| `/dev` | **拒否** |
| `/sys` | **拒否** |
| `/run` | **拒否** |
| `/home` | **拒否** |
| **`/dev/shm`** | **書けたまま** |
| **`/run/user/1000`** | **書けたまま** |

**別マウントのうち、呼び手が元々書けたものは境界内でも書けます。** これは `scope` に明記し、**試験で固定**しました（`remain writable`・`/dev/shm`・`reads are not confined` を含むことを検査）。

### 伝播の危険（塞いだ）

独立調査で「**ホストの `/` が `shared` だと、namespace 内の read-only 再マウントが外へ伝播してホストを読み取り専用にしうる**」という指摘を得ました。**実測すると、このホストの `/` は `private` なので漏れませんでした**（namespace 前後でホストは書き込み可能、`/` は前後とも `private`）。

**ただし「このホストでは漏れない」に依存してはいけない**ので、`unshare` に **`--propagation private`** を追加しました。**これで `/` が `shared` なホストでも伝播しません。**

### 読取は閉じていない（変わらず）

`/dev/shm` が書ける以上、**書込境界は「root tree と、呼び手が元々書けなかった別マウント」に限られます**。**読取は一切閉じていません。** `scope` に `reads are not confined` と書きました。

独立調査が挙げた「読取を閉じる」手段（tmpfs でパスを覆う、Landlock、fd の失効、`/proc/<pid>/root` の遮断）は、**いずれも本実装の範囲外**です。特に：

- **既に開かれた fd 経由の読取は、user namespace では防げません**（root か Landlock/seccomp が必要）
- **hard link や別経路の bind mount は、全木の走査なしには否定できません**

**これらは「未実装」ではなく「未測定であり、主張しない」ものとして残します。**

## 追記（2）：`basis=host-boundary` が「namespace が実際に走った」証拠を持つようにした

**独立レビュー（Step Code `step-5-preview`）が、本実装の実在する穴を指摘しました。**

### 穴：launcher を組めたことが「境界が効いた」と同一視されていた

`boundary != nil` は「launcher を build できた」ことしか意味しません。**script を飛ばして child を直接 exec する偽の `unshare`** を PATH に置くと、**実測で `claimed=true`・`basis=host-boundary` になりました。**

| 構成 | `claimed` | `basis` |
| --- | --- | --- |
| **script を飛ばす `unshare`** | **`true`** | **`host-boundary`** ← 穴 |

**「launcher を名指しした」ことと「その script が走った」ことは別**でした。

### 塞ぎ方：script が nonce を書き、entry がそれを読む

1. entry が `ALLOW_PROOF` に nonce（`sha256:` + state と firstState から決定的に導出）を入れる
2. script がそれを許可stateディレクトリへ `.tei-boundary-proof` として書く
3. entry が**そのファイルを読んで一致を確認**し、無ければ `unperformed`
4. 加えて script は root が本当に ro になったかを**書いてみて**確かめ、駄目なら exit 96
5. script の段階は 95〜99 の終了コードを使い、その範囲なら `unperformed`

### 再実測

| 構成 | 終了 | `claimed` | `repro` | `basis` |
| --- | --- | --- | --- | --- |
| **script を飛ばす `unshare`** | **1** | **`false`** | **`unperformed`** | `entry-post-hoc` |
| **本物の `unshare`** | **0** | **`true`** | **`reproduced`** | **`host-boundary`** |

理由の文言：`the state boundary left no proof that it ran, so its confinement was not shown`

### レビュー指摘の採否（2件目）

| 指摘 | 判定 |
| --- | --- |
| `driveSelfBuildIn` の argv 組み立てで添字がずれ、`sh` placeholder を上書きする | **誤り**（`boundary.Launcher` 全体をコピーしてから child を append しており、placeholder は保持される。実測でも本物の launcher は動作） |
| `host-boundary` が「namespace が実際に走った」ことを確かめていない | **実在**（上記のとおり塞いだ） |

**2件目を実際に再現してから直した**ので、**指摘を信じたのではなく、穴を観測してから塞いでいます。**
