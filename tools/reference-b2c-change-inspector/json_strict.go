package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// inspectJSONKeys rejects repeated object keys at every depth before the
// regular typed decoder sees the input. JSON field names that differ only
// by case are also rejected: encoding/json accepts case-insensitive matches
// for struct fields, which otherwise permits shadowed control flags.
func inspectJSONKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := walkJSONObjectKeys(dec); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return fmt.Errorf("trailing JSON value: %w", err)
	}
	return nil
}

func walkJSONObjectKeys(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		keys := map[string]bool{}
		for dec.More() {
			keyToken, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is not a string")
			}
			canonical := strings.ToLower(key)
			if keys[canonical] {
				return fmt.Errorf("duplicate or case-shadowed JSON key %q", key)
			}
			keys[canonical] = true
			if err := walkJSONObjectKeys(dec); err != nil {
				return err
			}
		}
		_, err := dec.Token() // closing brace, checked by Decoder
		return err
	case '[':
		for dec.More() {
			if err := walkJSONObjectKeys(dec); err != nil {
				return err
			}
		}
		_, err := dec.Token() // closing bracket, checked by Decoder
		return err
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
}
