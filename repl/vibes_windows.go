//go:build windows

package repl

import (
	"fmt"
	"time"
)

// vibesPrompt on Windows falls back to a static prompt (no live timer).
func vibesPrompt(from string, timeLimit time.Duration, qNum int) (answer string, timedOut bool) {
	fmt.Printf("  #%d  %s  ->  ~dec\n  > ", qNum, drillColorValue(from))
	var line string
	fmt.Scanln(&line)
	return line, false
}
