package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"mymoments/internal/holiday"
	"mymoments/internal/model"
	"mymoments/internal/recur"
	"mymoments/internal/store"
)

type contactItemInput struct {
	Kind          string  `json:"kind"`
	Label         *string `json:"label"`
	Value         string  `json:"value"`
	IsPrimaryCall *bool   `json:"isPrimaryCall"`
	IsPrimarySms  *bool   `json:"isPrimarySms"`
}

type contactRelationInput struct {
	RelatedContactID int64   `json:"relatedContactId"`
	Relation         *string `json:"relation"`
}

type contactReminderInput struct {
	Kind      string  `json:"kind"`
	Title     *string `json:"title"`
	Frequency *string `json:"frequency"`
	DueDate   *string `json:"dueDate"`
	Color     *string `json:"color"`
	Enabled   *bool   `json:"enabled"`
}

type contactInput struct {
	Name           *string                 `json:"name"`
	Organization   *string                 `json:"organization"`
	Title          *string                 `json:"title"`
	Note           *string                 `json:"note"`
	BirthdayMonth  *int                    `json:"birthdayMonth"`
	BirthdayDay    *int                    `json:"birthdayDay"`
	BirthYear      *int                    `json:"birthYear"`
	ShowInCalendar *bool                   `json:"showInCalendar"`
	Category       *string                 `json:"category"`
	AvatarKey      *string                 `json:"avatarKey"`
	Wechat         *string                 `json:"wechat"`
	OfficePhone    *string                 `json:"officePhone"`
	OwnerScope     *string                 `json:"ownerScope"`
	Items          *[]contactItemInput     `json:"items"`
	Relations      *[]contactRelationInput `json:"relations"`
	Reminders      *[]contactReminderInput `json:"reminders"`
}

func (s *Server) handleContactsList(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	q := r.URL.Query()
	items, err := s.store.ListContacts(r.Context(), store.ContactFilter{
		ViewerID: u.ID,
		Query:    q.Get("q"),
		Category: q.Get("category"),
	})
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleContactGet(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	c, ok := s.loadContactFor(r, w)
	if !ok {
		return
	}
	if !canReadContact(u, c) {
		writeError(w, notFound("联系人不存在"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"contact": c})
}

func (s *Server) handleContactCreate(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	var in contactInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	c := &model.Contact{OwnerScope: "household", CreatedBy: u.ID, Items: []model.ContactItem{}, Relations: []model.ContactRelation{}, Reminders: []model.ContactReminder{}}
	if apiErr := applyContactInput(c, &in, true); apiErr != nil {
		writeError(w, apiErr)
		return
	}
	id, err := s.store.CreateContact(r.Context(), c)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	c.ID = id
	if err := s.syncContactBirthday(r.Context(), c); err != nil {
		writeError(w, serverError(err))
		return
	}
	if err := s.syncContactReminderTasks(r.Context(), c); err != nil {
		writeError(w, serverError(err))
		return
	}
	created, err := s.store.GetContact(r.Context(), id)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"contact": created})
}

func (s *Server) handleContactPatch(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	c, ok := s.loadContactFor(r, w)
	if !ok {
		return
	}
	if !canReadContact(u, c) {
		writeError(w, notFound("联系人不存在"))
		return
	}
	if !canWriteContact(u, c) {
		writeError(w, forbidden("没有修改该联系人的权限"))
		return
	}
	var in contactInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if apiErr := applyContactInput(c, &in, false); apiErr != nil {
		writeError(w, apiErr)
		return
	}
	if err := s.store.UpdateContact(r.Context(), c); err != nil {
		writeError(w, serverError(err))
		return
	}
	if err := s.syncContactBirthday(r.Context(), c); err != nil {
		writeError(w, serverError(err))
		return
	}
	if err := s.syncContactReminderTasks(r.Context(), c); err != nil {
		writeError(w, serverError(err))
		return
	}
	updated, err := s.store.GetContact(r.Context(), c.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"contact": updated})
}

type contactPinInput struct {
	Pinned *bool `json:"pinned"`
}

func (s *Server) handleContactPin(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	c, ok := s.loadContactFor(r, w)
	if !ok {
		return
	}
	if !canReadContact(u, c) {
		writeError(w, notFound("联系人不存在"))
		return
	}
	if !canWriteContact(u, c) {
		writeError(w, forbidden("没有修改该联系人的权限"))
		return
	}
	var in contactPinInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if in.Pinned == nil {
		writeError(w, badRequest("缺少置顶状态"))
		return
	}
	if err := s.store.SetContactPinned(r.Context(), c.ID, *in.Pinned); err != nil {
		writeError(w, serverError(err))
		return
	}
	c.Pinned = *in.Pinned
	writeJSON(w, http.StatusOK, map[string]any{"contact": c})
}

func (s *Server) handleContactDelete(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	c, ok := s.loadContactFor(r, w)
	if !ok {
		return
	}
	if !canReadContact(u, c) {
		writeError(w, notFound("联系人不存在"))
		return
	}
	if !canDeleteContact(u, c) {
		writeError(w, forbidden("只有创建者或管理员可以删除该联系人"))
		return
	}
	if err := s.deleteContactDerived(r.Context(), c); err != nil {
		writeError(w, serverError(err))
		return
	}
	if err := s.store.DeleteContact(r.Context(), c.ID); err != nil {
		writeError(w, serverError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleContactAssociations 返回联系人被 @ 的关联数量，供删除确认弹框展示。
func (s *Server) handleContactAssociations(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	c, ok := s.loadContactFor(r, w)
	if !ok {
		return
	}
	if !canReadContact(u, c) {
		writeError(w, notFound("联系人不存在"))
		return
	}
	counts, err := s.store.ContactMentionCounts(r.Context(), c.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"counts": counts, "hasAssociations": counts.Total() > 0})
}

type blockedContact struct {
	ID     int64                      `json:"id"`
	Name   string                     `json:"name"`
	Counts store.ContactMentionCounts `json:"counts"`
}

// handleContactsCleanup 删除所有「无任何 @ 关联」的联系人；存在任一关联则整体拦截。
func (s *Server) handleContactsCleanup(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	contacts, err := s.store.ListContacts(r.Context(), store.ContactFilter{ViewerID: u.ID})
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	deletable := make([]model.Contact, 0)
	blocked := make([]blockedContact, 0)
	for _, c := range contacts {
		if !canDeleteContact(u, &c) {
			continue
		}
		counts, err := s.store.ContactMentionCounts(r.Context(), c.ID)
		if err != nil {
			writeError(w, serverError(err))
			return
		}
		if counts.Total() > 0 {
			blocked = append(blocked, blockedContact{ID: c.ID, Name: c.Name, Counts: counts})
			continue
		}
		deletable = append(deletable, c)
	}
	if len(blocked) > 0 {
		writeJSON(w, http.StatusOK, map[string]any{"deleted": 0, "blocked": blocked})
		return
	}
	for _, c := range deletable {
		if err := s.deleteContactDerived(r.Context(), &c); err != nil {
			writeError(w, serverError(err))
			return
		}
		if err := s.store.DeleteContact(r.Context(), c.ID); err != nil {
			writeError(w, serverError(err))
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": len(deletable), "blocked": blocked})
}

// deleteContactDerived 清理联系人衍生的问候待办、生日与日历事件/提醒。
func (s *Server) deleteContactDerived(ctx context.Context, c *model.Contact) error {
	if err := s.store.DeleteTasksBySeriesPrefix(ctx, fmt.Sprintf("contact-greet-%d-", c.ID)); err != nil {
		return err
	}
	b, err := s.store.GetBirthdayByContact(ctx, c.ID)
	if err == nil && b != nil {
		for _, d := range b.Dates {
			if d.CalendarEventID != nil {
				_ = s.store.DeleteEvent(ctx, *d.CalendarEventID)
				_ = s.store.DeleteSourceReminders(ctx, "event", *d.CalendarEventID)
			}
		}
		_ = s.store.DeleteSourceReminders(ctx, "birthday", b.ID)
		_ = s.store.DeleteBirthday(ctx, b.ID)
	}
	return nil
}

func (s *Server) loadContactFor(r *http.Request, w http.ResponseWriter) (*model.Contact, bool) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return nil, false
	}
	c, err := s.store.GetContact(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, notFound("联系人不存在"))
			return nil, false
		}
		writeError(w, serverError(err))
		return nil, false
	}
	return c, true
}

func applyContactInput(c *model.Contact, in *contactInput, create bool) *APIError {
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" || len([]rune(name)) > 80 {
			return badRequest("请填写 1–80 个字符的姓名")
		}
		c.Name = name
	} else if create {
		return badRequest("请填写联系人姓名")
	}
	if in.Organization != nil {
		c.Organization = nonEmptyPtr(*in.Organization)
	}
	if in.Title != nil {
		c.Title = nonEmptyPtr(*in.Title)
	}
	if in.Note != nil {
		note := strings.TrimSpace(*in.Note)
		if len([]rune(note)) > 2000 {
			return badRequest("备注过长")
		}
		c.Note = nonEmptyPtr(note)
	}
	if err := applyBirthday(c, in); err != nil {
		return err
	}
	if in.Category != nil {
		c.Category = nonEmptyPtr(*in.Category)
	}
	if in.AvatarKey != nil {
		c.AvatarKey = nonEmptyPtr(*in.AvatarKey)
	}
	if in.Wechat != nil {
		c.Wechat = nonEmptyPtr(strings.TrimSpace(*in.Wechat))
	}
	if in.OfficePhone != nil {
		c.OfficePhone = nonEmptyPtr(strings.TrimSpace(*in.OfficePhone))
	}
	if in.OwnerScope != nil {
		if *in.OwnerScope != "household" && *in.OwnerScope != "private" {
			return badRequest("通讯录归属无效")
		}
		c.OwnerScope = *in.OwnerScope
	}
	if in.Items != nil {
		items, err := applyContactItems(*in.Items)
		if err != nil {
			return err
		}
		c.Items = items
	}
	if in.Relations != nil {
		c.Relations = applyContactRelations(*in.Relations)
	}
	if in.Reminders != nil {
		reminders, err := applyContactReminders(*in.Reminders)
		if err != nil {
			return err
		}
		c.Reminders = reminders
	}
	return nil
}

func applyBirthday(c *model.Contact, in *contactInput) *APIError {
	monthSet := in.BirthdayMonth != nil
	daySet := in.BirthdayDay != nil
	if monthSet != daySet {
		return badRequest("生日月份与日期需同时填写")
	}
	if monthSet {
		m, d := *in.BirthdayMonth, *in.BirthdayDay
		if m < 1 || m > 12 {
			return badRequest("生日月份需在 1–12 之间")
		}
		if d < 1 || d > 31 {
			return badRequest("生日日期无效")
		}
		last := time.Date(2024, time.Month(m)+1, 0, 0, 0, 0, 0, time.UTC).Day()
		if m == 2 {
			last = 29
		}
		if d > last {
			return badRequest("生日日期无效")
		}
		c.BirthdayMonth = in.BirthdayMonth
		c.BirthdayDay = in.BirthdayDay
	}
	if in.BirthYear != nil {
		if *in.BirthYear == 0 {
			c.BirthYear = nil
		} else {
			if *in.BirthYear < 1900 || *in.BirthYear > time.Now().Year() {
				return badRequest("出生年份无效")
			}
			c.BirthYear = in.BirthYear
		}
	}
	if in.ShowInCalendar != nil {
		c.ShowInCalendar = *in.ShowInCalendar
	}
	return nil
}

func applyContactItems(items []contactItemInput) ([]model.ContactItem, *APIError) {
	if len(items) > 50 {
		return nil, badRequest("联系项过多")
	}
	out := make([]model.ContactItem, 0, len(items))
	for _, it := range items {
		kind := strings.TrimSpace(it.Kind)
		if kind != "phone" && kind != "email" && kind != "address" {
			return nil, badRequest("联系项类型无效")
		}
		value := strings.TrimSpace(it.Value)
		if value == "" || len([]rune(value)) > 200 {
			return nil, badRequest("联系项内容不能为空且不超过 200 字")
		}
		if kind == "email" && !strings.Contains(value, "@") {
			return nil, badRequest("邮箱格式不正确")
		}
		item := model.ContactItem{Kind: kind, Label: nonEmptyPtr(valueOr(it.Label, "")), Value: value}
		if it.IsPrimaryCall != nil {
			item.IsPrimaryCall = *it.IsPrimaryCall
		}
		if it.IsPrimarySms != nil {
			item.IsPrimarySms = *it.IsPrimarySms
		}
		out = append(out, item)
	}
	return out, nil
}

func applyContactRelations(relations []contactRelationInput) []model.ContactRelation {
	out := make([]model.ContactRelation, 0, len(relations))
	for _, r := range relations {
		if r.RelatedContactID <= 0 {
			continue
		}
		relation := "emergency"
		if r.Relation != nil && strings.TrimSpace(*r.Relation) != "" {
			relation = strings.TrimSpace(*r.Relation)
		}
		out = append(out, model.ContactRelation{RelatedContactID: r.RelatedContactID, Relation: relation})
	}
	return out
}

func applyContactReminders(reminders []contactReminderInput) ([]model.ContactReminder, *APIError) {
	if len(reminders) > 30 {
		return nil, badRequest("提醒项过多")
	}
	out := make([]model.ContactReminder, 0, len(reminders))
	for _, r := range reminders {
		kind := strings.TrimSpace(r.Kind)
		if kind != "spring_festival" && kind != "traditional" && kind != "western" && kind != "custom" {
			return nil, badRequest("提醒类型无效")
		}
		rem := model.ContactReminder{Kind: kind, Enabled: true}
		if r.Enabled != nil {
			rem.Enabled = *r.Enabled
		}
		if r.Title != nil {
			rem.Title = strings.TrimSpace(*r.Title)
		}
		if r.Color != nil {
			rem.Color = nonEmptyPtr(strings.TrimSpace(*r.Color))
		}
		if kind == "custom" {
			freq := ""
			if r.Frequency != nil {
				freq = strings.TrimSpace(*r.Frequency)
			}
			if freq != "" && freq != "monthly" && freq != "weekly" && freq != "once" {
				return nil, badRequest("提醒频率无效")
			}
			rem.Frequency = freq
			if r.DueDate != nil {
				date := strings.TrimSpace(*r.DueDate)
				if date == "" {
					return nil, badRequest("请填写提醒日期")
				}
				if err := recur.ValidateDate(date); err != nil {
					return nil, badRequest("提醒日期格式需为 YYYY-MM-DD")
				}
				rem.DueDate = &date
			} else if rem.Enabled {
				return nil, badRequest("请填写提醒日期")
			}
			if rem.Frequency == "monthly" || rem.Frequency == "weekly" {
				raw := reminderRule(rem.Frequency)
				rem.RecurrenceRule = &raw
			}
		}
		out = append(out, rem)
	}
	return out, nil
}

func reminderRule(freq string) string {
	if freq == "weekly" {
		return `{"freq":"weekly","interval":1,"anchor":"due"}`
	}
	return `{"freq":"monthly","interval":1,"anchor":"due"}`
}

// syncContactBirthday 依据联系人生日与“显示到家庭日历”选项维护生日档案与日历事件。
// 联系人生日只写公历日期；若档案已有农历日期则保留。
func (s *Server) syncContactBirthday(ctx context.Context, c *model.Contact) error {
	existing, _ := s.store.GetBirthdayByContact(ctx, c.ID)
	hasBirthday := c.BirthdayMonth != nil && c.BirthdayDay != nil
	if !hasBirthday || !c.ShowInCalendar {
		if existing == nil {
			return nil
		}
		for _, d := range existing.Dates {
			if d.CalendarEventID != nil {
				_ = s.store.DeleteEvent(ctx, *d.CalendarEventID)
				_ = s.store.DeleteSourceReminders(ctx, "event", *d.CalendarEventID)
			}
		}
		_ = s.store.DeleteSourceReminders(ctx, "birthday", existing.ID)
		return s.store.DeleteBirthday(ctx, existing.ID)
	}
	solarOffset := 0
	var solarEventID *int64
	lunarDates := make([]model.BirthdayDate, 0, 1)
	if existing != nil {
		for _, d := range existing.Dates {
			if d.Calendar == "lunar" {
				lunarDates = append(lunarDates, d)
				continue
			}
			solarEventID = d.CalendarEventID
			if d.ReminderOffset != nil {
				solarOffset = *d.ReminderOffset
			}
		}
	}
	dates := []model.BirthdayDate{{
		Calendar:        "solar",
		Month:           *c.BirthdayMonth,
		Day:             *c.BirthdayDay,
		YearUnknown:     c.BirthYear == nil,
		BirthYear:       c.BirthYear,
		ReminderOffset:  &solarOffset,
		CalendarEventID: solarEventID,
	}}
	dates = append(dates, lunarDates...)
	b := &model.Birthday{
		Name:      c.Name,
		AvatarKey: c.AvatarKey,
		ContactID: &c.ID,
		CreatedBy: c.CreatedBy,
		Dates:     dates,
	}
	if existing != nil {
		b.ID = existing.ID
		b.CreatedBy = existing.CreatedBy
		if err := s.store.UpdateBirthday(ctx, b); err != nil {
			return err
		}
	} else {
		id, err := s.store.CreateBirthday(ctx, b)
		if err != nil {
			return err
		}
		b.ID = id
	}
	if err := s.syncBirthdayEvent(ctx, b); err != nil {
		return err
	}
	return s.syncBirthdayReminder(ctx, b)
}

// syncContactReminderTasks 依据联系人问候/提醒偏好生成对应待办。
func (s *Server) syncContactReminderTasks(ctx context.Context, c *model.Contact) error {
	if err := s.store.DeleteTasksBySeriesPrefix(ctx, fmt.Sprintf("contact-greet-%d-", c.ID)); err != nil {
		return err
	}
	loc, err := s.location(ctx)
	if err != nil {
		return err
	}
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	for i, r := range c.Reminders {
		if !r.Enabled {
			continue
		}
		switch r.Kind {
		case "spring_festival", "traditional", "western":
			for _, h := range holiday.PresetHolidays(r.Kind) {
				for _, d := range holiday.NextOccurrences(h, now.Year(), 3) {
					if d < today.Format("2006-01-02") {
						continue
					}
					title := "给" + c.Name + "拜年"
					if r.Kind != "spring_festival" {
						title = "给" + c.Name + "发节日问候·" + h.Name
					}
					if err := s.createContactTask(ctx, c, title, &d, nil, presetColor(r.Kind), fmt.Sprintf("contact-greet-%d-%s", c.ID, r.Kind)); err != nil {
						return err
					}
				}
			}
		case "custom":
			if r.DueDate == nil {
				continue
			}
			title := r.Title
			if title == "" {
				title = "给" + c.Name + "的提醒"
			}
			color := presetColor("custom")
			if r.Color != nil {
				color = *r.Color
			}
			if err := s.createContactTask(ctx, c, title, r.DueDate, r.RecurrenceRule, color, fmt.Sprintf("contact-greet-%d-custom-%d", c.ID, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Server) createContactTask(ctx context.Context, c *model.Contact, title string, due *string, rule *string, color, series string) error {
	t := &model.Task{
		Title:          title,
		Status:         "todo",
		Priority:       "normal",
		DueDate:        due,
		RecurrenceRule: rule,
		SeriesID:       &series,
		Color:          &color,
		Visibility:     "private",
		CreatedBy:      c.CreatedBy,
		Assignees:      []model.User{},
		Subtasks:       []model.Subtask{},
	}
	_, err := s.store.CreateTask(ctx, t)
	return err
}

func presetColor(kind string) string {
	switch kind {
	case "spring_festival":
		return "#be123c"
	case "traditional":
		return "#b45309"
	case "western":
		return "#0369a1"
	default:
		return "#7c3aed"
	}
}
