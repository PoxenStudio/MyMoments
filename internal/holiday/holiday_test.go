package holiday

import (
	"testing"
	"time"
)

func TestLunarToSolarKnownDates(t *testing.T) {
	cases := []struct {
		year, month, day int
		want             string
	}{
		{2024, 1, 1, "2024-02-10"},
		{2025, 1, 1, "2025-01-29"},
		{2023, 2, 1, "2023-02-20"}, // 平月：即使 2023 有闰二月也取第一个二月
	}
	for _, c := range cases {
		if got := Lunar(c.year, c.month, c.day); got != c.want {
			t.Fatalf("Lunar(%d,%d,%d)=%s, want %s", c.year, c.month, c.day, got, c.want)
		}
	}
}

func TestLunarLeapMonth(t *testing.T) {
	got := lunarToSolarMonth(2023, 2, 1, true).Format("2006-01-02")
	if got != "2023-03-22" {
		t.Fatalf("leap 2023-2-1=%s, want 2023-03-22", got)
	}
	if !lunarToSolarMonth(2023, 3, 1, true).IsZero() {
		t.Fatal("2023 has no leap third month")
	}
}

func TestLunarDayClamp(t *testing.T) {
	got := lunarToSolarMonth(2024, 3, 30, false)
	want := lunarToSolarMonth(2024, 3, 29, false)
	if !got.Equal(want) {
		t.Fatalf("day 30 should clamp to 29 in a 29-day month: %s vs %s", got, want)
	}
}

func TestNextLunarOccurrence(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	from := time.Date(2024, 12, 1, 0, 0, 0, 0, loc)
	next, lunarYear, ok := NextLunarOccurrence(1, 1, from)
	if !ok {
		t.Fatal("expected an occurrence")
	}
	if got := next.Format("2006-01-02"); got != "2025-01-29" {
		t.Fatalf("next=%s, want 2025-01-29", got)
	}
	if lunarYear != 2025 {
		t.Fatalf("lunarYear=%d, want 2025", lunarYear)
	}

	list := LunarOccurrences(1, 1, from, time.Date(2025, 12, 31, 0, 0, 0, 0, loc))
	if len(list) != 1 || list[0].Date.Format("2006-01-02") != "2025-01-29" {
		t.Fatalf("unexpected occurrences: %+v", list)
	}
}
