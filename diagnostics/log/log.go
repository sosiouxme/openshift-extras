package log

import "fmt"

type Level struct {
  Level int
  Prefix string
}
var (
  ErrorLevel = Level{0, "ERROR: "}
  WarnLevel = Level{1, "WARN: "}
  InfoLevel = Level{2, "Info: "}
  DebugLevel = Level{3, "debug: "}
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
    fmt.Println(l.Prefix + msg)
  }
}

func Error(msg string) {
  log(ErrorLevel, msg)
}

func Warn(msg string) {
  log(WarnLevel, msg)
}

func Info(msg string) {
  log(InfoLevel, msg)
}

func Debug(msg string) {
  log(DebugLevel, msg)
}
