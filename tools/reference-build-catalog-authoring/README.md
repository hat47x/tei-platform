# builder catalogの著 authoring 参照

`tei self-build -build-catalog` が読むcatalogを、**能力宣言から組み立てる**参照ツールです。

[SB-C0 §4.32](../../plan/design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)は、catalogを**製品の外のデータ**と決めました。Workspaceにもplugin構文にも欄を足さない、という決定です（両方のparserは未知の入力を拒否するため、欄を足すと古いbinaryがファイルを読めなくなる）。その決定が残した「誰がcatalogを書くか」を埋めるのがこのツールです。**製品の契約は変えません。**

## 何をするか

能力宣言（build toolが何をできるか）を読み、そこに書かれた**artifactのdigestを実ファイルから計算して**、catalog JSONを書きます。

**digestを宣言に書くことはできません。** 手で書いたdigestを許すと、リポジトリに無いbuilderを名乗るcatalogが作れてしまうからです。

## 使い方

```bash
go run ./tools/reference-build-catalog-authoring \
  -declaration build-declaration.json \
  -root . \
  -output /tmp/catalog.json
```

出力:

```json
{
  "catalogPath": "/tmp/catalog.json",
  "catalogDigest": "sha256:…",
  "providers": 2,
  "dependencies": 1,
  "targets": 2
}
```

`catalogDigest`は**書いたバイト**に対する`sha256`です。そのまま`tei self-build -build-catalog-digest`へ渡します。

## 能力宣言の形

```json
{
  "providers": [
    {
      "plugin": "tei.builder.go",
      "version": "1.2.0",
      "implementation": "go.builder",
      "contract": "tei.plugin-build",
      "contractMajor": 1,
      "artifact": "dist/builder",
      "recipeKinds": ["go-package"],
      "targets": ["host/linux/amd64"],
      "profiles": ["release"]
    }
  ],
  "dependencies": [
    {"kind": "build-artifact", "id": "tei.support.codec", "version": "1.4.0", "artifact": "dist/codec"}
  ],
  "targets": [
    {"target": "host/linux/amd64", "profiles": ["release", "debug"]}
  ]
}
```

| 欄 | 意味 |
| --- | --- |
| `artifact` | `-root`からの**相対パス**。このファイルのdigestがcatalogに入る |
| `authorityScope` | **書かない。** 常に`build-tool`（build toolとして名乗れる値は一つだけ） |
| `artifactDigest` | **書かない。** 書くと拒否される |

**この形式は製品の形式ではありません。** 互換性の約束も`formatVersion`もありません。製品が読むのは、このツールが出力するcatalogだけです。

> **宣言を配るなら、配る前に`formatVersion`を足す。** 版を付けないのは、**配るのが catalog だけだから**である（catalog は既に `formatVersion` を持ち、未知の版を拒否する）。宣言はこのツールと同じ場所で変わるので、版を名指す相手がいない。
>
> **後から足すことはできない。** 宣言は未知のキーを拒否する形で解析されるので、`formatVersion` を足した宣言は**拒否される**（実測）。つまり配布後に版を導入すると、その時点の利用者全員にとって破壊的変更になる。判断の時期は「必要になった時点」ではなく「**配布を決めた時点**」である。根拠は[SB-C0 §4.35](../../plan/design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)と[実測の記録](../../plan/evidence/TEI_GO_SELF_BUILD_DECLARATION_VERSIONING_2026-11-10.md)。

## 拒否するもの

| 入力 | 理由 |
| --- | --- |
| 宣言に`artifactDigest`を書く | 読んでいないバイトを名乗れなくする |
| `-root`の外を指す`artifact`（`../…`・絶対パス） | ルートが持たないファイルをcatalogに載せない |
| 存在しない・通常ファイルでない`artifact` | 同上 |
| 既存の`-output` | 上書きしない |
| 組み立てたcatalogが`ParseBuildCatalog`で読めない | entryが読めないcatalogを「catalogがある」と誤解させない |

## この参照が主張しないこと

- **catalogが正しいこと。** 名指しされたデータを使った、という記録しかない（[§4.22](../../plan/design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) K-3、SC0-26）
- **誰が能力を宣言してよいか**（権限）。catalogは「buildできると名乗るデータ」であり、名乗りの妥当性は範囲外
- **builderの実装が正しいこと**。artifactのdigestを計算するだけで、中身は見ない
- 既定の試行でのClaim（[§4.30](../../plan/design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)のまま。catalogを渡す一手が要る）

## 検証

```bash
go test ./tools/reference-build-catalog-authoring -count=1
```

**6件**：artifactのdigestが実ファイルから計算される／宣言の`artifactDigest`を拒否／`-root`外を拒否／存在しないartifactを拒否／既存出力を上書きしない／`-h`相当の出力に呼び手が要るdigestが入る。
