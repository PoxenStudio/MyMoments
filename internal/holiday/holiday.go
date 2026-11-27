// Package holiday 提供中国传统节日（农历）与常见西方节日的公历日期计算。
// 农历转换基于 1900–2100 年的标准数据表。
package holiday

import "time"

// lunarInfo 编码每个农历年：低 4 位为闰月（0 表示无闰月），
// 第 4–15 位按位标记 12 个月的大小（1=30 天，0=29 天），第 16 位为闰月大小。
var lunarInfo = []int{
	0x04bd8, 0x04ae0, 0x0a570, 0x054d5, 0x0d260, 0x0d950, 0x16554, 0x056a0, 0x09ad0, 0x055d2, // 1900-1909
	0x04ae0, 0x0a5b6, 0x0a4d0, 0x0d250, 0x1d255, 0x0b540, 0x0d6a0, 0x0ada2, 0x095b0, 0x14977, // 1910-1919
	0x04970, 0x0a4b0, 0x0b4b5, 0x06a50, 0x06d40, 0x1ab54, 0x02b60, 0x09570, 0x052f2, 0x04970, // 1920-1929
	0x06566, 0x0d4a0, 0x0ea50, 0x06e95, 0x05ad0, 0x02b60, 0x186e3, 0x092e0, 0x1c8d7, 0x0c950, // 1930-1939
	0x0d4a0, 0x1d8a6, 0x0b550, 0x056a0, 0x1a5b4, 0x025d0, 0x092d0, 0x0d2b2, 0x0a950, 0x0b557, // 1940-1949
	0x06ca0, 0x0b550, 0x15355, 0x04da0, 0x0a5b0, 0x14573, 0x052b0, 0x0a9a8, 0x0e950, 0x06aa0, // 1950-1959
	0x0aea6, 0x0ab50, 0x04b60, 0x0aae4, 0x0a570, 0x05260, 0x0f263, 0x0d950, 0x05b57, 0x056a0, // 1960-1969
	0x096d0, 0x04dd5, 0x04ad0, 0x0a4d0, 0x0d4d4, 0x0d250, 0x0d558, 0x0b540, 0x0b6a0, 0x195a6, // 1970-1979
	0x095b0, 0x049b0, 0x0a974, 0x0a4b0, 0x0b27a, 0x06a50, 0x06d40, 0x0af46, 0x0ab60, 0x09570, // 1980-1989
	0x04af5, 0x04970, 0x064b0, 0x074a3, 0x0ea50, 0x06b58, 0x05ac0, 0x0ab60, 0x096d5, 0x092e0, // 1990-1999
	0x0c960, 0x0d954, 0x0d4a0, 0x0da50, 0x07552, 0x056a0, 0x0abb7, 0x025d0, 0x092d0, 0x0cab5, // 2000-2009
	0x0a950, 0x0b4a0, 0x0baa4, 0x0ad50, 0x055d9, 0x04ba0, 0x0a5b0, 0x15176, 0x052b0, 0x0a930, // 2010-2019
	0x07954, 0x06aa0, 0x0ad50, 0x05b52, 0x04b60, 0x0a6e6, 0x0a4e0, 0x0d260, 0x0ea65, 0x0d530, // 2020-2029
	0x05aa0, 0x076a3, 0x096d0, 0x04afb, 0x04ad0, 0x0a4d0, 0x1d0b6, 0x0d250, 0x0d520, 0x0dd45, // 2030-2039
	0x0b5a0, 0x056d0, 0x055b2, 0x049b0, 0x0a577, 0x0a4b0, 0x0aa50, 0x1b255, 0x06d20, 0x0ada0, // 2040-2049
	0x14b63, 0x09370, 0x049f8, 0x04970, 0x064b0, 0x168a6, 0x0ea50, 0x06b20, 0x1a6c4, 0x0aae0, // 2050-2059
	0x0a2e0, 0x0d2e3, 0x0c960, 0x0d557, 0x0d4a0, 0x0da50, 0x05d55, 0x056a0, 0x0a6d0, 0x055d4, // 2060-2069
	0x052d0, 0x0a9b8, 0x0a950, 0x0b4a0, 0x0b6a6, 0x0ad50, 0x055a0, 0x0aba4, 0x0a5b0, 0x052b0, // 2070-2079
	0x0b273, 0x06930, 0x07337, 0x06aa0, 0x0ad50, 0x14b55, 0x04b60, 0x0a570, 0x054e4, 0x0d160, // 2080-2089
	0x0e968, 0x0d520, 0x0daa0, 0x16aa6, 0x056d0, 0x04ae0, 0x0a9d4, 0x0a2d0, 0x0d150, 0x0f252, // 2090-2099
	0x0d520, // 2100
}

const baseYear = 1900
const maxYear = 2100

func leapMonth(y int) int {
	return lunarInfo[y-baseYear] & 0xf
}

func leapDays(y int) int {
	if leapMonth(y) == 0 {
		return 0
	}
	if lunarInfo[y-baseYear]&0x10000 != 0 {
		return 30
	}
	return 29
}

func monthDays(y, m int) int {
	if lunarInfo[y-baseYear]&(0x10000>>m) != 0 {
		return 30
	}
	return 29
}

func yearDays(y int) int {
	sum := 0
	for m := 1; m <= 12; m++ {
		sum += monthDays(y, m)
	}
	return sum + leapDays(y)
}

// lunarToSolar 将农历日期转换为公历日期（按北京时区 UTC+8）。
// 闰月按平月处理：即使该年存在同名闰月，也只取第一个平月。
func lunarToSolar(y, m, d int) time.Time {
	return lunarToSolarMonth(y, m, d, false)
}

// lunarToSolarMonth 将农历日期转换为公历日期；leap 为 true 时才取闰月。
func lunarToSolarMonth(y, m, d int, leap bool) time.Time {
	if y < baseYear || y > maxYear || m < 1 || m > 12 || d < 1 {
		return time.Time{}
	}
	days := monthDays(y, m)
	if leap {
		if leapMonth(y) != m {
			return time.Time{}
		}
		days = leapDays(y)
	}
	if d > days {
		d = days
	}
	loc := time.FixedZone("CST", 8*3600)
	offset := 0
	for i := baseYear; i < y; i++ {
		offset += yearDays(i)
	}
	for i := 1; i < m; i++ {
		offset += monthDays(y, i)
	}
	if leap {
		offset += monthDays(y, m)
	}
	offset += d - 1
	base := time.Date(baseYear, 1, 31, 0, 0, 0, 0, loc) // 1900-01-31 为农历 1900 年正月初一
	return base.AddDate(0, 0, offset)
}

// date 构造公历日期（UTC，仅年月日）。
func date(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

// Lunar 返回指定农历年月日对应的公历日期字符串 YYYY-MM-DD。
func Lunar(y, m, d int) string {
	t := lunarToSolar(y, m, d)
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

// LunarOccurrence 是农历日期的一次公历发生。
type LunarOccurrence struct {
	Date      time.Time
	LunarYear int
}

func lunarDateIn(y, m, d int, loc *time.Location) time.Time {
	t := lunarToSolar(y, m, d)
	if t.IsZero() {
		return time.Time{}
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// NextLunarOccurrence 返回不早于 from 的下一次农历月日对应的公历日期及发生农历年。
// 闰月按平月处理；农历三十遇到 29 天的月份时按廿九计算。
func NextLunarOccurrence(month, day int, from time.Time) (time.Time, int, bool) {
	if month < 1 || month > 12 || day < 1 {
		return time.Time{}, 0, false
	}
	loc := from.Location()
	start := dateOnly(from)
	for lunarYear := start.Year() - 1; lunarYear <= maxYear; lunarYear++ {
		if lunarYear < baseYear {
			continue
		}
		candidate := lunarDateIn(lunarYear, month, day, loc)
		if candidate.IsZero() {
			continue
		}
		if !candidate.Before(start) {
			return candidate, lunarYear, true
		}
	}
	return time.Time{}, 0, false
}

// LunarOccurrences 返回 [from, to] 范围内农历月日的全部公历发生。
func LunarOccurrences(month, day int, from, to time.Time) []LunarOccurrence {
	if month < 1 || month > 12 || day < 1 {
		return nil
	}
	loc := from.Location()
	start := dateOnly(from)
	end := dateOnly(to)
	if end.Before(start) {
		return nil
	}
	out := make([]LunarOccurrence, 0, 2)
	seen := make(map[string]bool)
	for lunarYear := start.Year() - 1; lunarYear <= end.Year()+1; lunarYear++ {
		if lunarYear < baseYear || lunarYear > maxYear {
			continue
		}
		candidate := lunarDateIn(lunarYear, month, day, loc)
		if candidate.IsZero() || candidate.Before(start) || candidate.After(end) {
			continue
		}
		key := candidate.Format("2006-01-02")
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, LunarOccurrence{Date: candidate, LunarYear: lunarYear})
	}
	return out
}

// SpringFestival 返回指定公历年份的春节日期（农历正月初一）。
func SpringFestival(year int) string { return Lunar(year, 1, 1) }

// LanternFestival 元宵节（正月十五）。
func LanternFestival(year int) string { return Lunar(year, 1, 15) }

// DragonBoat 端午节（五月初五）。
func DragonBoat(year int) string { return Lunar(year, 5, 5) }

// MidAutumn 中秋节（八月十五）。
func MidAutumn(year int) string { return Lunar(year, 8, 15) }

// WinterSolstice 冬至（21 世纪通用公式，公历 12 月 21–23 日）。
func WinterSolstice(year int) string {
	y := year % 100
	day := int(float64(y)*0.2422+21.94) - y/4
	return date(year, 12, day).Format("2006-01-02")
}

// Qingming 清明（21 世纪通用公式，公历 4 月 4–6 日）。
func Qingming(year int) string {
	y := year % 100
	day := int(float64(y)*0.2422+4.81) - y/4
	return date(year, 4, day).Format("2006-01-02")
}

// NewYear 元旦。
func NewYear(year int) string { return date(year, 1, 1).Format("2006-01-02") }

// Christmas 圣诞节。
func Christmas(year int) string { return date(year, 12, 25).Format("2006-01-02") }

// Valentines 情人节。
func Valentines(year int) string { return date(year, 2, 14).Format("2006-01-02") }

// Holiday 描述一个可提醒的节日。
type Holiday struct {
	Key  string
	Name string
	Date func(int) string
}

// PresetHolidays 返回三类预设问候对应的节日清单。
func PresetHolidays(kind string) []Holiday {
	switch kind {
	case "spring_festival":
		return []Holiday{{Key: "spring_festival", Name: "春节", Date: SpringFestival}}
	case "traditional":
		return []Holiday{
			{Key: "lantern", Name: "元宵节", Date: LanternFestival},
			{Key: "qingming", Name: "清明节", Date: Qingming},
			{Key: "dragon_boat", Name: "端午节", Date: DragonBoat},
			{Key: "mid_autumn", Name: "中秋节", Date: MidAutumn},
			{Key: "winter_solstice", Name: "冬至", Date: WinterSolstice},
		}
	case "western":
		return []Holiday{
			{Key: "new_year", Name: "元旦", Date: NewYear},
			{Key: "valentines", Name: "情人节", Date: Valentines},
			{Key: "christmas", Name: "圣诞节", Date: Christmas},
		}
	}
	return nil
}

// NextOccurrences 返回某节日从 from 年份起的 count 次公历日期。
func NextOccurrences(h Holiday, fromYear, count int) []string {
	out := make([]string, 0, count)
	for y := fromYear; y < fromYear+count; y++ {
		if d := h.Date(y); d != "" {
			out = append(out, d)
		}
	}
	return out
}
