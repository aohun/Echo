package magpieclient

// The manager's job: one magpie web instance, running when Echo needs it
// and gone when Echo stops (docs/08 M2). It finds an instance already
// serving before it spawns one, spawns with `--no-open` and a key it
// generated itself, waits until the instance answers a Probe, restarts
// it with a backoff if it dies, and stops it with SIGTERM — never a kill
// in front of in-flight gateway requests.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Manager owns one spawned magpie web process.
type Manager struct {
	// Bin is the magpie binary's path.
	Bin string
	// Addr is where the instance listens, 127.0.0.1:<port>.
	Addr string
	// Key is the run key the instance takes from its environment. NewKey
	// generates one; a fixed key survives the instance's restarts.
	Key string
	// Home and config folders the instance is confined to, for tests: set
	// only when Echo means to isolate it (never a user's real home).
	Env []string
	// Log takes the instance's output.
	Log io.Writer

	mu      sync.Mutex
	cmd     *exec.Cmd
	done    chan error // the instance's Wait, read once by whoever gets there first
	exited  chan struct{}
	stopped bool
	ready   chan struct{}
}

// NewKey generates a run key: 32 random bytes as hex, well over
// MAGPIE_WEB_KEY's minimum of 16.
func NewKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("magpieclient: run key: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Detect asks whether an instance already serves at addr/key: a Probe
// that answers means someone is there — use it, don't spawn a second.
func Detect(ctx context.Context, addr, key string) (bool, Info, error) {
	c := New("http://"+addr, key)
	info, err := c.Probe(ctx)
	if err != nil {
		return false, Info{}, err
	}
	return true, info, nil
}

// Start spawns the instance (unless stopped already), waits until it
// answers a Probe, and leaves a watcher behind that restarts it with a
// backoff if it dies on its own. An instance that never comes up in 20s
// is an error carrying its last output.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopped {
		return fmt.Errorf("magpieclient: instance stopped")
	}
	if m.Key == "" {
		key, err := NewKey()
		if err != nil {
			return err
		}
		m.Key = key
	}
	if m.Addr == "" {
		m.Addr = "127.0.0.1:" + DefaultPort
	}
	m.ready = make(chan struct{})
	return m.spawn(ctx, 0)
}

// spawn runs the process and waits for readiness; backoff counts the
// restarts already spent, capping the wait and, at three, giving up.
// Callers hold m.mu.
func (m *Manager) spawn(ctx context.Context, backoff int) error {
	try := backoff
	if try > 3 {
		return fmt.Errorf("magpieclient: instance died three times, leaving it down")
	}
	if try > 0 {
		wait := time.Duration(1<<try) * time.Second
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	cmd := exec.CommandContext(ctx, m.Bin, "web", "--no-open", "--addr", m.Addr)
	cmd.Env = append(m.Env, "MAGPIE_WEB_KEY="+m.Key, "MAGPIE_WEB_RUNKEY="+m.Key)
	// Stop is an interrupt — the Ctrl-C magpie treats as quit, draining
	// its in-flight gateway requests — with a kill only if it ignores it.
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return cmd.Process.Signal(os.Interrupt)
		}
		return nil
	}
	cmd.WaitDelay = 10 * time.Second
	if m.Log != nil {
		cmd.Stdout = m.Log
		cmd.Stderr = m.Log
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("magpieclient: start %s: %w", m.Bin, err)
	}
	m.cmd = cmd
	m.done = make(chan error, 1)
	m.exited = make(chan struct{})
	go func() { m.done <- cmd.Wait() }()
	if err := m.waitReady(ctx, m.done); err != nil {
		return err
	}
	// The watcher: the only reader of done once the instance is up. An
	// instance that dies on its own closes exited, comes back with the
	// same key so a page paired to it finds it where it was.
	go func() {
		err := <-m.done
		// No lock here: Stop holds it and waits on exited, and this
		// channel has exactly one closer — this spawn's single reader.
		close(m.exited)
		m.mu.Lock()
		stopped := m.stopped
		m.mu.Unlock()
		if stopped || ctx.Err() != nil {
			return
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		if rerr := m.spawn(ctx, try+1); rerr != nil {
			if m.Log != nil {
				fmt.Fprintf(m.Log, "magpieclient: the instance left (%v); restart: %v\n", err, rerr)
			}
		}
	}()
	return nil
}

// waitReady polls Probe until the instance answers, the process dies
// (its error comes back), or the wait is up (20s, then the try's backoff
// on top).
func (m *Manager) waitReady(ctx context.Context, done <-chan error) error {
	c := New("http://"+m.Addr, m.Key)
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error
	for {
		_, err := c.Probe(ctx)
		if err == nil {
			close(m.ready)
			return nil
		}
		lastErr = err
		select {
		case e := <-done:
			close(m.exited)
			return fmt.Errorf("magpieclient: instance exited while coming up: %v", e)
		case <-time.After(250 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("magpieclient: instance not ready after 20s: last probe: %v", lastErr)
		}
	}
}

// Ready says the instance answered a Probe since its last (re)start.
func (m *Manager) Ready() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	select {
	case <-m.ready:
		return true
	default:
		return false
	}
}

// Stop ends the instance: an interrupt — the Ctrl-C magpie treats as
// quit, draining its in-flight gateway requests — and a wait for the
// instance to leave; the exec machinery kills only if the instance
// ignores the interrupt for WaitDelay (10s). No kill in front of
// in-flight requests.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = true
	if m.cmd == nil || m.cmd.Cancel == nil {
		return nil
	}
	err := m.cmd.Cancel()
	select {
	case <-m.exited:
	case <-time.After(15 * time.Second):
		err = fmt.Errorf("magpieclient: stop timed out")
	}
	m.cmd = nil
	return err
}

// JoinLink is the link a page would open: the origin, the key in its
// query. Echo never needs to open it, but tests and pairing logs do.
func (m *Manager) JoinLink() string {
	return "http://" + m.Addr + "/?k=" + m.Key
}

// TrimRunKey keeps a key out of a logged line: the links magpie prints
// carry it in plain sight.
func TrimRunKey(line string) string {
	if i := strings.Index(line, "/?k="); i >= 0 {
		return line[:i+4] + "…"
	}
	return line
}
