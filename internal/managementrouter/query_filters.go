package managementrouter

import (
	"fmt"
	"net/url"
	"strings"
)

var validStates = map[string]bool{
	"pending":  true,
	"firing":   true,
	"silenced": true,
}

// parseStateAndLabels returns the optional state filter and label matches.
// Any query param other than "state" is treated as a label match.
// Repeated query params for the same key are collected as multiple accepted
// values (OR semantics): ?severity=critical&severity=warning matches alerts
// whose severity is either "critical" or "warning".
// Returns an error if the state value is not one of the known states.
func parseStateAndLabels(q url.Values) (string, map[string][]string, error) {
	state := strings.ToLower(strings.TrimSpace(q.Get("state")))
	if state != "" && !validStates[state] {
		return "", nil, fmt.Errorf("invalid state filter %q: must be one of pending, firing, silenced", state)
	}

	labels := make(map[string][]string)
	for key, vals := range q {
		if key == "state" {
			continue
		}
		trimmedKey := strings.TrimSpace(key)
		for _, v := range vals {
			trimmed := strings.TrimSpace(v)
			if trimmed == "" {
				continue
			}
			labels[trimmedKey] = append(labels[trimmedKey], trimmed)
		}
	}
	return state, labels, nil
}
