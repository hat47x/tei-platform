# builder catalogの出所を探す（§4.16の前提の実測）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.16が「R-1の前提」として残した**catalogの出所**を、リポジトリ内で探した記録です。

**R-1（既定をin-processへ）を採るかどうかの判断は、まだ下していません。** 本記録は「出所が既にどこかにあるか」を確かめ、**無い**ことを示します。

## 何を探したか

`BuilderResolver`が答える問いは「contract・recipe kind・target・profileに適合するProviderはどれか」である。これに答えるには、各Providerについて最低でも次が要る。

| 必要な情報 | 何に使うか |
| --- | --- |
| `contract` / `contractMajor` | 要求との適合判定 |
| `recipeKinds` | recipe kindの適合判定 |
| `targets` / `profiles` | target・profileの適合判定 |
| `authorityScope` | build-toolとしての権限確認 |
| `artifactDigest` | 解決したBuilder identityに載せる |

## 探した結果

### 1. `internal/`にcatalogは無い

前周（§4.16）の実測どおり、四つのportの実装は`tools/`のfixtureにしかない。`internal/plugin.Requirement`は`Plugin`と`CompatibleMajor`だけを持つ。

### 2. Workspaceのplugin lockにも無い

`reference/workspace/.tei/plugin-lock.json`のplugin entryは**`id`・`version`・`fingerprint`の3つだけ**である（機械的に確認）。

```text
plugin entry keys: ['id', 'version', 'fingerprint']
  recipes in plugin entry: False
  recipeKinds in plugin entry: False
  targets in plugin entry: False
  profiles in plugin entry: False
  authorityScope in plugin entry: False
  contract in plugin entry: False
```

lockは「どのPluginをどの版で使うか」を固定するが、**そのPluginをどのBuilderが作れるかは持たない**。別の関心事である。

### 3. Plugin定義（`.tei`）にも無い

`internal/plugin.Definition`は`ID`・`Version`・`Runtime`・`Artifact`・`Provides`・`Requires`・`External`を持つ。**builderの選択方針を持つ欄は無い**。Pluginは「何を提供するか」を宣言するが、「誰がそれをbuildするか」は宣言しない。

### 4. リポジトリのどこにも`recipeKinds`を持つcatalogは無い

`internal/`・`cmd/`・`reference/`・`requirements/`・`plan/`のJSON・YAMLを`recipeKinds`で検索した結果、**0件**。`tools/reference-build-authoring-intent/testdata/catalog.json`だけが持つ。

### 5. ADRにも判断が無い

`plan/adr/`で`builder`に触れるのはADR-0028（共通知識面）とADR-0032（Plugin化の範囲）だが、どちらも**builder catalogの出所を定めていない**。ADR-0032は「BuilderもPlugin化できる」と述べるが、その場合のcatalogがどこから来るかは書いていない。

## 何が言えるか

**catalogの出所は、リポジトリのどこにも存在しない。** したがってR-1は「既にあるものを配線する」作業ではなく、**次のいずれかを新しく決める**作業になる。

| 案 | 内容 | 何が要るか |
| --- | --- | --- |
| C-1: Workspaceが宣言する | Workspaceのmanifestまたは`.tei`にbuilder catalogを書く | 形式の設計と、source index・plugin lockとの関係の整理 |
| C-2: 外部配布物として受け取る | 署名済みのcatalogを外から渡す | 供給（SB-0b）と同じTrust問題を抱える |
| C-3: 解決しない | catalogを必要としない範囲に留める | in-process解決を諦める（R-2据え置きの継続） |

**C-1とC-2は別のClaimを伴う。** C-1はWorkspace形式の拡張であり、C-2は供給のTrust（未検証）に依存する。どちらも本稿（入口の外形と拒否条件）の範囲外である。

## 何を言わないか

- **R-1を採らないとは言っていない。** 前提が無いと言っている。
- **C-3を推奨していない。** 「今すぐ選べるのはC-3だけ」というだけである。
- **in-process解決の価値を否定していない。** §4.16のとおり、切り替えてもL2-bにはならない。

## 検証

- `internal/plugin.Requirement`の欄: コードを読んで確認
- plugin lockのentry: `json`として読み、keyを列挙して確認
- `internal/plugin.Definition`の欄: コードを読んで確認
- `recipeKinds`の検索: `internal/`・`cmd/`・`reference/`・`requirements/`・`plan/`のJSON・YAMLで0件
- ADR: `plan/adr/`で`builder`を含むファイルを列挙し、内容を確認

## 限界

- 本記録は**探索**であり、コードも設計も変えない。
- 「無い」ことの証明は、探した範囲に限る。`research/`・`docs/`・Issue本文は機械的に検索していない。
- G4は`UNQUALIFIED`のまま。
