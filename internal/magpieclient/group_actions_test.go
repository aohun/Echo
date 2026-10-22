package magpieclient

// The Routing page's write actions over HTTP (docs/08 M4), tested with a
// fake magpie (Go server).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// groupsStateJSON is the groups document every action answers with.
const groupsStateJSON = `{"groups":[{"id":"fast","name":"Fast","routing":"ordered","ready":true,"memberInfo":[{"id":"openai/gpt-5","ready":true,"provider":"openai","name":"OpenAI","model":"gpt-5","on":2}]}],"models":[],"pools":[],"deciders":[],"found":true}`

func groupsActionServer(t *testing.T, wantPath string, check func(body string)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != wantPath {
			t.Errorf("asked %s %s, want POST %s", r.Method, r.URL.Path, wantPath)
		}
		check(readAllBody(r))
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(groupsStateJSON))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func decodeGroupsBody(t *testing.T, body string) map[string]any {
	t.Helper()
	var sent map[string]any
	if err := json.Unmarshal([]byte(body), &sent); err != nil {
		t.Fatalf("body %q: %v", body, err)
	}
	return sent
}

func TestGroupsFoundSendsOnAndAdoptsState(t *testing.T) {
	srv := groupsActionServer(t, "/api/groups/found", func(body string) {
		sent := decodeGroupsBody(t, body)
		if sent["on"] != true {
			t.Errorf("found body on %v", sent["on"])
		}
	})
	c := New(srv.URL, "k")
	out, err := c.GroupsFound(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Found {
		t.Error("found did not answer with the groups document")
	}
	if len(out.Groups) != 1 || out.Groups[0].ID != "fast" {
		t.Errorf("found answer groups %+v", out.Groups)
	}
}

func TestGroupsSaveSendsTheGroupAndItsFormerID(t *testing.T) {
	srv := groupsActionServer(t, "/api/groups/save", func(body string) {
		sent := decodeGroupsBody(t, body)
		if sent["id"] != "quick" || sent["from"] != "fast" {
			t.Errorf("save body id %v from %v", sent["id"], sent["from"])
		}
		if sent["routing"] != "rotate" {
			t.Errorf("save body routing %v", sent["routing"])
		}
	})
	c := New(srv.URL, "k")
	g := provider.Group{ID: "quick", Name: "Quick", Routing: "rotate"}
	if _, err := c.GroupsSave(context.Background(), g, "fast"); err != nil {
		t.Fatal(err)
	}
}

func TestGroupsSaveWithoutARenameOmitsFrom(t *testing.T) {
	srv := groupsActionServer(t, "/api/groups/save", func(body string) {
		sent := decodeGroupsBody(t, body)
		if _, ok := sent["from"]; ok {
			t.Errorf("save body carries from: %v", sent["from"])
		}
	})
	c := New(srv.URL, "k")
	g := provider.Group{ID: "quick", Name: "Quick", Routing: "rotate"}
	if _, err := c.GroupsSave(context.Background(), g, ""); err != nil {
		t.Fatal(err)
	}
}

func TestGroupsDeleteSendsTheIDs(t *testing.T) {
	srv := groupsActionServer(t, "/api/groups/delete", func(body string) {
		var sent struct {
			IDs []string `json:"ids"`
		}
		if err := json.Unmarshal([]byte(body), &sent); err != nil {
			t.Fatalf("delete body %q: %v", body, err)
		}
		if len(sent.IDs) != 2 || sent.IDs[0] != "a" || sent.IDs[1] != "b" {
			t.Errorf("delete body ids %v", sent.IDs)
		}
	})
	c := New(srv.URL, "k")
	if _, err := c.GroupsDelete(context.Background(), []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
}

func TestGroupsSwitchSendsTheIDAndOn(t *testing.T) {
	srv := groupsActionServer(t, "/api/groups/switch", func(body string) {
		sent := decodeGroupsBody(t, body)
		if sent["id"] != "fast" || sent["on"] != true {
			t.Errorf("switch body id %v on %v", sent["id"], sent["on"])
		}
	})
	c := New(srv.URL, "k")
	if _, err := c.GroupsSwitch(context.Background(), "fast", true); err != nil {
		t.Fatal(err)
	}
}

func TestGroupsArrangeSendsTheWholeOrder(t *testing.T) {
	srv := groupsActionServer(t, "/api/groups/arrange", func(body string) {
		var sent struct {
			Order []string `json:"order"`
		}
		if err := json.Unmarshal([]byte(body), &sent); err != nil {
			t.Fatalf("arrange body %q: %v", body, err)
		}
		if len(sent.Order) != 2 || sent.Order[0] != "fast" || sent.Order[1] != "quick" {
			t.Errorf("arrange body order %v", sent.Order)
		}
	})
	c := New(srv.URL, "k")
	if _, err := c.GroupsArrange(context.Background(), []string{"fast", "quick"}); err != nil {
		t.Fatal(err)
	}
}
