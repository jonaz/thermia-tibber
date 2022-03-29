package main

import "time"

func nextDelay() time.Duration {
	now := time.Now()
	return truncateHour(now).Add(time.Hour).Sub(now)
}

func truncateHour(t time.Time) time.Time {
	t = t.Truncate(time.Minute * 30)
	if t.Minute() > 0 {
		t = t.Add(time.Minute * -1).Truncate(time.Minute * 30)
	}
	return t
}
