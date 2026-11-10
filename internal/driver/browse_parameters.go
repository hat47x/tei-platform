package driver

import (
	"fmt"

	"tei/internal/application"
)

// browseQueryParametersは、browse Patternが宣言した演算子・page-sizeの受け口を実行時parameterへ反映する。
//
// Pattern.PageSizeを正本とし、呼出側がQueryのlimit用inputへ別の値を渡すことを許さない。
// 演算子はPatternが宣言した集合の外を拒否し、Query側の分岐が未知の値を黙って既定動作へ
// 落とす余地を作らない。offsetは呼出側が選ぶ頁位置で、省略時は先頭(0)とする。
// native SQLは解析も書換えもしない。入力の検査はExecutorを呼ぶ前に行い、失敗は空結果にしない。
func browseQueryParameters(pattern application.Pattern, parameters map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(parameters)+2)
	for name, value := range parameters {
		out[name] = value
	}

	for _, filter := range pattern.Filters {
		if filter.OperatorParameter == "" {
			continue
		}
		raw, ok := out[filter.OperatorParameter]
		if !ok {
			return nil, fmt.Errorf("browse Pattern %sのfilter %sの演算子parameter %sが指定されていません", pattern.ID, filter.Attribute, filter.OperatorParameter)
		}
		operator, ok := raw.(string)
		if !ok || !containsString(filter.Operators, operator) {
			return nil, fmt.Errorf("browse Pattern %sのfilter %sは演算子 %v を宣言していません(宣言=%v)", pattern.ID, filter.Attribute, raw, filter.Operators)
		}
	}

	if name := pattern.PageSizeParameter; name != "" {
		if raw, ok := out[name]; ok && !sameInteger(raw, pattern.PageSize) {
			return nil, fmt.Errorf("browse Pattern %sのpage-size parameter %sはPatternの宣言値 %d と異なる値で指定されました", pattern.ID, name, pattern.PageSize)
		}
		out[name] = int64(pattern.PageSize)
	}
	if name := pattern.PageOffsetParameter; name != "" {
		raw, ok := out[name]
		if !ok {
			out[name] = int64(0)
		} else {
			offset, isInteger := integerValue(raw)
			if !isInteger || offset < 0 {
				return nil, fmt.Errorf("browse Pattern %sのpage offset parameter %sは0以上の整数でなければなりません", pattern.ID, name)
			}
			out[name] = offset
		}
	}
	return out, nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func integerValue(raw any) (int64, bool) {
	switch v := raw.(type) {
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	}
	return 0, false
}

func sameInteger(raw any, want int) bool {
	got, ok := integerValue(raw)
	return ok && got == int64(want)
}
