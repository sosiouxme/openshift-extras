package log

import "fmt"

type Level int
const (
  ErrorLevel Level = iota
  WarnLevel
  InfoLevel
  DebugLevel
)

var logLevel Level = InfoLevel // default

func SetLevel(l Level) {
  logLevel = l
}

func log(l Level, msg string) {
  if l <= logLevel {
    fmt.Println(msg)
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
