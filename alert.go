// Coping

package main

import (
	"log"
)

type ServiceState struct {
	Passing    bool
	Alerted    bool
	StateCount int
}

func (s ServiceState) Status() string {
	if s.Passing {
		return "passing"
	} else {
		return "failing"
	}
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
		state = ServiceState{passing, false, 0}
	}

	// If state has changed then reset StateCount and Alerted
	if state.Passing != passing {
		state.Passing = passing
		state.Alerted = false
		state.StateCount = 0
	}

	state.StateCount++

	if state.StateCount >= settings.AlertCount && !state.Alerted {
		log.Printf("%s is now %v for %v cycles!\n", result.url, state.Status(), settings.AlertCount)
		state.Alerted = true
	}

	serviceState[result.url] = state
}
