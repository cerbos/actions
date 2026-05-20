// Copyright 2026 Zenauth Ltd.

package timer

import (
	"log/slog"
	"time"
)

type Timer struct {
	start time.Time
}

var _ slog.LogValuer = Timer{}

func Start() Timer {
	return Timer{start: time.Now()}
}

func (t Timer) LogValue() slog.Value {
	return slog.DurationValue(time.Since(t.start).Round(time.Millisecond))
}
