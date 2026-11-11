package management

// FindRow returns the row for one plugin id. It is used by the CLI's detail
// view, so the selection rule (exact id, no prefix match) lives beside the
// derivation rather than in the adapter.
func FindRow(report Report, pluginID string) (Row, bool) {
	for _, row := range report.Rows {
		if row.PluginID == pluginID {
			return row, true
		}
	}
	return Row{}, false
}

// DetailMessage is the one-sentence explanation of a row. A reason code is
// never turned into a permission claim here.
func DetailMessage(row Row) string {
	if len(row.ReasonCodes) == 0 {
		return row.PluginID + " は宣言・環境選択・解決のいずれも揃っています。"
	}
	return row.PluginID + " には次の理由コードがあります: " + joinCodes(row.ReasonCodes)
}

func joinCodes(codes []string) string {
	out := ""
	for i, code := range codes {
		if i > 0 {
			out += ", "
		}
		out += code
	}
	return out
}
