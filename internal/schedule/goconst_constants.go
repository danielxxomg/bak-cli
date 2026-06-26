package schedule

// String constants extracted to satisfy goconst (min-occurrences 3).
// These values appeared 3+ times across production code.

const (
	cronDailyAt2AM   = "0 2 * * *"
	scheduleDaily    = "daily"
	scheduleEvery12h = "every-12h"
	scheduleEvery6h  = "every-6h"
	scheduleWeekly   = "weekly"
)
