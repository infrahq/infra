package types

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Duration time.Duration

func (d *Duration) String() string {
	if d == nil {
		return "0s"
	}
	return time.Duration(*d).String()
}

func (d *Duration) Set(s string) error {
	v, err := parseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(v)
	return nil
}

func (d *Duration) Type() string {
	return "duration"
}

func parseDuration(raw string) (time.Duration, error) {
	if raw == "" {
		return 0, fmt.Errorf(`invalid duration ""`)
	}

	var result time.Duration

	years, rest, ok := strings.Cut(raw, "y")
	if ok {
		raw = rest
		v, err := strconv.ParseInt(years, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number of years: %v", years)
		}
		result += time.Duration(v) * 365 * 24 * time.Hour
	}

	weeks, rest, ok := strings.Cut(raw, "w")
	if ok {
		raw = rest
		v, err := strconv.ParseInt(weeks, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number of weeks: %v", weeks)
		}
		result += time.Duration(v) * 7 * 24 * time.Hour
	}

	days, rest, ok := strings.Cut(raw, "d")
	if ok {
		raw = rest
		v, err := strconv.ParseInt(days, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number of days: %v", days)
		}
		result += time.Duration(v) * 24 * time.Hour
	}

	if raw != "" {
		v, err := time.ParseDuration(raw)
		if err != nil {
			return 0, err
		}
		result += v
	}
	return result, nil
}
