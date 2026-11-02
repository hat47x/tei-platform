package bindings

import "fmt"

func inputObject(name string, input any) (map[string]any, error) {
	fields, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s input must be an object", name)
	}
	return fields, nil
}

func requireString(name string, fields map[string]any, key string) (string, error) {
	value, _ := fields[key].(string)
	if value == "" {
		return "", fmt.Errorf("%s requires %s", name, key)
	}
	return value, nil
}
