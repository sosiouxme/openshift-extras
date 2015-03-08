package log

import (
	"fmt"
	ct "github.com/daviddengcn/go-colortext"
	"strings"
)

type Level struct {
	Level  int
	Prefix string
	Color  ct.Color
	Bright bool
}

var (
	ErrorLevel = Level{0, "ERROR: ", ct.Red, true}
	WarnLevel  = Level{1, "WARN:  ", ct.Yellow, false}
	InfoLevel  = Level{2, "Info:  ", ct.None, true}
	DebugLevel = Level{3, "debug: ", ct.None, false}
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

func Summary() {
	fmt.Println("\nSummary of diagnostics execution:")
	if warningsSeen > 0 {
		log(WarnLevel, fmt.Sprintf("Warnings seen: %d", warningsSeen))
	}
	if errorsSeen > 0 {
		log(ErrorLevel, fmt.Sprintf("Errors seen: %d", errorsSeen))
	}
	if warningsSeen == 0 && errorsSeen == 0 {
		log(InfoLevel, "Completed with no errors or warnings seen.")
	}
}

func log(l Level, msg string) {
	if l.Level <= current.Level {
		ct.ChangeColor(l.Color, l.Bright, ct.None, false)
		fmt.Println(l.Prefix + strings.Replace(msg, "\n", "\n       ", -1))
		ct.ResetColor()
	}
}

func Error(msg string) {
	log(ErrorLevel, msg)
	errorsSeen += 1
}
func Errorf(msg string, a ...interface{}) {
	Error(fmt.Sprintf(msg, a...))
}

func Warn(msg string) {
	log(WarnLevel, msg)
	warningsSeen += 1
}
func Warnf(msg string, a ...interface{}) {
	Warn(fmt.Sprintf(msg, a...))
}

func Info(msg string) {
	log(InfoLevel, msg)
}
func Infof(msg string, a ...interface{}) {
	Info(fmt.Sprintf(msg, a...))
}

func Debug(msg string) {
	log(DebugLevel, msg)
}
func Debugf(msg string, a ...interface{}) {
	Debug(fmt.Sprintf(msg, a...))
}

func LimitLines(msg string, n int) string {
	lines := strings.SplitN(msg, "\n", n+1)
	if len(lines) == n+1 {
		lines[n] = "[...]"
	}
	return strings.Join(lines, "\n")
}
