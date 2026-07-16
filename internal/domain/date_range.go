package domain

import "time"

// DateRange описывает полуоткрытый временной интервал: [Start, End).
// Расчёт календарных диапазонов для today и yesterday относится к следующему этапу.
type DateRange struct {
	Start time.Time
	End   time.Time
}
