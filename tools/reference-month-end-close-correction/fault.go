package monthendclose

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CrashEnv names the test-only kill point: "point:n" kills the process the
// n-th time it reaches point. The kill is a real process termination; no
// deferred cleanup runs and open transactions are left to the server.
const CrashEnv = "TEI_MECC_CRASH_AT"

// StopEnv names the test-only stop point: "point:n" stops the whole process
// (SIGSTOP on Unix) the n-th time it reaches point. Every goroutine, the
// lease heartbeat included, stops until the test sends SIGCONT.
const StopEnv = "TEI_MECC_STOP_AT"

// ClockOffsetEnv skews this process's host clock (a Go duration such as
// "-10m"). The host clock is recorded but never decides a lease expiry.
const ClockOffsetEnv = "TEI_MECC_CLOCK_OFFSET"

// Markers written to stderr immediately before a kill or a stop.
const (
	CrashMarker = "mecc-crash-point"
	StopMarker  = "mecc-stop-point"
)

// Injection points.
const (
	// Step 2: the whole file is written but not committed.
	CrashImportBeforeCommit = "import-before-commit"
	// Step 3: detail rows of a chunk are written, the chunk is not committed.
	CrashCloseChunkMid = "close-chunk-mid"
	// Step 3: the next chunk is selected and about to be written.
	PointBeforeChunkWrite = "before-chunk-write"
	// Step 3: the close totals are written, not committed.
	CrashCloseBeforeComplete = "close-before-complete"
)

type pointSpec struct {
	once   sync.Once
	point  string
	target int
	mu     sync.Mutex
	seen   map[string]int
}

var crashSpec, stopSpec pointSpec

func (p *pointSpec) hit(env, name string) bool {
	p.once.Do(func() {
		p.seen = map[string]int{}
		spec := os.Getenv(env)
		if spec == "" {
			return
		}
		point, n, ok := strings.Cut(spec, ":")
		target, err := strconv.Atoi(n)
		if !ok || err != nil || target < 1 {
			fmt.Fprintf(os.Stderr, "ignored malformed %s=%q\n", env, spec)
			return
		}
		p.point, p.target = point, target
	})
	if p.point != name {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.seen[name]++
	return p.seen[name] == p.target
}

func crashPoint(name string) {
	if !crashSpec.hit(CrashEnv, name) {
		return
	}
	fmt.Fprintf(os.Stderr, "%s %s %d\n", CrashMarker, name, crashSpec.target)
	if p, err := os.FindProcess(os.Getpid()); err == nil {
		_ = p.Kill()
	}
	for {
		time.Sleep(time.Hour)
	}
}

// stopPoint stops the process at the point; it returns when the process is
// continued.
func stopPoint(name string) {
	if !stopSpec.hit(StopEnv, name) {
		return
	}
	fmt.Fprintf(os.Stderr, "%s %s %d\n", StopMarker, name, stopSpec.target)
	stopSelf()
}

// HostClock returns this process's host clock, skewed by ClockOffsetEnv.
func HostClock() func() time.Time {
	off, err := time.ParseDuration(os.Getenv(ClockOffsetEnv))
	if err != nil {
		off = 0
	}
	return func() time.Time { return time.Now().Add(off) }
}
