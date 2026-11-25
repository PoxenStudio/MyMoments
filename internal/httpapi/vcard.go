package httpapi

import (
	"io"
	"net/http"
	"regexp"
	"strings"

	"mymoments/internal/model"
)

// vcardPhone 是解析出的电话条目。
type vcardPhone struct {
	Value string
	Label string
}

// vcard 是解析出的单张联系人名片。
type vcard struct {
	Name         string
	Organization string
	Title        string
	Note         string
	Birthday     *string
	Phones       []vcardPhone
	Emails       []string
	Address      string
}

// handleContactsImport 从上传的 vCard 文件导入联系人。
func (s *Server) handleContactsImport(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, badRequest("请上传 vCard 文件"))
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, badRequest("请上传 vCard 文件"))
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 50<<20))
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	cards := parseVCards(string(data))
	created := make([]model.Contact, 0, len(cards))
	for _, card := range cards {
		if strings.TrimSpace(card.Name) == "" {
			continue
		}
		c := card.toContact()
		c.OwnerScope = "household"
		c.CreatedBy = u.ID
		id, err := s.store.CreateContact(r.Context(), c)
		if err != nil {
			continue
		}
		got, err := s.store.GetContact(r.Context(), id)
		if err != nil {
			continue
		}
		created = append(created, *got)
	}
	writeJSON(w, http.StatusOK, map[string]any{"created": len(created), "items": created})
}

func (v vcard) toContact() *model.Contact {
	c := &model.Contact{Name: v.Name, Items: []model.ContactItem{}, Relations: []model.ContactRelation{}, Reminders: []model.ContactReminder{}}
	if v.Organization != "" {
		c.Organization = &v.Organization
	}
	if v.Title != "" {
		c.Title = &v.Title
	}
	if v.Note != "" {
		c.Note = &v.Note
	}
	if v.Birthday != nil {
		if m, d, y := parseBirthday(*v.Birthday); m > 0 && d > 0 {
			c.BirthdayMonth = &m
			c.BirthdayDay = &d
			if y > 0 {
				c.BirthYear = &y
			}
		}
	}
	for _, p := range v.Phones {
		label := p.Label
		if label == "" {
			label = "私人"
		}
		c.Items = append(c.Items, model.ContactItem{Kind: "phone", Label: &label, Value: p.Value})
	}
	for _, e := range v.Emails {
		label := "邮箱"
		c.Items = append(c.Items, model.ContactItem{Kind: "email", Label: &label, Value: e})
	}
	if v.Address != "" {
		label := "地址"
		c.Items = append(c.Items, model.ContactItem{Kind: "address", Label: &label, Value: v.Address})
	}
	return c
}

// parseVCards 解析 vCard 2.1/3.0 文本，返回名片列表。
func parseVCards(content string) []vcard {
	lines := unfoldLines(content)
	cards := []vcard{}
	current := map[string][]string{}
	inCard := false
	for _, line := range lines {
		if strings.EqualFold(strings.TrimSpace(line), "BEGIN:VCARD") {
			inCard = true
			current = map[string][]string{}
			continue
		}
		if strings.EqualFold(strings.TrimSpace(line), "END:VCARD") {
			if inCard {
				cards = append(cards, buildVCard(current))
			}
			inCard = false
			continue
		}
		if !inCard {
			continue
		}
		idx := strings.IndexAny(line, ":;")
		if idx < 0 {
			continue
		}
		// 属性名（去掉分组前缀与参数）；参数以“;”分隔，值为冒号后内容。
		name := line[:idx]
		if semi := strings.Index(line, ":"); semi >= 0 && semi < idx {
			name = line[:semi]
		}
		name = propertyName(name)
		value := ""
		if colon := strings.Index(line, ":"); colon >= 0 {
			value = line[colon+1:]
		}
		current[name] = append(current[name], value)
	}
	return cards
}

func unfoldLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	raw := strings.Split(content, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		if (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && len(out) > 0 {
			out[len(out)-1] += strings.TrimPrefix(strings.TrimPrefix(line, " "), "\t")
			continue
		}
		out = append(out, line)
	}
	return out
}

func propertyName(raw string) string {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, ";"); i >= 0 {
		raw = raw[:i]
	}
	// 去掉形如 item1.FN 的分组前缀。
	if i := strings.LastIndex(raw, "."); i >= 0 {
		raw = raw[i+1:]
	}
	return strings.ToUpper(raw)
}

func buildVCard(m map[string][]string) vcard {
	var v vcard
	v.Name = stripQuotes(firstOf(m, "FN"))
	if v.Name == "" {
		if n := firstOf(m, "N"); n != "" {
			v.Name = nameFromN(n)
		}
	}
	v.Organization = unescape(firstOf(m, "ORG"))
	v.Title = unescape(firstOf(m, "TITLE"))
	v.Note = unescape(firstOf(m, "NOTE"))
	if b := firstOf(m, "BDAY"); b != "" {
		v.Birthday = &b
	}
	for _, tel := range m["TEL"] {
		if value, label := parseTel(tel); value != "" {
			v.Phones = append(v.Phones, vcardPhone{Value: value, Label: label})
		}
	}
	for _, em := range m["EMAIL"] {
		if e := stripQuotes(em); e != "" {
			v.Emails = append(v.Emails, e)
		}
	}
	if adr := firstOf(m, "ADR"); adr != "" {
		v.Address = addressFromADR(adr)
	}
	return v
}

func firstOf(m map[string][]string, key string) string {
	if vals, ok := m[key]; ok && len(vals) > 0 {
		return strings.TrimSpace(vals[0])
	}
	return ""
}

func nameFromN(n string) string {
	// N: 姓;名;中间名;前缀;后缀
	parts := strings.Split(n, ";")
	family := stripQuotes(parts[0])
	given := ""
	if len(parts) > 1 {
		given = stripQuotes(parts[1])
	}
	if family != "" && given != "" {
		return family + given
	}
	if family != "" {
		return family
	}
	return given
}

func parseTel(raw string) (value, label string) {
	// 先拆参数段（冒号前），提取 TYPE。
	params := ""
	if colon := strings.Index(raw, ":"); colon >= 0 {
		params = raw[:colon]
		value = stripQuotes(raw[colon+1:])
	} else {
		value = stripQuotes(raw)
	}
	upper := strings.ToUpper(params)
	switch {
	case strings.Contains(upper, "CELL"), strings.Contains(upper, "MOBILE"):
		label = "私人"
	case strings.Contains(upper, "WORK"):
		label = "办公"
	case strings.Contains(upper, "HOME"):
		label = "住宅"
	default:
		label = "私人"
	}
	return value, label
}

var nonDigits = regexp.MustCompile(`[^0-9]`)

func addressFromADR(adr string) string {
	// ADR: ;;街道;城市;地区;邮编;国家
	parts := strings.Split(adr, ";")
	var fields []string
	for i, p := range parts {
		if i < 2 { // 跳过 PO box 与扩展地址
			continue
		}
		if s := stripQuotes(p); s != "" {
			fields = append(fields, s)
		}
	}
	return strings.Join(fields, " ")
}

func parseBirthday(raw string) (month, day, year int) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, 0, 0
	}
	digits := nonDigits.ReplaceAllString(s, "")
	switch len(digits) {
	case 8: // YYYYMMDD
		year = atoi(digits[0:4])
		month = atoi(digits[4:6])
		day = atoi(digits[6:8])
	case 4: // MMDD
		month = atoi(digits[0:2])
		day = atoi(digits[2:4])
	default:
		return 0, 0, 0
	}
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return 0, 0, 0
	}
	return month, day, year
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func unescape(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\,`, ",")
	s = strings.ReplaceAll(s, `\;`, ";")
	return stripQuotes(s)
}

// stripQuotes 去掉 vCard 2.1 中包裹字段值的成对双引号，如 FN:"张三"。
func stripQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
}
