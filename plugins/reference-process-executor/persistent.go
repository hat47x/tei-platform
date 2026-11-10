package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
)

// This bounded framing and hold control are reference-only lifecycle probes,
// not a new production process ABI. The request itself retains the existing ABI.
type persistentFrame struct {
	Kind      string          `json:"kind"`
	RequestID string          `json:"requestId,omitempty"`
	Request   json.RawMessage `json:"request,omitempty"`
	Hold      bool            `json:"hold,omitempty"`
	Revision  string          `json:"revision,omitempty"`
	PID       int             `json:"pid,omitempty"`
	Response  json.RawMessage `json:"response,omitempty"`
}

func servePersistent(input io.Reader, output io.Writer, args []string) error {
	if hasExactArg(args, "--tei-reference-readiness-fail") {
		return errors.New("reference readiness refused")
	}
	encode := json.NewEncoder(output)
	if err := encode.Encode(persistentFrame{Kind: "ready", Revision: referenceRevision, PID: os.Getpid()}); err != nil {
		return err
	}
	scan := bufio.NewScanner(input)
	scan.Buffer(make([]byte, 4096), 64<<10)
	for scan.Scan() {
		var frame persistentFrame
		if err := json.Unmarshal(scan.Bytes(), &frame); err != nil || frame.Kind != "invoke" || frame.RequestID == "" {
			return errors.New("invalid reference frame")
		}
		var req processRequest
		if err := json.Unmarshal(frame.Request, &req); err != nil || req.RequestID != frame.RequestID {
			return errors.New("reference request correlation mismatch")
		}
		if frame.Hold {
			if err := encode.Encode(persistentFrame{Kind: "started", RequestID: frame.RequestID}); err != nil {
				return err
			}
			if !scan.Scan() {
				return errors.New("held reference request lost its continuation")
			}
			var release persistentFrame
			if err := json.Unmarshal(scan.Bytes(), &release); err != nil || release.Kind != "continue" || release.RequestID != frame.RequestID {
				return errors.New("invalid reference continuation")
			}
		}
		var response bytes.Buffer
		if err := serve(bytes.NewReader(frame.Request), &response, args); err != nil {
			return err
		}
		if err := encode.Encode(persistentFrame{Kind: "result", RequestID: frame.RequestID, Response: json.RawMessage(response.Bytes())}); err != nil {
			return err
		}
	}
	return scan.Err()
}
