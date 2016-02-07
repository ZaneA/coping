// Coping

package main

import (
	"net/http"
	"time"
)

// Stores the result of a "ping"
type FetchResult struct {
	Url      string
	Code     int
	Duration time.Duration
}

// Return whether the status is a pass or fail
func (result FetchResult) Passed() bool {
	if result.Duration > (1 * time.Second) {
		return false
	}

	if result.Code != 200 {
		return false
	}

	return true
}

// Convert a status into a PASS/WARN/FAIL string
func (result FetchResult) StatusString() (string, string) {
	if result.Passed() == true {
		return "PASS", "\x1b[1;32mPASS\x1b[0m"
	} else {
		if result.Code == -1 {
			return "FAIL", "\x1b[1;31mFAIL\x1b[0m"
		} else {
			return "WARN", "\x1b[0;33mWARN\x1b[0m"
		}
	}
}

// Check a service
func CheckService(url string, report chan FetchResult) {
	start := time.Now()
	res, err := http.Get(url)

	requestTime := time.Since(start)

	if err != nil {
		report <- FetchResult{url, -1, requestTime}
		return
	}

	report <- FetchResult{url, res.StatusCode, requestTime}
}
