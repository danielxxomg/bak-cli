//go:build !windows

package schedule

import (
	"testing"
)

// TestCronScheduler_TypeCheck verifies CronScheduler satisfies Scheduler.
func TestCronScheduler_TypeCheck(t *testing.T) {
	var s Scheduler = &CronScheduler{}
	if _, ok := s.(*CronScheduler); !ok {
		t.Fatal("CronScheduler does not satisfy Scheduler")
	}
}
