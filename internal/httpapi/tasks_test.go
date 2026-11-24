package httpapi

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
	"time"
)

func setupHousehold(t *testing.T) (string, *http.Client, int64) {
	t.Helper()
	ts, admin := newTestServer(t)
	base := ts.URL + "/api/v1"
	status, body := call(t, admin, http.MethodPost, base+"/setup", map[string]any{
		"username": "admin", "displayName": "管理员", "password": "password12345",
	})
	if status != http.StatusCreated {
		t.Fatalf("setup: %d %v", status, body)
	}
	user := body["user"].(map[string]any)
	return base, admin, int64(user["id"].(float64))
}

func addMember(t *testing.T, base string, admin *http.Client, username, display string) (*http.Client, int64) {
	t.Helper()
	status, body := call(t, admin, http.MethodPost, base+"/family/members", map[string]any{
		"username": username, "displayName": display, "relation": "家人",
	})
	if status != http.StatusCreated {
		t.Fatalf("create member: %d %v", status, body)
	}
	temp, _ := body["temporaryPassword"].(string)
	user := body["user"].(map[string]any)
	id := int64(user["id"].(float64))
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	status, body = call(t, client, http.MethodPost, base+"/auth/login", map[string]any{"username": username, "password": temp})
	if status != http.StatusOK {
		t.Fatalf("member login: %d %v", status, body)
	}
	status, _ = call(t, client, http.MethodPost, base+"/auth/password", map[string]any{
		"currentPassword": temp, "newPassword": "newpassword123",
	})
	if status != http.StatusNoContent {
		t.Fatalf("change password: %d", status)
	}
	return client, id
}

func todayString() string {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return time.Now().In(loc).Format("2006-01-02")
}

func tomorrowString() string {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return time.Now().In(loc).AddDate(0, 0, 1).Format("2006-01-02")
}

func TestTaskRecurrenceCompleteAndUndo(t *testing.T) {
	base, admin, adminID := setupHousehold(t)
	today := todayString()

	status, body := call(t, admin, http.MethodPost, base+"/tasks", map[string]any{
		"title":           "浇花",
		"dueDate":         today,
		"priority":        "high",
		"recurrenceRule":  `{"freq":"daily","interval":1,"anchor":"due"}`,
		"assigneeIds":     []int64{adminID},
		"subtasks":        []map[string]any{{"title": "阳台"}, {"title": "客厅"}},
		"reminderOffsets": []int{30},
	})
	if status != http.StatusCreated {
		t.Fatalf("create task: %d %v", status, body)
	}
	task := body["task"].(map[string]any)
	taskID := int64(task["id"].(float64))
	if task["seriesId"] == nil || task["occurrenceDate"] != today {
		t.Fatalf("unexpected recurrence fields: %v", task)
	}
	subs := task["subtasks"].([]any)
	if len(subs) != 2 {
		t.Fatalf("expected 2 subtasks: %v", subs)
	}

	status, body = call(t, admin, http.MethodGet, base+"/tasks?scope=open", nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 1 {
		t.Fatalf("list open: %d %v", status, body)
	}

	status, body = call(t, admin, http.MethodPost, base+"/tasks/"+itoa(taskID)+"/completion", map[string]any{"occurrenceDate": today})
	if status != http.StatusOK {
		t.Fatalf("complete: %d %v", status, body)
	}
	completed := body["task"].(map[string]any)
	if completed["status"] != "completed" {
		t.Fatalf("expected completed: %v", completed)
	}
	completion := completed["completion"].(map[string]any)
	if completion["doneByUserId"] == nil || completion["actualUserId"] == nil {
		t.Fatalf("completion should record both users: %v", completion)
	}
	next := body["nextTask"].(map[string]any)
	if next["dueDate"] != tomorrowString() {
		t.Fatalf("expected next due %s, got %v", tomorrowString(), next["dueDate"])
	}

	status, body = call(t, admin, http.MethodPost, base+"/tasks/"+itoa(taskID)+"/completion", map[string]any{"occurrenceDate": today})
	if status != http.StatusConflict {
		t.Fatalf("duplicate completion should conflict: %d %v", status, body)
	}

	status, body = call(t, admin, http.MethodDelete, base+"/tasks/"+itoa(taskID)+"/completion", nil)
	if status != http.StatusOK {
		t.Fatalf("undo: %d %v", status, body)
	}
	if body["nextTaskRemoved"] != true {
		t.Fatalf("expected next task removed: %v", body)
	}
	restored := body["task"].(map[string]any)
	if restored["status"] != "todo" || restored["completion"] != nil {
		t.Fatalf("unexpected restored task: %v", restored)
	}
	status, body = call(t, admin, http.MethodGet, base+"/tasks?scope=open", nil)
	if len(body["items"].([]any)) != 1 {
		t.Fatalf("expected single open instance: %v", body)
	}
}

func TestTaskSubtaskAndTemplate(t *testing.T) {
	base, admin, adminID := setupHousehold(t)

	status, body := call(t, admin, http.MethodPost, base+"/templates", map[string]any{
		"name":               "周末采购",
		"title":              "去超市",
		"priority":           "low",
		"defaultDueTime":     "10:00",
		"defaultAssigneeIds": []int64{adminID},
		"subtaskTemplate":    []string{"列清单", "买水果"},
	})
	if status != http.StatusCreated {
		t.Fatalf("create template: %d %v", status, body)
	}
	tpl := body["template"].(map[string]any)
	tplID := int64(tpl["id"].(float64))

	status, body = call(t, admin, http.MethodPost, base+"/tasks", map[string]any{"templateId": tplID, "dueDate": todayString()})
	if status != http.StatusCreated {
		t.Fatalf("create from template: %d %v", status, body)
	}
	task := body["task"].(map[string]any)
	if task["title"] != "去超市" || task["priority"] != "low" || task["dueTime"] != "10:00" {
		t.Fatalf("template defaults not applied: %v", task)
	}
	subs := task["subtasks"].([]any)
	if len(subs) != 2 {
		t.Fatalf("expected template subtasks: %v", subs)
	}
	taskID := int64(task["id"].(float64))
	subID := int64(subs[0].(map[string]any)["id"].(float64))

	status, body = call(t, admin, http.MethodPatch, base+"/tasks/"+itoa(taskID)+"/subtasks/"+itoa(subID), map[string]any{"done": true})
	if status != http.StatusOK {
		t.Fatalf("toggle subtask: %d %v", status, body)
	}
	updated := body["task"].(map[string]any)
	if updated["subtasks"].([]any)[0].(map[string]any)["done"] != true {
		t.Fatalf("subtask not done: %v", updated)
	}
}

func TestTaskVisibility(t *testing.T) {
	base, admin, adminID := setupHousehold(t)
	member, memberID := addMember(t, base, admin, "xiaoming", "小明")

	status, body := call(t, admin, http.MethodPost, base+"/tasks", map[string]any{
		"title": "私密日记", "visibility": "private",
	})
	if status != http.StatusCreated {
		t.Fatalf("create private: %d %v", status, body)
	}
	privateID := int64(body["task"].(map[string]any)["id"].(float64))

	status, body = call(t, member, http.MethodGet, base+"/tasks", nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 0 {
		t.Fatalf("member should not see private task: %d %v", status, body)
	}
	status, _ = call(t, member, http.MethodGet, base+"/tasks/"+itoa(privateID), nil)
	if status != http.StatusNotFound {
		t.Fatalf("member get private should 404: %d", status)
	}
	status, _ = call(t, member, http.MethodPost, base+"/tasks/"+itoa(privateID)+"/completion", nil)
	if status != http.StatusNotFound {
		t.Fatalf("member complete private should 404: %d", status)
	}

	status, body = call(t, admin, http.MethodPost, base+"/tasks", map[string]any{
		"title": "只给小明", "visibility": "selected", "assigneeIds": []int64{memberID},
	})
	if status != http.StatusCreated {
		t.Fatalf("create selected: %d %v", status, body)
	}
	selectedID := int64(body["task"].(map[string]any)["id"].(float64))
	status, body = call(t, member, http.MethodGet, base+"/tasks", nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 1 {
		t.Fatalf("member should see selected task: %d %v", status, body)
	}
	status, _ = call(t, member, http.MethodPost, base+"/tasks/"+itoa(selectedID)+"/completion", nil)
	if status != http.StatusOK {
		t.Fatalf("member should complete selected task: %d", status)
	}

	status, body = call(t, member, http.MethodPost, base+"/tasks", map[string]any{
		"title": "家务", "assigneeIds": []int64{adminID},
	})
	if status != http.StatusCreated {
		t.Fatalf("member create household task: %d %v", status, body)
	}
	memberTaskID := int64(body["task"].(map[string]any)["id"].(float64))
	status, _ = call(t, admin, http.MethodDelete, base+"/tasks/"+itoa(memberTaskID), nil)
	if status != http.StatusNoContent {
		t.Fatalf("admin should delete member task: %d", status)
	}
}

func TestTaskMediaUploadRangeAndPermission(t *testing.T) {
	base, admin, _ := setupHousehold(t)
	status, body := call(t, admin, http.MethodPost, base+"/tasks", map[string]any{"title": "成长记录"})
	if status != http.StatusCreated {
		t.Fatalf("create task: %d %v", status, body)
	}
	taskID := int64(body["task"].(map[string]any)["id"].(float64))

	var img bytes.Buffer
	p := image.NewRGBA(image.Rect(0, 0, 40, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 40; x++ {
			p.Set(x, y, color.RGBA{R: 120, G: 80, B: 40, A: 255})
		}
	}
	if err := png.Encode(&img, p); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	var payload bytes.Buffer
	mw := multipart.NewWriter(&payload)
	part, err := mw.CreateFormFile("file", "moment.png")
	if err != nil {
		t.Fatalf("form file: %v", err)
	}
	if _, err := part.Write(img.Bytes()); err != nil {
		t.Fatalf("write form: %v", err)
	}
	mw.Close()

	req, _ := http.NewRequest(http.MethodPost, base+"/tasks/"+itoa(taskID)+"/media", &payload)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := admin.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("upload status: %d", res.StatusCode)
	}

	status, body = call(t, admin, http.MethodGet, base+"/tasks/"+itoa(taskID)+"/media", nil)
	if status != http.StatusOK {
		t.Fatalf("media list: %d %v", status, body)
	}
	items := body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected one media: %v", items)
	}
	mediaID := int64(items[0].(map[string]any)["id"].(float64))
	if items[0].(map[string]any)["storage_key"] != nil {
		t.Fatalf("storage key must not leak: %v", items[0])
	}

	req, _ = http.NewRequest(http.MethodGet, base+"/media/"+itoa(mediaID), nil)
	res, err = admin.Do(req)
	if err != nil {
		t.Fatalf("get media: %v", err)
	}
	data := make([]byte, 16)
	n, _ := res.Body.Read(data)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/png" || n == 0 {
		t.Fatalf("unexpected media get: %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}

	req, _ = http.NewRequest(http.MethodGet, base+"/media/"+itoa(mediaID), nil)
	req.Header.Set("Range", "bytes=0-9")
	res, err = admin.Do(req)
	if err != nil {
		t.Fatalf("range: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusPartialContent {
		t.Fatalf("expected 206, got %d", res.StatusCode)
	}

	status, _ = call(t, admin, http.MethodGet, base+"/media/"+itoa(mediaID)+"/thumbnail", nil)
	if status != http.StatusOK {
		t.Fatalf("thumbnail: %d", status)
	}

	member, _ := addMember(t, base, admin, "xiaohong", "小红")
	status, body = call(t, admin, http.MethodPost, base+"/tasks", map[string]any{
		"title": "私密照片", "visibility": "private",
	})
	privateID := int64(body["task"].(map[string]any)["id"].(float64))
	if status != http.StatusCreated {
		t.Fatalf("private create: %d", status)
	}
	status, _ = call(t, member, http.MethodPost, base+"/tasks/"+itoa(privateID)+"/media", nil)
	if status != http.StatusNotFound {
		t.Fatalf("member upload to private should 404: %d", status)
	}
}

func TestTaskValidation(t *testing.T) {
	base, admin, _ := setupHousehold(t)
	cases := []map[string]any{
		{"title": ""},
		{"title": "x", "dueTime": "10:00"},
		{"title": "x", "dueDate": "2026-13-01"},
		{"title": "x", "dueDate": "2026-11-24", "recurrenceRule": `{"freq":"hourly"}`},
		{"title": "x", "visibility": "world"},
	}
	for _, payload := range cases {
		status, _ := call(t, admin, http.MethodPost, base+"/tasks", payload)
		if status != http.StatusBadRequest && status != http.StatusUnprocessableEntity {
			t.Fatalf("expected validation error for %v, got %d", payload, status)
		}
	}
	status, body := call(t, admin, http.MethodPost, base+"/tasks", map[string]any{
		"title": "有效任务", "dueDate": "2026-11-24", "recurrenceRule": `{"freq":"weekly","interval":2,"weekdays":[1,3]}`,
	})
	if status != http.StatusCreated {
		t.Fatalf("valid task rejected: %d %v", status, body)
	}
	if !strings.Contains(body["task"].(map[string]any)["recurrenceRule"].(string), "weekly") {
		t.Fatalf("unexpected rule: %v", body)
	}
}
