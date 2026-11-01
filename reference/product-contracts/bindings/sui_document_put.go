package bindings

import (
	"context"

	"tei/internal/core"
	"tei/internal/semantic/capabilityinvoke"
)

type SUIDocumentPut struct{}

func (SUIDocumentPut) Invoke(_ context.Context, id core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	fields, err := inputObject("sui_document_put", input)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	documentID, err := requireString("sui_document_put", fields, "documentId")
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	return capabilityinvoke.BusinessResult{
		Capability: id, Mode: capabilityinvoke.ResultOutput,
		Payload: map[string]any{"documentId": documentID, "version": 1},
	}, nil
}
