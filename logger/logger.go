package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/staticbackendhq/core/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

var once sync.Once

func newFileWriter(filename string) io.Writer {
	return &lumberjack.Logger{
		Filename: filename,
		MaxAge:   22,
	}
}

func Setup(cfg config.AppConfig) {
	once.Do(func() {
		level := parseLevel(cfg.LogConsoleLevel)
		if cfg.AppEnv == "dev" {
			level = slog.LevelDebug
		}
		// By default create console writer
		writers := []io.Writer{os.Stdout}

		if cfg.LogFilename != "" {
			writers = append(writers, newFileWriter(cfg.LogFilename))
		}

		multiWriters := io.MultiWriter(writers...)

		handler := slog.NewTextHandler(multiWriters, &slog.HandlerOptions{Level: level})

		slog.SetDefault(slog.New(handler))
	})
}

// FatalError logs a structured error message and exits the process.
func FatalError(msg string, err error, args ...any) {
	if err != nil {
		args = append(args, "error", err)
	}
	slog.Error(msg, args...)
	os.Exit(1)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug", "trace":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
