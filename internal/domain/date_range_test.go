package domain

import (
	"testing"
	"time"
)

func TestTodayRange(t *testing.T) {
	location := time.FixedZone("test", 3*60*60)
	now := time.Date(2026, time.July, 16, 14, 35, 20, 123, location)

	got := TodayRange(now)

	assertDateRange(t, got,
		time.Date(2026, time.July, 16, 0, 0, 0, 0, location),
		time.Date(2026, time.July, 17, 0, 0, 0, 0, location),
	)
}

func TestYesterdayRange(t *testing.T) {
	tests := []struct {
		name      string
		now       time.Time
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:      "обычный день",
			now:       time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC),
			wantStart: time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, time.July, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "переход на предыдущий месяц и год",
			now:       time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
			wantStart: time.Date(2025, time.December, 31, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertDateRange(t, YesterdayRange(tt.now), tt.wantStart, tt.wantEnd)
		})
	}
}

func TestDateRangeContainsUsesHalfOpenBoundaries(t *testing.T) {
	rangeStart := time.Date(2026, time.July, 16, 0, 0, 0, 0, time.UTC)
	rangeEnd := rangeStart.AddDate(0, 0, 1)
	dateRange := DateRange{Start: rangeStart, End: rangeEnd}

	tests := []struct {
		name  string
		value time.Time
		want  bool
	}{
		{name: "начало входит", value: rangeStart, want: true},
		{name: "момент внутри входит", value: rangeStart.Add(time.Hour), want: true},
		{name: "конец не входит", value: rangeEnd, want: false},
		{name: "момент перед началом не входит", value: rangeStart.Add(-time.Nanosecond), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dateRange.Contains(tt.value); got != tt.want {
				t.Fatalf("Contains() = %t, ожидалось %t", got, tt.want)
			}
		})
	}
}

func assertDateRange(t *testing.T, got DateRange, wantStart, wantEnd time.Time) {
	t.Helper()

	if !got.Start.Equal(wantStart) {
		t.Errorf("Start = %s, ожидалось %s", got.Start, wantStart)
	}
	if !got.End.Equal(wantEnd) {
		t.Errorf("End = %s, ожидалось %s", got.End, wantEnd)
	}
}
