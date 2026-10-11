# Xead cross-view Reference

GitHub #1031 / RMT-4 のread-only cross-view Referenceです。

Xead sourceとexact transcription contractを受け取り、Data・Function・Businessをまたぐsource-backed relationとderived potential pathを構成します。

既定出力はpublic-safeなaggregate accountingだけです。legacy ID値、source本文、個別relation tokenは出しません。

```bash
go run ./tools/reference-xead-cross-view \
  -input /path/to/local-model.xead \
  -contract reference/xead-source-transcription-contract.synthetic.json
```

個別source tokenとrelationをローカル調査で確認する場合だけ `-local-projection` を付けます。この出力はpublic Reference Model Evidenceへ保存しません。

```bash
go run ./tools/reference-xead-cross-view \
  -input /path/to/model.xead \
  -contract reference/xead-source-transcription-contract.synthetic.json \
  -local-projection
```

cross-viewはBusiness Action固有CRUD、actual Runtime effect、transaction、authorization、business ownershipを推定しません。Function-level IOTable CRUDからActionへ連結したedgeはpotential operationに限定します。
