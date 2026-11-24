package recur

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const dateLayout = "2006-01-02"

// Rule 是待办重复规则的 JSON 子集。
type Rule struct {
	Freq     string `json:"freq"`
	Interval int    `json:"interval"`
	Weekdays []int  `json:"weekdays"`
	Until    string `json:"until"`
	Anchor   string `json:"anchor"`
}

// Parse 解析并校验重复规则，空串表示不重复。
func Parse(raw string) (*Rule, error) {
	if raw == "" {
		return nil, nil
	}
	var r Rule
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return nil, errors.New("重复规则格式不正确")
	}
	if err := r.normalize(); err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *Rule) normalize() error {
	switch r.Freq {
	case "daily", "weekly", "monthly", "yearly":
	case "":
		return errors.New("缺少重复频率")
	default:
		return errors.New("重复频率无效")
	}
	if r.Interval == 0 {
		r.Interval = 1
	}
	if r.Interval < 1 || r.Interval > 365 {
		return errors.New("重复间隔需在 1–365 之间")
	}
	if r.Anchor == "" {
		r.Anchor = "due"
	}
	if r.Anchor != "due" && r.Anchor != "completion" {
		return errors.New("重复锚点无效")
	}
	if r.Until != "" {
		if _, err := time.Parse(dateLayout, r.Until); err != nil {
			return errors.New("重复结束日期无效")
		}
	}
	for _, d := range r.Weekdays {
		if d < 0 || d > 6 {
			return errors.New("重复星期无效")
		}
	}
	return nil
}

// Next 计算下一次发生日期；无下一次时返回 false。
func Next(rule *Rule, occurrenceDate, today string) (string, bool) {
	if rule == nil {
		return "", false
	}
	occ, err := time.Parse(dateLayout, occurrenceDate)
	if err != nil {
		return "", false
	}
	now, err := time.Parse(dateLayout, today)
	if err != nil {
		return "", false
	}
	var candidate time.Time
	if rule.Anchor == "completion" {
		candidate = fromCompletion(rule, now)
	} else {
		var ok bool
		candidate, ok = fromDue(rule, occ, now)
		if !ok {
			return "", false
		}
	}
	if rule.Until != "" {
		until, _ := time.Parse(dateLayout, rule.Until)
		if candidate.After(until) {
			return "", false
		}
	}
	return candidate.Format(dateLayout), true
}

func fromCompletion(rule *Rule, today time.Time) time.Time {
	switch rule.Freq {
	case "daily":
		return today.AddDate(0, 0, rule.Interval)
	case "weekly":
		return today.AddDate(0, 0, 7*rule.Interval)
	case "monthly":
		return addMonths(today, rule.Interval)
	default:
		return addYears(today, rule.Interval)
	}
}

func fromDue(rule *Rule, occ, today time.Time) (time.Time, bool) {
	switch rule.Freq {
	case "daily":
		step := rule.Interval
		c := occ.AddDate(0, 0, step)
		for c.Before(today) {
			c = c.AddDate(0, 0, step)
		}
		return c, true
	case "weekly":
		if len(rule.Weekdays) > 0 {
			c := occ.AddDate(0, 0, 1)
			for i := 0; i < 366*3; i++ {
				if !c.Before(today) && weekdayAllowed(rule.Weekdays, c.Weekday()) {
					return c, true
				}
				c = c.AddDate(0, 0, 1)
			}
			return time.Time{}, false
		}
		c := occ.AddDate(0, 0, 7*rule.Interval)
		for c.Before(today) {
			c = c.AddDate(0, 0, 7*rule.Interval)
		}
		return c, true
	case "monthly":
		c := addMonths(occ, rule.Interval)
		for c.Before(today) {
			c = addMonths(c, rule.Interval)
		}
		return c, true
	default:
		c := addYears(occ, rule.Interval)
		for c.Before(today) {
			c = addYears(c, rule.Interval)
		}
		return c, true
	}
}

func weekdayAllowed(days []int, day time.Weekday) bool {
	for _, d := range days {
		if int(day) == d {
			return true
		}
	}
	return false
}

func addMonths(d time.Time, months int) time.Time {
	y, m, day := d.Date()
	first := time.Date(y, m+time.Month(months), 1, 0, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1).Day()
	if day > last {
		day = last
	}
	return time.Date(first.Year(), first.Month(), day, 0, 0, 0, 0, time.UTC)
}

func addYears(d time.Time, years int) time.Time {
	target := time.Date(d.Year()+years, d.Month(), 1, 0, 0, 0, 0, time.UTC)
	last := target.AddDate(0, 1, -1).Day()
	day := d.Day()
	if day > last {
		day = last
	}
	return time.Date(target.Year(), target.Month(), day, 0, 0, 0, 0, time.UTC)
}

// Expand 返回规则在 [from, to] 内的发生日期（含首尾），anchor 为系列起始日期。
func Expand(rule *Rule, anchor, from, to string) []string {
	out := []string{}
	if rule == nil {
		return out
	}
	start, err := time.Parse(dateLayout, anchor)
	if err != nil {
		return out
	}
	begin, err1 := time.Parse(dateLayout, from)
	end, err2 := time.Parse(dateLayout, to)
	if err1 != nil || err2 != nil || end.Before(begin) {
		return out
	}
	if begin.Before(start) {
		begin = start
	}
	if rule.Until != "" {
		if until, err := time.Parse(dateLayout, rule.Until); err == nil && until.Before(end) {
			end = until
		}
	}
	if end.Before(begin) {
		return out
	}
	// 上限保护：窗口最多十年，避免异常规则拖慢请求。
	if end.Sub(begin) > 3660*24*time.Hour {
		end = begin.AddDate(10, 0, 0)
	}
	for d := begin; !d.After(end); d = d.AddDate(0, 0, 1) {
		if matches(rule, start, d) {
			out = append(out, d.Format(dateLayout))
		}
	}
	return out
}

func matches(rule *Rule, anchor, d time.Time) bool {
	interval := rule.Interval
	if interval < 1 {
		interval = 1
	}
	days := int(d.Sub(anchor).Hours() / 24)
	if days < 0 {
		return false
	}
	switch rule.Freq {
	case "daily":
		return days%interval == 0
	case "weekly":
		if len(rule.Weekdays) == 0 {
			return days%(7*interval) == 0
		}
		if !weekdayAllowed(rule.Weekdays, d.Weekday()) {
			return false
		}
		return (days/7)%interval == 0
	case "monthly":
		months := (d.Year()-anchor.Year())*12 + int(d.Month()-anchor.Month())
		if months < 0 || months%interval != 0 {
			return false
		}
		return d.Day() == clampDay(anchor.Day(), d.Year(), d.Month())
	default:
		years := d.Year() - anchor.Year()
		if years < 0 || years%interval != 0 {
			return false
		}
		return d.Month() == anchor.Month() && d.Day() == clampDay(anchor.Day(), d.Year(), d.Month())
	}
}

func clampDay(day int, year int, month time.Month) int {
	last := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day > last {
		return last
	}
	return day
}

// TodayIn 返回指定时区的今天。
func TodayIn(loc *time.Location, now time.Time) string {
	return now.In(loc).Format(dateLayout)
}

// ValidateDate 校验 date-only 字符串。
func ValidateDate(v string) error {
	if _, err := time.Parse(dateLayout, v); err != nil {
		return fmt.Errorf("日期格式需为 YYYY-MM-DD")
	}
	return nil
}
