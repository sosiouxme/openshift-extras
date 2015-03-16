package log

import (
	"fmt"
	ct "github.com/daviddengcn/go-colortext"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"runtime"
	"strings"
)

type Level struct {
	Level  int
	Prefix string
	Color  ct.Color
	Bright bool
}

var (
	ErrorLevel  = Level{0, "ERROR: ", ct.Red, true}
	WarnLevel   = Level{1, "WARN:  ", ct.Yellow, true}
	InfoLevel   = Level{2, "Info:  ", ct.None, false}
	NoticeLevel = Level{2, "[Note] ", ct.White, false}
	DebugLevel  = Level{3, "debug: ", ct.None, false}
)

var current Level = InfoLevel // default
var warningsSeen int = 0
var errorsSeen int = 0
var ttyOutput bool = true

func init() {
	if runtime.GOOS == "linux" && !terminal.IsTerminal(int(os.Stdout.Fd())) {
		// don't want color sequences in redirected output (logs, "less", etc.)
		ttyOutput = false
	}
}

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
		Log(WarnLevel, fmt.Sprintf("Warnings seen: %d", warningsSeen))
	}
	if errorsSeen > 0 {
		Log(ErrorLevel, fmt.Sprintf("Errors seen: %d", errorsSeen))
	}
	if warningsSeen == 0 && errorsSeen == 0 {
		Log(InfoLevel, "Completed with no errors or warnings seen.")
	}
}

func Log(l Level, msg string) {
	if l.Level <= current.Level {
		if ttyOutput {
			ct.ChangeColor(l.Color, l.Bright, ct.None, false)
		}
		fmt.Println(l.Prefix + strings.Replace(msg, "\n", "\n       ", -1))
		if ttyOutput {
			ct.ResetColor()
		}
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

func LimitLines(msg string, n int) string {
	lines := strings.SplitN(msg, "\n", n+1)
	if len(lines) == n+1 {
		lines[n] = "[...]"
	}
	return strings.Join(lines, "\n")
}
