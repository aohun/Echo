//go:build nogui && echo

package shell

// The stall watchdog: a frame of the window's UI that takes seconds means
// something on the main thread is stuck on a lock or a call — the sort of
// freeze a reader reports as "the app hangs" with nothing in the logs to
// say what held it. The frame's heartbeat is watched, and a stall writes
// every goroutine's stack where a later read can name the holder.

import (
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"
)

// frameBeat is the time of the last frame's start, stored by the frame
// itself on the main thread.
var frameBeat atomic.Int64 // unix nanos

// beatFrame marks a frame's start; called from the view, on the main thread.
func beatFrame() {
	frameBeat.Store(time.Now().UnixNano())
}

// watchStalls starts the watchdog; called once, when the window is built.
func watchStalls() {
	go func() {
		var dumpedAt int64
		for {
			time.Sleep(time.Second)
			now := frameBeat.Load()
			if now == 0 {
				continue // no frame yet drawn
			}
			stalled := time.Since(time.Unix(0, now))
			if stalled < 5*time.Second || dumpedAt == now {
				continue
			}
			dumpedAt = now // one dump per stall, not one a second
			dump := "echo: the window's frame has not moved for " +
				stalled.Round(time.Second).String() + "\n" + string(stacks())
			_ = os.WriteFile(filepath.Join(os.TempDir(), "echo-stall.log"), []byte(dump), 0o644)
		}
	}()
}

// stacks is every goroutine's stack, the main thread's first.
func stacks() []byte {
	buf := make([]byte, 1<<20)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return buf[:n]
		}
		buf = make([]byte, 2*len(buf))
	}
}
