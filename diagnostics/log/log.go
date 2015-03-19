package log

import (
	"errors"
	"fmt"
	ct "github.com/daviddengcn/go-colortext"
	"strings"
)

type Level struct {
	Level  int
	Name   string
	Prefix string
	Color  ct.Color
	Bright bool
}

var (
	ErrorLevel  = Level{0, "error", "ERROR: ", ct.Red, true}
	WarnLevel   = Level{1, "warn", "WARN:  ", ct.Yellow, true}
	InfoLevel   = Level{2, "info", "Info:  ", ct.None, false}
	NoticeLevel = Level{2, "note", "[Note] ", ct.White, false}
	DebugLevel  = Level{3, "debug", "debug: ", ct.None, false}
)

var current Level = InfoLevel // default
var warningsSeen int = 0
var errorsSeen int = 0

func SetLevel(level int) {
	switch level {
	case 0:
		current = ErrorLevel
	case 1:
		current = WarnLevel
	case 2:
		current = InfoLevel
	default:
		current = DebugLevel
	}
}

//
// Deal with different log formats
//
type loggerType interface {
	Write(Level, string)
	Finish()
}

var logger loggerType = &textLogger{} // default
func SetLogFormat(format string) error {
	logger = &textLogger{} // default
	switch format {
	case "json":
		logger = &jsonLogger{}
	case "yaml":
		logger = &yamlLogger{}
	case "text":
	default:
		return errors.New("Output format must be one of: text, json, yaml")
	}
	return nil
}

// Provide a summary at the end
func Summary() {
	Log(InfoLevel, "\nSummary of diagnostics execution:\n")
	if warningsSeen > 0 {
		Log(InfoLevel, fmt.Sprintf("Warnings seen: %d", warningsSeen))
	}
	if errorsSeen > 0 {
		Log(InfoLevel, fmt.Sprintf("Errors seen: %d", errorsSeen))
	}
	if warningsSeen == 0 && errorsSeen == 0 {
		Log(InfoLevel, "Completed with no errors or warnings seen.")
	}
}

func Log(l Level, msg string) {
	if l.Level <= current.Level {
		logger.Write(l, msg)
	}
	if l.Level == ErrorLevel.Level {
		errorsSeen += 1
	} else if l.Level == WarnLevel.Level {
		warningsSeen += 1
	}
}

func Notice(msg string) {
	Log(NoticeLevel, msg)
}
func Noticef(msg string, a ...interface{}) {
	Notice(fmt.Sprintf(msg, a...))
}

func Error(msg string) {
	Log(ErrorLevel, msg)
}
func Errorf(msg string, a ...interface{}) {
	Error(fmt.Sprintf(msg, a...))
}

func Warn(msg string) {
	Log(WarnLevel, msg)
}
func Warnf(msg string, a ...interface{}) {
	Warn(fmt.Sprintf(msg, a...))
}

func Info(msg string) {
	Log(InfoLevel, msg)
}
func Infof(msg string, a ...interface{}) {
	Info(fmt.Sprintf(msg, a...))
}

func Debug(msg string) {
	Log(DebugLevel, msg)
}
func Debugf(msg string, a ...interface{}) {
	Debug(fmt.Sprintf(msg, a...))
}

// turn excess lines into [...]
func LimitLines(msg string, n int) string {
	lines := strings.SplitN(msg, "\n", n+1)
	if len(lines) == n+1 {
		lines[n] = "[...]"
	}
	return strings.Join(lines, "\n")
}

func Finish() {
	logger.Finish()
}
