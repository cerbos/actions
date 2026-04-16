// Copyright 2026 Zenauth Ltd.

package log

import (
	"context"
	"log/slog"
)

type loggerKey struct{}

func Context(parent context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(parent, loggerKey{}, logger)
}

func Get(ctx context.Context) *slog.Logger {
	return ctx.Value(loggerKey{}).(*slog.Logger) //nolint:forcetypeassert
}

func With(ctx context.Context, args ...any) context.Context {
	return Context(ctx, Get(ctx).With(args...))
}

func Debug(ctx context.Context, message string, args ...any) {
	Get(ctx).DebugContext(ctx, message, args...)
}

func Info(ctx context.Context, message string, args ...any) {
	Get(ctx).InfoContext(ctx, message, args...)
}

func Warn(ctx context.Context, message string, args ...any) {
	Get(ctx).WarnContext(ctx, message, args...)
}

func Error(ctx context.Context, message string, args ...any) {
	Get(ctx).ErrorContext(ctx, message, args...)
}
