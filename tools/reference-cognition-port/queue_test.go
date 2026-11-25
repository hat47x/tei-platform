package cognitionport_test

// The queue exists so a background hint cannot block the normal operation, and
// so that two jobs for the same subject never run at once. These tests pin both
// suppression rules, the release of a key, and the property that matters most in
// practice: concurrent submissions from an editor and a build do not race into
// admitting two jobs for one subject.

import (
	"runtime"
	"sync"
	"testing"

	port "tei/tools/reference-cognition-port"
)

// runtimeYield hands the scheduler a chance to interleave a racing submission
// without depending on a sleep duration, which would make the test slow on a
// loaded machine and fast only on an idle one.
func runtimeYield() { runtime.Gosched() }

func TestQueueAdmitsAndRunsOneJob(t *testing.T) {
	queue, err := port.NewQueue(1)
	if err != nil {
		t.Fatal(err)
	}
	defer queue.Close()
	job, stats := queue.Submit("key-a")
	if !stats.Admitted || job == nil {
		t.Fatalf("the job was not admitted: %+v", stats)
	}
	job.Finish()
}

// A second submission for a key that is still active is coalesced, not queued
// behind it: an edit that does not change the subject must not start a second
// inference.
func TestQueueCoalescesOnTheKey(t *testing.T) {
	queue, err := port.NewQueue(4)
	if err != nil {
		t.Fatal(err)
	}
	defer queue.Close()
	first, firstStats := queue.Submit("key-a")
	if !firstStats.Admitted {
		t.Fatalf("the first job was not admitted: %+v", firstStats)
	}
	second, secondStats := queue.Submit("key-a")
	if second != nil || secondStats.Admitted || secondStats.Suppressed != port.SuppressedKey {
		t.Fatalf("the duplicate was not coalesced: %+v", secondStats)
	}
	if secondStats.Deduplicated != 1 {
		t.Fatalf("the deduplication was not counted: %+v", secondStats)
	}
	first.Finish()
}

// The bound is enforced, and a refused submission is counted rather than
// silently dropped: a caller that cannot see suppression cannot tell whether
// its background work is running at all.
func TestQueueEnforcesDepthAndCountsSuppression(t *testing.T) {
	queue, err := port.NewQueue(1)
	if err != nil {
		t.Fatal(err)
	}
	defer queue.Close()
	first, _ := queue.Submit("key-a")
	second, stats := queue.Submit("key-b")
	if second != nil || stats.Admitted || stats.Suppressed != port.SuppressedDepth {
		t.Fatalf("the second job was not refused for depth: %+v", stats)
	}
	if stats.DepthExceeded != 1 {
		t.Fatalf("the depth refusal was not counted: %+v", stats)
	}
	first.Finish()
}

// After a job finishes, its key is available again. A queue that never released
// a key would refuse every later edit to the same subject.
func TestQueueReleasesTheKeyAfterTheJobFinishes(t *testing.T) {
	queue, err := port.NewQueue(1)
	if err != nil {
		t.Fatal(err)
	}
	defer queue.Close()
	first, stats := queue.Submit("key-a")
	if !stats.Admitted {
		t.Fatalf("the first job was not admitted: %+v", stats)
	}
	first.Finish()
	second, secondStats := queue.Submit("key-a")
	if second == nil || !secondStats.Admitted {
		t.Fatalf("the key was not released: %+v", secondStats)
	}
	second.Finish()
}

// Finish is idempotent: a caller that releases on a defer path and again on an
// error path must not free a key twice and let a second job in behind a job that
// is still running.
func TestQueueFinishIsIdempotent(t *testing.T) {
	queue, err := port.NewQueue(1)
	if err != nil {
		t.Fatal(err)
	}
	defer queue.Close()
	job, _ := queue.Submit("key-a")
	job.Finish()
	job.Finish()
	next, stats := queue.Submit("key-a")
	if next == nil || !stats.Admitted {
		t.Fatalf("a double finish released the key twice: %+v", stats)
	}
	next.Finish()
}

// An empty key is refused rather than treated as a wildcard: an empty key would
// coalesce every job into one and silently drop work the caller asked for.
func TestQueueRefusesAnEmptyKey(t *testing.T) {
	queue, err := port.NewQueue(2)
	if err != nil {
		t.Fatal(err)
	}
	defer queue.Close()
	job, stats := queue.Submit("")
	if job != nil || stats.Admitted || stats.Suppressed != port.SuppressedKey {
		t.Fatalf("an empty key was not refused: %+v", stats)
	}
}

// Close refuses further submissions without cancelling a job: this queue has no
// goroutine, so a caller that wants cancellation cancels its own work.
func TestQueueCloseRefusesLaterSubmissions(t *testing.T) {
	queue, err := port.NewQueue(1)
	if err != nil {
		t.Fatal(err)
	}
	queue.Close()
	job, stats := queue.Submit("key-a")
	if job != nil || stats.Admitted || stats.Suppressed != port.SuppressedClosed {
		t.Fatalf("a closed queue admitted work: %+v", stats)
	}
}

// A depth below one is refused rather than replaced with a default: a caller
// asking for zero concurrency means something different from a caller that did
// not think about it.
func TestQueueRefusesANonPositiveDepth(t *testing.T) {
	if _, err := port.NewQueue(0); err == nil {
		t.Fatal("a depth of zero was accepted")
	}
	if _, err := port.NewQueue(-1); err == nil {
		t.Fatal("a negative depth was accepted")
	}
}

// Concurrent submissions must never admit two jobs for the same subject at the
// same time. An editor firing on every keystroke and a build finishing are not
// serialised among themselves, so this is the race that would produce duplicate
// inference in practice.
func TestQueueNeverAdmitsTwoJobsForOneSubjectConcurrently(t *testing.T) {
	queue, err := port.NewQueue(8)
	if err != nil {
		t.Fatal(err)
	}
	defer queue.Close()

	var mu sync.Mutex
	concurrent := map[string]int{}
	maxConcurrent := map[string]int{}
	admittedTotal := 0

	var group sync.WaitGroup
	for index := 0; index < 256; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			key := "key-" + string(rune('a'+index%4))
			job, stats := queue.Submit(key)
			if !stats.Admitted {
				return
			}
			mu.Lock()
			admittedTotal++
			concurrent[key]++
			if concurrent[key] > maxConcurrent[key] {
				maxConcurrent[key] = concurrent[key]
			}
			mu.Unlock()

			// Hold the job briefly so a racing submission for the same key has
			// a chance to arrive while this one is active.
			runtimeYield()

			mu.Lock()
			concurrent[key]--
			mu.Unlock()
			job.Finish()
		}(index)
	}
	group.Wait()

	if admittedTotal == 0 {
		t.Fatal("no job was admitted")
	}
	for key, peak := range maxConcurrent {
		if peak > 1 {
			t.Fatalf("%s had %d concurrent jobs; the coalescing raced", key, peak)
		}
	}
}
