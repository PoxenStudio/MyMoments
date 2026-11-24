package httpapi

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	"mymoments/internal/recur"
)

// resolveMentionIDs 合并客户端显式提及与正文中的 @姓名，并校验成员有效。
func (s *Server) resolveMentionIDs(ctx context.Context, body string, explicit []int64) ([]int64, *APIError) {
	users, err := s.store.ListUsers(ctx)
	if err != nil {
		return nil, serverError(err)
	}
	active := make(map[int64]string, len(users))
	for _, u := range users {
		if u.Status == "active" {
			active[u.ID] = u.DisplayName
		}
	}
	seen := make(map[int64]bool, len(explicit))
	out := make([]int64, 0, len(explicit))
	add := func(id int64) {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	for _, id := range explicit {
		if _, ok := active[id]; !ok {
			return nil, badRequest("被提及的成员不存在或已停用")
		}
		add(id)
	}
	for id, name := range active {
		if bodyMentionsName(body, name) {
			add(id)
		}
	}
	return out, nil
}

// resolveAudienceIDs 校验「指定成员可见」的成员均有效。
func (s *Server) resolveAudienceIDs(ctx context.Context, ids []int64) ([]int64, *APIError) {
	if len(ids) == 0 {
		return []int64{}, nil
	}
	users, err := s.store.ListUsers(ctx)
	if err != nil {
		return nil, serverError(err)
	}
	active := make(map[int64]bool, len(users))
	for _, u := range users {
		if u.Status == "active" {
			active[u.ID] = true
		}
	}
	seen := make(map[int64]bool, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if !active[id] {
			return nil, badRequest("可见成员不存在或已停用")
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out, nil
}

func bodyMentionsName(body, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	needle := "@" + name
	for from := 0; from < len(body); {
		idx := strings.Index(body[from:], needle)
		if idx < 0 {
			return false
		}
		pos := from + idx
		if mentionBoundaryBefore(body, pos) && mentionBoundaryAfter(body, pos+len(needle)) {
			return true
		}
		from = pos + len(needle)
	}
	return false
}

func mentionBoundaryBefore(body string, pos int) bool {
	if pos == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(body[:pos])
	// 邮箱、账号等紧邻字符的行内 @ 不作为提及。
	return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '@' || r == '.' ||
		r == '_' || r == '-' || r == '+')
}

func mentionBoundaryAfter(body string, pos int) bool {
	if pos >= len(body) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(body[pos:])
	return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-')
}

func validateVisibility(v string) *APIError {
	switch v {
	case "household", "selected", "private":
		return nil
	default:
		return badRequest("可见范围无效")
	}
}

// validatePastDate 允许空值；非空时必须是今天或过去的 YYYY-MM-DD。
func (s *Server) validatePastDate(value, label string) *APIError {
	if value == "" {
		return nil
	}
	if err := recur.ValidateDate(value); err != nil {
		return badRequest(label + "格式需为 YYYY-MM-DD")
	}
	if value > s.today() {
		return badRequest(label + "不能晚于今天")
	}
	return nil
}
