package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"tei/internal/plugin/transform"
)

const (
	engineRequestSchema  = "tei.reference.transform-engine-request/v1"
	engineResponseSchema = "tei.reference.transform-engine-response/v1"
	requestEnvelopeSlack = 1 << 20
)

type engineRequest struct {
	Schema string      `json:"schema"`
	Inputs []wireInput `json:"inputs"`
}

type wireInput struct {
	Name  string    `json:"name"`
	Value wireValue `json:"value"`
}

type wireField struct {
	Name  string    `json:"name"`
	Value wireValue `json:"value"`
}

type wireValue struct {
	Kind        transform.ValueKind `json:"kind"`
	Bool        *bool               `json:"bool,omitempty"`
	Number      *string             `json:"number,omitempty"`
	String      *string             `json:"string,omitempty"`
	Array       *[]wireValue        `json:"array,omitempty"`
	Object      *[]wireField        `json:"object,omitempty"`
	BytesBase64 *string             `json:"bytesBase64,omitempty"`

	present wireValueFieldPresence `json:"-"`
}

type wireValueFieldPresence uint8

const (
	wireValueBoolPresent wireValueFieldPresence = 1 << iota
	wireValueNumberPresent
	wireValueStringPresent
	wireValueArrayPresent
	wireValueObjectPresent
	wireValueBytesPresent
)

func (w *wireValue) UnmarshalJSON(raw []byte) error {
	type alias wireValue
	var decoded alias
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("wire value contains multiple JSON values")
		}
		return err
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	var present wireValueFieldPresence
	if _, ok := fields["bool"]; ok {
		present |= wireValueBoolPresent
	}
	if _, ok := fields["number"]; ok {
		present |= wireValueNumberPresent
	}
	if _, ok := fields["string"]; ok {
		present |= wireValueStringPresent
	}
	if _, ok := fields["array"]; ok {
		present |= wireValueArrayPresent
	}
	if _, ok := fields["object"]; ok {
		present |= wireValueObjectPresent
	}
	if _, ok := fields["bytesBase64"]; ok {
		present |= wireValueBytesPresent
	}
	*w = wireValue(decoded)
	w.present = present
	return nil
}

type engineResponse struct {
	Schema string    `json:"schema"`
	Output wireValue `json:"output"`
	Steps  int       `json:"steps"`
}

func main() {
	sourcePath := flag.String("source", "", "path to an exact deterministic transform Source")
	flag.Parse()
	if strings.TrimSpace(*sourcePath) == "" {
		fmt.Fprintln(os.Stderr, "-source is required")
		os.Exit(2)
	}

	sourceRaw, err := readBoundedFile(*sourcePath, transform.MaxSerializedSourceBytes)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	source, err := transform.ParseSourceJSON(sourceRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	requestRaw, err := readBounded(os.Stdin, requestLimit(source))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	response, err := execute(source, requestRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func execute(source transform.Source, requestRaw []byte) (engineResponse, error) {
	request, err := decodeRequest(requestRaw, source.Limits)
	if err != nil {
		return engineResponse{}, err
	}
	inputs := make(map[string]transform.Value, len(request.Inputs))
	budget := wireBudget{}
	for _, input := range request.Inputs {
		if strings.TrimSpace(input.Name) == "" {
			return engineResponse{}, fmt.Errorf("transform engine input name is required")
		}
		if _, exists := inputs[input.Name]; exists {
			return engineResponse{}, fmt.Errorf("transform engine input %q is declared more than once", input.Name)
		}
		if err := budget.addBytes(len(input.Name), source.Limits); err != nil {
			return engineResponse{}, fmt.Errorf("transform engine input %q: %w", input.Name, err)
		}
		value, err := decodeWireValue(input.Value, source.Limits, 0, &budget)
		if err != nil {
			return engineResponse{}, fmt.Errorf("transform engine input %q: %w", input.Name, err)
		}
		inputs[input.Name] = value
	}

	result, err := transform.Evaluate(source, inputs)
	if err != nil {
		return engineResponse{}, err
	}
	output, err := encodeWireValue(result.Output, source.Limits, 0)
	if err != nil {
		return engineResponse{}, fmt.Errorf("encode transform engine output: %w", err)
	}
	return engineResponse{
		Schema: engineResponseSchema,
		Output: output,
		Steps:  result.Steps,
	}, nil
}

func decodeRequest(raw []byte, limits transform.Limits) (engineRequest, error) {
	if len(raw) == 0 {
		return engineRequest{}, fmt.Errorf("transform engine request is empty")
	}
	if !utf8.Valid(raw) {
		return engineRequest{}, fmt.Errorf("transform engine request is not valid UTF-8")
	}
	if err := validateJSONStructure(raw, limits.MaxNestingDepth+8); err != nil {
		return engineRequest{}, fmt.Errorf("transform engine request JSON: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var request engineRequest
	if err := decoder.Decode(&request); err != nil {
		return engineRequest{}, fmt.Errorf("decode transform engine request: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return engineRequest{}, fmt.Errorf("transform engine request contains multiple values")
		}
		return engineRequest{}, fmt.Errorf("decode trailing transform engine request content: %w", err)
	}
	if request.Schema != engineRequestSchema {
		return engineRequest{}, fmt.Errorf("unsupported transform engine request schema %q", request.Schema)
	}
	if len(request.Inputs) > limits.MaxCollection {
		return engineRequest{}, fmt.Errorf("transform engine request inputs exceed maxCollection")
	}
	return request, nil
}

type wireBudget struct {
	nodes int
	bytes int
}

func (b *wireBudget) addNode(limits transform.Limits) error {
	b.nodes++
	if b.nodes > limits.MaxInputNodes {
		return fmt.Errorf("wire values exceed maxInputNodes")
	}
	return nil
}

func (b *wireBudget) addBytes(count int, limits transform.Limits) error {
	if count < 0 || count > limits.MaxInputBytes-b.bytes {
		return fmt.Errorf("wire values exceed maxInputBytes")
	}
	b.bytes += count
	return nil
}

func decodeWireValue(w wireValue, limits transform.Limits, depth int, budget *wireBudget) (transform.Value, error) {
	if depth > limits.MaxNestingDepth {
		return transform.Value{}, fmt.Errorf("wire value exceeds maxNestingDepth")
	}
	if err := budget.addNode(limits); err != nil {
		return transform.Value{}, err
	}
	switch w.Kind {
	case transform.ValueNull:
		if wireFieldCount(w) != 0 {
			return transform.Value{}, fmt.Errorf("null wire value contains another-kind field")
		}
		return transform.Null(), nil
	case transform.ValueBool:
		if w.Bool == nil || wireFieldCount(w) != 1 {
			return transform.Value{}, fmt.Errorf("bool wire value requires only bool")
		}
		return transform.Bool(*w.Bool), nil
	case transform.ValueNumber:
		if w.Number == nil || wireFieldCount(w) != 1 {
			return transform.Value{}, fmt.Errorf("number wire value requires only number")
		}
		if err := budget.addBytes(len(*w.Number), limits); err != nil {
			return transform.Value{}, err
		}
		return transform.Number(*w.Number), nil
	case transform.ValueString:
		if w.String == nil || wireFieldCount(w) != 1 {
			return transform.Value{}, fmt.Errorf("string wire value requires only string")
		}
		if err := budget.addBytes(len(*w.String), limits); err != nil {
			return transform.Value{}, err
		}
		return transform.String(*w.String), nil
	case transform.ValueBytes:
		if w.BytesBase64 == nil || wireFieldCount(w) != 1 {
			return transform.Value{}, fmt.Errorf("bytes wire value requires only bytesBase64")
		}
		decodedLen := strictBase64DecodedLength(*w.BytesBase64)
		if decodedLen > limits.MaxInputBytes-budget.bytes {
			return transform.Value{}, fmt.Errorf("wire values exceed maxInputBytes")
		}
		raw, err := base64.StdEncoding.Strict().DecodeString(*w.BytesBase64)
		if err != nil {
			return transform.Value{}, fmt.Errorf("bytesBase64: %w", err)
		}
		if err := budget.addBytes(len(raw), limits); err != nil {
			return transform.Value{}, err
		}
		return transform.Value{Kind: transform.ValueBytes, Bytes: raw}, nil
	case transform.ValueArray:
		if w.Array == nil || wireFieldCount(w) != 1 {
			return transform.Value{}, fmt.Errorf("array wire value requires only array")
		}
		if len(*w.Array) > limits.MaxCollection {
			return transform.Value{}, fmt.Errorf("wire array exceeds maxCollection")
		}
		values := make([]transform.Value, len(*w.Array))
		for i := range *w.Array {
			value, err := decodeWireValue((*w.Array)[i], limits, depth+1, budget)
			if err != nil {
				return transform.Value{}, fmt.Errorf("wire array item %d: %w", i, err)
			}
			values[i] = value
		}
		return transform.Value{Kind: transform.ValueArray, Array: values}, nil
	case transform.ValueObject:
		if w.Object == nil || wireFieldCount(w) != 1 {
			return transform.Value{}, fmt.Errorf("object wire value requires only object")
		}
		if len(*w.Object) > limits.MaxCollection {
			return transform.Value{}, fmt.Errorf("wire object exceeds maxCollection")
		}
		fields := make([]transform.Field, len(*w.Object))
		seen := map[string]struct{}{}
		for i, field := range *w.Object {
			if strings.TrimSpace(field.Name) == "" {
				return transform.Value{}, fmt.Errorf("wire object field name is required")
			}
			if _, exists := seen[field.Name]; exists {
				return transform.Value{}, fmt.Errorf("wire object field %q is declared more than once", field.Name)
			}
			seen[field.Name] = struct{}{}
			if err := budget.addBytes(len(field.Name), limits); err != nil {
				return transform.Value{}, err
			}
			value, err := decodeWireValue(field.Value, limits, depth+1, budget)
			if err != nil {
				return transform.Value{}, fmt.Errorf("wire object field %q: %w", field.Name, err)
			}
			fields[i] = transform.Field{Name: field.Name, Value: value}
		}
		return transform.Value{Kind: transform.ValueObject, Object: fields}, nil
	default:
		return transform.Value{}, fmt.Errorf("unsupported wire value kind %q", w.Kind)
	}
}

func encodeWireValue(value transform.Value, limits transform.Limits, depth int) (wireValue, error) {
	if depth > limits.MaxNestingDepth {
		return wireValue{}, fmt.Errorf("output wire value exceeds maxNestingDepth")
	}
	switch value.Kind {
	case transform.ValueNull:
		return wireValue{Kind: value.Kind}, nil
	case transform.ValueBool:
		v := value.Bool
		return wireValue{Kind: value.Kind, Bool: &v}, nil
	case transform.ValueNumber:
		v := value.Number
		return wireValue{Kind: value.Kind, Number: &v}, nil
	case transform.ValueString:
		v := value.String
		return wireValue{Kind: value.Kind, String: &v}, nil
	case transform.ValueBytes:
		v := base64.StdEncoding.EncodeToString(value.Bytes)
		return wireValue{Kind: value.Kind, BytesBase64: &v}, nil
	case transform.ValueArray:
		if len(value.Array) > limits.MaxCollection {
			return wireValue{}, fmt.Errorf("output array exceeds maxCollection")
		}
		items := make([]wireValue, len(value.Array))
		for i := range value.Array {
			item, err := encodeWireValue(value.Array[i], limits, depth+1)
			if err != nil {
				return wireValue{}, err
			}
			items[i] = item
		}
		return wireValue{Kind: value.Kind, Array: &items}, nil
	case transform.ValueObject:
		if len(value.Object) > limits.MaxCollection {
			return wireValue{}, fmt.Errorf("output object exceeds maxCollection")
		}
		fields := make([]wireField, len(value.Object))
		for i := range value.Object {
			wire, err := encodeWireValue(value.Object[i].Value, limits, depth+1)
			if err != nil {
				return wireValue{}, err
			}
			fields[i] = wireField{Name: value.Object[i].Name, Value: wire}
		}
		return wireValue{Kind: value.Kind, Object: &fields}, nil
	default:
		return wireValue{}, fmt.Errorf("unsupported output value kind %q", value.Kind)
	}
}

func strictBase64DecodedLength(encoded string) int {
	decodedLen := base64.StdEncoding.DecodedLen(len(encoded))
	if strings.HasSuffix(encoded, "==") {
		decodedLen -= 2
	} else if strings.HasSuffix(encoded, "=") {
		decodedLen--
	}
	if decodedLen < 0 {
		return 0
	}
	return decodedLen
}

func wireFieldCount(w wireValue) int {
	present := w.present
	if w.Bool != nil {
		present |= wireValueBoolPresent
	}
	if w.Number != nil {
		present |= wireValueNumberPresent
	}
	if w.String != nil {
		present |= wireValueStringPresent
	}
	if w.Array != nil {
		present |= wireValueArrayPresent
	}
	if w.Object != nil {
		present |= wireValueObjectPresent
	}
	if w.BytesBase64 != nil {
		present |= wireValueBytesPresent
	}
	count := 0
	for present != 0 {
		count += int(present & 1)
		present >>= 1
	}
	return count
}

func requestLimit(source transform.Source) int64 {
	// Typed wire JSON can be larger than the semantic input bytes because of
	// field names, structural metadata, and base64 expansion. Keep a separate
	// Host-owned envelope limit without changing Source-declared input limits.
	limit := int64(source.Limits.MaxInputBytes)*4 + requestEnvelopeSlack
	max := int64((^uint(0)) >> 1)
	if limit > max {
		return max
	}
	return limit
}

func readBoundedFile(path string, maxBytes int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return readBounded(file, int64(maxBytes))
}

func readBounded(reader io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return nil, fmt.Errorf("byte limit must be positive")
	}
	raw, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxBytes {
		return nil, fmt.Errorf("input exceeds byte limit")
	}
	return raw, nil
}

func validateJSONStructure(raw []byte, maxDepth int) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := walkJSONValue(decoder, 0, maxDepth); err != nil {
		return err
	}
	if token, err := decoder.Token(); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("JSON contains multiple values after %v", token)
	}
	return nil
}

func walkJSONValue(decoder *json.Decoder, depth, maxDepth int) error {
	if depth > maxDepth {
		return fmt.Errorf("JSON nesting exceeds limit")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]struct{}{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("JSON object key is not a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("JSON object key %q is declared more than once", key)
			}
			seen[key] = struct{}{}
			if err := walkJSONValue(decoder, depth+1, maxDepth); err != nil {
				return err
			}
		}
		closeToken, err := decoder.Token()
		if err != nil {
			return err
		}
		if closeToken != json.Delim('}') {
			return fmt.Errorf("JSON object is not terminated")
		}
	case '[':
		for decoder.More() {
			if err := walkJSONValue(decoder, depth+1, maxDepth); err != nil {
				return err
			}
		}
		closeToken, err := decoder.Token()
		if err != nil {
			return err
		}
		if closeToken != json.Delim(']') {
			return fmt.Errorf("JSON array is not terminated")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
	return nil
}

