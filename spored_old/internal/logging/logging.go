// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

type PlainHandler struct {
	level slog.Leveler
	attrs []slog.Attr
	mu    *sync.Mutex
}

func NewPlainHandler(level slog.Leveler) *PlainHandler {
	return &PlainHandler{
		level: level,
		mu:    &sync.Mutex{},
	}
}

func (h *PlainHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *PlainHandler) Handle(_ context.Context, r slog.Record) error {
	timeStr := r.Time.Format("15:04:05")
	levelStr := r.Level.String()
	if len(levelStr) == 4 {
		levelStr = "." + levelStr
	}

	msg := fmt.Sprintf("%s -=[%s]=- %s", timeStr, levelStr, r.Message)

	for _, a := range h.attrs {
		msg += fmt.Sprintf(" %s=%v", a.Key, a.Value)
	}

	r.Attrs(func(a slog.Attr) bool {
		msg += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		return true
	})

	h.mu.Lock()
	defer h.mu.Unlock()
	fmt.Fprintln(os.Stdout, msg)
	return nil
}

func (h *PlainHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PlainHandler{
		level: h.level,
		attrs: append(append([]slog.Attr{}, h.attrs...), attrs...),
		mu:    h.mu,
	}
}

func (h *PlainHandler) WithGroup(_ string) slog.Handler {
	return h
}
