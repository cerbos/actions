// Copyright 2021-2026 Zenauth Ltd.

package command

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lmittmann/tint"

	"github.com/cerbos/actions/internal/log"
)

func Run(command func(context.Context) error) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logLevel := slog.LevelInfo
	_ = logLevel.UnmarshalText([]byte(os.Getenv("LOG_LEVEL")))

	ctx = log.Context(ctx, slog.New(tint.NewHandler(os.Stderr, &tint.Options{Level: logLevel, TimeFormat: "15:04:05.000"})))

	if err := command(ctx); err != nil {
		log.Error(ctx, "Failed", "err", err)
		os.Exit(1) //nolint:gocritic
	}
}
