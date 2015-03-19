package log

import (
	"fmt"
	"gopkg.in/yaml.v2"
)

type yamlMessageType struct {
	Message string
	Level   string
}
type yamlLogger struct {
	logStarted  bool
	yamlMessage yamlMessageType
}

func (y *yamlLogger) Write(l Level, msg string) {
	y.yamlMessage.Level = l.Name
	y.yamlMessage.Message = msg
	b, _ := yaml.Marshal(&y.yamlMessage)
	fmt.Println("---\n" + string(b))
}
func (y *yamlLogger) Finish() {}
