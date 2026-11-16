package config

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

// Var documents one environment variable read by the application.
type Var struct {
	Group       string
	Name        string
	Default     string
	Description string
}

// loader reads typed values from an environment lookup. It records every
// variable it is asked for, which is what Reference returns, and collects all
// parse errors instead of stopping at the first one.
type loader struct {
	lookup func(string) (string, bool)
	group  string
	vars   []Var
	errs   []error
}

func (l *loader) section(name string) { l.group = name }

func (l *loader) raw(name string, fallback string, description string) string {
	l.vars = append(l.vars, Var{Group: l.group, Name: name, Default: fallback, Description: description})

	if value, ok := l.lookup(name); ok {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return fallback
}

func (l *loader) fail(name string, format string, args ...any) {
	l.errs = append(l.errs, fmt.Errorf("%s: %s", name, fmt.Sprintf(format, args...)))
}

func (l *loader) str(name string, fallback string, description string) string {
	return l.raw(name, fallback, description)
}

func (l *loader) integer(name string, fallback int, description string) int {
	value := l.raw(name, strconv.Itoa(fallback), description)
	parsed, err := strconv.Atoi(value)
	if err != nil {
		l.fail(name, "%q is not an integer", value)
		return fallback
	}
	return parsed
}

func (l *loader) boolean(name string, fallback bool, description string) bool {
	value := l.raw(name, strconv.FormatBool(fallback), description)
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		l.fail(name, "%q is not a boolean", value)
		return fallback
	}
	return parsed
}

func (l *loader) duration(name string, fallback time.Duration, description string) time.Duration {
	value := l.raw(name, fallback.String(), description)
	parsed, err := time.ParseDuration(value)
	if err != nil {
		l.fail(name, "%q is not a duration such as 30s or 5m", value)
		return fallback
	}
	return parsed
}

func (l *loader) enum(name string, fallback string, allowed []string, description string) string {
	value := l.raw(name, fallback, description+" One of: "+strings.Join(allowed, ", ")+".")
	for _, candidate := range allowed {
		if strings.EqualFold(value, candidate) {
			return candidate
		}
	}
	l.fail(name, "%q is not one of %s", value, strings.Join(allowed, ", "))
	return fallback
}

func (l *loader) list(name string, fallback string, description string) []string {
	value := l.raw(name, fallback, description)
	items := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func (l *loader) level(name string, fallback slog.Level, description string) slog.Level {
	value := l.raw(name, fallback.String(), description+" One of: DEBUG, INFO, WARN, ERROR.")
	var level slog.Level
	if err := level.UnmarshalText([]byte(value)); err != nil {
		l.fail(name, "%q is not a log level", value)
		return fallback
	}
	return level
}
