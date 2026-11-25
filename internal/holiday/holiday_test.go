package holiday

import "testing"

func TestKnownDates(t *testing.T) {
	cases := map[int]string{
		2023: "2023-01-22",
		2024: "2024-02-10",
		2025: "2025-01-29",
		2026: "2026-02-17",
		2027: "2027-02-06",
		2028: "2028-01-26",
	}
	for y, want := range cases {
		if got := SpringFestival(y); got != want {
			t.Errorf("SpringFestival(%d) = %s, want %s", y, got, want)
		}
	}
	// 端午、中秋：2026 端午=2026-06-19, 中秋=2026-09-25
	if got := DragonBoat(2026); got != "2026-06-19" {
		t.Errorf("DragonBoat(2026) = %s, want 2026-06-19", got)
	}
	if got := MidAutumn(2026); got != "2026-09-25" {
		t.Errorf("MidAutumn(2026) = %s, want 2026-09-25", got)
	}
}
