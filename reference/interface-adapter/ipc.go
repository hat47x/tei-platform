package interfaceadapter

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
)

const maxIPCFrame = 1 << 20

// IPCServer is the Desktop transport adapter. The Principal is fixed by the
// host when the server is composed; the message cannot carry or change it.
// This is a typed local IPC boundary, not Wails or WebView.
type IPCServer struct {
	Core      Core
	Principal Principal
	mu        sync.Mutex
	invoked   int
}

// ServeConn reads length-prefixed JSON requests and writes typed results until
// the peer closes the connection.
func (s *IPCServer) ServeConn(conn net.Conn) error {
	defer conn.Close()
	for {
		var request Request
		if err := readIPCFrame(conn, &request); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		s.mu.Lock()
		s.invoked++
		s.mu.Unlock()
		result := s.Core.Invoke(context.Background(), s.Principal, request)
		if err := writeIPCFrame(conn, result); err != nil {
			return err
		}
	}
}

// Invocations is the number of messages that reached Core.
func (s *IPCServer) Invocations() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.invoked
}

// IPCAdapter is the client side of the Desktop adapter.
type IPCAdapter struct {
	Conn net.Conn
}

// Execute sends one typed request and reads the typed result.
func (a IPCAdapter) Execute(request Request) (Result, error) {
	if err := writeIPCFrame(a.Conn, request); err != nil {
		return Result{}, err
	}
	var result Result
	if err := readIPCFrame(a.Conn, &result); err != nil {
		return Result{}, err
	}
	return result, nil
}

func writeIPCFrame(w io.Writer, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(encoded) > maxIPCFrame {
		return fmt.Errorf("interface adapter: IPC frame is too large")
	}
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(encoded)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err = w.Write(encoded)
	return err
}

func readIPCFrame(r io.Reader, value any) error {
	header := make([]byte, 4)
	if _, err := io.ReadFull(r, header); err != nil {
		return err
	}
	length := int(binary.BigEndian.Uint32(header))
	if length <= 0 || length > maxIPCFrame {
		return fmt.Errorf("interface adapter: invalid IPC frame length %d", length)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return err
	}
	return json.Unmarshal(payload, value)
}
