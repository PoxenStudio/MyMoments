package holiday

import (
	"context"
	"testing"
	"time"
)

func TestDayInfoLunarAndHoliday(t *testing.T) {
	svc := NewService("", "")
	cases := []struct {
		date     string
		lunar    string
		festival string
		holiday  string
		offDay   bool
		workday  bool
	}{
		{date: "2026-02-17", lunar: "正月", festival: "春节", holiday: "春节", offDay: true},
		{date: "2026-02-16", festival: "除夕", holiday: "春节", offDay: true},
		{date: "2026-02-14", holiday: "春节", workday: true},
		{date: "2026-09-25", lunar: "八月十五", festival: "中秋节", holiday: "中秋节", offDay: true},
		{date: "2026-04-05", festival: "清明节", holiday: "清明节", offDay: true},
		{date: "2025-10-11", holiday: "国庆节、中秋节", workday: true},
		{date: "2027-02-05", festival: "除夕", holiday: "春节", offDay: true},
		{date: "2027-02-14", holiday: "春节", workday: true},
		{date: "2026-03-01"},
	}
	for _, tc := range cases {
		got := svc.DayInfo(tc.date)
		if tc.lunar != "" && got.Lunar != tc.lunar {
			t.Errorf("%s lunar = %q, want %q", tc.date, got.Lunar, tc.lunar)
		}
		if got.Lunar == "" {
			t.Errorf("%s lunar should not be empty", tc.date)
		}
		if got.Festival != tc.festival {
			t.Errorf("%s festival = %q, want %q", tc.date, got.Festival, tc.festival)
		}
		if got.Holiday != tc.holiday {
			t.Errorf("%s holiday = %q, want %q", tc.date, got.Holiday, tc.holiday)
		}
		if got.OffDay != tc.offDay {
			t.Errorf("%s offDay = %v, want %v", tc.date, got.OffDay, tc.offDay)
		}
		if got.Workday != tc.workday {
			t.Errorf("%s workday = %v, want %v", tc.date, got.Workday, tc.workday)
		}
	}
}

func TestRangeOrderAndLength(t *testing.T) {
	svc := NewService("", "")
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	days, err := svc.Range(context.Background(), from, to)
	if err != nil {
		t.Fatalf("Range: %v", err)
	}
	if len(days) != 31 {
		t.Fatalf("len(days) = %d, want 31", len(days))
	}
	if days[0].Date != "2026-01-01" || days[30].Date != "2026-01-31" {
		t.Fatalf("unexpected range boundaries: %s ~ %s", days[0].Date, days[30].Date)
	}
	if !days[0].OffDay || days[0].Holiday != "元旦" {
		t.Fatalf("2026-01-01 should be 元旦 off day, got %+v", days[0])
	}
}

func TestQingmingFormula(t *testing.T) {
	cases := map[int]string{
		2024: "2024-04-04",
		2025: "2025-04-04",
		2026: "2026-04-05",
		2027: "2027-04-05",
	}
	for year, want := range cases {
		if got := Qingming(year); got != want {
			t.Errorf("Qingming(%d) = %s, want %s", year, got, want)
		}
	}
}
