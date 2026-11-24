package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"mymoments/internal/config"
	"mymoments/internal/store"
)

func newTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(context.Background(), "file:"+filepath.Join(dir, "test.db")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	cfg := config.Config{
		DataDir:       dir,
		DefaultTZ:     "Asia/Shanghai",
		SessionSecret: []byte("test-secret-test-secret-test-secret"),
		SessionTTL:    time.Hour,
	}
	srv, err := New(cfg, st)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	t.Cleanup(func() {
		ts.Close()
		_ = st.Close()
	})
	return ts, client
}

func call(t *testing.T, client *http.Client, method, url string, body any) (int, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	out := map[string]any{}
	if len(raw) > 0 && res.Header.Get("Content-Type") == "application/json; charset=utf-8" {
		_ = json.Unmarshal(raw, &out)
	}
	return res.StatusCode, out
}

func TestSetupLoginFlow(t *testing.T) {
	ts, client := newTestServer(t)
	base := ts.URL + "/api/v1"

	status, body := call(t, client, http.MethodGet, base+"/bootstrap", nil)
	if status != http.StatusOK || body["needsSetup"] != true {
		t.Fatalf("expected needsSetup, got %d %v", status, body)
	}

	status, body = call(t, client, http.MethodPost, base+"/setup", map[string]any{
		"householdName": "林家",
		"username":      "admin",
		"displayName":   "管理员",
		"password":      "password12345",
	})
	if status != http.StatusCreated {
		t.Fatalf("setup failed: %d %v", status, body)
	}
	household, _ := body["household"].(map[string]any)
	if household == nil || household["name"] != "林家" {
		t.Fatalf("unexpected household: %v", body["household"])
	}
	user, _ := body["user"].(map[string]any)
	if user == nil || user["role"] != "admin" {
		t.Fatalf("unexpected user: %v", body["user"])
	}

	status, body = call(t, client, http.MethodGet, base+"/bootstrap", nil)
	if status != http.StatusOK || body["needsSetup"] != false || body["user"] == nil {
		t.Fatalf("expected session user, got %d %v", status, body)
	}

	status, _ = call(t, client, http.MethodPost, base+"/auth/logout", nil)
	if status != http.StatusNoContent {
		t.Fatalf("logout: %d", status)
	}
	status, body = call(t, client, http.MethodGet, base+"/bootstrap", nil)
	if status != http.StatusOK || body["user"] != nil {
		t.Fatalf("expected anonymous, got %d %v", status, body)
	}

	status, _ = call(t, client, http.MethodPost, base+"/auth/login", map[string]any{"username": "admin", "password": "wrong-password"})
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", status)
	}
	status, body = call(t, client, http.MethodPost, base+"/auth/login", map[string]any{"username": "ADMIN", "password": "password12345"})
	if status != http.StatusOK {
		t.Fatalf("login failed: %d %v", status, body)
	}
}

func TestSetupConflict(t *testing.T) {
	ts, client := newTestServer(t)
	base := ts.URL + "/api/v1"
	payload := map[string]any{"username": "admin", "displayName": "管理员", "password": "password12345"}
	if status, _ := call(t, client, http.MethodPost, base+"/setup", payload); status != http.StatusCreated {
		t.Fatalf("first setup: %d", status)
	}
	status, body := call(t, client, http.MethodPost, base+"/setup", payload)
	if status != http.StatusConflict {
		t.Fatalf("expected conflict, got %d %v", status, body)
	}
}

func TestMemberPermissionAndPasswordChange(t *testing.T) {
	ts, admin := newTestServer(t)
	base := ts.URL + "/api/v1"
	if status, _ := call(t, admin, http.MethodPost, base+"/setup", map[string]any{
		"username": "admin", "displayName": "管理员", "password": "password12345",
	}); status != http.StatusCreated {
		t.Fatal("setup failed")
	}

	status, body := call(t, admin, http.MethodPost, base+"/family/members", map[string]any{
		"username": "xiaoming", "displayName": "小明", "relation": "儿子",
	})
	if status != http.StatusCreated {
		t.Fatalf("create member: %d %v", status, body)
	}
	temp, _ := body["temporaryPassword"].(string)
	if temp == "" {
		t.Fatal("expected temporary password")
	}

	jar, _ := cookiejar.New(nil)
	member := &http.Client{Jar: jar}
	status, body = call(t, member, http.MethodPost, base+"/auth/login", map[string]any{"username": "xiaoming", "password": temp})
	if status != http.StatusOK {
		t.Fatalf("member login: %d %v", status, body)
	}
	user, _ := body["user"].(map[string]any)
	if user["mustChangePassword"] != true {
		t.Fatalf("expected mustChangePassword: %v", user)
	}

	status, _ = call(t, member, http.MethodGet, base+"/family/members", nil)
	if status != http.StatusForbidden {
		t.Fatalf("expected password change required, got %d", status)
	}

	status, _ = call(t, member, http.MethodPost, base+"/auth/password", map[string]any{
		"currentPassword": temp, "newPassword": "newpassword123",
	})
	if status != http.StatusNoContent {
		t.Fatalf("change password: %d", status)
	}
	status, _ = call(t, member, http.MethodGet, base+"/family/members", nil)
	if status != http.StatusOK {
		t.Fatalf("members after change: %d", status)
	}
	status, _ = call(t, member, http.MethodPost, base+"/family/members", map[string]any{
		"username": "other", "displayName": "其他",
	})
	if status != http.StatusForbidden {
		t.Fatalf("member must not create members: %d", status)
	}
}

func TestLastAdminCannotBeRemoved(t *testing.T) {
	ts, admin := newTestServer(t)
	base := ts.URL + "/api/v1"
	_, body := call(t, admin, http.MethodPost, base+"/setup", map[string]any{
		"username": "admin", "displayName": "管理员", "password": "password12345",
	})
	user, _ := body["user"].(map[string]any)
	id := int64(user["id"].(float64))

	status, body := call(t, admin, http.MethodPatch, base+"/family/members/"+itoa(id), map[string]any{"status": "inactive"})
	if status != http.StatusBadRequest {
		t.Fatalf("self deactivation should fail: %d %v", status, body)
	}
	status, body = call(t, admin, http.MethodPatch, base+"/family/members/"+itoa(id), map[string]any{"role": "member"})
	if status != http.StatusConflict {
		t.Fatalf("last admin demotion should conflict: %d %v", status, body)
	}
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
