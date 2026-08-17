// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

type Color string
const (
	Reset Color = "\033[0m"
	Red Color = "\033[31m"
	Green Color = "\033[32m"
	Yellow Color = "\033[33m"
	Blue Color = "\033[34m"
)

type PlainHandler struct {
	level slog.Leveler
	attrs []slog.Attr
	mu    *sync.Mutex
	w     io.Writer
}

func NewPlainHandler(level slog.Leveler, w io.Writer) *PlainHandler {
	if w == nil {
		w = os.Stdout
	}
	return &PlainHandler{
		level: level,
		mu:    &sync.Mutex{},
		w:     w,
	}
}

func (h *PlainHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *PlainHandler) Handle(_ context.Context, r slog.Record) error {
	timeStr := r.Time.Format("15:04:05")
	levelStr := r.Level.String()
	switch levelStr {
	case "DEBUG":
		levelStr = string(Blue) + "  [DEBUG]  " + string(Reset)
	case "INFO":
		levelStr = string(Green) + " -[.INFO]- " + string(Reset)
	case "WARN":
		levelStr = string(Yellow) + "!-[.WARN]-!" + string(Reset)
	case "ERROR":
		levelStr = string(Red) + "X=[ERROR]=X" + string(Reset)
	}

	msg := fmt.Sprintf("%s %s %s", timeStr, levelStr, r.Message)

	for _, a := range h.attrs {
		msg += fmt.Sprintf(" %s=%v", a.Key, a.Value)
	}

	r.Attrs(func(a slog.Attr) bool {
		msg += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		return true
	})

	h.mu.Lock()
	defer h.mu.Unlock()
	fmt.Fprintln(h.w, msg)
	return nil
}

func (h *PlainHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PlainHandler{
		level: h.level,
		attrs: append(append([]slog.Attr{}, h.attrs...), attrs...),
		mu:    h.mu,
		w:     h.w,
	}
}

func (h *PlainHandler) WithGroup(_ string) slog.Handler {
	return h
}
