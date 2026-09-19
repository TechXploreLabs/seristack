package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	conf "github.com/TechXploreLabs/seristack/internal/config"
)

func TestSplitHeader(t *testing.T) {
	got := splitHeader(" developers, platform , ,platform-admin ")
	want := []string{"developers", "platform", "platform-admin"}
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSplitHeaderEmpty(t *testing.T) {
	if got := splitHeader(" , , "); len(got) != 0 {
		t.Fatalf("got %#v, want empty", got)
	}
}

func TestCheckAccessNoRules(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	if !checkAccess(req, nil, "ALL") {
		t.Fatal("no access rules should allow the request")
	}
}

func TestCheckAccessAny(t *testing.T) {
	rules := []conf.AccessRule{
		{HeaderName: "X-Auth-Request-Groups", HeaderValue: []string{"developers"}},
		{HeaderName: "X-Auth-Request-Roles", HeaderValue: []string{"admin"}},
	}

	tests := []struct {
		name, groups, roles string
		want                bool
	}{
		{"group matches", "developers", "", true},
		{"role matches", "", "admin", true},
		{"comma separated group matches", "users, developers, platform", "", true},
		{"nothing matches", "users", "viewer", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-Auth-Request-Groups", tt.groups)
			req.Header.Set("X-Auth-Request-Roles", tt.roles)
			if got := checkAccess(req, rules, "ANY"); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckAccessAll(t *testing.T) {
	rules := []conf.AccessRule{
		{HeaderName: "X-Auth-Request-Groups", HeaderValue: []string{"platform"}},
		{HeaderName: "X-Auth-Request-Roles", HeaderValue: []string{"admin"}},
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Auth-Request-Groups", "platform")
	req.Header.Set("X-Auth-Request-Roles", "admin")
	if !checkAccess(req, rules, "ALL") {
		t.Fatal("both matching rules should allow the request")
	}

	req.Header.Set("X-Auth-Request-Roles", "viewer")
	if checkAccess(req, rules, "ALL") {
		t.Fatal("one non-matching rule should deny ALL access")
	}
}

func TestCheckAccessCaseInsensitive(t *testing.T) {
	rules := []conf.AccessRule{
		{HeaderName: "X-Auth-Request-Groups", HeaderValue: []string{"platform"}},
		{HeaderName: "X-Auth-Request-Roles", HeaderValue: []string{"admin"}},
	}
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Auth-Request-Groups", "platform")
	req.Header.Set("X-Auth-Request-Roles", "admin")
	if !checkAccess(req, rules, "all") {
		t.Fatal(`"all" should behave like "ALL"`)
	}
}

func TestExtractIdentity(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Auth-Request-User", "alice")
	req.Header.Set("X-Auth-Request-Groups", "developers,platform")
	req.Header.Set("Authorization", "Bearer secret")

	got := extractIdentity(req, []string{"Authorization"})

	if got["X-Auth-Request-User"] != "alice" {
		t.Errorf("user = %q, want alice", got["X-Auth-Request-User"])
	}
	if got["X-Auth-Request-Groups"] != "developers,platform" {
		t.Errorf("groups = %q, want developers,platform", got["X-Auth-Request-Groups"])
	}
	if _, ok := got["Authorization"]; ok {
		t.Error("Authorization should be excluded")
	}
}

func TestSubstituteVarsFromQueryAndForm(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/test?name=alice&name=last",
		strings.NewReader("command=go+test"),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		t.Fatal(err)
	}

	got := substituteVars(req)
	if got["name"] != "last" {
		t.Errorf("name = %q, want last", got["name"])
	}
	if got["command"] != "go test" {
		t.Errorf("command = %q, want go test", got["command"])
	}
}

func TestSubstituteVarsFromJSON(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/test",
		strings.NewReader(`{"name":"alice","count":3,"enabled":true}`),
	)
	req.Header.Set("Content-Type", "application/json")

	got := substituteVars(req)
	if got["name"] != "alice" {
		t.Errorf("name = %q, want alice", got["name"])
	}
	if got["count"] != "3" {
		t.Errorf("count = %q, want 3", got["count"])
	}
	if got["enabled"] != "true" {
		t.Errorf("enabled = %q, want true", got["enabled"])
	}
}

func TestRegisterHandlerRejectsWrongMethodBeforeExecution(t *testing.T) {
	stack := conf.Stack{Name: "stack2", Method: "POST"}
	mux := http.NewServeMux()

	RegisterHandler(mux, stack, map[string]*conf.Stack{"stack2": &stack}, ".", nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/stack2", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestRegisterHandlerDeniesUnauthorizedStack(t *testing.T) {
	stack := conf.Stack{
		Name:   "admin-stack",
		Method: "POST",
		Access: []conf.AccessRule{
			{
				HeaderName:  "X-Auth-Request-Groups",
				HeaderValue: []string{"platform-admin"},
			},
		},
	}
	mux := http.NewServeMux()
	RegisterHandler(mux, stack, map[string]*conf.Stack{"admin-stack": &stack}, ".", nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/admin-stack", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
