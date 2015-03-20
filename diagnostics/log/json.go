package log

import (
	"encoding/json"
	"fmt"
)

type mapss map[string]string
type jsonLogger struct {
	logStarted  bool
	logFinished bool
	jsonMessage mapss
}

func (j *jsonLogger) Write(l Level, msg string) {
	if j.logStarted {
		fmt.Println(",")
	} else {
		fmt.Println("[")
		j.jsonMessage = make(mapss)
	}
	j.logStarted = true
	j.jsonMessage["message"] = msg
	j.jsonMessage["level"] = l.Name
	b, _ := json.MarshalIndent(j.jsonMessage, "  ", "  ")
	fmt.Print("  " + string(b))
}
func (j *jsonLogger) Finish() {
	if j.logStarted {
		fmt.Println("\n]")
	} else if !j.logFinished {
		fmt.Println("[]")
	}
	j.logFinished = true
}
