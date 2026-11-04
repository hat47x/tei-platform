# 可変長・可変文字集合の内容導出ID：共通契約案（2026-11）

- **文書の性格**: 実装前設計・未採用。採用済み仕様、既存ID移行の決定、テスト実施結果ではない。
- **作業の正本**: [#2814](https://github.com/hat47x/tei-platform/issues/2814)（受入条件・進捗）。
- **確認基準**: `main` commit `1f9f7d5d1876a0d0c420568bdf06becd96923b21`（2026-11-04 JST）。同時点の現行Referenceを変更しない。
- **関連する採用済み境界**: [#1120](https://github.com/hat47x/tei-platform/issues/1120)（論理ID・型付き参照）、[#1595](https://github.com/hat47x/tei-platform/issues/1595)（Canonical Capability ID）、[#2094](https://github.com/hat47x/tei-platform/issues/2094)（派生名の衝突検出）、[全体構成](../../docs/architecture.md)、[管理名前空間案](PLUGIN_NAMESPACE_AND_OPTIONAL_COMPOSITION_2026-10.md)。
- **既存の責務**: [#951](https://github.com/hat47x/tei-platform/issues/951)（Plugin/Contract identity）、[#943](https://github.com/hat47x/tei-platform/issues/943)（Artifact/Build/Lock）。これらの第二のAuthority・Resolver・Lockを作らない。
- **非主張**: 既存SHA-256やDigestが置換済み、新方式が採用済み、テスト/CI/Provider/Studioが成功済み、短縮IDが暗号学的完全性や権限を証明する、とは主張しない。

## 1. 必要性と解決する失敗

数字だけの外部ID、URL用の短い文字列、ケース非区別のDB、ファイル名、人が照合する短縮表示には異なる長さ・許容文字集合がある。一方、同じ内容の照合に使う暗号学的Digestと、編集を跨いで保持する論理IDと、個々の業務レコードのキーは異なる概念である。単なる`SHA256(hex)[:N]`は、暗黙の衝突リスク、異なるルールの混同、文字数と情報量の誤認を生む。

**中心原則**: 内容からのID生成を選択できるようにしながら、同一性の意味、生成規則、物理的な文字列表現を別の契約として保持する。

- `logical identity` / `qualified semantic ref`: 意味上の正本対象を識別。改名・内容更新時の継続性は#1120が決める。ハッシュを必須にしない。
- `content digest`: 規定した正規化済みbyteとアルゴリズムの照合値。対象の内容が変われば変化し得る。
- `hash-derived ID`: 内容/目的/スキームに拘束して生成する識別値。衝突確認と用途ごとの適格性が必要。
- `display alias`: 短縮・表示に使う補助。完全な正本IDへの単独逆解決、Authority、意味の受理を含意しない。
- `artifact integrity / publisher signature`: exact artifact byte・署名・Trust等を別途検証。短いIDから代替しない。

## 2. 最小型と責務（名前・JSONは候補）

| 概念 | 必要な情報 | 意味 |
| --- | --- | --- |
| `HashDigest` | algorithmId + algorithmVersion/profile + outputBits + rawBytes | バイナリの照合値。**encodingを含めない** |
| `HashIdScheme` | schemeId/version, inputKind, canonicalizationProfile/version, purpose, namespace/scope, generator, output rule, constraints | 入力と結果の再現規則。独立に識別・固定 |
| `HashId` | scheme identity, value, scope, optional source revision/provenance | 規則を伴う内容導出ID。裸の文字列の同値比較をしない |
| `HashIdText` | scheme identity, alphabet/encoding and serialized text | Targetに渡す表記。文字列表現だけで意味・権限を確定しない |
| `TargetIdConstraint` | allowed alphabet, min/max/exact length, case/collation, reserved names, path/URL safety | 外部制約。Schemeが表せてもTargetへ適用可能とは限らない |

`tei.std.hashId.*`は提案中の管理名前空間であり、現行`std.*`の自動改名も、Kernel固定型への組込みも決めない。標準のContractはProvider/Pluginとして再利用可能であることを優先する。UIやHostは不要。

## 3. 同一性の導出手順

```text
Source / approved input
  -> canonicalize(inputKind, profileVersion) -> canonicalBytes
  -> domain-separate(purpose, scope, inputKind, version)
  -> derive(mode, algorithm, digestBits OR alphabet+length)
  -> HashDigest or HashId
  -> target validation + collision check -> optional physical ID
```

- **入力を勝手に正規化しない**。raw bytes、JSON canonical、Unicode NFC等は異なる`canonicalizationProfile`を持つ。JSON等の重複キー・不正UTF-8・非対応型・不明versionは拒否。Unicodeの正規化表はversion固定。実装間で同じbyteを作る。
- **曖昧性を閉じる**。入力type、scope、purpose、正規化版を長さ付きのバイナリframe（big-endian整数、定めたUTF-8）で区切る。文字列連結の区切り曖昧性を許さない。例: `frame(x) = u64be(byteLength(x)) || x`。`a|bc`と`ab|c`が衝突してはいけない。
- **ドメインを分離する**。`tei.hashId.content.v1`と`tei.hashId.exact.v1`、各`purpose`、`inputKind`、`scope`、`canonicalizationProfile`を入力/Customizationへ機械可読に束縛する。単なる`hash(plainString)`を全用途で共有しない。
- **Schemeの版を固定する**。文字集合の**並び順**、case policy、出力長、XOF消費順序、leading zero、padding、ビット順も同じ仕様の一部。変更時は別schemeとして扱い、旧値を再解釈しない。
- **同一性の持続性**。内容から生成したIDは内容に依存する。内容が変わっても保持する意味ID、業務レコードID、外部の既存IDは別途保持・関連付ける。

### 3.1 モードA：`encodedDigest`

```json
{
  "schemeId": "tei.std.hashId.v1",
  "mode": "encodedDigest",
  "purpose": "example.artifact",
  "scope": "example.release",
  "inputKind": "canonical-binary",
  "canonicalization": "raw-bytes/v1",
  "generator": {"algorithm": "cshake256", "outputBits": 160},
  "presentation": {"encoding": "base32-rfc4648-lower-unpadded"}
}
```

バイナリDigestを固定ビット長で作成し、Base16/Base32/Base64url等で**可逆に**表現する。encodingだけを切り替えた場合、Digestのバイナリ値は変えない。結果の文字列をDigestの代わりに無条件に比較しない。標準Base-NにはRFC 4648の非使用ビット0・case/padding規則を適用し、非正規の別表記を拒否する。

Base10/Base36/Base62の固定長変換も許すが、RFC 4648準拠とは称さない。big-endianの**非負整数**として符号なしに変換し、左側をalphabet[0]で指定文字数まで埋める。逆変換はbitsを超える値、非正規長、先頭ゼロ規則違反を拒否する。例えば128 bitはBase10で39桁、Base36で25桁、Base62で22桁、RFC 4648 Base32（paddingなし）で26文字、Base64url（paddingなし）で22文字を要する。

SHA-256は256 bit固定。短縮する場合は`sha256-truncated/v1`のように**明示的な別プロファイル**とし、SHA-256フルDigestと誤表示しない。可変出力のSHAKE256/cSHAKE256/BLAKE3は指定byteを導出できる。ただしXOFで出力長を変えたときのprefix関係に注意し、非互換な長さ変更はSchemeの新しい版として扱う。

### 3.2 モードB：`exactCharacters`

```json
{
  "schemeId": "tei.std.hashId.v1",
  "mode": "exactCharacters",
  "purpose": "example.external-record",
  "scope": "example.application",
  "inputKind": "canonical-binary",
  "canonicalization": "raw-bytes/v1",
  "generator": {"algorithm": "cshake256"},
  "presentation": {
    "alphabet": "0123456789abcdefghjkmnpqrstvwxyz",
    "length": 32,
    "unit": "unicode-scalar"
  }
}
```

**このモードはDigestの可逆エンコーディングではない。** アルファベット集合`A`と正確な長さ`L`が固定され、XOF出力から一様に文字を選んで`A^L`通りの文字列へ写像する。

正確な候補アルゴリズム（PoCで固定する）:
1. 結果の文脈（`mode`、アルファベットの順序を含む正準表現、`L`）をdomain-separateしてXOFを初期化する。
2. `k`を`256^k >= A`となる最小の正整数、`M=256^k`、`T=floor(M/A)*A`とする。
3. XOFの出力を**連続したk byte big-endian単位**で読み、`x>=T`を棄却、そうでなければ`alphabet[x mod A]`を1文字追加する。
4. `L`文字で停止。上限内に達しない場合はエラー。拒否した候補のXOF消費位置も再現規則に含む。

この方法なら除算の剰余による文字頻度の偏りを回避できる。任意のUnicodeアルファベットを要求する場合、文字数の単位は**Unicode scalar value**であり、grapheme cluster/画面表示幅とは異なる。Unicode版、正規化、紛らわしい字形、結合文字、制御/方向制御文字、DB照合順序は独立Policyで検査する。PoCの最初の相互運用ProfileはASCII alphabetから開始し、Unicode完全対応は別の反例・ターゲット検証を条件とする。重複文字・空集合・上限超過・無効scalarは拒否する。

### 3.3 桁数・情報量・衝突

`A`種類・`L`文字なら最大`M=A^L`種類、理想的な情報量は `H=L*log2(A)` bit。誕生日近似は（生成対象数`n`が`sqrt(M)`より十分小さいとき）`P(collision) ≈ n(n-1)/(2M)`。一般には`1-exp(-n(n-1)/(2M))`を用いて概算する。**暗号強度・原像抵抗性そのものと同一ではない**。固定Digestを短縮すれば利用できる識別空間も縮小する。

例：数字20桁は約66.4 bit。Base32で32文字なら160 bit相当。事前の`ExpectedPopulation`と`MaxCollisionProbability`などはTarget/用途Policyが指定し、危険な短縮を表示警告だけで許さないPolicyでは生成を拒否する。生成後も永続側の一意制約/重複検査が必要。自動saltや連番再発行で既存意味IDを黙って変更せず、衝突時は両候補とScheme/Scopeを示してFail Closedする。再試行・割当を許す用途では、採用したsalt/nonce・再割当の記録とAuthorityを別に定義する。

## 4. 方式の比較と導入順

| 候補 | 出力 | 導入位置 / 条件 |
| --- | --- | --- |
| SHA-256 | 256bit固定 | 既存互換・Golden比較。変更時は明示移行 |
| SHAKE256 (FIPS 202) | XOF | 可変長の基本比較 |
| cSHAKE256 (SP 800-185) | customizable XOF | 用途・契約を分離する新規ID候補。ただし仕様をPoCで決定 |
| BLAKE3 | XOF | 高速/大量処理時のProvider候補。SHAKE/cSHAKEとの同値性は主張しない |

外部標準: [FIPS 202](https://csrc.nist.gov/pubs/fips/202/final)、[SP 800-185](https://csrc.nist.gov/pubs/sp/800/185/final)、[RFC 4648](https://www.rfc-editor.org/rfc/rfc4648)。SP 800-185r1は2026-10-08公開の**初期公開草案**であり、既存正式版とは区別する。BLAKE3は[公開仕様](https://github.com/BLAKE3-team/BLAKE3-specs)を参照。

導入は(1)契約とGolden/負例 → (2)SHA-256/SHAKE256/cSHAKE256の限定Reference → (3)exactCharactersの文字集合・長さ試験 → (4)BLAKE3/Unicode/Target連携比較。固定アルゴリズムのKernel強制や新しいRegistryはしない。

## 5. 正本と実現先の境界

- 正本の`qualified ref`、構成のLogical Instance ID、Runtime View ID、Source Token、業務主キー、Artifact Digestを別種として扱う。表示IDでGraphの旧選択や異Scopeを再解決しない（#1120/#1124）。
- Plugin/Bundleの公開fingerprintやRuntime EngineのLock/署名を一律置換しない。旧識別子と新Schemeの**並存比較**は可能だが、同一性の結論は当該Owner/検証で確定する（#943/#951）。
- Targetへ適用する前に文字数、文字の許容性、予約語、case folding、Unicode normalization、DB collation、Path、URL、ファイル名の制約を確認。外部環境で`A`と`a`が同一視されるなら`A^L`の空間をそのまま評価しない。
- 内容を公開から推測できる deterministic hash IDは秘密値・認可証明ではない。署名・トークン・秘密鍵・認可情報は別の責務。
- 生成手順をAIが構成しても、生成したSchemeと入力/出力の宣言・選択されたProvider・検査結果を図表とログから辿れ、隠れたフックや暗黙の再ハッシュを設けない。

## 6. 最小受入試験表（Go/TypeScript参照試験を一部追加。全体受入は未実施）

| ID | 入力 / 条件 | 期待する証拠 |
| --- | --- | --- |
| H01 | raw byte同一、同一Scheme、二回生成 | binary/text完全一致 |
| H02 | 同一DigestのBase16/Base32/Base64url切替 | 復号byte一致、表記の違いだけを記録 |
| H03 | 128bit・Base10/36/62、先頭ゼロ | 固定長・逆変換、非正規文字列を拒否 |
| H04 | 任意Alphabet、Aが256の約数でない、length変更 | rejection samplingのGolden、modulo biasなし、長さ固定 |
| H05 | 同一値でもpurpose/scope/inputKind/profileが異なる | 異Schemeとして比較拒否。用途分離の正規化を再現 |
| H06 | UTF-8/NFC/NFD/重複JSONキー/曖昧frame | 固定profileで一致/差異/拒否を説明。曖昧連結なし |
| H07 | 数字20桁で大規模母集団、Target要求と不足 | 衝突リスク診断・Policyによる拒否 |
| H08 | case-insensitive DB、全角/大小文字、予約ファイル名 | 物理Target衝突・違反を適用前に拒否 |
| H09 | 生成後の同一文字列衝突、異scope/曖昧lookup | 黙った再割当・別対象上書きをしない |
| H10 | 旧Canonical ID、SHA-256 Fingerprint、Artifact/Scenario受入 | 非回帰、旧schemeの再発行なし |
| H11 | Go/TypeScript、OS・Unicode実装差、Golden fixtures | 同一input canonical bytes/同一output。非互換profileを拒否 |
| H12 | Studio/CLI/Provider複数Consumer | 同じScheme解決と診断。UI view IDを正本へ昇格させない |

Golden vectorはアルゴリズム別の**公開既知テストベクトル**と、TEI envelope/schemeの独立した合成vectorを分ける。双方が同じ誤った実装を共有しただけでは合格としない。Go参照と型付きTypeScript参照、Node.js独立SHAKE256実装・共有Golden fixtureを [`tools/reference-hash-id/`](../../tools/reference-hash-id/) に追加した。cSHAKE256の暗号PrimitiveはNIST公開ベクトルを用いるGo試験で独立検査する。TypeScript実装の暗号計算は注入されたProviderが担当し、TypeScriptコンパイラがない環境では型チェック試験を明示Skipする。**Go/TS/Nodeの同一HEADでの合格、独立cSHAKEのTEI frame照合、実DB/Windows/ブラウザ検証はCIと個別受入根拠が揃うまで未実証**。この表は全体受入の計画である。

## 7. 未決・停止条件

1. **製品の要件化**: 本設計から要求へ移すのはOwnerが用途・対象・非互換性を受理した後。まず#2814で比較。
2. **Exact-modeのUnicode Profile**: scalar vs grapheme、正規化版、DB collation、悪意ある文字列はPoC対象。ASCIIだけで本番の任意Unicode適合を宣言しない。
3. **識別とセキュリティの等価視を禁止**: 署名対象、Canonical ID、短縮表示、Artifact Digestを暗黙置換しそうな場合は作業停止。
4. **変更の計測**: 既存の契約の合成だけで同じ用途を実装できれば、新しいCore原語・新DB/Registryを増やさない。再利用できない失敗を実Consumerで示せたときだけ拡張する。
5. **資格**: 実装の合成試験が成功しても、実業務データの同一性、外部Provider、Owner意味受理、現在の実行権限を証明しない。

**現状（2026-11-04）**: `tools/reference-hash-id/` にraw-bytes/v1限定のGo Reference（生成、復号、Target制約・割当Snapshot比較）と、Host非依存の型付きTypeScript Reference（暗号PrimitiveはAdapter責務）、Go/Node共有Golden fixtureを追加した。SHA-256/SHAKE256の独立Node/Python系GoldenとNIST cSHAKE256公開ベクトルの試験を実装済み。Go1.26.10をローカル作業環境へ取得できず、Go全体テストの実績はCIで判定する。TypeScript型検査はコンパイラが利用可能な環境でのみ実行される。PR #2815の正確なHEADに対応するCI結果を見ずに成功とみなさない。全体受入、実Target同時登録、Unicode/OS、cSHAKE256の独立TypeScript Provider、既存Digest移行は未実証。本番適用はなし。
