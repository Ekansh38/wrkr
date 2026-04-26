//go:build windows

package repl

import (
	"fmt"
	"time"
)

// vibesPrompt on Windows falls back to a static prompt (no live timer).
func vibesPrompt(from string, timeLimit time.Duration) (answer string, timedOut bool) {
	fmt.Printf("  %s  ->  ~dec\n  > ", drillColorValue(from))
	var line string
	fmt.Scanln(&line)
	return line, false
}
