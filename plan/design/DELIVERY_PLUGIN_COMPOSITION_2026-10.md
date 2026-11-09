# 提供形態のプラグイン構成 設計案（2026-10）

- 状態: 実装前設計（ADR-0036の採用方針を具体化）
- 日付: 2026-10-19
- 関連: #2020 / #2019 / #1043 / #1223 / ADR-0036
- 目的: 個人・SIer・大規模SaaSの違いを、形態別の実装ではなくプラグインの選択と構成で実現する。

## 1. 構成の単位

```text
Canonical meaning / Core contracts
        |
Delivery Profile（選択と検証だけ）
        |
selected plugins
  - tenant-isolation
  - authentication / authority
  - storage / data-residency
  - distribution / export
  - audit / observability
        |
application / target
```

Coreは意味・契約・実行境界を持ち、提供形態の名前を持たない。提供形態は、必要な能力を持つプラグインの組として表す。

## 2. 提供形態と選択する能力

| 提供形態 | 主な能力 | 形態固有の実装 |
| --- | --- | --- |
| 個人 | ローカル保存、オフライン、単一利用者、簡易配布 | 作らない。既存pluginを選ぶ |
| SIer | 顧客別配布、エクスポート、権限委譲、監査証拠 | 作らない。既存pluginを選ぶ |
| 大規模SaaS | テナント分離、段階更新、監査、データ所在 | 作らない。基盤pluginを選ぶ |

## 3. SaaS向けテナント分離の基盤機構

テナント分離は、SaaSアプリケーションごとに実装せず、共通の基盤機構として提供する。アプリケーションはテナントの識別とスコープを受け取り、基盤機構を通して保存・参照・実行を行う。

基盤機構の契約候補は次のとおり。

- `TenantID`: 現在のテナントを明示する。
- `TenantScope`: テナントごとの保存領域・読み書きの境界。
- `RequireTenant`: 要求テナントと現在のスコープが一致しなければ拒否する。
- `CrossTenantDenied`: 越境の参照・書込みを、業務上の失敗と混ぜずに拒否する。
- `Audit`: 越境の試行、拒否、現在のテナントを記録する。
- `FailClosed`: テナントを確定できない場合は、空の結果や既定テナントへ落とさない。

保存領域は、テナントごとのrootを分ける。パス結合やシンボリックリンクでroot外へ出ないよう、`os.Root`相当の相対操作を使う。

## 4. Delivery Profileの契約

Profileは、提供形態の名前ではなく、必要な能力と選択するpluginを宣言する。

```json
{
  "schema": "tei.reference.delivery-profile/v0",
  "id": "saas-default",
  "requiredCapabilities": ["tenant-isolation", "audit", "authentication"],
  "plugins": [
    {"id": "tenant-isolation", "version": "1"},
    {"id": "audit", "version": "1"},
    {"id": "authentication", "version": "1"}
  ]
}
```

Profileの検証は次を行う。

- 必須能力が、選択したpluginの能力で満たされるか。
- 同じplugin idの重複や、解決できないversionがないか。
- 能力の欠落、権限の不一致、分離保証の不足をfail closedにする。
- Profile自体を意味の正本にしない。

## 5. 最小PoC

1つの受理済み意味（例: 受注の読み取り）と同じアプリケーションを使い、2つのテナントを用意する。

- テナントAとBで、同じ意味・同じアプリケーションを使う。
- テナントAのスコープから、テナントBの保存領域を参照・書込みできない。
- 越境の試行を`CrossTenantDenied`として拒否し、業務上の「データなし」と混同しない。
- テナントを確定できない要求をfail closedにする。
- 形態別のアプリケーション実装を追加せず、tenant-isolation pluginの選択だけで成立することを確認する。

## 6. 非目標

- 課金・請求・契約管理の実装。
- 全クラウド・全認証基盤への対応。
- 法規制・監査基準への適合証明。
- DB・外部APIを含む完全なテナント分離保証（PoCでは保存領域と参照境界に限定）。

## 7. 未決

- テナント分離をDBの行レベル認可まで強制するか、アプリケーション境界に留めるか。
- 監査証拠の保存先と保存期間。
- pluginの配布・版・失効を、生成器registryとどう揃えるか。
- 個人→SaaS、SIer納品→SaaSの移行手順。
