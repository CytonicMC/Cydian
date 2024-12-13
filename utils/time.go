package utils

import "time"

func PointerNow() *time.Time {
	now := time.Now()
	return &now
}
