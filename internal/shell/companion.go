//go:build nogui && echo

package shell

import (
	"cmp"
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/yetone/magpie/internal/magpieclient"
)

// The bundled magpie web. The installed app carries its own instance (make
// echo-app-bundled puts the helper next to the executable), so the user
// installs one thing and the shell reads a magpie web it started itself
// (docs/08). A dev run keeps the seam's environment — the verify scripts —
// and a test or probe build has no helper next to its binary, so neither
// spawns anything.
const (
	companionHelper  = "echo-magpie-web"
	companionWebPort = "3438" // what the experiment's verify runs used
	companionGwPort  = "3498"
)

// companionState is the run state kept between launches: the instance's
// ports and its run key, so an instance a previous launch left answering
// is reused rather than replaced, and an agent's configs keep pointing
// at a gateway whose address the drift-reapply keeps reasserting anyway.
type companionState struct {
	Web string `json:"web"` // the web API's port
	Gw  string `json:"gw"` // the gateway's port
	Key string `json:"key"` // the run key, both sides of the pair read it
}

// teeCompanionLog sends the shell's log to a file in the app's own data
// dir, beside a Finder launch that has no stderr to show it on. Only a
// bundled app does this — a dev run logs to the terminal it came from.
// The file starts over past two megabytes, so a launch that logs much
// cannot grow it forever.
func teeCompanionLog() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(exe), companionHelper)); err != nil {
		return // not a bundled app: dev run, test or probe
	}
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return
	}
	path := filepath.Join(dir, "echo.log")
	if info, err := os.Stat(path); err == nil && info.Size() > 2<<20 {
		_ = os.Remove(path)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	log.SetOutput(io.MultiWriter(os.Stderr, f))
}

// startCompanion points the remote seam at the bundled magpie web while
// the window comes up: the instance the saved state names is reused when
// it answers its probe, a fresh one is started otherwise. The channel
// closes once the seam is decided — set, or left unset on purpose — and
// the first read waits for it, because remoteGateway decides once
// (sync.Once): a seam set late would leave the whole session embedded.
//
// The write stops the instance this launch started, and is nil when it
// reused one, pointed at the environment's own, or found no helper.
func startCompanion() (<-chan struct{}, func()) {
	ready := make(chan struct{})
	var mu sync.Mutex
	var started *exec.Cmd
	go func() {
		defer close(ready)
		if cmd := companionStart(); cmd != nil {
			mu.Lock()
			started = cmd
			mu.Unlock()
		}
	}()
	return ready, func() {
		mu.Lock()
		if started != nil {
			_ = started.Process.Kill()
		}
		mu.Unlock()
	}
}

// companionStart is the deciding itself, run off the main thread so the
// window need not wait for an instance to come up. It returns the
// instance it spawned, nil otherwise.
func companionStart() *exec.Cmd {
	if os.Getenv(remoteEnvAddr) != "" {
		return nil // the environment's own instance wins
	}
	exe, err := os.Executable()
	if err != nil {
		return nil
	}
	helper := filepath.Join(filepath.Dir(exe), companionHelper)
	if _, err := os.Stat(helper); err != nil {
		return nil // not a bundled app: dev run, test or probe
	}

	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		log.Printf("echo: companion: no user data dir: %v", err)
		return nil
	}
	st := readCompanionState(filepath.Join(dir, "companion.json"))
	if st.Key == "" {
		if st.Key, err = magpieclient.NewKey(); err != nil {
			log.Printf("echo: companion: %v", err)
			return nil
		}
	}

	addr := "http://127.0.0.1:" + cmp.Or(st.Web, companionWebPort)
	cmd := (*exec.Cmd)(nil)
	if companionHealthy(addr, st.Key) {
		log.Printf("echo: companion: reusing the instance on %s", addr)
	} else {
		web, gw := companionPort(cmp.Or(st.Web, companionWebPort)), companionPort(cmp.Or(st.Gw, companionGwPort))
		cmd, err = companionSpawn(helper, web, gw, st.Key, filepath.Join(dir, companionHelper+".log"))
		if err != nil {
			log.Printf("echo: companion: %v", err)
			return nil
		}
		st.Web, st.Gw = web, gw
		addr = "http://127.0.0.1:" + web
		if err := companionWait(addr, st.Key); err != nil {
			log.Printf("echo: companion: %v", err)
			return nil // the reads fall back to embedded
		}
		if err := writeCompanionState(filepath.Join(dir, "companion.json"), st); err != nil {
			log.Printf("echo: companion: %v", err)
		}
		log.Printf("echo: companion: magpie web on %s, gateway on 127.0.0.1:%s", addr, st.Gw)
	}

	os.Setenv(remoteEnvAddr, addr)
	os.Setenv(remoteEnvKey, st.Key)
	return cmd
}

// companionHealthy says the instance answers its probe with the key.
func companionHealthy(addr, key string) bool {
	c := magpieclient.New(addr, key)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := c.Probe(ctx)
	return err == nil
}

// companionWait waits until a freshly started instance answers, for the
// window appears only after this: the seam decides once, so a half-up
// instance would leave the session embedded for good.
func companionWait(addr, key string) error {
	c := magpieclient.New(addr, key)
	deadline := time.Now().Add(15 * time.Second)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, err := c.Probe(ctx)
		cancel()
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// companionSpawn starts the helper: web mode, no browser to open, the
// gateway on its own port, both reading the run key from the environment.
// Its output lands in a log next to the state, so a companion that
// misbehaves can be read about.
func companionSpawn(helper, webPort, gwPort, key, logPath string) (*exec.Cmd, error) {
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		f = nil // the log is a courtesy; run without it
	}
	cmd := exec.Command(helper, "web", "--no-open", "--addr", "127.0.0.1:"+webPort)
	cmd.Env = append(os.Environ(),
		"MAGPIE_WEB_KEY="+key,
		"MAGPIE_WEB_RUNKEY="+key,
		"MAGPIE_ADDR=127.0.0.1:"+gwPort,
	)
	if f != nil {
		cmd.Stdout, cmd.Stderr = f, f
	} else {
		cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	}
	if err := cmd.Start(); err != nil {
		if f != nil {
			_ = f.Close()
		}
		return nil, err
	}
	return cmd, nil
}

// companionPort picks a free port, preferring the one named: the saved
// state's when it is free again, a fresh one when something else holds it.
func companionPort(preferred string) string {
	if l, err := net.Listen("tcp", "127.0.0.1:"+preferred); err == nil {
		_ = l.Close()
		return preferred
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return preferred // let the spawn report the bind failure
	}
	defer l.Close()
	_, port, _ := net.SplitHostPort(l.Addr().String())
	return port
}

// readCompanionState reads the saved run state; anything unreadable or
// incomplete starts over with the defaults.
func readCompanionState(path string) companionState {
	var st companionState
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	return st
}

func writeCompanionState(path string, st companionState) error {
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
