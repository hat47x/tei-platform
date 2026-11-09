package ordermigrationdualdb

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CrashEnv names the test-only injection point. Its value is "point:n": the
// process kills itself the n-th time it reaches point. The kill is a real
// process termination (SIGKILL on Unix, TerminateProcess on Windows); no
// deferred cleanup runs and open transactions are left to the server.
const CrashEnv = "TEI_OMDD_CRASH_AT"

// CrashMarker is written to stderr immediately before the kill.
const CrashMarker = "omdd-crash-point"

// Crash points.
const (
	CrashChunkMid             = "chunk-mid"
	CrashAfterPrimaryCommit   = "after-primary-commit"
	CrashBeforeSend           = "before-send"
	CrashAfterInventoryCommit = "after-inventory-commit"
	CrashBeforeCompensation   = "before-compensation"
	// Step 4: a legacy transaction is applied and recorded in the primary,
	// the slot is not advanced yet.
	CrashAfterChangeApplied = "after-change-applied"
	// Step 6: a reverse-sync group is committed in the legacy database, the
	// primary outbox is not marked delivered yet.
	CrashAfterReverseLegacyCommit = "after-reverse-legacy-commit"
)

var crash struct {
	once   sync.Once
	point  string
	target int
	mu     sync.Mutex
	seen   map[string]int
}

func crashPoint(name string) {
	crash.once.Do(func() {
		crash.seen = map[string]int{}
		spec := os.Getenv(CrashEnv)
		if spec == "" {
			return
		}
		point, n, ok := strings.Cut(spec, ":")
		target, err := strconv.Atoi(n)
		if !ok || err != nil || target < 1 {
			fmt.Fprintf(os.Stderr, "ignored malformed %s=%q\n", CrashEnv, spec)
			return
		}
		crash.point, crash.target = point, target
	})
	if crash.point != name {
		return
	}
	crash.mu.Lock()
	crash.seen[name]++
	hit := crash.seen[name] == crash.target
	crash.mu.Unlock()
	if !hit {
		return
	}
	fmt.Fprintf(os.Stderr, "%s %s %d\n", CrashMarker, name, crash.target)
	if p, err := os.FindProcess(os.Getpid()); err == nil {
		_ = p.Kill()
	}
	// Kill is asynchronous on some platforms; never continue past the point.
	for {
		time.Sleep(time.Hour)
	}
}
