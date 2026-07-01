package main

import "time"

func parseStoredTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func formatStoredTime(value time.Time) string {
	if value.IsZero() {
		value = time.Now()
	}
	return value.Format(time.RFC3339Nano)
}
