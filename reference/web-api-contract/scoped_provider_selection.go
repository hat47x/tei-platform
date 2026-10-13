package webapicontract

import "reflect"

// RequireScopedCanonicalExecutor is the composition boundary for a dynamic
// Provider discovered by a Web/CLI/Desktop Host. A legacy BrowseQueryExecutor
// is NOT accepted as a scoped Provider simply because it can execute SQL.
//
// This runtime interface check is only a guard against accidental fallback;
// successful matching never proves actual row-level enforcement.
func RequireScopedCanonicalExecutor(candidate any) (ScopedCanonicalExecutor, error) {
	provider, ok := candidate.(ScopedCanonicalExecutor)
	if !ok || provider == nil {
		return nil, ErrScopedQueryUnavailable
	}
	value := reflect.ValueOf(provider)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if value.IsNil() {
			return nil, ErrScopedQueryUnavailable
		}
	}
	return provider, nil
}
