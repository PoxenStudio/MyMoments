package httpapi

import (
	"net/http"
	"testing"
)

func TestTimelineAggregationAndCursor(t *testing.T) {
	base, admin, _ := setupHousehold(t)
	today := todayString()
	yesterday := addDays(today, -1)
	tomorrow := addDays(today, 1)

	status, body := call(t, admin, http.MethodPost, base+"/tasks", map[string]any{
		"title": "买菜", "dueDate": today,
	})
	if status != http.StatusCreated {
		t.Fatalf("create task: %d %v", status, body)
	}
	taskID := int64(body["task"].(map[string]any)["id"].(float64))
	if status, body = call(t, admin, http.MethodPost, base+"/tasks/"+itoa(taskID)+"/completion", map[string]any{}); status != http.StatusOK && status != http.StatusCreated {
		t.Fatalf("complete task: %d %v", status, body)
	}

	if status, body = call(t, admin, http.MethodPost, base+"/events", map[string]any{
		"title": "晚饭", "startLocal": today + " 18:00:00",
	}); status != http.StatusCreated {
		t.Fatalf("create event: %d %v", status, body)
	}
	if status, body = call(t, admin, http.MethodPost, base+"/events", map[string]any{
		"title": "明天出游", "startLocal": tomorrow + " 09:00:00",
	}); status != http.StatusCreated {
		t.Fatalf("create future event: %d %v", status, body)
	}
	if status, body = call(t, admin, http.MethodPost, base+"/messages", map[string]any{
		"body": "昨天散步", "occurrenceDate": yesterday,
	}); status != http.StatusCreated {
		t.Fatalf("create message: %d %v", status, body)
	}
	if status, body = call(t, admin, http.MethodPost, base+"/notes", map[string]any{
		"title": "菜谱", "memoryDate": today,
	}); status != http.StatusCreated {
		t.Fatalf("create note: %d %v", status, body)
	}

	status, body = call(t, admin, http.MethodGet, base+"/timeline?direction=past&limit=2", nil)
	if status != http.StatusOK {
		t.Fatalf("timeline past: %d %v", status, body)
	}
	items := body["items"].([]any)
	if len(items) != 2 || body["nextCursor"] == "" || body["hasMore"] != true {
		t.Fatalf("expected first page with cursor: %v", body)
	}
	seen := map[string]bool{}
	types := map[string]bool{}
	for _, raw := range items {
		entry := raw.(map[string]any)
		seen[entry["occurrenceKey"].(string)] = true
		types[entry["sourceType"].(string)] = true
		if entry["occurredOn"].(string) > today {
			t.Fatalf("past entry after anchor: %v", entry)
		}
	}
	cursor := body["nextCursor"].(string)
	status, body = call(t, admin, http.MethodGet, base+"/timeline?direction=past&limit=10&cursor="+cursor, nil)
	if status != http.StatusOK {
		t.Fatalf("timeline page 2: %d %v", status, body)
	}
	for _, raw := range body["items"].([]any) {
		entry := raw.(map[string]any)
		key := entry["occurrenceKey"].(string)
		if seen[key] {
			t.Fatalf("duplicate entry across pages: %s", key)
		}
		seen[key] = true
		types[entry["sourceType"].(string)] = true
	}
	for _, want := range []string{"task_completion", "event", "message", "note"} {
		if !types[want] {
			t.Fatalf("timeline missing %s: %v", want, types)
		}
	}

	status, body = call(t, admin, http.MethodGet, base+"/timeline?direction=future&types=event&anchor="+today, nil)
	if status != http.StatusOK {
		t.Fatalf("timeline future: %d %v", status, body)
	}
	future := body["items"].([]any)
	if len(future) != 2 {
		t.Fatalf("expected today and tomorrow events: %v", future)
	}
	if future[0].(map[string]any)["occurredOn"] != today || future[1].(map[string]any)["occurredOn"] != tomorrow {
		t.Fatalf("future events not ascending: %v", future)
	}

	status, _ = call(t, admin, http.MethodGet, base+"/timeline?cursor=not-a-cursor", nil)
	if status != http.StatusBadRequest {
		t.Fatalf("invalid cursor should fail: %d", status)
	}
}
