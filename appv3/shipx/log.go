package shipx

import (
	"fmt"
	"log/slog"

	"github.com/xgfone/ship/v5"
)

func NewLog(log *slog.Logger) ship.Logger {
	return &shipLog{log: log}
}

type shipLog struct {
	log *slog.Logger
}

func (sl *shipLog) Tracef(format string, args ...any) {
	sl.logf(slog.LevelDebug, format, args...)
}

func (sl *shipLog) Debugf(format string, args ...any) {
	sl.logf(slog.LevelDebug, format, args...)
}

func (sl *shipLog) Infof(format string, args ...any) {
	sl.logf(slog.LevelInfo, format, args...)
}

func (sl *shipLog) Warnf(format string, args ...any) {
	sl.logf(slog.LevelWarn, format, args...)
}

func (sl *shipLog) Errorf(format string, args ...any) {
	sl.logf(slog.LevelError, format, args...)
}

func (sl *shipLog) logf(level slog.Level, format string, args ...any) {
	if !sl.log.Enabled(nil, level) {
		return
	}

	size := len(args)
	if size == 0 {
		sl.log.Log(nil, level, format)
		return
	}

	var not bool
	attrs := make([]slog.Attr, 0, size)
	for _, arg := range args {
		if attr, ok := arg.(slog.Attr); ok {
			attrs = append(attrs, attr)
		} else {
			not = true
			break
		}
	}
	if not {
		msg := fmt.Sprintf(format, args...)
		sl.log.Log(nil, level, msg)
	} else {
		sl.log.LogAttrs(nil, level, format, attrs...)
	}
}
