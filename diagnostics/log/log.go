package log

import (
  "strings"
  "fmt"
  ct "github.com/daviddengcn/go-colortext"
)

type Level struct {
  Level int
  Prefix string
  Color ct.Color
  Bright bool
}
var (
  ErrorLevel = Level{0, "ERROR: ", ct.Red, true}
  WarnLevel  = Level{1, "WARN:  ", ct.Yellow, false}
  InfoLevel  = Level{2, "Info:  ", ct.None, true}
  DebugLevel = Level{3, "debug: ", ct.None, false}
)

var current Level = InfoLevel // default

func SetLevel(l int) {
  switch {
  case l == 0:
    current = ErrorLevel
  case l == 1:
    current = WarnLevel
  case l == 2:
    current = InfoLevel
  default:
    current = DebugLevel
  }
}

func log(l Level, msg string) {
  if l.Level <= current.Level {
    ct.ChangeColor(l.Color, l.Bright, ct.None, false)
    fmt.Println(l.Prefix + strings.Replace(msg, "\n", "\n       ",-1))
    ct.ResetColor()
  }
}

func Error(msg string) {
  log(ErrorLevel, msg)
}
func Errorf(msg string, a ...interface{}) {
  Error(fmt.Sprintf(msg, a...))
}

func Warn(msg string) {
  log(WarnLevel, msg)
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
  lines := strings.SplitN(msg, "\n", n)
  if len(lines) == n {
    lines[n-1] = "[...]"
  }
  return strings.Join(lines, "\n")
}
