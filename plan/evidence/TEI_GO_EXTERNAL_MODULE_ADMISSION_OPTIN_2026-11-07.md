# 外部Go Module入力の明示opt-in受入（案C）の参照実装

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-Bに対応する参照実装です。方針の選択肢は[外部Go Module入力の受入契約](../design/EXTERNAL_GO_MODULE_ADMISSION_CONTRACT_2026-11.md)に整理し、本案C（段階導入）を**既定案**として実装しました。

**既定の挙動は変えていません。**`ResolveSourceClosure`は従来どおり外部Module根を拒否します。明示的に供給を渡した呼び出しだけが、外部Moduleメンバーを含む閉包を受け取ります。

## 二つの入口

| 入口 | 方針 | 外部Moduleを選ぶ対象 |
| --- | --- | --- |
| `ResolveSourceClosure`（既存） | `repository-only` | 拒否（変更なし） |
| `ResolveQualifiedSourceClosure`（新規）・供給なし | `repository-only` | 拒否。理由を`refusedReason`へ記録 |
| `ResolveQualifiedSourceClosure`・供給あり | `external-module-opt-in` | 選択されたModuleメンバーを`module:<path>@<version>/<file>`で受入 |

`ResolveQualifiedSourceClosure`は、既存`ResolveSourceClosure`を置き換えません。既存の呼び出し元（`tools/reference-plugin-self-build`等）の挙動と検証根拠をそのまま保ちます。

## 受入記録

`ClosureAdmission`は、その閉包が**どの方針で、何を根拠に**作られたかを残します。

| 欄 | 意味 |
| --- | --- |
| `policy` | `repository-only` または `external-module-opt-in` |
| `supplyRoot` | 使った供給ディレクトリ（opt-inのときだけ） |
| `repositoryMembers` / `externalMembers` | メンバー数。返した一覧と一致することをテストが固定 |
| `externalModules` | 選択されたModule識別子（`path@version`） |
| `goSumBinding` | Module byteを何が拘束したか |
| `qualified` | 資格を出したか（拒否時は`false`） |
| `qualificationBasis` | 何を根拠に資格を出したか |
| `refusedReason` | 拒否した理由（拒否時だけ） |
| `closureDigest` | 返したメンバーから再計算できる閉包digest |
| `boundary` | この記録が扱わない範囲（供給元・署名・権限・受理） |

### `goSumBinding`が主張すること、しないこと

`goSumBinding`は「**Go toolchainが`go.sum`と一致しないModule byteを拒否する**ので、その検査に依拠した」と記録します。本resolverが同じbyteを再検証した、とは書きません。独立したbyte照合は[SB-0bの参照実装](../../reference-go-module-supply-verify/README.md)が担います。

## 実装上の判断

1. **外部メンバーは名前空間で分ける**。`module:`前置を予約し、リポジトリ内のパスが`module:`で始まる場合は拒否します。パス衝突で「リポジトリのファイル」と「Moduleのファイル」を取り違えないためです。
2. **外部メンバーの場所はmodule相対**。cacheの物理位置を名前に入れません（[SB-A1](TEI_GO_SOURCE_CLOSURE_OBSERVATION_2026-11-07.md)と同じ規則）。
3. **Module識別子に版が無い場合は拒否**。版の無いModuleは内容同一性を持てません。
4. **供給なしで外部Moduleを選ぶ対象は、部分的な閉包を返さず拒否**。`members`は`nil`、digestは空、`qualified`は`false`で、`refusedReason`に理由を入れます。
5. **resolver自体が失敗した場合も`refusedReason`を残す**。供給が無い隔離環境では`go list`が先に失敗するため、その場合も「方針と理由」が読めるようにしました（テストが捕捉）。
6. **Windowsの8.3短縮パス対策**は[SB-A1](TEI_GO_SOURCE_CLOSURE_OBSERVATION_2026-11-07.md)と同じ正規化を共有します。

## 2026-11-07の検証結果

- 環境: WSL2 Ubuntu 22.04 / Go 1.27.3 linux/amd64、およびWindows 11 / Go 1.27.0

| 試験 | 期待 | 実測 |
| --- | --- | --- |
| `cmd/tei`・供給なし | 拒否、`policy=repository-only`、`qualified=false`、理由あり | PASS |
| `cmd/tei`・供給あり | 受入、`policy=external-module-opt-in`、外部メンバー数とModule数が観測と一致 | PASS |
| 外部Moduleを選ばない対象・供給なし | 受入、`policy=repository-only`、外部メンバー0 | PASS |
| リポジトリ内が`module:`前置を使う | 拒否（`reserved external member namespace`） | PASS（Linux） |

補足: `module:`はWindowsのパスに使えない文字`:`を含むため、同試験はWindowsでskipします。リポジトリのパスとして表現できない以上、Windowsでは衝突自体が起こりません。

## 限界

- 供給元・発行者・署名・Trustは**未検証**です。`go.sum`自体を誰が受理したかの根拠も別途必要です（設計文書の「未取得の観測」）。
- 既定の`ResolveSourceClosure`は変えていません。したがって`cmd/tei`を既定経路で扱う限り、閉包は依然として`rejected`です。
- 本Referenceを製品経路（`internal/plugin.BuildSource`等）へ接続していません。接続は所有者の判断です。
- Linux amd64での観測が主で、Windowsはテスト通過のみ確認しています。

## 実行した検証

- `gofmt -l tools/internal/gobuilder`：差分なし
- `go vet ./tools/internal/gobuilder`：PASS
- `go test ./tools/internal/gobuilder -count=1`：PASS（既存試験を含む。WindowsとWSLの両方）

**NOT RUN**：製品経路への接続、`go.sum`のTrust根拠の設計、Stage比較（SB-C）、CI、trusted full checkout。
