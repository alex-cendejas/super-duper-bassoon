package clock

import "time"

// SystemClock is the real-time clock using the system clock.
type SystemClock struct{}

func (SystemClock) Now() time.Time      { return time.Now() }
func (SystemClock) Sleep(d time.Duration) { time.Sleep(d) }
