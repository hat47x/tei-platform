# state分離の「host-boundary」は、この環境で実際に張れる（S2の実現可能性）

[#4486](https://github.com/hat47x/tei-platform/issues/4486) の S2 は「Linux限定の任意の境界で、書込可能なのを選択したstateと成果物ディレクトリだけにする」ことを求めています。**本記録は、それがこの環境で実際に張れることを確かめ、entryへ組み込む際の障害を特定します。**

**まだ実装していません。** 何が動いて、何が邪魔をするかの実測です。

## 前提

| 項目 | 値 |
| --- | --- |
| 環境 | WSL2 Ubuntu-22.04、kernel 6.18 |
| user namespace | **使える**（`unshare -rm true` 成功） |
| S1の事後検査 | 実装済み（[記録](TEI_GO_SELF_BUILD_STATE_SEPARATION_POSTHOC_2026-11-13.md)） |

## 張り方（実測した手順）

user + mount namespace の中で、**root を読み取り専用にし、許可したディレクトリだけを書き込み可能に戻す**。

```sh
unshare -rm sh -c '
  mount -o remount,bind,ro / || exit 97          # ここ以外を読み取り専用にする
  for d in "$ALLOW_STATE" "$ALLOW_ART"; do
    mkdir -p "$d"
    mount --bind "$d" "$d"
    mount -o remount,bind,rw "$d"                # 許可した場所だけ戻す
  done
  exec "$@"
' sh <child> <child args...>
```

## 実測

| 試したこと | 結果 |
| --- | --- |
| `mount -o remount,bind,ro /`（namespace内） | **成功** |
| `mount --bind / $T/rootbind`（別の場所へbind） | **失敗**（`wrong fs type, bad option, bad superblock`） |
| root ro の後に、許可ディレクトリを `remount,bind,rw` | **成功** |
| **同梱child を境界内で実行** | **exit 0、成果物とstateを作成**（正例が通る） |
| 許可外のパスへの書込 | **`OUTSIDE_WRITE_BLOCKED`**（ファイルは作られない） |
| 許可した state への書込 | **`ALLOWED_WRITE_OK`** |

**正例は通り、境界外の書込は落ちます。** [#4486](https://github.com/hat47x/tei-platform/issues/4486) が B2 で観測した `read-only file system` と同じ効果を、**root の再マウントだけで再現できました**。

## entry へ組み込む際の障害（特定した）

`driveSelfBuild` は**起動した実行物の digest**を `Invoked.ExecutableDigest` に記録します（`cmd/tei/cli_self_build.go`）。**境界を張るために `unshare` を起動すると、記録されるのは `unshare` の digest になります。**

**すると条件5のbuilder同一性（1回目と2回目の実行物の比較）が壊れます。** 1回目は child、2回目は `unshare` になり、`the two runs used different builder executables` で `mismatched` になります。

**したがって S2 には、起動するのは launcher だが digest は child のものとして記録する経路が要ります。** 例えば `driveSelfBuild` に launcher の前置を渡し、digest は `-child` のパスから取り続ける形です。**これは小さな変更ですが、`Invoked.Executable` と `Argv` の意味を変えるので、記録の意味を決めてから入れます。**

## していないこと

- **entryへの配線・flag・試験は未実装です**（本記録は実現可能性の実測）
- **1回目のbuildは境界の外**です（S2は2回目について述べています）
- **読取は制限していません。** 書込だけです
- **ネットワーク・常駐Process・環境変数**は境界に入れていません（#4486 の未決のまま）
- `linux/amd64`（WSL2）の1回です。**`darwin`・Windowsは対象外**（S2はLinux限定）
- **`qualified` は `false`**、G4は `UNQUALIFIED` のままです

## 2026-11-13 追記：上の「境界外の書込」の証拠を取り直した

**上の実測のうち「許可外のパスへの書込 → `OUTSIDE_WRITE_BLOCKED`」は、証拠として不正確でした。** 最初の探査スクリプトで**変数を設定し忘れ**、空のパスへ書こうとして失敗した結果を「ブロックされた」と読んでいました。**結論は変わりませんが、根拠を差し替えます。**

取り直した実測（明示的なパス、エラー文を表示）：

| 試したこと | 結果 |
| --- | --- |
| namespace 内で `/` を `remount,bind,ro` | **成功** |
| 許可ディレクトリへの書込 | **成功**（`ALLOWED_OK`） |
| 許可外のパスへの書込 | **`Read-only file system` で失敗**（ファイルは作られない） |
| `/root` への書込 | **`Permission denied`**（別の理由。境界の証拠には使わない） |

**許可外の書込は、`Read-only file system` という明示的な理由で落ちます。** これは「書込境界が効いている」ことの直接の証拠であり、**当初の（空パスによる）観測より強い**ものです。

**記録の性格：** 一度書いた測定の**根拠が弱いと分かった時点で差し替える**。結論を守るために根拠を残しません。
