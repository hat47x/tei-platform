# ISSUE-000003: 既存認証仕様を保持したCompatibility Referenceを設計する

- Status: Proposed

## 背景

業務システムの移行では、標準的なID / password認証であっても、ユーザー無効化、有効期限、ログイン失敗回数、ロック、組織属性など、システム固有の制御が長年積み重なっています。

TEIが既存システム移行に使えるためには、最初に標準認証へ作り替えることを要求せず、既存仕様を保持したままCapabilityへ接続できる必要があります。

## 関連するProduct Requirement

- RQ-008 Capability単位の置換
- RQ-009 既存認証・認可仕様の保持
- RQ-015 AIによる既存知識の探索
- RQ-017 二重管理の最小化
- RQ-021 Reference / Conformance

## 検討するReference

小さい既存システム風の認証モデルを用意し、少なくとも次を含めます。

- user ID
- password credential reference
- disabled state
- validity period
- login failure count
- lock state
- organization / role等のauthorization fact
- system-specific ruleを一つ以上

値そのものをTEI標準schemaへ移し替えるのではなく、既存field / functionをCapability contractへMappingする形を優先します。

## 確かめること

- `authenticate`等のCapability contractと既存実装を分離できる
- field / function mappingで既存modelを利用できる
- custom typeやplugin hookへ逃がすべき箇所を明示できる
- builtin / generated / plugin / existing implementationをCapability単位で差し替えられる
- AI / Editorが既存Mappingを読み取り、同じ質問を繰り返さずに不足を確認できる土台がある
- 認証仕様を変えずに移行した後、段階的に標準実装へ置き換えられる

## 非目標

- 一つの標準User schemaへ全システムを統一すること
- production用IdP製品を内製すること
- password secretそのものをCanonical definitionへ保持すること

## 完了条件

「既存認証は複雑だからTEI標準へ作り替える必要がある」ではなく、「既存仕様を保ったままTEIのCapability境界へ接続できる」ことをReferenceとConformanceで説明できる設計が成立すること。
