package semantic

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// Fingerprintは、Semanticの意味だけを正規化して指紋にする。
// Application・Package・投影は含めない。Packageの取り外しで意味が変わらないことを照合するために使う。
func Fingerprint(namespaces []Namespace) (string, error) {
	normalized := make([]Namespace, 0, len(namespaces))
	for _, namespace := range namespaces {
		normalized = append(normalized, normalizeNamespace(namespace))
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].ID < normalized[j].ID })
	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func normalizeNamespace(namespace Namespace) Namespace {
	out := namespace
	out.Types = append([]Type(nil), namespace.Types...)
	out.Data = append([]Data(nil), namespace.Data...)
	out.Capabilities = append([]Capability(nil), namespace.Capabilities...)
	out.Validations = append([]Validation(nil), namespace.Validations...)
	sort.Slice(out.Types, func(i, j int) bool { return out.Types[i].ID < out.Types[j].ID })
	sort.Slice(out.Data, func(i, j int) bool { return out.Data[i].ID < out.Data[j].ID })
	sort.Slice(out.Capabilities, func(i, j int) bool { return out.Capabilities[i].ID < out.Capabilities[j].ID })
	sort.Slice(out.Validations, func(i, j int) bool { return out.Validations[i].ID < out.Validations[j].ID })
	for i := range out.Data {
		out.Data[i].Relations = append([]Relation(nil), namespace.Data[i].Relations...)
		sort.Slice(out.Data[i].Relations, func(a, b int) bool {
			return out.Data[i].Relations[a].ID < out.Data[i].Relations[b].ID
		})
	}
	return out
}
