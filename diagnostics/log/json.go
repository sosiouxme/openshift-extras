package log

import (
	"encoding/json"
	"fmt"
)

type jsonMessageType struct {
	Message string `json:"message"`
	Level   string `json:"level"`
}
type jsonLogger struct {
	logStarted  bool
	logFinished bool
	jsonMessage jsonMessageType
}

func (j *jsonLogger) Write(l Level, msg string) {
	if j.logStarted {
		fmt.Println(",")
	} else {
		fmt.Println("[")
	}
	j.logStarted = true
	j.jsonMessage.Message = msg
	j.jsonMessage.Level = l.Name
	b, _ := json.MarshalIndent(j.jsonMessage, "  ", "  ")
	fmt.Print(string(b))
}
func (j *jsonLogger) Finish() {
	if j.logStarted {
		fmt.Println("\n]")
	} else if !j.logFinished {
		fmt.Println("[]")
	}
	j.logFinished = true
}
