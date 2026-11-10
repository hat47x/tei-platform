package compositionprocessbridge

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"sync"
	"time"

	pruntime "tei/internal/plugin/runtime"
)

type workerFrame struct {
	Kind      string          `json:"kind"`
	RequestID string          `json:"requestId,omitempty"`
	Request   json.RawMessage `json:"request,omitempty"`
	Hold      bool            `json:"hold,omitempty"`
	Revision  string          `json:"revision,omitempty"`
	PID       int             `json:"pid,omitempty"`
	Response  json.RawMessage `json:"response,omitempty"`
}
type workerEvent struct {
	frame workerFrame
	err   error
}

// A serial reference transport owns one long-lived child. A cancelled exchange
// retires the entire stream: unread replies can never become another call's result.
// This adapter grants no business authority and does not retry accepted intents.
type persistentWorker struct {
	cmd      *exec.Cmd
	in       io.WriteCloser
	events   chan workerEvent
	done     chan struct{}
	stopScan chan struct{}
	gate     chan struct{}
	mu       sync.Mutex
	exitErr  error
	stopOnce sync.Once
}

func startPersistentWorker(ctx context.Context, path, expectedDigest, revision string, failReady bool) (*persistentWorker, error) {
	args := []string{persistentArgvMarker, "--tei-reference-persistent"}
	if failReady {
		args = append(args, "--tei-reference-readiness-fail")
	}
	cmd, closeImage, err := preparePersistentCommand(ctx, path, expectedDigest, args)
	if err != nil {
		return nil, err
	}
	defer closeImage()
	return startPersistentCommand(ctx, cmd, revision)
}

func startPersistentCommand(ctx context.Context, cmd *exec.Cmd, revision string) (*persistentWorker, error) {
	w := &persistentWorker{cmd: cmd, events: make(chan workerEvent, 2), done: make(chan struct{}), stopScan: make(chan struct{}), gate: make(chan struct{}, 1)}
	var err error
	w.in, err = w.cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := w.cmd.StdoutPipe()
	if err != nil {
		_ = w.in.Close()
		return nil, err
	}
	// Never retain child stderr, which could contain request data.
	w.cmd.Stderr = io.Discard
	if err := w.cmd.Start(); err != nil {
		_ = w.in.Close()
		_ = out.Close()
		return nil, err
	}
	w.gate <- struct{}{}
	go func() {
		scan := bufio.NewScanner(out)
		scan.Buffer(make([]byte, 4096), 64<<10)
		for scan.Scan() {
			var frame workerFrame
			err := json.Unmarshal(scan.Bytes(), &frame)
			select {
			case w.events <- workerEvent{frame, err}:
			case <-w.stopScan:
				return
			}
			if err != nil {
				return
			}
		}
		err := scan.Err()
		if err == nil {
			err = io.EOF
		}
		select {
		case w.events <- workerEvent{err: err}:
		case <-w.stopScan:
		}
	}()
	go func() { err := w.cmd.Wait(); w.mu.Lock(); w.exitErr = err; w.mu.Unlock(); close(w.done) }()
	frame, err := w.receive(ctx)
	if err != nil || frame.Kind != "ready" || frame.Revision != revision || frame.PID != w.cmd.Process.Pid {
		return nil, errors.Join(errors.New("reference process failed readiness"), w.stop())
	}
	return w, nil
}
func (w *persistentWorker) alive() bool {
	select {
	case <-w.done:
		return false
	default:
		return true
	}
}
func (w *persistentWorker) receive(ctx context.Context) (workerFrame, error) {
	select {
	case event := <-w.events:
		return event.frame, event.err
	case <-ctx.Done():
		return workerFrame{}, ctx.Err()
	case <-w.done:
		return workerFrame{}, errors.New("reference worker exited")
	}
}
func (w *persistentWorker) invoke(ctx context.Context, req pruntime.Request, hold bool, started func(), release <-chan struct{}) (pruntime.Response, error) {
	select {
	case <-w.gate:
	case <-ctx.Done():
		return pruntime.Response{}, ctx.Err()
	}
	defer func() { w.gate <- struct{}{} }()
	if !w.alive() {
		return pruntime.Response{}, errors.New("reference worker unavailable")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return pruntime.Response{}, err
	}
	if len(body) > 48<<10 {
		return pruntime.Response{}, errors.New("reference request exceeds frame budget")
	}
	if err := json.NewEncoder(w.in).Encode(workerFrame{Kind: "invoke", RequestID: req.RequestID, Request: body, Hold: hold}); err != nil {
		_ = w.stop()
		return pruntime.Response{}, errors.New("reference stream write failed")
	}
	frame, err := w.receive(ctx)
	if err == nil && hold {
		if frame.Kind != "started" || frame.RequestID != req.RequestID {
			err = errors.New("invalid reference start frame")
		} else {
			if started != nil {
				started()
			}
			select {
			case <-release:
				err = json.NewEncoder(w.in).Encode(workerFrame{Kind: "continue", RequestID: req.RequestID})
			case <-ctx.Done():
				err = ctx.Err()
			}
			if err == nil {
				frame, err = w.receive(ctx)
			}
		}
	}
	if err != nil {
		_ = w.stop()
		return pruntime.Response{}, err
	}
	if frame.Kind != "result" || frame.RequestID != req.RequestID {
		_ = w.stop()
		return pruntime.Response{}, errors.New("reference response correlation mismatch")
	}
	var response pruntime.Response
	if err := json.Unmarshal(frame.Response, &response); err != nil || response.RequestID != req.RequestID || response.ProtocolVersion != req.ProtocolVersion || response.CapabilityID != req.CapabilityID || response.DescriptorFingerprint != req.DescriptorFingerprint || response.Plugin != req.Plugin {
		_ = w.stop()
		return pruntime.Response{}, errors.New("reference response identity mismatch")
	}
	return response, nil
}
func (w *persistentWorker) stop() error {
	w.stopOnce.Do(func() { _ = w.in.Close(); close(w.stopScan) })
	select {
	case <-w.done:
	case <-time.After(time.Second):
		_ = w.cmd.Process.Kill()
		select {
		case <-w.done:
		case <-time.After(2 * time.Second):
			return errors.New("reference process exit not confirmed")
		}
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.exitErr
}

// persistentArgvMarker is the argv value the reference executor must preserve.
const persistentArgvMarker = "--tei-reference-marker=argv value preserved"
