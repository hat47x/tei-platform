# ISSUE-000078: 外部Reference Modelの転写適合性を検証する

- 状態: In Progress
- 性格: Product Value / Reference / Conformance / migration pressure
- 主担当要求: RQ-001 / 002 / 003 / 004 / 010 / 011 / 012 / 013 / 021 / 022 / 025 / 026 / 029 / 031 / 032 / 033

## 目的

成熟した外部Reference ModelをTEIの意味モデルへ移すとき、TEIの表現力、Source traceability、Target independence、Business cross-view、Runtime authority、AI再入場性が保たれるかを検証する。特定の外部製品との互換性を実装する作業ではない。

## 公開境界

外部モデル、手引、source snapshot、その抜粋や変換物は、この公開Repositoryへ追加しない。公開利用条件が確認され、個別審査を終えるまで、外部資料由来の詳細な分析・転写記録も公開対象にしない。

## 完了条件

- source constructとTEI上の対応・非対応・未解決点を区別できる。
- source traceabilityとTEI側のcanonical identityを分離できる。
- business meaning、interaction、target/runtime realization、external actualityの境界を保持できる。
- 合成または独自作成した入力だけで確認した点と、実Provider・実環境で確認した点を分けて記録する。
- 転写・実行・投影できない要素と残る不確実性を明示する。

## 防ぐ失敗

- source constructや暗黙前提を根拠なく捨てる、または推測で埋める。
- source固有のID・分類・画面操作をTEIのcanonical meaningへそのまま昇格する。
- Referenceの読み込み成功を、元Runtimeとの互換性や実環境での動作証拠として扱う。
- 利用条件が未確認の外部資料やその変換物を公開する。

詳しい作業メモと転写資料はprivate resources側に保管する。ここでは公開可能な目的、境界、完了条件のみを管理する。
