package magpieclient

// The manager's tests run against a fake magpie web: the test binary
// itself, in helper mode — its init sees the env and plays the instance
// instead of running the tests (a process test of a process manager).

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"
)

func init() {
	if os.Getenv("MAGPIE_FAKE_WEB") == "" {
		return
	}
	port := ""
	for i, a := range os.Args {
		if a == "--addr" && i+1 < len(os.Args) {
			port = strings.TrimPrefix(os.Args[i+1], "127.0.0.1:")
		}
	}
	if port == "" {
		os.Exit(2)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/settings", func(rw http.ResponseWriter, r *http.Request) {
		ck, err := r.Cookie("magpie_web_" + port)
		if err != nil || ck.Value != os.Getenv("MAGPIE_WEB_KEY") {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{"version":"fake","gateway":"http://127.0.0.1:1","dir":"/tmp/fake"}`))
	})
	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		os.Exit(3)
	}
	go http.Serve(ln, mux)
	fmt.Println("magpie web on http://127.0.0.1:" + port + "/?k=" + os.Getenv("MAGPIE_WEB_KEY"))
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	os.Exit(0)
}

func freePort(t *testing.T) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	return port
}

func newTestManager(t *testing.T) *Manager {
	m := &Manager{
		Bin:  os.Args[0],
		Addr: "127.0.0.1:" + freePort(t),
		Env:  []string{"MAGPIE_FAKE_WEB=1"},
		Log:  testLog{t},
	}
	t.Cleanup(func() { m.Stop() })
	return m
}

type testLog struct{ t *testing.T }

func (l testLog) Write(p []byte) (int, error) { return len(p), nil }

func TestSpawnBecomesReadyThenDetectThenStops(t *testing.T) {
	m := newTestManager(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if !m.Ready() {
		t.Fatal("Start returned before the instance was ready")
	}
	there, info, err := Detect(ctx, m.Addr, m.Key)
	if err != nil || !there {
		t.Fatalf("Detect: %v (%v)", err, info)
	}
	if info.Version != "fake" {
		t.Errorf("Detect read version %q", info.Version)
	}
	if err := m.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestStopIsIdempotent(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Stop(); err != nil {
		t.Fatalf("first Stop: %v", err)
	}
	if err := m.Stop(); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
}

func TestSpawnRefusesAfterStop(t *testing.T) {
	m := newTestManager(t)
	if err := m.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := m.Start(context.Background()); err == nil {
		t.Fatal("Start after Stop succeeded")
	}
}

func TestDetectFindsNothingWhenNothingServes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	there, _, err := Detect(ctx, "127.0.0.1:1", "k")
	if there {
		t.Fatal("Detect found an instance on port 1")
	}
	if err == nil {
		t.Error("Detect swallowed the connection error")
	}
}

func TestNewKeyIsLongHex(t *testing.T) {
	k, err := NewKey()
	if err != nil {
		t.Fatal(err)
	}
	if len(k) < 32 {
		t.Errorf("run key %q is shorter than MAGPIE_WEB_KEY's minimum even before hex", k)
	}
	for _, c := range k {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("run key %q is not hex", k)
			break
		}
	}
}

func TestTrimRunKeyRedactsTheLink(t *testing.T) {
	line := "● magpie web on http://127.0.0.1:3437/?k=secret-key-abcdef"
	if got := TrimRunKey(line); strings.Contains(got, "secret") {
		t.Errorf("the logged line still carries the key: %q", got)
	}
	if got := TrimRunKey("no link here"); got != "no link here" {
		t.Errorf("TrimRunKey changed a line without a link: %q", got)
	}
}

// compile-time: the fake's settings answer decodes through the client.
func TestFakeSettingsDecode(t *testing.T) {
	var s Settings
	if err := json.Unmarshal([]byte(`{"version":"fake","gateway":"http://127.0.0.1:1","dir":"/tmp/fake"}`), &s); err != nil {
		t.Fatal(err)
	}
	if s.Version != "fake" {
		t.Errorf("version %q", s.Version)
	}
}
