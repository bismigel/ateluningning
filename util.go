package main

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// CapitalizeFirstLetter uppercases the first letter of s.
func CapitalizeFirstLetter(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// formatDuration renders a duration in a human-readable form.
func formatDuration(d time.Duration) string {
	s := int(d.Seconds())
	days := s / (24 * 3600)
	s %= 24 * 3600
	hours := s / 3600
	s %= 3600
	minutes := s / 60
	seconds := s % 60

	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d days", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d hours", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%d minutes", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%d seconds", seconds))
	}

	return strings.Join(parts, ", ")
}

// humanJoin renders a list with an Oxford "or": "a, b, c, or d".
func humanJoin(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", or " + items[len(items)-1]
	}
}
