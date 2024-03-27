package httputil

import (
	"log/slog"
)

type (
	LevelLogger interface { // slog.Logger
		Info(msg string, args ...any)
		Warn(msg string, args ...any)
		Error(msg string, args ...any)
	}
	sLogger struct {
		logger LevelLogger
	}
	fMap = map[string]any
)

func newLogger(l LevelLogger) sLogger {
	return sLogger{logger: l}
}
func (l sLogger) log() LevelLogger {
	if l.logger != nil {
		return l.logger
	}
	return slog.Default()
}
func (l sLogger) Info(msg string, fields ...fMap) {
	l.log().Info(msg, logArgs(fields...)...)
}
func (l sLogger) Warn(msg string, fields ...fMap) {
	l.log().Warn(msg, logArgs(fields...)...)
}
func (l sLogger) Error(msg string, fields ...fMap) {
	l.log().Error(msg, logArgs(fields...)...)
}
func logArgs(fields ...fMap) []any {
	var args []any
	for _, f := range fields {
		for k, v := range f {
			args = append(args, k, v)
		}
	}
	return args
}
