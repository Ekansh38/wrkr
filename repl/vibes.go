//go:build !windows

package repl

// vibesPrompt implements a self-contained input loop for vibes mode with a
// live countdown timer. It bypasses liner entirely and talks to the raw
// terminal directly (liner has already configured raw/noecho mode for us).
//
// We use O_NONBLOCK on stdin so we can poll for input while updating the
// timer display without leaving a goroutine blocked on os.Stdin.Read after
// we return. Blocking mode is restored before we return, so liner's next
// Prompt call is unaffected.
//
// Display layout (two lines we own completely):
//
//	  0x80  ->  ~dec  [4s]    <- question + live timer
//	  > hello_                <- input we echo ourselves

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/fatih/color"
)

func vibesPrompt(from string, timeLimit time.Duration, qNum int) (answer string, timedOut bool) {
	urgentStyle := color.New(color.FgRed, color.Bold).SprintFunc()

	fd := uintptr(os.Stdin.Fd())

	// Save original flags and enable O_NONBLOCK for polling.
	origFlags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_GETFL, 0)
	nonblockOK := errno == 0
	if nonblockOK {
		syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_SETFL, origFlags|syscall.O_NONBLOCK) //nolint:errcheck
		defer syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_SETFL, origFlags)              //nolint:errcheck
	}

	deadline := time.Now().Add(timeLimit)
	var buf []byte
	lastSecs := -1
	initialized := false

	timerLabel := func() string {
		remaining := time.Until(deadline)
		secs := int(remaining.Seconds()) + 1
		if secs < 0 {
			secs = 0
		}
		if secs <= 2 {
			return urgentStyle(fmt.Sprintf("[%ds]", secs))
		}
		return dimGray(fmt.Sprintf("[%ds]", secs))
	}

	// redraw rewrites both lines in place.
	// force=true: always redraw (buffer changed after a keypress).
	// force=false: only redraw when the displayed second changes.
	redraw := func(force bool) {
		remaining := time.Until(deadline)
		secs := int(remaining.Seconds()) + 1
		if secs < 0 {
			secs = 0
		}
		if !force && secs == lastSecs {
			return
		}
		lastSecs = secs
		tl := timerLabel()
		qLabel := dimGray(fmt.Sprintf("#%d", qNum))
		if !initialized {
			fmt.Printf("  %s  %s  ->  ~dec  %s\n  > %s",
				qLabel, drillColorValue(from), tl, string(buf))
			initialized = true
		} else {
			// Cursor is on the input line. Go up 1, rewrite question, come
			// back, rewrite input. We own both lines - no liner state to corrupt.
			fmt.Printf("\033[1A\r  %s  %s  ->  ~dec  %s\033[K\n\r  > %s\033[K",
				qLabel, drillColorValue(from), tl, string(buf))
		}
	}

	redraw(true)

	b := make([]byte, 1)
	for {
		if time.Now().After(deadline) {
			fmt.Println()
			return string(buf), true
		}

		n, readErr := os.Stdin.Read(b)
		if n > 0 {
			switch b[0] {
			case '\r', '\n':
				fmt.Println()
				return string(buf), false
			case 127, 8: // Backspace / Delete
				if len(buf) > 0 {
					buf = buf[:len(buf)-1]
					redraw(true)
				}
			case 3, 4: // Ctrl-C / Ctrl-D — quit
				fmt.Println()
				return ":q", false
			default:
				if b[0] >= 32 && b[0] < 127 {
					buf = append(buf, b[0])
					redraw(true)
				}
			}
		} else if readErr != nil && nonblockOK {
			// EAGAIN / EWOULDBLOCK — no data yet; sleep and tick the timer.
			time.Sleep(20 * time.Millisecond)
			redraw(false)
		}
	}
}
