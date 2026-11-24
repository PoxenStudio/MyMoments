package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"mymoments/internal/model"
	"mymoments/internal/recur"
	"mymoments/internal/store"
)

type templateInput struct {
	Name               *string   `json:"name"`
	Title              *string   `json:"title"`
	Description        *string   `json:"description"`
	Icon               *string   `json:"icon"`
	Color              *string   `json:"color"`
	Priority           *string   `json:"priority"`
	DefaultAssigneeIDs *[]int64  `json:"defaultAssigneeIds"`
	DefaultDueTime     *string   `json:"defaultDueTime"`
	RecurrenceRule     *string   `json:"recurrenceRule"`
	ReminderOffset     *int      `json:"reminderOffset"`
	SubtaskTemplate    *[]string `json:"subtaskTemplate"`
	Enabled            *bool     `json:"enabled"`
	SortOrder          *int      `json:"sortOrder"`
}

func (s *Server) handleTemplatesList(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListTemplates(r.Context())
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleTemplateCreate(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	var in templateInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	t := &model.TaskTemplate{Enabled: true, CreatedBy: u.ID, DefaultAssigneeIDs: []int64{}, SubtaskTemplate: []string{}}
	if apiErr := s.applyTemplateInput(r, t, &in); apiErr != nil {
		writeError(w, apiErr)
		return
	}
	id, err := s.store.CreateTemplate(r.Context(), t)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	t.ID = id
	writeJSON(w, http.StatusCreated, map[string]any{"template": t})
}

func (s *Server) handleTemplatePatch(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	t, apiErr := s.loadTemplate(r, id)
	if t == nil {
		if apiErr != nil {
			writeError(w, apiErr)
		}
		return
	}
	u, _ := currentUser(r)
	if t.CreatedBy != u.ID && u.Role != "admin" {
		writeError(w, forbidden("只有创建者或管理员可以修改模板"))
		return
	}
	var in templateInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if apiErr := s.applyTemplateInput(r, t, &in); apiErr != nil {
		writeError(w, apiErr)
		return
	}
	if err := s.store.UpdateTemplate(r.Context(), t); err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"template": t})
}

func (s *Server) handleTemplateDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	t, apiErr := s.loadTemplate(r, id)
	if t == nil {
		if apiErr != nil {
			writeError(w, apiErr)
		}
		return
	}
	u, _ := currentUser(r)
	if t.CreatedBy != u.ID && u.Role != "admin" {
		writeError(w, forbidden("只有创建者或管理员可以删除模板"))
		return
	}
	if err := s.store.DeleteTemplate(r.Context(), id); err != nil {
		writeError(w, serverError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) loadTemplate(r *http.Request, id int64) (*model.TaskTemplate, *APIError) {
	t, err := s.store.GetTemplate(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, notFound("模板不存在")
		}
		return nil, serverError(err)
	}
	return t, nil
}

func (s *Server) applyTemplateInput(r *http.Request, t *model.TaskTemplate, in *templateInput) *APIError {
	if in.Name != nil {
		t.Name = strings.TrimSpace(*in.Name)
	}
	if t.Name == "" || len([]rune(t.Name)) > 60 {
		return badRequest("请填写 1–60 个字符的模板名称")
	}
	if in.Title != nil {
		t.Title = strings.TrimSpace(*in.Title)
	}
	if t.Title == "" {
		t.Title = t.Name
	}
	if len([]rune(t.Title)) > 120 {
		return badRequest("模板标题过长")
	}
	if in.Description != nil {
		desc := strings.TrimSpace(*in.Description)
		if len([]rune(desc)) > 2000 {
			return badRequest("模板说明过长")
		}
		t.Description = nonEmptyPtr(desc)
	}
	if in.Icon != nil {
		t.Icon = nonEmptyPtr(*in.Icon)
	}
	if in.Color != nil {
		t.Color = nonEmptyPtr(*in.Color)
	}
	if in.Priority != nil {
		if !validPriority(*in.Priority) {
			return badRequest("优先级无效")
		}
		t.Priority = nonEmptyPtr(*in.Priority)
	}
	if in.DefaultDueTime != nil {
		if err := checkTime(*in.DefaultDueTime); err != nil {
			return err
		}
		t.DefaultDueTime = nonEmptyPtr(*in.DefaultDueTime)
	}
	if in.ReminderOffset != nil {
		if *in.ReminderOffset < 0 || *in.ReminderOffset > 10080 {
			return badRequest("提醒提前量无效")
		}
		t.ReminderOffset = in.ReminderOffset
	}
	if in.RecurrenceRule != nil {
		raw := strings.TrimSpace(*in.RecurrenceRule)
		if raw == "" {
			t.RecurrenceRule = nil
		} else {
			if _, err := recur.Parse(raw); err != nil {
				return badRequest(err.Error())
			}
			t.RecurrenceRule = &raw
		}
	}
	if in.DefaultAssigneeIDs != nil {
		users, err := s.resolveAssignees(r, *in.DefaultAssigneeIDs)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok {
				return apiErr
			}
			return serverError(err)
		}
		ids := make([]int64, 0, len(users))
		for _, u := range users {
			ids = append(ids, u.ID)
		}
		t.DefaultAssigneeIDs = ids
	}
	if in.SubtaskTemplate != nil {
		out := []string{}
		for _, s := range *in.SubtaskTemplate {
			title := strings.TrimSpace(s)
			if title == "" {
				continue
			}
			if len([]rune(title)) > 120 {
				return badRequest("子任务标题过长")
			}
			out = append(out, title)
		}
		t.SubtaskTemplate = out
	}
	if in.Enabled != nil {
		t.Enabled = *in.Enabled
	}
	if in.SortOrder != nil {
		t.SortOrder = *in.SortOrder
	}
	return nil
}
