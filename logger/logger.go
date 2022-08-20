package logger

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/staticbackendhq/core/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	*zerolog.Logger
}

var (
	logger Logger
	once   sync.Once
)

func newFileWriter(filename string) io.Writer {
	return &lumberjack.Logger{
		Filename: filename,
	}
}

func Get(cfg config.AppConfig) *Logger {
	once.Do(func() {
		// By default create console writer
		writers := []io.Writer{zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.Stamp}}

		if cfg.LogFilename != "" {
			writers = append(writers, newFileWriter(cfg.LogFilename))
		}

		if cfg.LogConsoleLevel != "" {
			level, err := zerolog.ParseLevel(cfg.LogConsoleLevel)
			if err != nil {
				panic(err)
			}

			zerolog.SetGlobalLevel(level)
		}

		if cfg.AppEnv == "dev" {
			zerolog.SetGlobalLevel(zerolog.TraceLevel)
		}

		multiWriters := io.MultiWriter(writers...)

		zeroLogger := zerolog.New(multiWriters).With().Timestamp().Logger()

		logger = Logger{&zeroLogger}
	})

	return &logger
}
