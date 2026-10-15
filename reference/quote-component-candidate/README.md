# 見積もりプレビューの型付き境界候補（WIT・未採用）

管理課題：[#1145](https://github.com/hat47x/tei-platform/issues/1145)。
基準は[ADR-0032](../../plan/adr/0032-composition-kernel-replacement.md)と[既存のGo／JavaScript比較](../quote-cross-language/README.md)。このディレクトリの`quote.wit`は**採用前のAdapter候補**であり、TEIの正式なService Contract、Plugin ABI、Providerの受理証明ではない。

## 選んだConsumerと比較の問い

想定Consumerは、受注入力画面の**読み取り専用・オフライン見積もりプレビュー**である。数量・単価・SKUを表示中に小計を計算するが、受注確定、顧客情報取得、認可の付与、DB更新、外部通信はしない。

Go Server／CLIとNode.jsでは現状のProvider＋既存Transportで業務意味を確認済み。Desktop WebViewとBrowserについては、**同一の型付き実装成果物を別Hostで動かしたい**という具体的な配布要求が成立したとき、Component Modelに追加価値があるかを試す。WITがなければ実現できないと最初から結論しない。ブラウザのCore Wasm、JSモジュール、Host Adapterでも成立するなら、増えるランタイム・配布・ビルド費用と比較して採否を決める。

| 比較案 | 得られること | 欠点・未確認 |
| --- | --- | --- |
| Native Go＋Node.js JS | 同じ意味契約に従った案件個別Providerを容易に独立開発 | 物理成果物と言語別実装を共有しない |
| JS UI側の直接計算 | BrowserとWebViewへの組込みが簡単 | 業務実装をJSに限定。UI内の計算をサーバー認可と同一視できない |
| Core Wasm＋JS Host関数 | 小型の純計算をBrowserなどで実行 | エクスポートやメモリ型、文字列変換の個別Adapterが必要 |
| WIT＋Component Model | 型付きの呼出し形・結果・失敗とHost importを明示し、異言語・異Hostで同一Componentを扱う候補 | Host/Guest生成器、Component runtimeまたはブラウザ向け変換、成果物管理が追加で必要 |

## 型付き候補で表せる意味

WITの`quote-preview`は`quote-request`を受け、`result<quote-result, quote-rejection>`を返す。SKU、通貨、明細集合、整数数量／整数最小通貨単位／小計と、**業務拒否**を型として表す。整数は`u64`に限定し、途中をJSの不正確な`number`へ落とさない。`line-count`は`u32`である。

現行のProviderからは「invalid or unsupported read-only quote」という一種類の業務上の拒否が返るため、今回のWIT候補も`rejected`の一種類だけとする。Providerが返す業務拒否と、Host実行失敗・権限不足・成果物不一致・世代失効を同一の`result`へ変換しない。後者は外側の実行境界で扱う。

**WITの型だけでは表せないもの**：`JPY`以外の拒否、1〜50明細、数量1〜100、単価0〜10億、小計1兆以下、SKUの空白・UTF-8長、入力JSONの重複キー・null・欠損・大文字小文字・指数表記など。これらは既存の業務意味契約と入力境界に残す。特にJSONからWITのRecordへ変換した**後**には、失われた重複キーや元の整数の綴りを復元できない。WITを追加してもHostは既存の入力検証を省略できない。

## 最小Worldと権限

`quote-preview-pure`にはWITの`import`を宣言しない。読み取り専用の純粋計算として、ファイルシステム・ネットワーク・時計・主体・秘密情報のHost関数を追加しない。ただし**WIT上にimportがないこと自体は、Hostの実隔離・Runtime Authority・信頼できるArtifactを保証しない**。同一のService Contractの提供元選択は既存Kernel／Host Authorityを使い、第二Resolver／Lock／Authorityを設けない。

契約ID `reference.order.quote-lines/v1` とWIT package `tei:quote-preview-candidate@0.1.0` は**別の識別子**である。この候補は明示的Adapterを通じて既存契約へ対応付けるための材料であり、WIT名称・versionを正本へ昇格させない。

## 今回の試験と明示的な未達

`quote_boundary_test.go`は固定23件の候補業務Oracleを読み、正常8件のデータを**事前に検証済みという条件**でWITの数値型へ損失なく投影し、数量×単価が独立期待値と一致するか確認する。拒否15件はOracle状態の保持のみ確認する。重複キー・ケース違いと2^53超のJSON数値を**標準Decoderへ通すと元の情報が失われ得ること**を反例とし、入力検査を省略できないことをテストにする。

WIT構造については静的な境界チェックを行い、`wasm-tools`が利用できる場合だけ公式Parserで`wasm-tools component wit quote.wit`を実行する。利用不可なら**文法検証は未実施**とログに残す。

```sh
go test -count=1 ./reference/quote-component-candidate
go vet ./reference/quote-component-candidate
wasm-tools component wit reference/quote-component-candidate/quote.wit
```

本ラウンドで**Componentバイナリのビルド、WIT binding生成、実Hostでのロード・実行、Browser transpile、複数実Hostの受理を完了したとは主張しない**。次に採用を検討するための実機判定は次の順序とする。

1. 具体Consumerが同一Component成果物を本当に二つ以上のHostで必要とするかを確認する。不要ならWITを必須化しない。
2. `wasm-tools`でWIT構文を検証し、Guest bindingを生成し、`jco`等のBrowser向け変換後とNative Hostで同じ未受理23候補ベクトルを実行する。NodeのCLI動作をBrowser実証へ転用しない。
3. HostがProvider選択・現行権限・Artifact bytes・Generationを既存の機構で検証でき、Typed errorが業務拒否だけを表し、Host失敗を区別できるか確認する。
4. 成果物サイズ、追加依存、起動・定常負荷、更新単位を観測する。案件に具体的なSLOがない限り一律閾値を課さない。

## 参考資料

- WebAssembly Component Modelの[WIT仕様](https://github.com/WebAssembly/component-model/blob/main/design/mvp/WIT.md)
- Bytecode Allianceの[JavaScript／Component Model](https://component-model.bytecodealliance.org/language-support/building-a-simple-component/javascript.html)
- Bytecode Allianceの[wasm-tools](https://github.com/bytecodealliance/wasm-tools/blob/main/README.md)


## 実Componentを二つのJavaScript Hostで実行する候補実験（#1861）

実行スクリプト：[`e2e/run.sh`](e2e/run.sh)。この検証は上の`quote_boundary_test.go`とは別で、外部npmと実Chromiumを用いるため**通常のGo全体試験には組み込まない**。`node`、`npm`、`sha256sum`および環境変数`TEI_BROWSER_EXECUTABLE`で指定した実Chrome/Chromiumが必須。

```sh
export TEI_BROWSER_EXECUTABLE="$(command -v google-chrome)"
bash reference/quote-component-candidate/e2e/run.sh
```

`jco`を一時領域へ取得し、`quote.wit`を実解析して`guest.mjs`から**Componentバイナリ**を作る。`jco wit`で実際のimportsを確認し、`jco transpile`によって得た同じ成果物由来のCore WasmとJS AdapterをNodeと実Chromium（Playwrightで起動）で検証する。未受理の固定候補23件について、受理8件、拒否15件の期待結果を保持する。ただし**入力がWITへ表現できず変換前に拒否される例**と、**Guestが業務拒否する例**は分離して数える。Host起動・成果物ロード・実行が失敗した場合を「業務拒否成功」とみなさない。

この比較は**NodeとBrowserという二つの異なるJavaScript実行環境**の動作であり、Wasmtimeなど異なるNative Component Hostとの同一バイナリの直接実行を証明しない。さらに`jco transpile`は実Componentバイナリから別の実行可能成果物を生成するため、BrowserがComponent形式のバイナリを直接実行したとも主張しない。両Hostが共通のビルド入力Componentを持つという限定的な証拠である。

Guestの純計算に`import`を宣言しないことは、製品としての起動権限、Artifact Load時の真正性、配備先の信頼、OS隔離、生成世代の保持を意味しない。独立したHostの比較、機能面以外の費用測定、正式な受理／採用判断は残る。実験のCIが未完了または失敗した時点では、この段落を実行成功の証拠にしてはならない。
