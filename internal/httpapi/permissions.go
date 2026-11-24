package httpapi

import "mymoments/internal/model"

func hasAssignee(t *model.Task, userID int64) bool {
	for _, a := range t.Assignees {
		if a.ID == userID {
			return true
		}
	}
	return false
}

func canReadTask(u *model.User, t *model.Task) bool {
	switch t.Visibility {
	case "private":
		return t.CreatedBy == u.ID
	case "selected":
		return t.CreatedBy == u.ID || hasAssignee(t, u.ID)
	default:
		return true
	}
}

func canWriteTask(u *model.User, t *model.Task) bool {
	switch t.Visibility {
	case "private":
		return t.CreatedBy == u.ID
	case "selected":
		return t.CreatedBy == u.ID || hasAssignee(t, u.ID)
	default:
		return true
	}
}

func canDeleteTask(u *model.User, t *model.Task) bool {
	return t.CreatedBy == u.ID || (u.Role == "admin" && t.Visibility != "private")
}
