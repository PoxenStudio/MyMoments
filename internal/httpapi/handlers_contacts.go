package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"mymoments/internal/model"
	"mymoments/internal/recur"
	"mymoments/internal/store"
)

type contactItemInput struct {
	Kind  string  `json:"kind"`
	Label *string `json:"label"`
	Value string  `json:"value"`
}

type contactInput struct {
	Name         *string             `json:"name"`
	Organization *string             `json:"organization"`
	Title        *string             `json:"title"`
	Note         *string             `json:"note"`
	BirthdayOn   *string             `json:"birthdayOn"`
	Category     *string             `json:"category"`
	AvatarKey    *string             `json:"avatarKey"`
	OwnerScope   *string             `json:"ownerScope"`
	Items        *[]contactItemInput `json:"items"`
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
	c := &model.Contact{OwnerScope: "household", CreatedBy: u.ID, Items: []model.ContactItem{}}
	if apiErr := applyContactInput(c, &in, true); apiErr != nil {
		writeError(w, apiErr)
		return
	}
	id, err := s.store.CreateContact(r.Context(), c)
	if err != nil {
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
	updated, err := s.store.GetContact(r.Context(), c.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"contact": updated})
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
	if err := s.store.DeleteContact(r.Context(), c.ID); err != nil {
		writeError(w, serverError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	if in.BirthdayOn != nil {
		date := strings.TrimSpace(*in.BirthdayOn)
		if date != "" {
			if err := recur.ValidateDate(date); err != nil {
				return badRequest("生日日期格式需为 YYYY-MM-DD")
			}
		}
		c.BirthdayOn = nonEmptyPtr(date)
	}
	if in.Category != nil {
		c.Category = nonEmptyPtr(*in.Category)
	}
	if in.AvatarKey != nil {
		c.AvatarKey = nonEmptyPtr(*in.AvatarKey)
	}
	if in.OwnerScope != nil {
		if *in.OwnerScope != "household" && *in.OwnerScope != "private" {
			return badRequest("通讯录归属无效")
		}
		c.OwnerScope = *in.OwnerScope
	}
	if in.Items != nil {
		if len(*in.Items) > 50 {
			return badRequest("联系项过多")
		}
		items := make([]model.ContactItem, 0, len(*in.Items))
		for _, it := range *in.Items {
			kind := strings.TrimSpace(it.Kind)
			if kind != "phone" && kind != "email" && kind != "address" {
				return badRequest("联系项类型无效")
			}
			value := strings.TrimSpace(it.Value)
			if value == "" || len([]rune(value)) > 200 {
				return badRequest("联系项内容不能为空且不超过 200 字")
			}
			if kind == "email" && !strings.Contains(value, "@") {
				return badRequest("邮箱格式不正确")
			}
			items = append(items, model.ContactItem{Kind: kind, Label: nonEmptyPtr(valueOr(it.Label, "")), Value: value})
		}
		c.Items = items
	}
	return nil
}
