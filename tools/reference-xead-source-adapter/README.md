# Xead / Xeaf Source Adapter Reference

GitHub #950のsource-faithful adapter候補です。

`Observe`はXMLをローカル観測用の`ObservedDocument`へ読み込みます。未知要素や未解釈属性も捨てません。Observed Documentには元sourceの属性値やテキストを含み得るため、public Evidenceとして保存しません。

`PublicReceipt`は、artifact digest、element / attribute名の集計、locator digest、source fact種別、authority limitだけを残します。元sourceの属性値やテキストは出力しません。

独自fixtureのテストでは、同一bytesの決定性、未知constructの保持、公開receiptからのsource値除去、legacy key系source factのauthority limitを確認します。

Repository外のexact `MfgOrder.xead` / `MfgOrder.xeaf`にも適用し、既存EvidenceのSHA-256と主要件数が一致すること、全12,771 elementが既知constructとしてaccountingされることを確認しました。原model本体やObserved DocumentはRepositoryへ追加していません。

この実装はproduction ImporterでもCanonical authorityでもありません。Plugin登録境界はISSUE-000080 / GitHub #951へ分離しています。
