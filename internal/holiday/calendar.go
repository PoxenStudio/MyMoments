package holiday

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed china_holidays.json
var chinaHolidaysJSON []byte

const dateLayout = "2006-01-02"

// LunarDate 表示公历日期对应的农历日期。
type LunarDate struct {
	Year  int  `json:"year"`
	Month int  `json:"month"`
	Day   int  `json:"day"`
	Leap  bool `json:"leap"`
}

// DayInfo 描述公历某一天的农历、节日与休假信息。
type DayInfo struct {
	Date       string `json:"date"`
	Lunar      string `json:"lunar"`
	LunarMonth int    `json:"lunarMonth"`
	LunarDay   int    `json:"lunarDay"`
	LunarLeap  bool   `json:"lunarLeap"`
	Festival   string `json:"festival,omitempty"`
	Holiday    string `json:"holiday,omitempty"`
	OffDay     bool   `json:"offDay"`
	Workday    bool   `json:"workday"`
}

type statutoryDay struct {
	Name     string `json:"name"`
	Date     string `json:"date"`
	IsOffDay bool   `json:"isOffDay"`
}

type statutoryFile struct {
	Year int            `json:"year"`
	Days []statutoryDay `json:"days"`
}

type dayEntry struct {
	name   string
	offDay bool
}

// Service 维护农历转换与法定节假日数据。
// 内置 2024–2027 年数据，并可从远程数据源（默认 holiday-cn）按年更新。
type Service struct {
	mu        sync.RWMutex
	days      map[string]dayEntry
	builtin   map[int]bool
	loaded    map[int]bool
	refreshed map[int]bool
	fetching  map[int]bool

	dataDir string
	dataURL string
	client  *http.Client
	now     func() time.Time
}

// NewService 创建节假日服务。dataDir 用于缓存远程数据，dataURL 支持 {year} 或 %d 占位。
func NewService(dataDir, dataURL string) *Service {
	s := &Service{
		days:      make(map[string]dayEntry),
		builtin:   make(map[int]bool),
		loaded:    make(map[int]bool),
		refreshed: make(map[int]bool),
		fetching:  make(map[int]bool),
		dataDir:   dataDir,
		dataURL:   dataURL,
		client:    &http.Client{Timeout: 4 * time.Second},
		now:       time.Now,
	}
	if len(chinaHolidaysJSON) > 0 {
		var f statutoryFile
		if err := json.Unmarshal(chinaHolidaysJSON, &f); err == nil {
			for _, d := range f.Days {
				s.days[d.Date] = dayEntry{name: d.Name, offDay: d.IsOffDay}
				if y := parseYear(d.Date); y > 0 {
					s.builtin[y] = true
				}
			}
		}
	}
	s.loadDisk()
	go s.refreshUpcoming()
	return s
}

// refreshUpcoming 在后台刷新当前及未来两年的数据，避免阻塞日历首屏。
func (s *Service) refreshUpcoming() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	year := s.now().Year()
	for i := 0; i <= 2; i++ {
		s.ensure(ctx, year+i, true)
	}
}

func parseYear(date string) int {
	t, err := time.Parse(dateLayout, date)
	if err != nil {
		return 0
	}
	return t.Year()
}

// loadDisk 读取缓存目录中的年度数据；同一日期以磁盘数据为准（通常是较新的公告）。
func (s *Service) loadDisk() {
	if s.dataDir == "" {
		return
	}
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(s.dataDir, entry.Name()))
		if err != nil {
			continue
		}
		var f statutoryFile
		if json.Unmarshal(body, &f) != nil {
			continue
		}
		for _, d := range f.Days {
			s.days[d.Date] = dayEntry{name: d.Name, offDay: d.IsOffDay}
		}
		if y := parseYear(strings.TrimSuffix(entry.Name(), ".json") + "-01-01"); y > 0 {
			s.loaded[y] = true
		}
	}
}

// Range 返回闭区间 [from, to] 内每天的农历与休假信息。
func (s *Service) Range(ctx context.Context, from, to time.Time) ([]DayInfo, error) {
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	to = time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	if to.Before(from) {
		return nil, fmt.Errorf("结束日期早于开始日期")
	}
	total := int(to.Sub(from).Hours()/24) + 1
	if total > 400 {
		return nil, fmt.Errorf("日期范围不能超过 400 天")
	}
	years := make(map[int]bool)
	for t := from; !t.After(to); t = t.AddDate(0, 0, 1) {
		years[t.Year()] = true
	}
	for y := range years {
		s.ensure(ctx, y, false)
	}
	out := make([]DayInfo, 0, total)
	for t := from; !t.After(to); t = t.AddDate(0, 0, 1) {
		out = append(out, s.DayInfo(t.Format(dateLayout)))
	}
	return out, nil
}

// DayInfo 返回某一天（YYYY-MM-DD）的农历与休假信息。
func (s *Service) DayInfo(dateStr string) DayInfo {
	info := DayInfo{Date: dateStr}
	t, err := time.Parse(dateLayout, dateStr)
	if err != nil {
		return info
	}
	if lunar, ok := SolarToLunar(t); ok {
		info.LunarMonth = lunar.Month
		info.LunarDay = lunar.Day
		info.LunarLeap = lunar.Leap
		info.Lunar = lunarLabel(lunar)
		info.Festival = traditionalFestival(lunar, t)
	}
	s.mu.RLock()
	if entry, ok := s.days[dateStr]; ok {
		info.Holiday = entry.name
		info.OffDay = entry.offDay
		info.Workday = !entry.offDay
	}
	s.mu.RUnlock()
	return info
}

func (s *Service) ensure(ctx context.Context, year int, force bool) {
	if s.dataURL == "" || year < 2020 || year > s.now().Year()+3 {
		return
	}
	if !s.beginFetch(year, force) {
		return
	}
	defer s.endFetch(year)

	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, s.yearURL(year), nil)
	if err != nil {
		return
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return
	}
	var f statutoryFile
	if json.Unmarshal(body, &f) != nil || len(f.Days) == 0 {
		return
	}
	s.mu.Lock()
	for _, d := range f.Days {
		s.days[d.Date] = dayEntry{name: d.Name, offDay: d.IsOffDay}
	}
	s.loaded[year] = true
	s.mu.Unlock()
	if s.dataDir != "" {
		if os.MkdirAll(s.dataDir, 0o755) == nil {
			_ = os.WriteFile(filepath.Join(s.dataDir, strconv.Itoa(year)+".json"), body, 0o644)
		}
	}
}

func (s *Service) beginFetch(year int, force bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fetching[year] || s.refreshed[year] {
		return false
	}
	// 非强制刷新仅在数据缺失时请求远程；已内置或已缓存的年份不阻塞请求。
	if !force && (s.builtin[year] || s.loaded[year]) {
		return false
	}
	// 历史年份无需反复刷新。
	if year < s.now().Year() && (s.builtin[year] || s.loaded[year]) {
		return false
	}
	s.fetching[year] = true
	return true
}

func (s *Service) endFetch(year int) {
	s.mu.Lock()
	delete(s.fetching, year)
	s.refreshed[year] = true
	s.mu.Unlock()
}

func (s *Service) yearURL(year int) string {
	if strings.Contains(s.dataURL, "{year}") {
		return strings.ReplaceAll(s.dataURL, "{year}", strconv.Itoa(year))
	}
	if strings.Contains(s.dataURL, "%d") {
		return fmt.Sprintf(s.dataURL, year)
	}
	return strings.TrimRight(s.dataURL, "/") + "/" + strconv.Itoa(year) + ".json"
}

// SolarToLunar 将公历日期转换为农历日期。
func SolarToLunar(t time.Time) (LunarDate, bool) {
	cur := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	base := time.Date(baseYear, 1, 31, 0, 0, 0, 0, time.UTC)
	offset := int(cur.Sub(base).Hours() / 24)
	if offset < 0 {
		return LunarDate{}, false
	}
	year := baseYear
	for year <= maxYear {
		days := yearDays(year)
		if offset < days {
			break
		}
		offset -= days
		year++
	}
	if year > maxYear {
		return LunarDate{}, false
	}
	for _, m := range monthsOfYear(year) {
		if offset < m.days {
			return LunarDate{Year: year, Month: m.month, Day: offset + 1, Leap: m.leap}, true
		}
		offset -= m.days
	}
	return LunarDate{}, false
}

type lunarMonthInfo struct {
	month int
	leap  bool
	days  int
}

func monthsOfYear(year int) []lunarMonthInfo {
	leap := leapMonth(year)
	out := make([]lunarMonthInfo, 0, 13)
	for m := 1; m <= 12; m++ {
		out = append(out, lunarMonthInfo{month: m, days: monthDays(year, m)})
		if leap == m {
			out = append(out, lunarMonthInfo{month: m, leap: true, days: leapDays(year)})
		}
	}
	return out
}

var lunarMonthNames = [...]string{"", "正月", "二月", "三月", "四月", "五月", "六月", "七月", "八月", "九月", "十月", "冬月", "腊月"}

var lunarDayNames = [...]string{
	"", "初一", "初二", "初三", "初四", "初五", "初六", "初七", "初八", "初九", "初十",
	"十一", "十二", "十三", "十四", "十五", "十六", "十七", "十八", "十九", "二十",
	"廿一", "廿二", "廿三", "廿四", "廿五", "廿六", "廿七", "廿八", "廿九", "三十",
}

func lunarLabel(l LunarDate) string {
	if l.Month < 1 || l.Month > 12 || l.Day < 1 || l.Day > 30 {
		return ""
	}
	prefix := ""
	if l.Leap {
		prefix = "闰"
	}
	month := prefix + lunarMonthNames[l.Month]
	if l.Day == 1 {
		return month
	}
	return month + lunarDayNames[l.Day]
}

func traditionalFestival(l LunarDate, t time.Time) string {
	switch {
	case l.Month == 1 && l.Day == 1:
		return "春节"
	case l.Month == 1 && l.Day == 15:
		return "元宵节"
	case l.Month == 5 && l.Day == 5:
		return "端午节"
	case l.Month == 7 && l.Day == 7:
		return "七夕"
	case l.Month == 8 && l.Day == 15:
		return "中秋节"
	case l.Month == 9 && l.Day == 9:
		return "重阳节"
	case l.Month == 12 && l.Day == 8:
		return "腊八节"
	}
	if next, ok := SolarToLunar(t.AddDate(0, 0, 1)); ok && next.Month == 1 && next.Day == 1 {
		return "除夕"
	}
	dateStr := t.Format(dateLayout)
	if dateStr == Qingming(t.Year()) {
		return "清明节"
	}
	if dateStr == WinterSolstice(t.Year()) {
		return "冬至"
	}
	return ""
}
