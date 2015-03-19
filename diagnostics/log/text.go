package log

import (
	"fmt"
	ct "github.com/daviddengcn/go-colortext"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"runtime"
	"strings"
)

var ttyOutput bool = true

func init() {
	if runtime.GOOS == "linux" && !terminal.IsTerminal(int(os.Stdout.Fd())) {
		// don't want color sequences in redirected output (logs, "less", etc.)
		ttyOutput = false
	}
}

type textLogger struct{}

func (t *textLogger) Write(l Level, msg string) {
	if ttyOutput {
		ct.ChangeColor(l.Color, l.Bright, ct.None, false)
	}
	fmt.Println(l.Prefix + strings.Replace(msg, "\n", "\n       ", -1))
	if ttyOutput {
		ct.ResetColor()
	}
}
func (t *textLogger) Finish() {}
