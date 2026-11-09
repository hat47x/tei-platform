# Relationの削除・改名・分割に伴う過去参照の表示（#1126）

状態：**変更前のCanonical Worldから過去のRelation参照を取り出す読み取り専用の拡張**。Source、Shared Acceptanceの受理判断、DBスキーマ、DDL、Runtime authorityは変更しない。

## 背景

`tools/internal/changeanalysis`の`Diff`は、Relationの追加・変更・削除を`relation` subjectとして独立に列挙する。しかし`Impact`は後状態だけを辿り、`removed`のsubjectを対象外にしていた。このため、Relationを削除した場合や、改名・分割で旧Relationが消えた場合、旧Relationが指していた端点Data、key属性、Persistence Mapping、利用Patternへの参照が「影響なし」と誤認され得る。候補キー・関数従属性では`ImpactWithBefore`が旧Worldの参照を`previous.*`として保持していたが、Relationは同じ扱いを受けていなかった。

受入条件の「relationのrename・split・deletionで影響表示」に対し、後状態だけの解析では旧参照が欠ける。本拡張はその欠落を埋めるが、Sourceへの書き戻しや物理スキーマへの作用は行わない。

## 機構

`ImpactWithBefore(before, after, changes)`は、`relation`が`removed`または`changed`のとき、変更前のWorldから`relationEdges(before, id)`で次の参照を取り出し、`previous.`を前置して返す。

- 所有Data（`previous.semantic.data.relation`）
- target Data（`previous.semantic.relation.target`）
- keyのsource／target属性（`previous.semantic.relation.key.*`）
- 端点を実現するPersistence Mappingとrelation storage（`previous.persistence.*`）
- そのRelationを明示参照するApplication Pattern（`previous.application.pattern.relation`）

後状態の宣言への参照は従来どおり`semantic.*`で示す。改名・分割は「旧Relationの削除」と「新Relationの追加」として現れるため、旧側は`previous.*`、新側は`semantic.*`の両方を同時に示せる。

判断できない事項は成功へ読み替えず、次の未解決として残す。

- `RELATION_RETRACTION_REQUIRES_REVIEW`：削除・変更の確認事項
- `RELATION_PREVIOUS_DECLARATION_MISSING`：旧Relationの宣言位置・端点を変更前のWorldから特定できない場合
- `RELATION_PREVIOUS_STORAGE_NOT_AUTHORED`：旧Relationの保存方法が端点のMappingからもrelation storageからも導けない場合

## 安全境界

- 本拡張は読み取り専用であり、Relationや規則を正本Sourceから削除しない。物理FK・UNIQUE・Index・Migration、Provider、Runtime authorityを変更しない。
- `previous.*`の辺は**変更前の宣言**を表し、新しいWorldにその参照が存在することを主張しない。
- 旧Worldで参照を解決できたことや、候補キーと物理PK／UNIQUEの属性集合の一致表示は、業務キーの正しさ、Relationの保存方法の妥当性、実DBの制約の存在を証明しない。
- 実業務データ、実行Target、人間の承認・権限の確認は別の段階に残る。

## 試験

参照Workspaceの`sales.order.lines`を対象に、削除のときに旧端点・relation storage・利用Patternの辺が`previous.*`として残ること、後状態だけの`Impact`が旧Relationを知らないこと、cardinality変更で新側と旧側の両方が示されること、解決できない旧Relation IDを「影響なし」にしないこと、解析が決定的で正本Worldを変更しないことを検証する。
