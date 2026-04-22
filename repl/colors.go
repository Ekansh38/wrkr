package repl

import (
	"regexp"
	"strings"

	"github.com/fatih/color"

	"wrkr/engine"
)

// ── Style definitions ─────────────────────────────────────────────────────────

var (
	boldWhite = color.New(color.Bold, color.FgHiWhite).SprintFunc()
	dimGray   = color.New(color.FgHiBlack).SprintFunc()

	styleError       = color.New(color.FgRed, color.Bold).SprintFunc()
	styleAutocorrect = color.New(color.FgYellow).SprintFunc()
	styleVarName     = color.New(color.FgYellow).SprintFunc()
	styleModeLabel   = color.New(color.Bold).SprintFunc()

	styleHex = color.New(color.FgCyan).SprintFunc()
	styleBin = color.New(color.FgGreen).SprintFunc()
	styleOct = color.New(color.FgYellow).SprintFunc()

	// Per-mode prompt color.
	modeColor = map[string]func(...interface{}) string{
		"dec":   color.New(color.FgHiWhite).SprintFunc(),
		"size":  color.New(color.FgBlue).SprintFunc(),
		"hex":   color.New(color.FgCyan).SprintFunc(),
		"bin":   color.New(color.FgGreen).SprintFunc(),
		"oct":   color.New(color.FgYellow).SprintFunc(),
		"bytes": color.New(color.FgHiWhite).SprintFunc(),
		"bits":  color.New(color.FgMagenta).SprintFunc(),
	}
)

// modePrompt returns the "[mode] > " prompt string colored for the current mode.
// Note: liner does not account for invisible ANSI bytes when tracking cursor
// position, so very long input lines may wrap slightly off on some terminals.
// This is a known liner limitation; for a personal tool it is acceptable.
func modePrompt() string {
	fn, ok := modeColor[engine.CurrentMode]
	if !ok {
		fn = modeColor["dec"]
	}
	return fn("["+engine.CurrentMode+"]") + " > "
}

// colorizeResult applies colors to a formatted result string:
//
//   - 0x… cyan   0b… green   0o… yellow
//   - "number  [hint]"  →  bold number  +  dim bracket
//   - "number unit"     →  bold number  +  dim unit label  (size/bytes/bits modes)
//   - plain number      →  bold
func colorizeResult(s string) string {
	low := strings.ToLower(s)

	// Base-prefixed outputs.
	if strings.HasPrefix(low, "0x") {
		return applyBaseColor(s, styleHex)
	}
	if strings.HasPrefix(s, "0b") {
		return applyBaseColor(s, styleBin)
	}
	if strings.HasPrefix(s, "0o") {
		return applyBaseColor(s, styleOct)
	}

	// "number  [hint bracket]" — dec mode smart hint.
	reHint := regexp.MustCompile(`^([-\d.]+)(  \[.+\])$`)
	if m := reHint.FindStringSubmatch(s); m != nil {
		return boldWhite(m[1]) + dimGray(m[2])
	}

	// "number unit" — size / bytes / bits / conversion target labels.
	reUnit := regexp.MustCompile(`^([-\d.]+)\s+([A-Za-z]+)$`)
	if m := reUnit.FindStringSubmatch(s); m != nil {
		return boldWhite(m[1]) + " " + dimGray(m[2])
	}

	return boldWhite(s)
}

// applyBaseColor colors the base value and dims the tag (e.g. "0xFF  [Hex]").
func applyBaseColor(s string, styleFn func(...interface{}) string) string {
	reTag := regexp.MustCompile(`^(.*?)(  \[.+\])$`)
	if m := reTag.FindStringSubmatch(s); m != nil {
		return styleFn(m[1]) + dimGray(m[2])
	}
	return styleFn(s)
}

// colorMode returns the mode name colored in its associated color.
func colorMode(mode string) string {
	fn, ok := modeColor[mode]
	if !ok {
		return styleModeLabel(mode)
	}
	return fn(mode)
}
