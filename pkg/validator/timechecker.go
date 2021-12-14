package validator

import (
	"errors"
	"fmt"
	"time"
)

// CheckTimeBeforeNow : check if the given string is a valid duration string or utc datetime string,
// and check if the time before now.
func CheckTimeBeforeNow(str string) (time.Time, error) {
	now := time.Now()

	// str equals now()
	if str == "now()" {
		return now, nil
	}
	// str is relative time duration, like -20h5m2s
	if d, err := time.ParseDuration(str); err == nil {
		t := now.Add(d)
		if t.Before(now) {
			return t, nil
		}
		return time.Time{}, errors.New("time should before now")
	}
	// str is absolute time format with utc datetime string, like 2021-10-02T15:04:05Z
	if t, err := time.Parse("2006-01-02T15:04:05Z", str); err == nil {
		if t.Before(now) {
			return t, nil
		}
		return time.Time{}, errors.New("time should before now")
	}
	return time.Time{}, errors.New("invalid string")
}

// CheckDurationPositive : check if the given string is a valid duration string,
// and check if the duration is positive. A positive duration like 5h30m, while the negative like -5h30m
func CheckDurationPositive(str string) (time.Duration, error) {
	if d, err := time.ParseDuration(str); err != nil {
		return 0, fmt.Errorf("invaild duration string")
	} else {
		now := time.Now()
		if now.Add(d).Before(now) {
			return 0, fmt.Errorf("negative duration string")
		}
		return d, nil
	}
}
