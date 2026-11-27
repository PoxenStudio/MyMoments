package httpapi

import (
	"net/http"
	"testing"
	"time"
)

func TestCalendarEventCRUDAndVisibility(t *testing.T) {
	base, admin, _ := setupHousehold(t)
	today := todayString()

	status, body := call(t, admin, http.MethodPost, base+"/events", map[string]any{
		"title":      "家庭聚餐",
		"startLocal": today + " 18:00",
		"endLocal":   today + " 20:00",
		"location":   "家里",
		"color":      "#a16207",
	})
	if status != http.StatusCreated {
		t.Fatalf("create event: %d %v", status, body)
	}
	event := body["event"].(map[string]any)
	eventID := int64(event["id"].(float64))
	if event["startLocal"] != today+" 18:00:00" || event["occurrenceDate"] != today {
		t.Fatalf("unexpected event: %v", event)
	}

	status, body = call(t, admin, http.MethodGet, base+"/events?from="+today+"&to="+today, nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 1 {
		t.Fatalf("list events: %d %v", status, body)
	}

	status, body = call(t, admin, http.MethodPatch, base+"/events/"+itoa(eventID), map[string]any{
		"title": "家庭聚餐（改）", "endLocal": today + " 21:00",
	})
	if status != http.StatusOK {
		t.Fatalf("patch event: %d %v", status, body)
	}
	if body["event"].(map[string]any)["title"] != "家庭聚餐（改）" {
		t.Fatalf("title not updated: %v", body)
	}

	status, body = call(t, admin, http.MethodPost, base+"/events", map[string]any{
		"title": "私密安排", "startLocal": today + " 09:00", "visibility": "private",
	})
	if status != http.StatusCreated {
		t.Fatalf("create private: %d %v", status, body)
	}
	privateID := int64(body["event"].(map[string]any)["id"].(float64))

	member, _ := addMember(t, base, admin, "xiaoli", "小李")
	status, body = call(t, member, http.MethodGet, base+"/events?from="+today+"&to="+today, nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 1 {
		t.Fatalf("member should only see household event: %d %v", status, body)
	}
	status, _ = call(t, member, http.MethodGet, base+"/events/"+itoa(privateID), nil)
	if status != http.StatusNotFound {
		t.Fatalf("member private event should 404: %d", status)
	}

	status, _ = call(t, admin, http.MethodDelete, base+"/events/"+itoa(eventID), nil)
	if status != http.StatusNoContent {
		t.Fatalf("delete event: %d", status)
	}
}

func TestEventRecurrenceExceptions(t *testing.T) {
	base, admin, _ := setupHousehold(t)
	today := todayString()
	second := addDays(today, 7)
	third := addDays(today, 14)
	until := addDays(today, 28)

	status, body := call(t, admin, http.MethodPost, base+"/events", map[string]any{
		"title":           "每周读书会",
		"startLocal":      today + " 19:00",
		"endLocal":        today + " 20:30",
		"recurrenceRule":  `{"freq":"weekly","interval":1,"anchor":"due"}`,
		"reminderOffsets": []int{30},
	})
	if status != http.StatusCreated {
		t.Fatalf("create series: %d %v", status, body)
	}
	event := body["event"].(map[string]any)
	eventID := int64(event["id"].(float64))
	if event["seriesId"] == nil {
		t.Fatalf("series id missing: %v", event)
	}

	listURL := base + "/events?from=" + today + "&to=" + until
	status, body = call(t, admin, http.MethodGet, listURL, nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 5 {
		t.Fatalf("expected 5 occurrences: %d %v", status, body)
	}

	status, _ = call(t, admin, http.MethodDelete, base+"/events/"+itoa(eventID)+"?scope=this&occurrence="+second, nil)
	if status != http.StatusNoContent {
		t.Fatalf("cancel occurrence: %d", status)
	}
	status, body = call(t, admin, http.MethodGet, listURL, nil)
	items := body["items"].([]any)
	if len(items) != 4 {
		t.Fatalf("cancel not applied: %d %v", status, body)
	}
	for _, raw := range items {
		if raw.(map[string]any)["occurrenceDate"] == second {
			t.Fatalf("cancelled occurrence still visible")
		}
	}

	status, body = call(t, admin, http.MethodPatch, base+"/events/"+itoa(eventID), map[string]any{
		"scope": "this", "occurrenceDate": third, "title": "读书会（改期）",
	})
	if status != http.StatusOK {
		t.Fatalf("override occurrence: %d %v", status, body)
	}
	status, body = call(t, admin, http.MethodGet, listURL, nil)
	found := false
	for _, raw := range body["items"].([]any) {
		item := raw.(map[string]any)
		if item["occurrenceDate"] == third && item["title"] == "读书会（改期）" {
			found = true
		}
	}
	if !found {
		t.Fatalf("override not applied: %v", body)
	}

	status, _ = call(t, admin, http.MethodDelete, base+"/events/"+itoa(eventID)+"?scope=all", nil)
	if status != http.StatusNoContent {
		t.Fatalf("delete series: %d", status)
	}
	status, body = call(t, admin, http.MethodGet, listURL, nil)
	if len(body["items"].([]any)) != 0 {
		t.Fatalf("series not removed: %v", body)
	}
}

func TestBirthdayLunarDates(t *testing.T) {
	base, admin, _ := setupHousehold(t)
	status, body := call(t, admin, http.MethodPost, base+"/birthdays", map[string]any{
		"name": "外婆",
		"dates": []map[string]any{
			{"calendar": "solar", "month": 2, "day": 29, "birthYear": 1940, "reminderOffset": 0},
			{"calendar": "lunar", "month": 1, "day": 1, "yearUnknown": true},
		},
	})
	if status != http.StatusCreated {
		t.Fatalf("create dual birthday: %d %v", status, body)
	}
	birthday := body["birthday"].(map[string]any)
	dates := birthday["dates"].([]any)
	if len(dates) != 2 {
		t.Fatalf("expected two dates: %v", birthday)
	}
	var lunarReminder any = "missing"
	for _, raw := range dates {
		item := raw.(map[string]any)
		if item["calendar"] == "lunar" {
			lunarReminder = item["reminderOffset"]
		}
	}
	if lunarReminder != nil {
		t.Fatalf("lunar reminder must be optional: %v", birthday)
	}

	status, body = call(t, admin, http.MethodGet, base+"/birthdays?from=2027-01-01&to=2027-12-31", nil)
	if status != http.StatusOK {
		t.Fatalf("list birthday occurrences: %d %v", status, body)
	}
	items := body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("unexpected birthdays: %v", body)
	}
	occurrences := items[0].(map[string]any)["occurrences"].([]any)
	foundSolar, foundLunar := false, false
	for _, raw := range occurrences {
		occ := raw.(map[string]any)
		switch occ["calendar"] {
		case "solar":
			foundSolar = occ["date"] == "2027-02-28"
		case "lunar":
			foundLunar = true
		}
	}
	if !foundSolar || !foundLunar {
		t.Fatalf("unexpected occurrences: %v", occurrences)
	}
}

func TestBirthdaysAndContacts(t *testing.T) {
	base, admin, _ := setupHousehold(t)
	today := todayString()
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Now().In(loc)
	month := int(now.Month())
	day := now.Day()

	status, body := call(t, admin, http.MethodPost, base+"/contacts", map[string]any{
		"name": "王老师", "organization": "实验小学", "category": "学校",
		"items": []map[string]any{
			{"kind": "phone", "label": "手机", "value": "13800000000"},
			{"kind": "email", "value": "wang@example.com"},
		},
	})
	if status != http.StatusCreated {
		t.Fatalf("create contact: %d %v", status, body)
	}
	contact := body["contact"].(map[string]any)
	contactID := int64(contact["id"].(float64))
	if len(contact["items"].([]any)) != 2 {
		t.Fatalf("contact items missing: %v", contact)
	}
	status, body = call(t, admin, http.MethodGet, base+"/contacts?q=实验", nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 1 {
		t.Fatalf("search contact: %d %v", status, body)
	}

	status, body = call(t, admin, http.MethodPost, base+"/birthdays", map[string]any{
		"name": "王老师",
		"dates": []map[string]any{
			{"calendar": "solar", "month": month, "day": day, "birthYear": 1980, "reminderOffset": 0},
		},
		"contactId": contactID, "note": "喜欢兰花",
	})
	if status != http.StatusCreated {
		t.Fatalf("create birthday: %d %v", status, body)
	}
	birthday := body["birthday"].(map[string]any)
	birthdayID := int64(birthday["id"].(float64))
	if birthday["daysUntil"].(float64) != 0 {
		t.Fatalf("expected birthday today: %v", birthday)
	}
	if birthday["age"] == nil {
		t.Fatalf("expected age: %v", birthday)
	}

	status, body = call(t, admin, http.MethodGet, base+"/birthdays?source=contact", nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 1 {
		t.Fatalf("list birthdays: %d %v", status, body)
	}

	status, body = call(t, admin, http.MethodGet, base+"/events?source=all&from="+today+"&to="+addDays(today, 31), nil)
	found := false
	for _, raw := range body["items"].([]any) {
		item := raw.(map[string]any)
		if item["source"] == "birthday" && item["allDay"] == true {
			found = true
		}
	}
	if !found {
		t.Fatalf("birthday calendar event missing: %d %v", status, body)
	}

	status, body = call(t, admin, http.MethodPatch, base+"/birthdays/"+itoa(birthdayID), map[string]any{
		"dates": []map[string]any{
			{"calendar": "solar", "month": month, "day": day, "yearUnknown": true, "birthYear": 0},
		},
	})
	if status != http.StatusOK {
		t.Fatalf("patch birthday: %d %v", status, body)
	}
	if body["birthday"].(map[string]any)["age"] != nil {
		t.Fatalf("unknown year must not expose age: %v", body)
	}

	status, _ = call(t, admin, http.MethodDelete, base+"/birthdays/"+itoa(birthdayID), nil)
	if status != http.StatusNoContent {
		t.Fatalf("delete birthday: %d", status)
	}
	status, body = call(t, admin, http.MethodGet, base+"/events?source=all&from="+today+"&to="+addDays(today, 31), nil)
	if len(body["items"].([]any)) != 0 {
		t.Fatalf("birthday event not removed: %v", body)
	}

	status, body = call(t, admin, http.MethodPost, base+"/contacts", map[string]any{
		"name": "私人联系人", "ownerScope": "private",
	})
	if status != http.StatusCreated {
		t.Fatalf("create private contact: %d %v", status, body)
	}
	privateID := int64(body["contact"].(map[string]any)["id"].(float64))
	member, _ := addMember(t, base, admin, "xiaozhang", "小张")
	status, body = call(t, member, http.MethodGet, base+"/contacts", nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 1 {
		t.Fatalf("member should only see household contact: %d %v", status, body)
	}
	status, _ = call(t, member, http.MethodGet, base+"/contacts/"+itoa(privateID), nil)
	if status != http.StatusNotFound {
		t.Fatalf("private contact should 404: %d", status)
	}

	status, body = call(t, admin, http.MethodGet, base+"/contacts", nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 2 {
		t.Fatalf("admin should see both contacts: %d %v", status, body)
	}
}

func addDays(date string, days int) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, days).Format("2006-01-02")
}
