// Coping

package main

import (
	"fmt"
	"time"
)

type ServiceState struct {
	Code       int
	Passing    bool
	Alerted    bool
	StateCount int
}

var serviceState map[string]ServiceState

func init() {
	serviceState = make(map[string]ServiceState)
}

// Alert about a result
func MaybeAlert(settings *Settings, result FetchResult) {
	state, ok := serviceState[result.url]

	passing := result.Passed()

	if !ok && passing {
		// Default state of passing so just ignore
		return
	}

	if !ok {
		state = ServiceState{result.code, passing, false, 0}
	}

	// If state has changed then reset StateCount and Alerted
	if state.Code != result.code {
		state.Code = result.code
		state.Passing = result.Passed()
		state.Alerted = false
		state.StateCount = 0
	}

	state.StateCount++

	if state.StateCount >= settings.AlertCount && !state.Alerted {
		// Alert output to be fed into another program
		fmt.Printf("%v;%s;%v;%v;%v;%v\n", time.Now().Unix(), result.url, result.code, result.Duration, result.StatusString(false), settings.AlertCount)
		state.Alerted = true
	}

	serviceState[result.url] = state
}
