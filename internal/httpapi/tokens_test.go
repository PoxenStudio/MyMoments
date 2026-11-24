package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
)

func formatID(id int64) string { return strconv.FormatInt(id, 10) }

func callBearer(t *testing.T, client *http.Client, method, url, token string, body any) (int, map[string]any) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer res.Body.Close()
	var out map[string]any
	if res.StatusCode != http.StatusNoContent {
		_ = json.NewDecoder(res.Body).Decode(&out)
	}
	return res.StatusCode, out
}

func TestAPITokenScopes(t *testing.T) {
	base, admin, _ := setupHousehold(t)
	status, body := call(t, admin, http.MethodPost, base+"/tasks", map[string]any{"title": "交水电费"})
	if status != http.StatusCreated {
		t.Fatalf("create task: %d %v", status, body)
	}
	taskID := body["task"].(map[string]any)["id"].(float64)

	status, body = call(t, admin, http.MethodPost, base+"/tokens", map[string]any{
		"name":          "AI 只读",
		"scopes":        map[string][]string{"tasks": {"read"}, "family": {"read"}},
		"expiresInDays": 30,
	})
	if status != http.StatusCreated {
		t.Fatalf("create token: %d %v", status, body)
	}
	raw := body["token"].(string)
	prefix := body["item"].(map[string]any)["prefix"].(string)
	if raw[:3] != "mm_" || prefix != raw[:10] {
		t.Fatalf("unexpected token prefix: %q %q", raw, prefix)
	}
	scopes := body["item"].(map[string]any)["scopes"].(map[string]any)
	if len(scopes["tasks"].([]any)) != 1 {
		t.Fatalf("unexpected scopes: %v", scopes)
	}

	status, body = callBearer(t, admin, http.MethodGet, base+"/tasks", raw, nil)
	if status != http.StatusOK {
		t.Fatalf("token read tasks: %d %v", status, body)
	}
	status, _ = callBearer(t, admin, http.MethodPost, base+"/tasks", raw, map[string]any{"title": "无权创建"})
	if status != http.StatusForbidden {
		t.Fatalf("token write tasks should be forbidden, got %d", status)
	}
	status, _ = callBearer(t, admin, http.MethodGet, base+"/household", raw, nil)
	if status != http.StatusOK {
		t.Fatalf("token read family: %d", status)
	}
	status, _ = callBearer(t, admin, http.MethodPatch, base+"/household", raw, map[string]any{"name": "改名"})
	if status != http.StatusForbidden {
		t.Fatalf("token write family should be forbidden, got %d", status)
	}
	status, _ = callBearer(t, admin, http.MethodGet, base+"/tokens", raw, nil)
	if status != http.StatusForbidden {
		t.Fatalf("token should not manage tokens, got %d", status)
	}

	// 删除权限需要显式授予。
	status, body = call(t, admin, http.MethodPost, base+"/tokens", map[string]any{
		"name":   "AI 可删除",
		"scopes": map[string][]string{"tasks": {"read", "delete"}},
	})
	if status != http.StatusCreated {
		t.Fatalf("create delete token: %d %v", status, body)
	}
	deleteRaw := body["token"].(string)
	status, _ = callBearer(t, admin, http.MethodDelete, base+"/tasks/"+formatID(int64(taskID)), deleteRaw, nil)
	if status != http.StatusNoContent {
		t.Fatalf("token delete task: %d", status)
	}

	// 撤销后立即失效。
	status, body = call(t, admin, http.MethodGet, base+"/tokens", nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 2 {
		t.Fatalf("list tokens: %d %v", status, body)
	}
	var revokeID int64
	for _, rawItem := range body["items"].([]any) {
		item := rawItem.(map[string]any)
		if item["prefix"] == prefix {
			revokeID = int64(item["id"].(float64))
		}
	}
	if revokeID == 0 {
		t.Fatalf("token to revoke not found: %v", body)
	}
	status, _ = call(t, admin, http.MethodDelete, base+"/tokens/"+formatID(revokeID), nil)
	if status != http.StatusNoContent {
		t.Fatalf("revoke token: %d", status)
	}
	status, _ = callBearer(t, admin, http.MethodGet, base+"/tasks", raw, nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("revoked token should be rejected, got %d", status)
	}

	status, _ = call(t, admin, http.MethodPost, base+"/tokens", map[string]any{
		"name": "bad", "scopes": map[string][]string{"tasks": {"read"}, "bogus": {"read"}},
	})
	if status != http.StatusBadRequest {
		t.Fatalf("invalid scope should fail, got %d", status)
	}
}

func TestSearch(t *testing.T) {
	base, admin, _ := setupHousehold(t)
	call(t, admin, http.MethodPost, base+"/tasks", map[string]any{"title": "买生日蛋糕"})
	call(t, admin, http.MethodPost, base+"/notes", map[string]any{"title": "蛋糕口味", "body": "草莓奶油"})
	call(t, admin, http.MethodPost, base+"/contacts", map[string]any{"name": "蛋糕店老板"})
	call(t, admin, http.MethodPost, base+"/messages", map[string]any{"body": "记得订生日蛋糕"})

	status, body := call(t, admin, http.MethodGet, base+"/search?q=%E8%9B%8B%E7%B3%95", nil)
	if status != http.StatusOK {
		t.Fatalf("search: %d %v", status, body)
	}
	items := body["items"].([]any)
	if len(items) < 4 {
		t.Fatalf("expected matches across modules, got %v", items)
	}
	types := map[string]bool{}
	for _, raw := range items {
		types[raw.(map[string]any)["type"].(string)] = true
	}
	for _, want := range []string{"task", "note", "contact", "message"} {
		if !types[want] {
			t.Fatalf("missing %s in search results: %v", want, types)
		}
	}

	status, body = call(t, admin, http.MethodGet, base+"/search?q=", nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 0 {
		t.Fatalf("empty search: %d %v", status, body)
	}
}
