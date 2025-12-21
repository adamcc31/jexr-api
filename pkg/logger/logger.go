package logger

import (
	"log/slog"
	"os"
)

var Log *slog.Logger

func Init() {
	// JSON handler for production-ready logging
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	Log = slog.New(handler)
}
