package httpapi

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"mymoments/internal/auth"
	"mymoments/internal/model"
	"mymoments/internal/recur"
	"mymoments/internal/store"
)

type taskInput struct {
	Title           *string         `json:"title"`
	Description     *string         `json:"description"`
	Status          *string         `json:"status"`
	Priority        *string         `json:"priority"`
	StartDate       *string         `json:"startDate"`
	DueDate         *string         `json:"dueDate"`
	DueTime         *string         `json:"dueTime"`
	TemplateID      *int64          `json:"templateId"`
	RecurrenceRule  *string         `json:"recurrenceRule"`
	Color           *string         `json:"color"`
	Icon            *string         `json:"icon"`
	Visibility      *string         `json:"visibility"`
	AssigneeIDs     *[]int64        `json:"assigneeIds"`
	Mentions        *[]mentionInput `json:"mentions"`
	Subtasks        *[]subtaskInput `json:"subtasks"`
	ReminderOffsets *[]int          `json:"reminderOffsets"`
	Archived        *bool           `json:"archived"`
}

type subtaskInput struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func (s *Server) handleTasksList(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	q := r.URL.Query()
	f := store.TaskFilter{
		ViewerID:   u.ID,
		Scope:      q.Get("scope"),
		Priority:   q.Get("priority"),
		Query:      q.Get("q"),
		From:       q.Get("from"),
		To:         q.Get("to"),
		Order:      q.Get("order"),
		AssigneeID: queryInt(q.Get("assignee")),
		TemplateID: queryInt(q.Get("templateId")),
		Limit:      int(queryInt(q.Get("limit"))),
		Offset:     int(queryInt(q.Get("offset"))),
	}
	if q.Get("assignee") == "me" {
		f.AssigneeID = u.ID
	}
	items, err := s.store.ListTasks(r.Context(), f)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	today := s.today()
	counts, err := s.store.CountTasks(r.Context(), u.ID, today)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "counts": counts, "today": today})
}

func (s *Server) handleTaskGet(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	task, _ := s.loadTaskFor(r, w)
	if task == nil {
		return
	}
	if !canReadTask(u, task) {
		writeError(w, notFound("待办不存在"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": task})
}

func (s *Server) handleTaskCreate(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	var in taskInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	task, err := s.buildNewTask(r, u, &in)
	if err != nil {
		writeError(w, err)
		return
	}
	mentions, apiErr := s.resolveMentions(r.Context(), u.ID, taskMentionSource(task), valueMentions(in.Mentions))
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}
	task.Mentions = mentions
	id, err := s.store.CreateTask(r.Context(), task)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	task.ID = id
	if err := s.syncTaskReminders(r, task); err != nil {
		writeError(w, serverError(err))
		return
	}
	created, err := s.store.GetTask(r.Context(), id)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"task": created})
}

func (s *Server) handleTaskPatch(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	task, _ := s.loadTaskFor(r, w)
	if task == nil {
		return
	}
	if !canReadTask(u, task) {
		writeError(w, notFound("待办不存在"))
		return
	}
	if !canWriteTask(u, task) {
		writeError(w, forbidden("没有修改该待办的权限"))
		return
	}
	var in taskInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if apiErr := applyTaskPatch(task, &in); apiErr != nil {
		writeError(w, apiErr)
		return
	}
	if in.AssigneeIDs != nil {
		users, err := s.resolveAssignees(r, *in.AssigneeIDs)
		if err != nil {
			writeError(w, err)
			return
		}
		task.Assignees = users
	}
	if in.Subtasks != nil {
		subs := make([]model.Subtask, 0, len(*in.Subtasks))
		for i, st := range *in.Subtasks {
			title := strings.TrimSpace(st.Title)
			if title == "" {
				continue
			}
			if len([]rune(title)) > 120 {
				writeError(w, badRequest("子任务标题过长"))
				return
			}
			subs = append(subs, model.Subtask{ID: st.ID, TaskID: task.ID, Title: title, Done: st.Done, Position: i})
		}
		task.Subtasks = subs
	}
	if in.Archived != nil {
		if *in.Archived {
			if !canDeleteTask(u, task) {
				writeError(w, forbidden("只有创建者或管理员可以归档该待办"))
				return
			}
			now := time.Now().Unix()
			task.ArchivedAt = &now
		} else {
			task.ArchivedAt = nil
		}
	}
	mentions, apiErr := s.resolveMentions(r.Context(), u.ID, taskMentionSource(task), valueMentions(in.Mentions))
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}
	task.Mentions = mentions
	if err := s.store.UpdateTask(r.Context(), task); err != nil {
		writeError(w, serverError(err))
		return
	}
	if in.ReminderOffsets != nil {
		task.ReminderOffsets = normalizeOffsets(*in.ReminderOffsets)
		if err := s.syncTaskReminders(r, task); err != nil {
			writeError(w, serverError(err))
			return
		}
	}
	updated, err := s.store.GetTask(r.Context(), task.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": updated})
}

func (s *Server) handleTaskDelete(w http.ResponseWriter, r *http.Request) {
	task, _ := s.loadTaskFor(r, w)
	if task == nil {
		return
	}
	u, _ := currentUser(r)
	if !canReadTask(u, task) {
		writeError(w, notFound("待办不存在"))
		return
	}
	if !canDeleteTask(u, task) {
		writeError(w, forbidden("只有创建者或管理员可以删除该待办"))
		return
	}
	media, err := s.store.ListMedia(r.Context(), task.ID, nil)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	if err := s.store.DeleteTask(r.Context(), task.ID); err != nil {
		writeError(w, serverError(err))
		return
	}
	_ = s.store.DeleteTaskReminders(r.Context(), task.ID)
	s.removeMediaFiles(media)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSubtaskPatch(w http.ResponseWriter, r *http.Request) {
	task, _ := s.loadTaskFor(r, w)
	if task == nil {
		return
	}
	u, _ := currentUser(r)
	if !canReadTask(u, task) {
		writeError(w, notFound("待办不存在"))
		return
	}
	if !canWriteTask(u, task) {
		writeError(w, forbidden("没有修改该待办的权限"))
		return
	}
	subID, err := parseID(r, "subId")
	if err != nil {
		writeError(w, err)
		return
	}
	var body struct {
		Done bool `json:"done"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, err)
		return
	}
	if err := s.store.SetSubtaskDone(r.Context(), task.ID, subID, body.Done); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, notFound("子任务不存在"))
			return
		}
		writeError(w, serverError(err))
		return
	}
	updated, err := s.store.GetTask(r.Context(), task.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": updated})
}

func (s *Server) handleTaskComplete(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	task, _ := s.loadTaskFor(r, w)
	if task == nil {
		return
	}
	if !canReadTask(u, task) {
		writeError(w, notFound("待办不存在"))
		return
	}
	if !canWriteTask(u, task) {
		writeError(w, forbidden("没有完成该待办的权限"))
		return
	}
	var body struct {
		OccurrenceDate string `json:"occurrenceDate"`
		ActualUserID   *int64 `json:"actualUserId"`
	}
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &body); err != nil {
			writeError(w, err)
			return
		}
	}
	if task.Status == "completed" || task.Completion != nil {
		writeError(w, conflict("该待办已完成"))
		return
	}
	occurrence := strings.TrimSpace(body.OccurrenceDate)
	if occurrence == "" {
		occurrence = firstNonEmpty(task.OccurrenceDate, task.DueDate)
	}
	if occurrence == "" {
		occurrence = s.today()
	}
	if err := recur.ValidateDate(occurrence); err != nil {
		writeError(w, badRequest("完成日期格式不正确"))
		return
	}
	actual := u.ID
	if body.ActualUserID != nil && *body.ActualUserID > 0 {
		if _, err := s.store.GetUserByID(r.Context(), *body.ActualUserID); err != nil {
			writeError(w, badRequest("实际完成者不存在"))
			return
		}
		actual = *body.ActualUserID
	}
	c := &model.TaskCompletion{
		TaskID:         task.ID,
		OccurrenceDate: occurrence,
		SeriesID:       task.SeriesID,
		DoneByUserID:   &u.ID,
		ActualUserID:   &actual,
	}
	next, err := s.buildNextOccurrence(task, occurrence)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.store.CompleteTask(r.Context(), task, c, next); err != nil {
		if errors.Is(err, store.ErrTaskCompleted) {
			writeError(w, conflict("该待办已完成"))
			return
		}
		writeError(w, serverError(err))
		return
	}
	if next != nil {
		if err := s.syncTaskReminders(r, next); err != nil {
			writeError(w, serverError(err))
			return
		}
	}
	updated, err := s.store.GetTask(r.Context(), task.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	var nextOut *model.Task
	if next != nil {
		if loaded, err := s.store.GetTask(r.Context(), next.ID); err == nil {
			nextOut = loaded
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": updated, "nextTask": nextOut, "completion": updated.Completion})
}

func (s *Server) handleTaskUndo(w http.ResponseWriter, r *http.Request) {
	task, _ := s.loadTaskFor(r, w)
	if task == nil {
		return
	}
	u, _ := currentUser(r)
	if !canReadTask(u, task) {
		writeError(w, notFound("待办不存在"))
		return
	}
	if !canWriteTask(u, task) {
		writeError(w, forbidden("没有撤销该待办的权限"))
		return
	}
	if task.Completion == nil {
		writeError(w, notFound("没有可撤销的完成记录"))
		return
	}
	removed, err := s.store.UndoCompletion(r.Context(), task, task.Completion.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	if task.Completion.NextTaskID != nil && removed {
		_ = s.store.DeleteTaskReminders(r.Context(), *task.Completion.NextTaskID)
	}
	if err := s.syncTaskReminders(r, task); err != nil {
		writeError(w, serverError(err))
		return
	}
	updated, err := s.store.GetTask(r.Context(), task.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": updated, "nextTaskRemoved": removed})
}

func (s *Server) loadTaskFor(r *http.Request, w http.ResponseWriter) (*model.Task, *APIError) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return nil, err.(*APIError)
	}
	task, err := s.store.GetTask(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, notFound("待办不存在"))
			return nil, nil
		}
		writeError(w, serverError(err))
		return nil, nil
	}
	return task, nil
}

func (s *Server) buildNewTask(r *http.Request, u *model.User, in *taskInput) (*model.Task, error) {
	task := &model.Task{
		Status:          "todo",
		Priority:        "normal",
		Visibility:      "household",
		CreatedBy:       u.ID,
		Assignees:       []model.User{},
		Subtasks:        []model.Subtask{},
		ReminderOffsets: []int{},
	}
	var tplDueTime *string
	if in.TemplateID != nil && *in.TemplateID > 0 {
		tpl, err := s.store.GetTemplate(r.Context(), *in.TemplateID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, badRequest("模板不存在")
			}
			return nil, serverError(err)
		}
		task.TemplateID = &tpl.ID
		task.Title = tpl.Title
		if task.Title == "" {
			task.Title = tpl.Name
		}
		task.Description = tpl.Description
		task.Priority = valueOr(tpl.Priority, "normal")
		task.Icon = tpl.Icon
		task.Color = tpl.Color
		tplDueTime = tpl.DefaultDueTime
		task.RecurrenceRule = tpl.RecurrenceRule
		if tpl.ReminderOffset != nil {
			task.ReminderOffsets = []int{*tpl.ReminderOffset}
		}
		if len(tpl.DefaultAssigneeIDs) > 0 {
			users, err := s.resolveAssignees(r, tpl.DefaultAssigneeIDs)
			if err != nil {
				return nil, err
			}
			task.Assignees = users
		}
		for _, title := range tpl.SubtaskTemplate {
			task.Subtasks = append(task.Subtasks, model.Subtask{Title: title})
		}
	}
	if in.Title != nil {
		task.Title = strings.TrimSpace(*in.Title)
	}
	if task.Title == "" || len([]rune(task.Title)) > 120 {
		return nil, badRequest("请填写 1–120 个字符的标题")
	}
	if in.Description != nil {
		desc := strings.TrimSpace(*in.Description)
		if len([]rune(desc)) > 5000 {
			return nil, badRequest("说明过长")
		}
		if desc != "" {
			task.Description = &desc
		}
	}
	if in.Priority != nil {
		if !validPriority(*in.Priority) {
			return nil, badRequest("优先级无效")
		}
		task.Priority = *in.Priority
	}
	if in.Status != nil {
		if !validStatus(*in.Status) {
			return nil, badRequest("状态无效")
		}
		task.Status = *in.Status
	}
	if in.StartDate != nil {
		if err := checkDate(*in.StartDate); err != nil {
			return nil, err
		}
		task.StartDate = nonEmptyPtr(*in.StartDate)
	}
	if in.DueDate != nil {
		if err := checkDate(*in.DueDate); err != nil {
			return nil, err
		}
		task.DueDate = nonEmptyPtr(*in.DueDate)
	}
	if in.DueTime != nil {
		if err := checkTime(*in.DueTime); err != nil {
			return nil, err
		}
		task.DueTime = nonEmptyPtr(*in.DueTime)
	}
	if task.DueTime != nil && task.DueDate == nil {
		return nil, badRequest("设置截止时间前请先选择截止日期")
	}
	if task.DueTime == nil && task.DueDate != nil && tplDueTime != nil {
		task.DueTime = tplDueTime
	}
	if in.Visibility != nil {
		if !validVisibility(*in.Visibility) {
			return nil, badRequest("可见范围无效")
		}
		task.Visibility = *in.Visibility
	}
	if in.Color != nil {
		task.Color = nonEmptyPtr(*in.Color)
	}
	if in.Icon != nil {
		task.Icon = nonEmptyPtr(*in.Icon)
	}
	if in.AssigneeIDs != nil {
		users, err := s.resolveAssignees(r, *in.AssigneeIDs)
		if err != nil {
			return nil, err
		}
		task.Assignees = users
	}
	if in.Subtasks != nil {
		for i, st := range *in.Subtasks {
			title := strings.TrimSpace(st.Title)
			if title == "" {
				continue
			}
			if len([]rune(title)) > 120 {
				return nil, badRequest("子任务标题过长")
			}
			task.Subtasks = append(task.Subtasks, model.Subtask{Title: title, Position: i})
		}
	}
	if in.RecurrenceRule != nil && strings.TrimSpace(*in.RecurrenceRule) != "" {
		rule, err := recur.Parse(*in.RecurrenceRule)
		if err != nil {
			return nil, badRequest(err.Error())
		}
		if rule == nil {
			return nil, badRequest("重复规则格式不正确")
		}
		if task.DueDate == nil {
			return nil, badRequest("重复待办需要设置截止日期")
		}
		raw := strings.TrimSpace(*in.RecurrenceRule)
		task.RecurrenceRule = &raw
	}
	if in.ReminderOffsets != nil {
		task.ReminderOffsets = normalizeOffsets(*in.ReminderOffsets)
	} else if task.DueDate != nil {
		if h, err := s.store.GetHousehold(r.Context()); err == nil {
			task.ReminderOffsets = []int{h.DefaultReminderOffset}
		}
	}
	if task.RecurrenceRule != nil {
		series, err := auth.RandomToken(12)
		if err != nil {
			return nil, serverError(err)
		}
		task.SeriesID = &series
		occ := firstNonEmpty(task.DueDate, task.StartDate)
		if occ == "" {
			occ = s.today()
		}
		task.OccurrenceDate = &occ
	} else if task.DueDate != nil {
		occ := *task.DueDate
		task.OccurrenceDate = &occ
	}
	if task.Status == "completed" {
		now := time.Now().Unix()
		task.CompletedAt = &now
	}
	return task, nil
}

func applyTaskPatch(task *model.Task, in *taskInput) *APIError {
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" || len([]rune(title)) > 120 {
			return badRequest("请填写 1–120 个字符的标题")
		}
		task.Title = title
	}
	if in.Description != nil {
		desc := strings.TrimSpace(*in.Description)
		if len([]rune(desc)) > 5000 {
			return badRequest("说明过长")
		}
		task.Description = nonEmptyPtr(desc)
	}
	if in.Priority != nil {
		if !validPriority(*in.Priority) {
			return badRequest("优先级无效")
		}
		task.Priority = *in.Priority
	}
	if in.Status != nil {
		if !validStatus(*in.Status) {
			return badRequest("状态无效")
		}
		task.Status = *in.Status
		if *in.Status != "completed" {
			task.CompletedAt = nil
		}
	}
	if in.StartDate != nil {
		if err := checkDate(*in.StartDate); err != nil {
			return err
		}
		task.StartDate = nonEmptyPtr(*in.StartDate)
	}
	if in.DueDate != nil {
		if err := checkDate(*in.DueDate); err != nil {
			return err
		}
		task.DueDate = nonEmptyPtr(*in.DueDate)
	}
	if in.DueTime != nil {
		if err := checkTime(*in.DueTime); err != nil {
			return err
		}
		task.DueTime = nonEmptyPtr(*in.DueTime)
	}
	if task.DueTime != nil && task.DueDate == nil {
		return badRequest("设置截止时间前请先选择截止日期")
	}
	if in.Visibility != nil {
		if !validVisibility(*in.Visibility) {
			return badRequest("可见范围无效")
		}
		task.Visibility = *in.Visibility
	}
	if in.Color != nil {
		task.Color = nonEmptyPtr(*in.Color)
	}
	if in.Icon != nil {
		task.Icon = nonEmptyPtr(*in.Icon)
	}
	if in.TemplateID != nil {
		task.TemplateID = in.TemplateID
	}
	if in.RecurrenceRule != nil {
		raw := strings.TrimSpace(*in.RecurrenceRule)
		if raw == "" {
			task.RecurrenceRule = nil
			task.SeriesID = nil
		} else {
			if _, err := recur.Parse(raw); err != nil {
				return badRequest(err.Error())
			}
			if task.DueDate == nil {
				return badRequest("重复待办需要设置截止日期")
			}
			task.RecurrenceRule = &raw
			if task.SeriesID == nil {
				series, err := auth.RandomToken(12)
				if err != nil {
					return serverError(err)
				}
				task.SeriesID = &series
			}
		}
	}
	if task.RecurrenceRule != nil {
		occ := firstNonEmpty(task.DueDate, task.StartDate)
		if occ != "" {
			task.OccurrenceDate = &occ
		}
	}
	if in.ReminderOffsets != nil {
		task.ReminderOffsets = normalizeOffsets(*in.ReminderOffsets)
	}
	return nil
}

func (s *Server) buildNextOccurrence(task *model.Task, occurrence string) (*model.Task, error) {
	rule, err := recur.Parse(firstNonEmpty(task.RecurrenceRule))
	if err != nil || rule == nil {
		return nil, nil
	}
	nextDate, ok := recur.Next(rule, occurrence, s.today())
	if !ok {
		return nil, nil
	}
	next := &model.Task{
		Title:           task.Title,
		Description:     task.Description,
		Status:          "todo",
		Priority:        task.Priority,
		TemplateID:      task.TemplateID,
		RecurrenceRule:  task.RecurrenceRule,
		SeriesID:        task.SeriesID,
		DueDate:         &nextDate,
		DueTime:         task.DueTime,
		Color:           task.Color,
		Icon:            task.Icon,
		Visibility:      task.Visibility,
		CreatedBy:       task.CreatedBy,
		Assignees:       task.Assignees,
		Subtasks:        task.Subtasks,
		ReminderOffsets: task.ReminderOffsets,
	}
	next.OccurrenceDate = &nextDate
	if task.StartDate != nil && task.DueDate != nil {
		shift := daysBetween(*task.DueDate, nextDate)
		if shifted, err := shiftDate(*task.StartDate, shift); err == nil {
			next.StartDate = &shifted
		}
	} else {
		next.StartDate = task.StartDate
	}
	for i := range next.Subtasks {
		next.Subtasks[i].Done = false
	}
	return next, nil
}

func (s *Server) resolveAssignees(r *http.Request, ids []int64) ([]model.User, error) {
	seen := map[int64]bool{}
	out := []model.User{}
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		u, err := s.store.GetUserByID(r.Context(), id)
		if err != nil {
			return nil, badRequest("指派的成员不存在")
		}
		if u.Status != "active" {
			return nil, badRequest("不能指派已停用的成员")
		}
		out = append(out, *u)
	}
	return out, nil
}

func (s *Server) syncTaskReminders(r *http.Request, task *model.Task) error {
	specs, err := s.reminderSpecs(r, task)
	if err != nil {
		return err
	}
	return s.store.ReplaceTaskReminders(r.Context(), task, specs)
}

func (s *Server) reminderSpecs(r *http.Request, task *model.Task) ([]store.ReminderSpec, error) {
	if task.DueDate == nil || len(task.ReminderOffsets) == 0 {
		return nil, nil
	}
	hhmm := "09:00"
	if task.DueTime != nil {
		hhmm = *task.DueTime
	}
	loc, err := s.location(r.Context())
	if err != nil {
		return nil, err
	}
	due, err := time.ParseInLocation("2006-01-02 15:04", *task.DueDate+" "+hhmm, loc)
	if err != nil {
		return nil, err
	}
	recipients := make([]int64, 0, len(task.Assignees))
	for _, a := range task.Assignees {
		recipients = append(recipients, a.ID)
	}
	if len(recipients) == 0 {
		recipients = []int64{task.CreatedBy}
	}
	specs := make([]store.ReminderSpec, 0, len(recipients)*len(task.ReminderOffsets))
	for _, uid := range recipients {
		for _, offset := range task.ReminderOffsets {
			at := due.Unix() - int64(offset)*60
			specs = append(specs, store.ReminderSpec{UserID: uid, RemindAt: at})
		}
	}
	return specs, nil
}

// taskMentionSource 合并待办标题与说明，作为 @ 提及的扫描正文。
func taskMentionSource(t *model.Task) string {
	parts := make([]string, 0, 2)
	parts = append(parts, t.Title)
	if t.Description != nil {
		parts = append(parts, *t.Description)
	}
	return strings.Join(parts, "\n")
}

func (s *Server) today() string {
	loc, err := s.location(context.Background())
	if err != nil {
		return time.Now().Format("2006-01-02")
	}
	return recur.TodayIn(loc, time.Now())
}

func (s *Server) location(ctx context.Context) (*time.Location, error) {
	h, err := s.store.GetHousehold(ctx)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return time.LoadLocation(s.cfg.DefaultTZ)
		}
		return nil, err
	}
	loc, err := time.LoadLocation(h.Timezone)
	if err != nil {
		return time.LoadLocation(s.cfg.DefaultTZ)
	}
	return loc, nil
}

func normalizeOffsets(in []int) []int {
	seen := map[int]bool{}
	out := []int{}
	for _, v := range in {
		if v < 0 || v > 10080 || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Ints(out)
	return out
}

func validPriority(v string) bool {
	return v == "low" || v == "normal" || v == "high" || v == "urgent"
}

func validStatus(v string) bool {
	return v == "todo" || v == "in_progress" || v == "completed"
}

func validVisibility(v string) bool {
	return v == "household" || v == "selected" || v == "private"
}

func checkDate(v string) *APIError {
	if v == "" {
		return nil
	}
	if err := recur.ValidateDate(v); err != nil {
		return badRequest("日期格式需为 YYYY-MM-DD")
	}
	return nil
}

func checkTime(v string) *APIError {
	if v == "" {
		return nil
	}
	if len(v) != 5 || v[2] != ':' {
		return badRequest("时间格式需为 HH:MM")
	}
	hh, err1 := strconv.Atoi(v[:2])
	mm, err2 := strconv.Atoi(v[3:])
	if err1 != nil || err2 != nil || hh > 23 || mm > 59 {
		return badRequest("时间格式需为 HH:MM")
	}
	return nil
}

func nonEmptyPtr(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func valueOr(v *string, fallback string) string {
	if v == nil || *v == "" {
		return fallback
	}
	return *v
}

func firstNonEmpty(values ...*string) string {
	for _, v := range values {
		if v != nil && *v != "" {
			return *v
		}
	}
	return ""
}

func queryInt(v string) int64 {
	n, _ := strconv.ParseInt(v, 10, 64)
	return n
}

func daysBetween(from, to string) int {
	a, err1 := time.Parse("2006-01-02", from)
	b, err2 := time.Parse("2006-01-02", to)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int(b.Sub(a).Hours() / 24)
}

func shiftDate(date string, days int) (string, error) {
	d, err := time.Parse("2006-01-02", date)
	if err != nil || days == 0 {
		return date, err
	}
	return d.AddDate(0, 0, days).Format("2006-01-02"), nil
}
