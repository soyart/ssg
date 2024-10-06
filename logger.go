package ssg

import (
	"io"
	"log/slog"
)

type Logger struct {
	Log    *slog.Logger
	LogErr *slog.Logger
}

func SiteLogger(
	site string,
	logOut io.Writer,
	logErr io.Writer,
) Logger {
	o := slog.
		New(slog.NewTextHandler(
			logOut,
			&slog.HandlerOptions{
				AddSource: false,
				Level:     slog.LevelDebug,
			}),
		).
		With("site", site)

	e := slog.
		New(slog.NewJSONHandler(
			logOut,
			&slog.HandlerOptions{
				AddSource: false,
				Level:     slog.LevelDebug,
			}),
		).
		With("site", site)

	return Logger{
		Log:    o,
		LogErr: e,
	}
}
