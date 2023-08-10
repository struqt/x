package logging

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

var setupOnce sync.Once

func setup() {
	setupOnce.Do(func() {
		zerologr.NameSeparator = "/"
		zerologr.NameFieldName = "n"
		zerologr.VerbosityFieldName = "v"
		zerologr.SetMaxV(LogVerbosity)
	})
}

var (
	LogRotateMBytes     uint16 = 16
	LogRotateFiles      uint16 = 64
	LogVerbosity               = 0
	LogConsoleThreshold        = int8(zerolog.ErrorLevel)
)

func NewLogger(path string) logr.Logger {
	verbose := NewLumberjack(LogRotateMBytes, LogRotateFiles, path)
	console := NewThresholdConsole()
	logger := NewZerolog(verbose, console)
	setup()
	return zerologr.New(logger)
}

func NewLumberjack(fileMBytes uint16, fileCount uint16, path string) *lumberjack.Logger {
	logger := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    int(fileMBytes),
		MaxBackups: int(fileCount),
		LocalTime:  false,
		Compress:   true,
	}
	return logger
}

func NewZerolog(writers ...io.Writer) *zerolog.Logger {
	multi := zerolog.MultiLevelWriter(writers...)
	logger := zerolog.New(multi).With().Timestamp().Caller().Logger()
	return &logger
}

type ThresholdWriter struct {
	threshold zerolog.Level
	writer    zerolog.LevelWriter
}

func (t *ThresholdWriter) Write(bytes []byte) (n int, err error) {
	return t.WriteLevel(zerolog.NoLevel, bytes)
}

func (t *ThresholdWriter) WriteLevel(level zerolog.Level, bytes []byte) (n int, err error) {
	if level >= t.threshold {
		return t.writer.WriteLevel(level, bytes)
	}
	return len(bytes), nil
}

func NewThresholdConsole() *ThresholdWriter {
	console := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	}
	return &ThresholdWriter{
		writer:    zerolog.MultiLevelWriter(console),
		threshold: zerolog.Level(LogConsoleThreshold),
	}
}
