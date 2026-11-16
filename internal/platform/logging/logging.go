// Package logging builds the application logger and carries request-scoped
// loggers through a context.
package logging

import (
	"context"
	"io"
	"log/slog"
)

type contextKey struct{}

// New creates a logger writing to w. Format is "json" or "text".
func New(w io.Writer, level slog.Level, format string) *slog.Logger {
	options := &slog.HandlerOptions{Level: level}
	if format == "json" {
		return slog.New(slog.NewJSONHandler(w, options))
	}
	return slog.New(slog.NewTextHandler(w, options))
}

// WithContext returns a context carrying logger.
func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext returns the request-scoped logger, or the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(contextKey{}).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}
