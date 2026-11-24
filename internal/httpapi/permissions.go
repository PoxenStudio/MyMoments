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

func hasEventAssignee(ev *model.CalendarEvent, userID int64) bool {
	for _, a := range ev.Assignees {
		if a.ID == userID {
			return true
		}
	}
	return false
}

func canReadEvent(u *model.User, ev *model.CalendarEvent) bool {
	switch ev.Visibility {
	case "private":
		return ev.CreatedBy == u.ID
	case "selected":
		return ev.CreatedBy == u.ID || hasEventAssignee(ev, u.ID)
	default:
		return true
	}
}

func canWriteEvent(u *model.User, ev *model.CalendarEvent) bool {
	return canReadEvent(u, ev)
}

func canDeleteEvent(u *model.User, ev *model.CalendarEvent) bool {
	return ev.CreatedBy == u.ID || (u.Role == "admin" && ev.Visibility != "private")
}

func canReadContact(u *model.User, c *model.Contact) bool {
	return c.OwnerScope != "private" || c.CreatedBy == u.ID
}

func canWriteContact(u *model.User, c *model.Contact) bool {
	return canReadContact(u, c)
}

func canDeleteContact(u *model.User, c *model.Contact) bool {
	return c.CreatedBy == u.ID || (u.Role == "admin" && c.OwnerScope != "private")
}

func containsID(ids []int64, id int64) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func canReadMessage(u *model.User, m *model.MessagePost) bool {
	switch m.Visibility {
	case "private":
		return m.AuthorID == u.ID
	case "selected":
		return m.AuthorID == u.ID || containsID(m.AudienceIDs, u.ID)
	default:
		return true
	}
}

func canWriteMessage(u *model.User, m *model.MessagePost) bool {
	return canReadMessage(u, m)
}

func canDeleteMessage(u *model.User, m *model.MessagePost) bool {
	return m.AuthorID == u.ID || (u.Role == "admin" && m.Visibility != "private")
}

func canReadNote(u *model.User, n *model.Note) bool {
	switch n.Visibility {
	case "private":
		return n.AuthorID == u.ID
	case "selected":
		return n.AuthorID == u.ID || containsID(n.AudienceIDs, u.ID)
	default:
		return true
	}
}

func canWriteNote(u *model.User, n *model.Note) bool {
	return canReadNote(u, n)
}

func canDeleteNote(u *model.User, n *model.Note) bool {
	return n.AuthorID == u.ID || (u.Role == "admin" && n.Visibility != "private")
}
