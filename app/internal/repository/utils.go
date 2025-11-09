package repository

import "time"

func monthsInclusive(a, b time.Time) int {
	if b.Before(a) {
		return 0
	}
	days := int(b.Sub(a).Hours()/24) + 1
	months := (days + 29) / 30
	return months
}
