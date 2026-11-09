//go:build !unix

package monthendclose

import "time"

// stopSelf cannot stop the whole process on this platform; it blocks the
// calling goroutine instead. The real-DB tests that need a stop run on Unix.
func stopSelf() {
	for {
		time.Sleep(time.Hour)
	}
}
