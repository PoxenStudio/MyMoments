package httpapi

import (
	"net/http"
	"testing"
)

func TestMessagesMentionsAndVisibility(t *testing.T) {
	base, admin, adminID := setupHousehold(t)
	member, memberID := addMember(t, base, admin, "xiaoli", "小李")
	other, _ := addMember(t, base, admin, "xiaowang", "小王")
	today := todayString()

	status, body := call(t, admin, http.MethodPost, base+"/messages", map[string]any{
		"body":           "周末一起做饭 @小李",
		"occurrenceDate": today,
		"mentions":       []map[string]any{{"type": "user", "id": memberID}},
	})
	if status != http.StatusCreated {
		t.Fatalf("create message: %d %v", status, body)
	}
	post := body["message"].(map[string]any)
	postID := int64(post["id"].(float64))
	if post["occurrenceDate"] != today || post["replyCount"].(float64) != 0 {
		t.Fatalf("unexpected post: %v", post)
	}

	status, body = call(t, member, http.MethodGet, base+"/notifications", nil)
	if status != http.StatusOK {
		t.Fatalf("member notifications: %d %v", status, body)
	}
	items := body["items"].([]any)
	if len(items) != 1 || body["unread"].(float64) != 1 {
		t.Fatalf("expected one unread mention notification: %v", body)
	}
	notif := items[0].(map[string]any)
	if notif["sourceType"] != "message" || int64(notif["sourceId"].(float64)) != postID {
		t.Fatalf("unexpected notification: %v", notif)
	}
	notifID := int64(notif["id"].(float64))
	if status, _ = call(t, member, http.MethodPatch, base+"/notifications/"+itoa(notifID)+"/read", nil); status != http.StatusNoContent {
		t.Fatalf("mark read: %d", status)
	}
	status, body = call(t, member, http.MethodGet, base+"/notifications", nil)
	if status != http.StatusOK || body["unread"].(float64) != 0 {
		t.Fatalf("unread should be zero: %d %v", status, body)
	}

	status, body = call(t, other, http.MethodGet, base+"/messages", nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 1 {
		t.Fatalf("household message visible to all: %d %v", status, body)
	}

	status, body = call(t, member, http.MethodPost, base+"/messages/"+itoa(postID)+"/replies", map[string]any{
		"body": "好的 @管理员", "mentions": []map[string]any{{"type": "user", "id": adminID}},
	})
	if status != http.StatusCreated {
		t.Fatalf("create reply: %d %v", status, body)
	}
	status, body = call(t, admin, http.MethodGet, base+"/notifications", nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 1 {
		t.Fatalf("admin should receive reply mention: %d %v", status, body)
	}
	status, body = call(t, member, http.MethodGet, base+"/messages/"+itoa(postID), nil)
	if status != http.StatusOK || body["message"].(map[string]any)["replyCount"].(float64) != 1 {
		t.Fatalf("reply not attached: %d %v", status, body)
	}

	status, body = call(t, admin, http.MethodPost, base+"/messages", map[string]any{
		"body": "只给自己看 @小李", "visibility": "private", "mentions": []map[string]any{{"type": "user", "id": memberID}},
	})
	if status != http.StatusCreated {
		t.Fatalf("create private: %d %v", status, body)
	}
	privateID := int64(body["message"].(map[string]any)["id"].(float64))
	status, _ = call(t, member, http.MethodGet, base+"/messages/"+itoa(privateID), nil)
	if status != http.StatusNotFound {
		t.Fatalf("private message should 404: %d", status)
	}
	status, body = call(t, member, http.MethodGet, base+"/notifications", nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 1 {
		t.Fatalf("private mention must not notify: %d %v", status, body)
	}

	status, body = call(t, admin, http.MethodPost, base+"/messages", map[string]any{
		"body": "指定小李可见", "visibility": "selected", "audienceIds": []int64{memberID},
	})
	if status != http.StatusCreated {
		t.Fatalf("create selected: %d %v", status, body)
	}
	selectedID := int64(body["message"].(map[string]any)["id"].(float64))
	status, _ = call(t, member, http.MethodGet, base+"/messages/"+itoa(selectedID), nil)
	if status != http.StatusOK {
		t.Fatalf("audience member should read selected: %d", status)
	}
	status, _ = call(t, other, http.MethodGet, base+"/messages/"+itoa(selectedID), nil)
	if status != http.StatusNotFound {
		t.Fatalf("non-audience should not read selected: %d", status)
	}

	status, _ = call(t, admin, http.MethodPost, base+"/messages", map[string]any{
		"body": "未来补记", "occurrenceDate": tomorrowString(),
	})
	if status != http.StatusBadRequest {
		t.Fatalf("future occurrence date should fail: %d", status)
	}

	status, _ = call(t, other, http.MethodDelete, base+"/messages/"+itoa(postID), nil)
	if status != http.StatusForbidden {
		t.Fatalf("non-author cannot delete: %d", status)
	}
	status, _ = call(t, admin, http.MethodDelete, base+"/messages/"+itoa(postID), nil)
	if status != http.StatusNoContent {
		t.Fatalf("author delete: %d", status)
	}
	status, body = call(t, member, http.MethodGet, base+"/notifications", nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 0 {
		t.Fatalf("notifications should be removed with message: %d %v", status, body)
	}
}

func TestNotesChecklistVisibilityAndMentions(t *testing.T) {
	base, admin, adminID := setupHousehold(t)
	member, memberID := addMember(t, base, admin, "xiaoli", "小李")
	other, _ := addMember(t, base, admin, "xiaowang", "小王")
	today := todayString()

	status, body := call(t, admin, http.MethodPost, base+"/notes", map[string]any{
		"title":      "旅行清单",
		"body":       "记得带 @小李 的相机",
		"color":      "sand",
		"category":   "出行",
		"memoryDate": today,
		"mentions":   []map[string]any{{"type": "user", "id": memberID}},
		"items": []map[string]any{
			{"text": "充电器", "checked": false},
			{"text": "相机", "checked": true},
		},
	})
	if status != http.StatusCreated {
		t.Fatalf("create note: %d %v", status, body)
	}
	note := body["note"].(map[string]any)
	noteID := int64(note["id"].(float64))
	if note["memoryDate"] != today || len(note["items"].([]any)) != 2 {
		t.Fatalf("unexpected note: %v", note)
	}

	status, body = call(t, member, http.MethodGet, base+"/notes?q=旅行", nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 1 {
		t.Fatalf("member should see note: %d %v", status, body)
	}
	status, body = call(t, member, http.MethodGet, base+"/notifications", nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 1 {
		t.Fatalf("note mention should notify member: %d %v", status, body)
	}

	status, body = call(t, admin, http.MethodPatch, base+"/notes/"+itoa(noteID), map[string]any{
		"items": []map[string]any{
			{"text": "充电器", "checked": true},
			{"text": "相机", "checked": true},
			{"text": "雨伞", "checked": false},
		},
		"pinned": true,
	})
	if status != http.StatusOK {
		t.Fatalf("patch note: %d %v", status, body)
	}
	updated := body["note"].(map[string]any)
	if len(updated["items"].([]any)) != 3 || updated["pinned"] != true {
		t.Fatalf("note not updated: %v", updated)
	}

	status, body = call(t, admin, http.MethodPost, base+"/notes", map[string]any{
		"title": "私密日记", "visibility": "private",
	})
	if status != http.StatusCreated {
		t.Fatalf("create private note: %d %v", status, body)
	}
	privateID := int64(body["note"].(map[string]any)["id"].(float64))
	status, _ = call(t, member, http.MethodGet, base+"/notes/"+itoa(privateID), nil)
	if status != http.StatusNotFound {
		t.Fatalf("private note should 404: %d", status)
	}

	status, body = call(t, admin, http.MethodPost, base+"/notes", map[string]any{
		"title": "给管理员的通知", "visibility": "selected", "audienceIds": []int64{adminID},
		"body": "自己看",
	})
	if status != http.StatusCreated {
		t.Fatalf("create selected note: %d %v", status, body)
	}
	selectedID := int64(body["note"].(map[string]any)["id"].(float64))
	status, _ = call(t, other, http.MethodGet, base+"/notes/"+itoa(selectedID), nil)
	if status != http.StatusNotFound {
		t.Fatalf("non-audience cannot read selected note: %d", status)
	}

	status, _ = call(t, other, http.MethodDelete, base+"/notes/"+itoa(noteID), nil)
	if status != http.StatusForbidden {
		t.Fatalf("non-author cannot delete note: %d", status)
	}
	if status, _ = call(t, admin, http.MethodDelete, base+"/notes/"+itoa(noteID), nil); status != http.StatusNoContent {
		t.Fatalf("delete note: %d", status)
	}
	status, body = call(t, member, http.MethodGet, base+"/notifications", nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 0 {
		t.Fatalf("note notification should be removed: %d %v", status, body)
	}
}
