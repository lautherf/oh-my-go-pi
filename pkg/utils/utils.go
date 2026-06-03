package utils

import (
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func OmpDir() string {
	home, _ := os.UserHomeDir()
	return home + "/.omp"
}

type LogConfig struct {
	Level  string
	File   string
	Pretty bool
}

func InitLogger(cfg LogConfig) {
	lvl, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)

	var writers []io.Writer
	consoleOut := zerolog.ConsoleWriter{Out: os.Stderr, NoColor: !cfg.Pretty}
	writers = append(writers, consoleOut)

	if cfg.File != "" {
		if err := os.MkdirAll(OmpDir(), 0755); err == nil {
			f, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err == nil {
				fileOut := zerolog.ConsoleWriter{Out: f, NoColor: true}
				writers = append(writers, fileOut)
			}
		}
	}

	multi := io.MultiWriter(writers...)
	log.Logger = zerolog.New(multi).With().Timestamp().Logger()
}
