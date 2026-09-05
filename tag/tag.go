// Package tag builds Apprise tag expressions for per-recipient notification routing.
package tag

import (
	"fmt"
	"strings"
)

// Person returns prefix concatenated with personID (e.g. starapp_uid_42).
func Person(prefix string, personID int) string {
	return fmt.Sprintf("%s%d", prefix, personID)
}

// Join builds a comma-separated OR tag expression for the given person ids.
func Join(prefix string, personIDs []int) string {
	if len(personIDs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(personIDs))
	for _, id := range personIDs {
		parts = append(parts, Person(prefix, id))
	}
	return strings.Join(parts, ",")
}
