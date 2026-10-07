package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Radiqeee/taskapi/internal/auth"
	"github.com/Radiqeee/taskapi/internal/store"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := New(store.NewMemory(), auth.NewTokenManager("test-secret", time.Hour))
	srv := httptest.NewServer(h.Router())
	t.Cleanup(srv.Close)
	return srv
}

// do sends a JSON request and decodes the JSON response into out (if non-nil).
func do(t *testing.T, srv *httptest.Server, method, path, token string, body, out any) int {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, srv.URL+path, &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("decode %s %s: %v", method, path, err)
		}
	}
	return resp.StatusCode
}

type authResp struct {
	Token string `json:"token"`
}

type taskResp struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func register(t *testing.T, srv *httptest.Server, email string) string {
	t.Helper()
	var r authResp
	if code := do(t, srv, "POST", "/api/v1/auth/register", "", map[string]string{"email": email, "password": "password123"}, &r); code != http.StatusCreated {
		t.Fatalf("register: got %d", code)
	}
	return r.Token
}

func TestAuth(t *testing.T) {
	srv := newTestServer(t)
	register(t, srv, "a@example.com")

	creds := map[string]string{"email": "A@example.com", "password": "password123"}
	if code := do(t, srv, "POST", "/api/v1/auth/register", "", creds, nil); code != http.StatusConflict {
		t.Errorf("duplicate register (case-insensitive): got %d, want 409", code)
	}

	var r authResp
	if code := do(t, srv, "POST", "/api/v1/auth/login", "", creds, &r); code != http.StatusOK || r.Token == "" {
		t.Errorf("login: got %d, token %q", code, r.Token)
	}

	bad := map[string]string{"email": "a@example.com", "password": "wrongpass1"}
	if code := do(t, srv, "POST", "/api/v1/auth/login", "", bad, nil); code != http.StatusUnauthorized {
		t.Errorf("wrong password: got %d, want 401", code)
	}

	short := map[string]string{"email": "b@example.com", "password": "short"}
	if code := do(t, srv, "POST", "/api/v1/auth/register", "", short, nil); code != http.StatusBadRequest {
		t.Errorf("short password: got %d, want 400", code)
	}
}

func TestTasksRequireAuth(t *testing.T) {
	srv := newTestServer(t)
	for _, token := range []string{"", "not-a-jwt"} {
		if code := do(t, srv, "GET", "/api/v1/tasks", token, nil, nil); code != http.StatusUnauthorized {
			t.Errorf("token %q: got %d, want 401", token, code)
		}
	}
}

func TestTaskCRUD(t *testing.T) {
	srv := newTestServer(t)
	token := register(t, srv, "a@example.com")

	var created taskResp
	if code := do(t, srv, "POST", "/api/v1/tasks", token, map[string]string{"title": "Write tests"}, &created); code != http.StatusCreated {
		t.Fatalf("create: got %d", code)
	}
	if code := do(t, srv, "POST", "/api/v1/tasks", token, map[string]string{"title": ""}, nil); code != http.StatusBadRequest {
		t.Errorf("create without title: got %d, want 400", code)
	}

	path := "/api/v1/tasks/" + jsonNum(created.ID)

	var updated taskResp
	if code := do(t, srv, "PATCH", path, token, map[string]bool{"done": true}, &updated); code != http.StatusOK {
		t.Fatalf("update: got %d", code)
	}
	if !updated.Done || updated.Title != "Write tests" {
		t.Errorf("update: got %+v, want done with title unchanged", updated)
	}

	do(t, srv, "POST", "/api/v1/tasks", token, map[string]string{"title": "Second"}, nil)

	var list struct {
		Items []taskResp `json:"items"`
	}
	do(t, srv, "GET", "/api/v1/tasks?done=true", token, nil, &list)
	if len(list.Items) != 1 || list.Items[0].ID != created.ID {
		t.Errorf("filter done=true: got %+v", list.Items)
	}
	do(t, srv, "GET", "/api/v1/tasks?limit=1", token, nil, &list)
	if len(list.Items) != 1 || list.Items[0].Title != "Second" {
		t.Errorf("limit=1 should return newest task: got %+v", list.Items)
	}
	if code := do(t, srv, "GET", "/api/v1/tasks?limit=1000", token, nil, nil); code != http.StatusBadRequest {
		t.Errorf("limit over max: got %d, want 400", code)
	}

	if code := do(t, srv, "DELETE", path, token, nil, nil); code != http.StatusNoContent {
		t.Errorf("delete: got %d", code)
	}
	if code := do(t, srv, "GET", path, token, nil, nil); code != http.StatusNotFound {
		t.Errorf("get after delete: got %d, want 404", code)
	}
}

func TestTasksAreIsolatedBetweenUsers(t *testing.T) {
	srv := newTestServer(t)
	alice := register(t, srv, "alice@example.com")
	bob := register(t, srv, "bob@example.com")

	var task taskResp
	do(t, srv, "POST", "/api/v1/tasks", alice, map[string]string{"title": "Private"}, &task)
	path := "/api/v1/tasks/" + jsonNum(task.ID)

	if code := do(t, srv, "GET", path, bob, nil, nil); code != http.StatusNotFound {
		t.Errorf("bob reads alice's task: got %d, want 404", code)
	}
	if code := do(t, srv, "DELETE", path, bob, nil, nil); code != http.StatusNotFound {
		t.Errorf("bob deletes alice's task: got %d, want 404", code)
	}

	var list struct {
		Items []taskResp `json:"items"`
	}
	do(t, srv, "GET", "/api/v1/tasks", bob, nil, &list)
	if len(list.Items) != 0 {
		t.Errorf("bob's list contains %d tasks, want 0", len(list.Items))
	}
}

func jsonNum(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
