package domain

import "time"

// DateRange описывает полуоткрытый временной интервал: [Start, End).
type DateRange struct {
	Start time.Time
	End   time.Time
}

// CalendarDayRange возвращает диапазон календарного дня во временной зоне day.
func CalendarDayRange(day time.Time) DateRange {
	year, month, date := day.Date()
	location := day.Location()

	return DateRange{
		Start: time.Date(year, month, date, 0, 0, 0, 0, location),
		End:   time.Date(year, month, date+1, 0, 0, 0, 0, location),
	}
}

// TodayRange возвращает диапазон текущего календарного дня.
func TodayRange(now time.Time) DateRange {
	return CalendarDayRange(now)
}

// YesterdayRange возвращает диапазон предыдущего календарного дня.
func YesterdayRange(now time.Time) DateRange {
	return CalendarDayRange(now.AddDate(0, 0, -1))
}

// Contains сообщает, входит ли момент времени в диапазон [Start, End).
func (r DateRange) Contains(value time.Time) bool {
	return !value.Before(r.Start) && value.Before(r.End)
}
