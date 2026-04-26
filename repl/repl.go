package repl

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/expr-lang/expr"
	"github.com/fatih/color"
	"github.com/peterh/liner"

	"github.com/Ekansh38/wrkr/drill"
	"github.com/Ekansh38/wrkr/engine"
)

// containsFormatFn returns true if the expression contains a bare format-function call
// (hex/bin/oct/dec and their width variants) - used to generate hints on compile errors.
func containsFormatFn(s string) bool {
	for _, fn := range []string{"hex(", "bin(", "oct(", "octal(", "dec(", "decimal("} {
		if strings.Contains(s, fn) {
			return true
		}
	}
	// Width-specific: bin8..bin512, hex8..hex128, oct8..oct64
	for _, prefix := range []string{"bin", "hex", "oct"} {
		for _, w := range []string{"8(", "16(", "32(", "64(", "128(", "256(", "512("} {
			if strings.Contains(s, prefix+w) {
				return true
			}
		}
	}
	return false
}

func printHelp(topic string) {
	fmt.Println()
	switch strings.ToLower(topic) {
	case "math", "geometry", "gamedev":
		fmt.Println("math")
		fmt.Println()
		fmt.Println("  sin(x)  cos(x)  tan(x)  hypot(a,b)  pi")
		fmt.Println("  sqrt(x)  abs(x)  pow(base,exp)  min(a,b)  max(a,b)")
		fmt.Println("  round(x)  floor(x)  ceil(x)")
		fmt.Println("  log(x)  log2(x)  log10(x)")
		fmt.Println()
		fmt.Println("  hypot(3,4)         -> 5")
		fmt.Println("  pow(2,10)          -> 1024")
		fmt.Println("  floor(log2(1000))  -> 9")
		fmt.Println("  sin(pi/2)          -> 1")
	case "systems", "computer", "hardware":
		fmt.Println("base literals")
		fmt.Println()
		fmt.Println("  input:")
		fmt.Println("    0xFF  0b1010  0o17           prefix")
		fmt.Println("    FF hex  101 bin  17 octal    natural (suffix = base of digits)")
		fmt.Println("    \\xFF  \\b1010  \\o17           typo shorthand")
		fmt.Println("    1_000_000  0b1011_1011        _ separators stripped before eval")
		fmt.Println()
		fmt.Println("  output:")
		fmt.Println("    hex(255)  bin(255)  octal(255)  dec(0xFF)    function")
		fmt.Println("    255 to hex  0xFF to bin                      to keyword")
		fmt.Println("    0x123 hex to bin  0b1010 bin to hex          annotated source")
		fmt.Println()
		fmt.Println("  bitwise:  & | ^ ~ << >>  (standard C precedence)")
		fmt.Println("  bswap16/32/64(x)   byte-swap for endianness conversion")
		fmt.Println("  popcount(x)        count set bits (Hamming weight)")
		fmt.Println("    0xFF & 0x0F              -> 15      (low nibble)")
		fmt.Println("    (0xAB >> 4) & 0xF        -> 10      (high nibble)")
		fmt.Println("    1 << 5                   -> 32      (set bit 5)")
		fmt.Println("    0x12345 & ~(4096-1)      -> 73728   (page-align)")
		fmt.Println("    ~0                       -> -1      (all bits set, int64)")
		fmt.Println()
		fmt.Println("  data sizes:  b  bit  kb  mb  gb  tb")
		fmt.Println("    5 mb               -> 5242880")
		fmt.Println("    2 * tb / (4 * kb)  -> 536870912")
	case "units", "conversions":
		fmt.Println("units")
		fmt.Println()
		fmt.Println("  distance:  m  km  cm  mm  mi  ft  in")
		fmt.Println("  data:      b  bit  kb  mb  gb  tb")
		fmt.Println()
		fmt.Println("  <number> <unit> to <unit>")
		fmt.Println()
		fmt.Println("  50 mi to km   -> 80.4672 km")
		fmt.Println("  100 ft to m   -> 30.48 m")
		fmt.Println("  1 gb to mb    -> 1024 MB")
		fmt.Println("  1 mb to bits  -> 8388608 bits")
		fmt.Println("  30 cm to in   -> 11.811... in")
		fmt.Println()
		fmt.Println("  result ignores current output mode")
	case "modes", "state", "mode":
		fmt.Println("output modes")
		fmt.Println()
		fmt.Println("  mode <name>    switch")
		fmt.Println("  mode           query current")
		fmt.Println()
		fmt.Println("  mode   terminal                          clipboard")
		fmt.Println("  dec    1048576  [1 MB]                   1048576")
		fmt.Println("  size   1 MB                              1")
		fmt.Println("  bytes  1048576 B                         1048576")
		fmt.Println("  bits   8388608 bits                      8388608")
		fmt.Println("  hex    0x100000  [Hex]                   0x100000")
		fmt.Println("  bin    0b100000000000000000000  [Bin]    0b100000000000000000000")
		fmt.Println("  oct    0o4000000  [Oct]                  0o4000000")
		fmt.Println()
		fmt.Println("  two's complement modes (zero-padded, negatives as bit pattern):")
		fmt.Println("  bin8/16/32/64/128/256/512")
		fmt.Println("  hex8/16/32/64/128")
		fmt.Println("  oct8/16/32/64")
		fmt.Println()
		fmt.Println("  also available as functions: bin32(-5)  hex64(-1)  oct8(-1)")
		fmt.Println()
		fmt.Println("  dec mode adds [1 MB] hint when expression uses a data unit.")
		fmt.Println("  suppressed when units cancel (e.g. mb/gb*1000 = dimensionless).")
		fmt.Println()
		fmt.Println("  bare 'hex'/'bin' evaluate as expressions. only 'mode hex' switches.")
		fmt.Println()
		fmt.Println("  mode all   show dec + hex + bin simultaneously")
	case "types", "type", "integers", "int":
		fmt.Println("type modes  (integer semantics, orthogonal to format mode)")
		fmt.Println()
		fmt.Println("  type <name>    set integer type constraint")
		fmt.Println("  type           query current")
		fmt.Println()
		fmt.Println("  type auto      default: pure float64 math, no wrapping")
		fmt.Println()
		fmt.Println("  unsigned:  u8  u16  u32  u64  u128")
		fmt.Println("  signed:    s8  s16  s32  s64  s128")
		fmt.Println()
		fmt.Println("  with type u8 active:")
		fmt.Println("    255 + 1    -> 0  [u8 ovf]    overflow detected + wrapped")
		fmt.Println("    200 + 50   -> 250  [u8]")
		fmt.Println()
		fmt.Println("  with type s8 active:")
		fmt.Println("    127 + 1    -> -128  [s8 ovf]")
		fmt.Println("    -5 + 10    -> 5  [s8]")
		fmt.Println()
		fmt.Println("  cast functions (explicit, no global mode needed):")
		fmt.Println("    u8(246)          -> 246     unsigned cast")
		fmt.Println("    s8(246)          -> -10     signed reinterpret")
		fmt.Println("    u8(256)          -> 0       overflow wraps")
		fmt.Println("    s16(-32769)      -> 32767   wrap to s16 max")
		fmt.Println()
		fmt.Println("  to keyword (inline cast):")
		fmt.Println("    246 to u8        -> 246")
		fmt.Println("    0b11110110 to s8 -> -10")
		fmt.Println("    _ to u32         applies type to last result")
		fmt.Println()
		fmt.Println("  cast functions return float64 - compose with arithmetic:")
		fmt.Println("    u8(200) + u8(100)    -> 44  (300 wrapped to u8)")
		fmt.Println()
		fmt.Println("  type mode is independent of format mode:")
		fmt.Println("    mode hex + type u8 -> results shown in hex, wrapped to u8")
	case "vars", "variables", "memory":
		fmt.Println("variables")
		fmt.Println()
		fmt.Println("  block = 4096")
		fmt.Println("  page  = 4 * kb")
		fmt.Println("  journal = 128 * mb")
		fmt.Println()
		fmt.Println("  journal / block")
		fmt.Println("  (512 * mb) / block")
		fmt.Println()
		fmt.Println("  vars          list")
		fmt.Println("  del block     remove")
		fmt.Println()
		fmt.Println("  _ holds the last numeric result (not persisted)")
		fmt.Println()
		fmt.Println("  saved to ~/.wrkr_vars.json, reloaded on next launch")
		fmt.Println("  mode names (hex, bin, dec...) are reserved, cannot be used as var names")
	case "settings", "setting", "config":
		fmt.Println("settings")
		fmt.Println()
		fmt.Println("  setting                            show all settings")
		fmt.Println()
		fmt.Println("  setting clipboard on|off           toggle clipboard copy (default: on)")
		fmt.Println()
		fmt.Println("  setting grouping on|off            _ separators in output + clipboard")
		fmt.Println("  setting grouping display on|off    _ separators in terminal only")
		fmt.Println("  setting grouping clipboard on|off  _ separators in clipboard only")
		fmt.Println()
		fmt.Println("  setting prefix on|off              0x/0b/0o prefix in output + clipboard")
		fmt.Println("  setting prefix display on|off      prefix in terminal only")
		fmt.Println("  setting prefix clipboard on|off    prefix in clipboard only")
		fmt.Println()
		fmt.Println("  defaults: grouping display on, grouping clipboard off,")
		fmt.Println("            prefix display on, prefix clipboard off (raw)")
		fmt.Println()
		fmt.Println("  settings are saved to ~/.wrkr_config.json")
	case "drill":
		fmt.Println("drill - binary/hex/decimal fluency trainer")
		fmt.Println()
		fmt.Println("  drill    start an interactive session")
		fmt.Println()
		fmt.Println("  Games:")
		fmt.Println("    1) convert    Q&A: type the conversion")
		fmt.Println("    2) flashcard  see answer for 1.5s, then recall from memory")
		fmt.Println("    3) vibes      estimate hex/bin values against the clock")
		fmt.Println("    4) sprint     60-second timed blitz")
		fmt.Println("    5) bit scan   given a hex value, which bit position is set?")
		fmt.Println("    6) other      bonus games (hex ops: 0xA + 0x7 = ?)")
		fmt.Println()
		fmt.Println("  Modes (convert / flashcard / sprint):")
		fmt.Println("    1) nibble  (0-15)      master the 16 core hex facts first")
		fmt.Println("    2) powers  (2^0-2^15)  essential for fast decomposition")
		fmt.Println("    3) byte    (0-255)     full 8-bit range")
		fmt.Println("    4) random  mix of all three")
		fmt.Println()
		fmt.Println("  Answer formats:")
		fmt.Println("    hex:  0xF  or bare with a-f letter (F, b4)")
		fmt.Println("    bin:  0b1010  or bare 0s and 1s (1010)")
		fmt.Println("    dec:  plain digits (15, 255)")
		fmt.Println("    bit:  plain number - 0 = LSB (e.g. 7)")
		fmt.Println()
		fmt.Println("  Stats saved to ~/.wrkr_drill.json")
		fmt.Println("  Recommended: nibble -> hex until instant, powers -> bin, byte -> hex")
	case "all":
		printHelp("math")
		printHelp("systems")
		printHelp("units")
		printHelp("modes")
		printHelp("types")
		printHelp("vars")
		printHelp("settings")
		printHelp("drill")
	default:
		fmt.Println("help math      trig, logs, pow")
		fmt.Println("help systems   base literals and conversion")
		fmt.Println("help units     unit conversion")
		fmt.Println("help modes     output modes")
		fmt.Println("help types     integer type modes (u8, s16, ...)")
		fmt.Println("help vars      variables")
		fmt.Println("help settings  clipboard and other settings")
		fmt.Println("help drill     drill mode games and answer formats")
		fmt.Println("help all       everything")
	}
}

// openInEditor writes initial content to a temp file, opens $EDITOR, and
// returns the saved content. Falls back to vi if EDITOR is unset.
func openInEditor(initial string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	f, err := os.CreateTemp("", "wrkr-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	if initial != "" {
		f.WriteString(initial)
	}
	f.Close()

	cmd := exec.Command(editor, f.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	data, err := os.ReadFile(f.Name())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// drillQuit returns true if the input is a quit command.
func drillQuit(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "q" || s == ":q"
}

// drillColorValue colors a value string by its base prefix.
func drillColorValue(s string) string {
	low := strings.ToLower(s)
	switch {
	case strings.HasPrefix(low, "0x"):
		return styleHex(s)
	case strings.HasPrefix(low, "0b"):
		return styleBin(s)
	default:
		return boldWhite(s)
	}
}

// drillColorBase colors the target base label.
func drillColorBase(base string) string {
	switch base {
	case "hex":
		return styleHex(base)
	case "bin":
		return styleBin(base)
	case "bit":
		return color.New(color.FgMagenta).Sprint(base)
	}
	return boldWhite(base)
}

// drillStreakStyle colors the streak number - escalates as it grows.
func drillStreakStyle(n int) string {
	s := fmt.Sprintf("%d", n)
	switch {
	case n >= 10:
		return color.New(color.FgGreen, color.Bold).Sprint(s)
	case n >= 5:
		return color.New(color.FgYellow).Sprint(s)
	default:
		return dimGray(s)
	}
}

func parseDrillMode(raw string) (drill.Mode, bool) {
	switch strings.TrimSpace(raw) {
	case "1":
		return drill.ModeNibble, true
	case "2":
		return drill.ModePowers, true
	case "3":
		return drill.ModeByte, true
	case "4":
		return drill.ModeRandom, true
	}
	return 0, false
}

func parseDrillConv(raw string) (drill.Conv, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "h", "hex":
		return drill.ConvToHex, true
	case "b", "bin":
		return drill.ConvToBin, true
	case "d", "dec":
		return drill.ConvToDec, true
	}
	return 0, false
}

func showDrillSummary(stats *drill.Stats, nCorrect, nWrong, bestStreak int, game string) {
	total := nCorrect + nWrong
	if total == 0 {
		return
	}
	pct := 100 * nCorrect / total
	fmt.Printf("  %d correct  %d wrong  %d%%",
		nCorrect, nWrong, pct,
	)
	if bestStreak > 0 {
		fmt.Printf("  best streak %s", drillStreakStyle(bestStreak))
	}
	fmt.Println()
	fmt.Println()
	stats.LastSession = &drill.SessionSummary{
		Correct: nCorrect,
		Wrong:   nWrong,
		Game:    game,
	}
}

func runConvertDrill(line *liner.State, mode drill.Mode, conv drill.Conv, stats *drill.Stats) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	gen := drill.NewGenerator(mode, conv, rng)
	gen.ApplyWeakSpotBias(stats.MissedCounts)
	correctStyle := color.New(color.FgGreen, color.Bold).SprintFunc()
	var streak, bestStreak, nCorrect, nWrong int
	fmt.Println()
	for {
		q := gen.Next()
		fmt.Printf("  %s  →  %s\n", drillColorValue(q.From), drillColorBase(q.ToBase))
		ans, err := line.Prompt("  > ")
		ans = strings.TrimSpace(ans)
		if err != nil || drillQuit(ans) {
			fmt.Println()
			break
		}
		if q.Check(ans) {
			streak++
			nCorrect++
			stats.Record(q.Value, q.ToBase, true)
			if streak > bestStreak {
				bestStreak = streak
			}
			if streak > 1 {
				fmt.Printf("  %s  %s\n\n", correctStyle("✓"), drillStreakStyle(streak))
			} else {
				fmt.Printf("  %s\n\n", correctStyle("✓"))
			}
		} else {
			prevStreak := streak
			streak = 0
			nWrong++
			stats.Record(q.Value, q.ToBase, false)
			if prevStreak > 1 {
				fmt.Printf("  %s  %s  %s\n\n",
					styleError("✗"),
					drillColorValue(q.CorrectAnswer()),
					dimGray(fmt.Sprintf("(lost %d)", prevStreak)),
				)
			} else {
				fmt.Printf("  %s  %s\n\n",
					styleError("✗"),
					drillColorValue(q.CorrectAnswer()),
				)
			}
		}
	}
	showDrillSummary(stats, nCorrect, nWrong, bestStreak, "convert")
}

func runFlashcardDrill(line *liner.State, mode drill.Mode, conv drill.Conv, stats *drill.Stats) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	gen := drill.NewGenerator(mode, conv, rng)
	gen.ApplyWeakSpotBias(stats.MissedCounts)
	correctStyle := color.New(color.FgGreen, color.Bold).SprintFunc()
	var streak, bestStreak, nCorrect, nWrong int
	fmt.Println()
	for {
		q := gen.Next()
		// Show question + answer for 1.5 seconds, then erase the answer line.
		fmt.Printf("  %s  →  %s:  %s\n",
			drillColorValue(q.From), drillColorBase(q.ToBase), drillColorValue(q.CorrectAnswer()))
		time.Sleep(1500 * time.Millisecond)
		fmt.Print("\033[1A\033[2K") // cursor up 1, clear line
		fmt.Printf("  %s  →  %s\n", drillColorValue(q.From), drillColorBase(q.ToBase))
		ans, err := line.Prompt("  > ")
		ans = strings.TrimSpace(ans)
		if err != nil || drillQuit(ans) {
			fmt.Println()
			break
		}
		if q.Check(ans) {
			streak++
			nCorrect++
			stats.Record(q.Value, q.ToBase, true)
			if streak > bestStreak {
				bestStreak = streak
			}
			if streak > 1 {
				fmt.Printf("  %s  %s\n\n", correctStyle("✓"), drillStreakStyle(streak))
			} else {
				fmt.Printf("  %s\n\n", correctStyle("✓"))
			}
		} else {
			prevStreak := streak
			streak = 0
			nWrong++
			stats.Record(q.Value, q.ToBase, false)
			if prevStreak > 1 {
				fmt.Printf("  %s  %s  %s\n\n",
					styleError("✗"),
					drillColorValue(q.CorrectAnswer()),
					dimGray(fmt.Sprintf("(lost %d)", prevStreak)),
				)
			} else {
				fmt.Printf("  %s  %s\n\n",
					styleError("✗"),
					drillColorValue(q.CorrectAnswer()),
				)
			}
		}
	}
	showDrillSummary(stats, nCorrect, nWrong, bestStreak, "flashcard")
}

func runApproxDrill(_ *liner.State, tol drill.VibesTolerance, timeLimit time.Duration, stats *drill.Stats) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	gen := drill.NewApproxGenerator(rng, tol)
	correctStyle := color.New(color.FgGreen, color.Bold).SprintFunc()
	var streak, bestStreak, nCorrect, nWrong int
	fmt.Println()
	var tolDesc string
	switch tol {
	case drill.VibesExact:
		tolDesc = fmt.Sprintf("exact match  limit %gs", timeLimit.Seconds())
	default:
		tolDesc = fmt.Sprintf("within %d%%  limit %gs", int(tol), timeLimit.Seconds())
	}
	fmt.Printf("  %s\n\n", dimGray(tolDesc))
	for {
		q := gen.Next()
		t0 := time.Now()
		ans, tOut := vibesPrompt(q.From, timeLimit)
		elapsed := time.Since(t0)
		ans = strings.TrimSpace(ans)
		if drillQuit(ans) {
			fmt.Println()
			break
		}

		took := fmt.Sprintf("%.1fs", elapsed.Seconds())

		if tOut {
			// Too slow - auto-fail regardless of answer.
			prevStreak := streak
			streak = 0
			nWrong++
			stats.Record(q.Value, "dec", false)
			if prevStreak > 1 {
				fmt.Printf("  %s  %s  %s  %s\n\n",
					styleError("✗"),
					boldWhite(fmt.Sprintf("%d", q.Value)),
					dimGray(fmt.Sprintf("too slow (%s)", took)),
					dimGray(fmt.Sprintf("(lost %d)", prevStreak)),
				)
			} else {
				fmt.Printf("  %s  %s  %s\n\n",
					styleError("✗"),
					boldWhite(fmt.Sprintf("%d", q.Value)),
					dimGray(fmt.Sprintf("too slow (%s)", took)),
				)
			}
			continue
		}

		if q.Check(ans) {
			streak++
			nCorrect++
			stats.Record(q.Value, "dec", true)
			if streak > bestStreak {
				bestStreak = streak
			}
			exact := dimGray(fmt.Sprintf("= %d", q.Value))
			if streak > 1 {
				fmt.Printf("  %s  %s  %s  %s\n\n", correctStyle("✓"), exact, dimGray(took), drillStreakStyle(streak))
			} else {
				fmt.Printf("  %s  %s  %s\n\n", correctStyle("✓"), exact, dimGray(took))
			}
		} else {
			prevStreak := streak
			streak = 0
			nWrong++
			stats.Record(q.Value, "dec", false)
			if prevStreak > 1 {
				fmt.Printf("  %s  %s  %s  %s  %s\n\n",
					styleError("✗"),
					boldWhite(fmt.Sprintf("%d", q.Value)),
					dimGray(fmt.Sprintf("(ok: %s)", q.RangeHint())),
					dimGray(took),
					dimGray(fmt.Sprintf("(lost %d)", prevStreak)),
				)
			} else {
				fmt.Printf("  %s  %s  %s  %s\n\n",
					styleError("✗"),
					boldWhite(fmt.Sprintf("%d", q.Value)),
					dimGray(fmt.Sprintf("(ok: %s)", q.RangeHint())),
					dimGray(took),
				)
			}
		}
	}
	showDrillSummary(stats, nCorrect, nWrong, bestStreak, "vibes")
}

func runHexOpsDrill(line *liner.State, stats *drill.Stats) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	gen := drill.NewHexOpsGenerator(rng)
	correctStyle := color.New(color.FgGreen, color.Bold).SprintFunc()
	var streak, bestStreak, nCorrect, nWrong int
	fmt.Println()
	fmt.Printf("  %s\n\n", dimGray("answer in hex (e.g. 0x1F or 1F)"))
	for {
		q := gen.Next()
		fmt.Printf("  %s  =  ?\n", color.New(color.FgCyan).Sprint(q.Prompt))
		t0 := time.Now()
		ans, err := line.Prompt("  > ")
		elapsed := time.Since(t0)
		ans = strings.TrimSpace(ans)
		if err != nil || drillQuit(ans) {
			fmt.Println()
			break
		}
		tookStr := dimGray(fmt.Sprintf("%.1fs", elapsed.Seconds()))
		if q.Check(ans) {
			streak++
			nCorrect++
			if streak > bestStreak {
				bestStreak = streak
			}
			if streak > 1 {
				fmt.Printf("  %s  %s  %s  %s\n\n", correctStyle("✓"), dimGray(q.CorrectAnswer()), tookStr, drillStreakStyle(streak))
			} else {
				fmt.Printf("  %s  %s  %s\n\n", correctStyle("✓"), dimGray(q.CorrectAnswer()), tookStr)
			}
		} else {
			prevStreak := streak
			streak = 0
			nWrong++
			if prevStreak > 1 {
				fmt.Printf("  %s  %s  %s  %s\n\n",
					styleError("✗"),
					boldWhite(q.CorrectAnswer()),
					tookStr,
					dimGray(fmt.Sprintf("(lost %d)", prevStreak)),
				)
			} else {
				fmt.Printf("  %s  %s  %s\n\n",
					styleError("✗"),
					boldWhite(q.CorrectAnswer()),
					tookStr,
				)
			}
		}
	}
	showDrillSummary(stats, nCorrect, nWrong, bestStreak, "hex ops")
}

func runSprintDrill(line *liner.State, mode drill.Mode, conv drill.Conv, stats *drill.Stats) {
	const duration = 60 * time.Second
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	gen := drill.NewGenerator(mode, conv, rng)
	gen.ApplyWeakSpotBias(stats.MissedCounts)
	correctStyle := color.New(color.FgGreen, color.Bold).SprintFunc()
	var nCorrect, nWrong int
	start := time.Now()
	fmt.Println()
	fmt.Printf("  %s\n\n", dimGray("60 seconds - go!"))
	for {
		elapsed := time.Since(start)
		if elapsed >= duration {
			break
		}
		remaining := duration - elapsed
		secs := int(remaining.Seconds()) + 1
		q := gen.Next()
		fmt.Printf("  %s  %s  →  %s\n",
			dimGray(fmt.Sprintf("[%02d]", secs)),
			drillColorValue(q.From),
			drillColorBase(q.ToBase),
		)
		ans, err := line.Prompt("  > ")
		ans = strings.TrimSpace(ans)
		if err != nil || drillQuit(ans) {
			fmt.Println()
			showDrillSummary(stats, nCorrect, nWrong, 0, "sprint")
			return
		}
		if time.Since(start) >= duration {
			fmt.Println()
			break
		}
		if q.Check(ans) {
			nCorrect++
			stats.Record(q.Value, q.ToBase, true)
			fmt.Printf("  %s\n\n", correctStyle("✓"))
		} else {
			nWrong++
			stats.Record(q.Value, q.ToBase, false)
			fmt.Printf("  %s  %s\n\n", styleError("✗"), drillColorValue(q.CorrectAnswer()))
		}
	}
	fmt.Printf("  %s  ", boldWhite("Time!"))
	showDrillSummary(stats, nCorrect, nWrong, 0, "sprint")
}

func runBitScanDrill(line *liner.State, stats *drill.Stats) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	gen := drill.NewGenerator(drill.ModePowers, drill.ConvToBitPos, rng)
	correctStyle := color.New(color.FgGreen, color.Bold).SprintFunc()
	var streak, bestStreak, nCorrect, nWrong int
	fmt.Println()
	fmt.Printf("  %s\n\n", dimGray("which bit position is set? (0 = LSB)"))
	for {
		q := gen.Next()
		fmt.Printf("  %s  →  bit position\n", drillColorValue(q.From))
		ans, err := line.Prompt("  > ")
		ans = strings.TrimSpace(ans)
		if err != nil || drillQuit(ans) {
			fmt.Println()
			break
		}
		if q.Check(ans) {
			streak++
			nCorrect++
			stats.Record(q.Value, "bit", true)
			if streak > bestStreak {
				bestStreak = streak
			}
			if streak > 1 {
				fmt.Printf("  %s  %s\n\n", correctStyle("✓"), drillStreakStyle(streak))
			} else {
				fmt.Printf("  %s\n\n", correctStyle("✓"))
			}
		} else {
			prevStreak := streak
			streak = 0
			nWrong++
			stats.Record(q.Value, "bit", false)
			if prevStreak > 1 {
				fmt.Printf("  %s  %s  %s\n\n",
					styleError("✗"),
					boldWhite(q.CorrectAnswer()),
					dimGray(fmt.Sprintf("(lost %d)", prevStreak)),
				)
			} else {
				fmt.Printf("  %s  %s\n\n",
					styleError("✗"),
					boldWhite(q.CorrectAnswer()),
				)
			}
		}
	}
	showDrillSummary(stats, nCorrect, nWrong, bestStreak, "bit scan")
}

// runDrill runs an interactive drill session using the existing liner instance.
func runDrill(line *liner.State) {
	stats := drill.LoadStats()
	stats.UpdateStreak()
	defer func() { drill.SaveStats(stats) }()

	fmt.Println()

	// Streak + last session summary.
	if stats.Streak > 1 {
		fmt.Printf("  %s  day %s\n",
			dimGray("streak"),
			drillStreakStyle(stats.Streak),
		)
	}
	if stats.LastSession != nil {
		ls := stats.LastSession
		total := ls.Correct + ls.Wrong
		pctStr := ""
		if total > 0 {
			pctStr = fmt.Sprintf(" (%d%%)", 100*ls.Correct/total)
		}
		fmt.Printf("  %s  last: %d/%d%s - %s\n",
			dimGray("stats"),
			ls.Correct, total, pctStr,
			dimGray(ls.Game),
		)
	}
	if missed := stats.TopMissed(3); len(missed) > 0 {
		fmt.Printf("  %s", dimGray("weak:  "))
		for i, m := range missed {
			if i > 0 {
				fmt.Printf("  ")
			}
			fmt.Printf("%s %s", m.Display, dimGray(fmt.Sprintf("×%d", m.Count)))
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("  Game:")
	fmt.Println("    1) convert    binary ↔ hex ↔ decimal Q&A")
	fmt.Println("    2) flashcard  see answer briefly, then recall")
	fmt.Println("    3) vibes      estimate hex/bin values against the clock")
	fmt.Println("    4) sprint     60-second timed blitz")
	fmt.Println("    5) bit scan   which bit position is set?")
	fmt.Println("    6) other      bonus games")
	fmt.Println()
	fmt.Printf("  %s to quit\n", dimGray(":q"))
	fmt.Println()

	gameRaw, err := line.Prompt("  game [1-6]: ")
	if err != nil || drillQuit(gameRaw) {
		fmt.Println()
		return
	}
	game := strings.TrimSpace(gameRaw)

	// Bit scan: no mode/conv selection needed.
	if game == "5" {
		runBitScanDrill(line, &stats)
		return
	}

	// Other: bonus games submenu.
	if game == "6" {
		fmt.Println()
		fmt.Println("  Other:")
		fmt.Println("    1) hex ops  arithmetic & bitwise in hex  (0xA + 0x7 = ?)")
		fmt.Println()
		otherRaw, err := line.Prompt("  game [1]: ")
		if err != nil || drillQuit(otherRaw) {
			fmt.Println()
			return
		}
		switch strings.TrimSpace(otherRaw) {
		case "1":
			runHexOpsDrill(line, &stats)
		default:
			fmt.Println(styleError("  invalid - enter 1"))
			fmt.Println()
		}
		return
	}

	// Vibes: pick precision and time limit, then go.
	if game == "3" {
		fmt.Println()
		fmt.Println("  Precision:")
		fmt.Println("    1) rough  +-25%  magnitude + gut feel")
		fmt.Println("    2) close  +-10%  read the leading digit(s)")
		fmt.Println("    3) tight  +-5%   almost exact")
		fmt.Println("    4) exact  0      nail it")
		fmt.Println()
		tolRaw, err := line.Prompt("  precision [1-4]: ")
		if err != nil || drillQuit(tolRaw) {
			fmt.Println()
			return
		}
		var tol drill.VibesTolerance
		switch strings.TrimSpace(tolRaw) {
		case "1":
			tol = drill.VibesRough
		case "2":
			tol = drill.VibesClose
		case "3":
			tol = drill.VibesTight
		case "4":
			tol = drill.VibesExact
		default:
			fmt.Println(styleError("  invalid - enter 1, 2, 3, or 4"))
			fmt.Println()
			return
		}
		fmt.Println()
		fmt.Printf("  %s\n", dimGray("time limit per question (default 5):"))
		fmt.Println()
		timeLimitRaw, err := line.Prompt("  seconds [5]: ")
		if err != nil || drillQuit(timeLimitRaw) {
			fmt.Println()
			return
		}
		timeLimit := 5 * time.Second
		if ts := strings.TrimSpace(timeLimitRaw); ts != "" {
			if n, e := fmt.Sscanf(ts, "%g", new(float64)); n == 1 && e == nil {
				var secs float64
				fmt.Sscanf(ts, "%g", &secs)
				if secs >= 1 && secs <= 60 {
					timeLimit = time.Duration(secs * float64(time.Second))
				}
			}
		}
		runApproxDrill(line, tol, timeLimit, &stats)
		return
	}

	// Convert, flashcard, sprint need a mode.
	fmt.Println()
	fmt.Println("  Mode:")
	fmt.Println("    1) nibble  (0–15)")
	fmt.Println("    2) powers  (2^0–2^15)")
	fmt.Println("    3) byte    (0–255)")
	fmt.Println("    4) random  mix")
	fmt.Println()

	modeRaw, err := line.Prompt("  mode [1-4]: ")
	if err != nil || drillQuit(modeRaw) {
		fmt.Println()
		return
	}
	mode, ok := parseDrillMode(modeRaw)
	if !ok {
		fmt.Println(styleError("  invalid mode - enter 1, 2, 3, or 4"))
		fmt.Println()
		return
	}

	// Convert, flashcard, sprint all need a target base.
	convRaw, err := line.Prompt("  to [h/b/d]: ")
	if err != nil || drillQuit(convRaw) {
		fmt.Println()
		return
	}
	conv, ok := parseDrillConv(convRaw)
	if !ok {
		fmt.Println(styleError("  invalid - enter h, b, or d"))
		fmt.Println()
		return
	}

	switch game {
	case "1":
		runConvertDrill(line, mode, conv, &stats)
	case "2":
		runFlashcardDrill(line, mode, conv, &stats)
	case "4":
		runSprintDrill(line, mode, conv, &stats)
	default:
		fmt.Println(styleError("  invalid game - enter 1–5"))
		fmt.Println()
	}
}

// Run starts the interactive REPL.
func Run() {
	validTokens := engine.GetValidTokens()

	// Use a closure so that after :e we can reassign line and the defer still
	// closes the current instance (not the one captured at defer time).
	var line *liner.State

	setupLiner := func() {
		line = liner.NewLiner()
		line.SetCtrlCAborts(true)
		line.SetCompleter(func(input string) []string {
			// Split into the already-typed prefix and the token being completed.
			lastBoundary := strings.LastIndexAny(input, " \t(,+-*/^%")
			prefix := input
			before := ""
			if lastBoundary >= 0 {
				before = input[:lastBoundary+1]
				prefix = input[lastBoundary+1:]
			}
			lp := strings.ToLower(prefix)

			// How many whitespace-separated words are fully typed before the
			// current token?  This drives context-aware completions.
			fields := strings.Fields(strings.ToLower(input))
			numCompleted := len(fields)
			if len(fields) > 0 && !strings.HasSuffix(input, " ") {
				numCompleted-- // last field is still being typed
			}

			var candidates []string
			commandContext := false // allow empty-prefix completions in command positions

			if numCompleted == 0 {
				// Completing the first word: commands + expression tokens.
				if prefix == "" {
					return nil
				}
				candidates = append([]string{
					"help", "mode", "type", "setting", "vars", "del", "debug",
					"drill", "exit", "quit", "clear", ":e", ":q",
				}, engine.GetCompletionTokens()...)
			} else {
				cmd := fields[0]
				switch cmd {
				case "mode":
					commandContext = true
					if numCompleted == 1 {
						for k := range engine.ModeMap {
							candidates = append(candidates, k)
						}
					}
				case "type":
					commandContext = true
					if numCompleted == 1 {
						for k := range engine.TypeModeMap {
							candidates = append(candidates, k)
						}
					}
				case "setting":
					commandContext = true
					switch numCompleted {
					case 1:
						candidates = []string{"clipboard", "grouping", "prefix"}
					case 2:
						if len(fields) >= 2 {
							switch fields[1] {
							case "clipboard":
								candidates = []string{"on", "off"}
							case "grouping", "prefix":
								candidates = []string{"on", "off", "display", "clipboard"}
							}
						}
					case 3:
						if len(fields) >= 3 {
							switch fields[1] {
							case "grouping", "prefix":
								if fields[2] == "display" || fields[2] == "clipboard" {
									candidates = []string{"on", "off"}
								}
							}
						}
					}
				case "help":
					commandContext = true
					if numCompleted == 1 {
						candidates = []string{
							"math", "systems", "units", "modes", "types",
							"vars", "settings", "drill", "all",
						}
					}
				case "del":
					commandContext = true
					if numCompleted == 1 {
						for k := range engine.UserVars {
							candidates = append(candidates, k)
						}
					}
				default:
					// Expression context: only complete non-empty prefixes.
					if prefix == "" {
						return nil
					}
					candidates = engine.GetValidTokens()
				}
			}

			if !commandContext && prefix == "" {
				return nil
			}

			var out []string
			for _, c := range candidates {
				if strings.HasPrefix(strings.ToLower(c), lp) && strings.ToLower(c) != lp {
					out = append(out, before+c)
				}
			}
			sort.Strings(out)
			return out
		})
	}

	setupLiner()
	defer func() { line.Close() }()

	// Restore persisted settings (format mode, type mode, clipboard, grouping, prefix).
	{
		cfg := engine.ReadAppConfig()
		if cfg.FormatMode != "" {
			if m, ok := engine.ModeMap[cfg.FormatMode]; ok {
				engine.CurrentMode = m
			}
		}
		if cfg.TypeMode != "" {
			if t, ok := engine.TypeModeMap[cfg.TypeMode]; ok {
				engine.CurrentTypeMode = t
			}
		}
		if cfg.Clipboard != nil {
			engine.ClipboardEnabled = *cfg.Clipboard
		}
		if cfg.GroupingDisplay != nil {
			engine.GroupingDisplay = *cfg.GroupingDisplay
		}
		if cfg.GroupingClipboard != nil {
			engine.GroupingClipboard = *cfg.GroupingClipboard
		}
		if cfg.PrefixDisplay != nil {
			engine.PrefixDisplay = *cfg.PrefixDisplay
		}
		if cfg.PrefixClipboard != nil {
			engine.PrefixClipboard = *cfg.PrefixClipboard
		}
	}

	fmt.Println("wrkr  help all / exit")

	// Saved variable prompt.
	if saved, _ := engine.ReadSavedVars(); saved != nil {
		keys := make([]string, 0, len(saved.Vars))
		for k := range saved.Vars {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		if engine.ReadAutoload() {
			engine.ApplySavedVars(saved.Vars)
			validTokens = engine.GetValidTokens()
			fmt.Println()
			fmt.Printf("  %d variable(s) loaded\n", len(saved.Vars))
			for _, k := range keys {
				fmt.Printf("    %s  =  %s\n",
					styleVarName(fmt.Sprintf("%-12s", k)),
					boldWhite(engine.FormatDecimal(saved.Vars[k])),
				)
			}
			fmt.Println()
		} else {
			fmt.Println()
			fmt.Printf("  %d saved variable(s)\n", len(saved.Vars))
			for _, k := range keys {
				fmt.Printf("    %s  =  %s\n",
					styleVarName(fmt.Sprintf("%-12s", k)),
					boldWhite(engine.FormatDecimal(saved.Vars[k])),
				)
			}
			fmt.Println()
			fmt.Println("  [Enter] load & remember    [s] this session only    [d] delete")
			fmt.Println()

			choice, _ := line.Prompt("> ")
			switch strings.ToLower(strings.TrimSpace(choice)) {
			case "d", "delete":
				engine.DeletePersistedVars()
				fmt.Println("  variables deleted")
			case "s", "session":
				engine.ApplySavedVars(saved.Vars)
				validTokens = engine.GetValidTokens()
				fmt.Printf("  loaded for this session\n")
			default:
				engine.ApplySavedVars(saved.Vars)
				validTokens = engine.GetValidTokens()
				engine.SetAutoload(true)
				fmt.Printf("  loaded, will autoload from now on  %s\n",
					dimGray("(edit ~/.wrkr_config.json to disable)"))
			}
			fmt.Println()
		}
	}

	var scriptQueue []string
	var lastEditorContent string

	for {
		var rawInput string

		if len(scriptQueue) > 0 {
			rawInput = scriptQueue[0]
			scriptQueue = scriptQueue[1:]
			line.AppendHistory(rawInput)
			fmt.Printf("\n%s %s\n", dimGray(">"), rawInput)
		} else {
			if engine.CurrentTypeMode != "auto" {
				fmt.Printf("\n%s\n", colorMode("["+engine.CurrentMode+"/"+engine.CurrentTypeMode+"]"))
			} else {
				fmt.Printf("\n%s\n", colorMode("["+engine.CurrentMode+"]"))
			}
			var err error
			rawInput, err = line.Prompt("> ")
			if err != nil {
				if err == liner.ErrPromptAborted {
					scriptQueue = nil
					fmt.Println(dimGray("  :q to quit"))
					continue
				}
				if err == io.EOF {
					return
				}
				continue
			}
			rawInput = strings.TrimSpace(rawInput)
			if rawInput == "" {
				continue
			}
			line.AppendHistory(rawInput)
		}

		lowerInput := strings.ToLower(rawInput)

		// Built-in commands.

		if lowerInput == "exit" || lowerInput == "quit" || lowerInput == ":q" || lowerInput == "q" {
			return
		}

		if lowerInput == "clear" {
			fmt.Print("\033[H\033[2J")
			continue
		}

		if strings.HasPrefix(lowerInput, "help") {
			parts := strings.SplitN(lowerInput, " ", 2)
			topic := "general"
			if len(parts) > 1 {
				topic = strings.TrimSpace(parts[1])
			}
			printHelp(topic)
			continue
		}

		// :e - open expression(s) in $EDITOR. Each non-empty line runs as a
		// separate command so you can chain assignments and expressions.
		if lowerInput == ":e" {
			var histBuf bytes.Buffer
			line.WriteHistory(&histBuf)
			line.Close()

			editorContent, editorErr := openInEditor(lastEditorContent)

			setupLiner()
			line.ReadHistory(&histBuf)

			if editorErr != nil {
				fmt.Println(styleError("editor error: " + editorErr.Error()))
				continue
			}

			var editorLines []string
			for _, l := range strings.Split(editorContent, "\n") {
				if t := strings.TrimSpace(l); t != "" && !strings.HasPrefix(t, "#") {
					editorLines = append(editorLines, t)
				}
			}
			if len(editorLines) == 0 {
				continue
			}
			lastEditorContent = editorContent

			// Queue all lines. The first one falls through into this iteration;
			// the rest are prepended to scriptQueue for subsequent iterations.
			rawInput = editorLines[0]
			lowerInput = strings.ToLower(rawInput)
			line.AppendHistory(rawInput)
			if len(editorLines) > 1 {
				scriptQueue = append(editorLines[1:], scriptQueue...)
			}
			fmt.Printf("\n%s %s\n", dimGray(">"), rawInput)
			// fall through to expression processing below
		}

		// Debug: show only changed pipeline steps + expanded constants.
		if strings.HasPrefix(lowerInput, "debug ") {
			debugExpr := strings.TrimSpace(rawInput[6:])
			s0 := debugExpr
			s1 := engine.StripNumericSeparators(s0)
			s2 := engine.FixBaseTypos(s1)
			s3 := engine.FixNakedBases(s2)
			s4 := strings.ReplaceAll(s3, " into ", " to ")
			s4 = strings.ReplaceAll(s4, " in to ", " to ")
			s5 := engine.ProcessConversions(s4)
			s6 := engine.ProcessFormatting(s5)
			s7 := engine.FixImplicitMultiplication(s6)
			s8 := engine.RewriteBitwiseOps(s7)
			s9 := engine.TranslateBases(s8)
			s10 := engine.ExpandConstants(s8) // unit names -> numbers (pre-translate view)

			steps := []struct{ label, val string }{
				{"input   ", s0},
				{"sep     ", s1},
				{"typos   ", s2},
				{"bases   ", s3},
				{"keywords", s4},
				{"convert ", s5},
				{"format  ", s6},
				{"multiply", s7},
				{"bitwise ", s8},
				{"ast     ", s9},
				{"expanded", s10},
			}

			fmt.Println()
			prev := ""
			for _, step := range steps {
				if step.val == prev {
					continue // skip unchanged steps
				}
				arrow := "  "
				if prev != "" {
					arrow = dimGray("->")
				}
				fmt.Printf("  %s  %s %s\n", dimGray(step.label), arrow, boldWhite(step.val))
				prev = step.val
			}

			env := engine.GetMergedEnv()
			if prog, err := expr.Compile(s9, expr.Env(env)); err == nil {
				if res, err := expr.Run(prog, env); err == nil {
					var resultStr string
					switch v := res.(type) {
					case float64:
						resultStr = engine.FormatDecimal(v)
					case float32:
						resultStr = engine.FormatDecimal(float64(v))
					case int:
						resultStr = engine.FormatDecimal(float64(v))
					case int64:
						resultStr = engine.FormatDecimal(float64(v))
					default:
						resultStr = fmt.Sprintf("%v", v)
					}
					fmt.Printf("\n  %s  %s\n", dimGray("result  "), colorizeResult(resultStr))
				}
			}
			fmt.Println()
			continue
		}

		// List all user-defined variables.
		if lowerInput == "vars" {
			if len(engine.UserVars) == 0 {
				fmt.Println("no variables defined.  try: block = 4096")
			} else {
				keys := make([]string, 0, len(engine.UserVars))
				for k := range engine.UserVars {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Printf("  %s  =  %s\n",
						styleVarName(fmt.Sprintf("%-12s", k)),
						boldWhite(engine.FormatDecimal(engine.UserVars[k].(float64))),
					)
				}
			}
			continue
		}

		// Delete a user-defined variable.
		if strings.HasPrefix(lowerInput, "del ") {
			varName := strings.TrimSpace(rawInput[4:])
			if varName == "" {
				fmt.Println(styleError("usage: del <name>"))
				continue
			}
			if engine.DeleteVar(varName) {
				validTokens = engine.GetValidTokens()
				engine.PersistVars()
				fmt.Printf("deleted %s\n", styleVarName(varName))
			} else {
				fmt.Printf("%s  (not a user variable, use 'vars' to list)\n",
					styleError("unknown: "+varName))
			}
			continue
		}

		// Mode query / switch.

		if lowerInput == "mode" {
			fmt.Printf("current mode: %s\n", colorMode(engine.CurrentMode))
			continue
		}
		if strings.HasPrefix(lowerInput, "mode ") {
			modeCmd := strings.TrimSpace(strings.TrimPrefix(lowerInput, "mode "))
			if modeCmd == "help" {
				printHelp("modes")
			} else if newMode, ok := engine.ModeMap[modeCmd]; ok {
				engine.CurrentMode = newMode
				cfg := engine.ReadAppConfig()
				cfg.FormatMode = newMode
				engine.SaveAppConfig(cfg)
				fmt.Printf("mode -> %s\n", colorMode(newMode))
			} else {
				fmt.Printf("%s\n  tip: help modes\n", styleError("unknown mode: "+modeCmd))
			}
			continue
		}

		// Type mode query / switch.

		if lowerInput == "type" {
			if engine.CurrentTypeMode == "auto" {
				fmt.Printf("current type: %s  (pure float64, no wrapping)\n", engine.CurrentTypeMode)
			} else {
				fmt.Printf("current type: %s\n", engine.CurrentTypeMode)
			}
			continue
		}
		if strings.HasPrefix(lowerInput, "type ") {
			typeCmd := strings.TrimSpace(strings.TrimPrefix(lowerInput, "type "))
			if typeCmd == "help" {
				printHelp("types")
			} else if newType, ok := engine.TypeModeMap[typeCmd]; ok {
				engine.CurrentTypeMode = newType
				cfg := engine.ReadAppConfig()
				cfg.TypeMode = newType
				engine.SaveAppConfig(cfg)
				if newType == "auto" {
					fmt.Printf("type -> %s  (pure float64, no wrapping)\n", newType)
				} else {
					fmt.Printf("type -> %s\n", newType)
				}
			} else {
				fmt.Printf("%s\n  tip: help types\n", styleError("unknown type: "+typeCmd))
			}
			continue
		}

		// Settings.

		if lowerInput == "setting" || strings.HasPrefix(lowerInput, "setting ") {
			parts := strings.Fields(rawInput)
			if len(parts) < 2 {
				// Show a table of all current settings.
				onOff := func(b bool) string {
					if b {
						return "on"
					}
					return "off"
				}
				fmt.Println()
				fmt.Printf("  %-12s  %s\n", "clipboard", onOff(engine.ClipboardEnabled))
				fmt.Printf("  %-12s  display %-3s   clipboard %s\n", "grouping",
					onOff(engine.GroupingDisplay), onOff(engine.GroupingClipboard))
				fmt.Printf("  %-12s  display %-3s   clipboard %s\n", "prefix",
					onOff(engine.PrefixDisplay), onOff(engine.PrefixClipboard))
				fmt.Println()
				continue
			}
			switch strings.ToLower(parts[1]) {
			case "clipboard":
				if len(parts) == 2 {
					status := "on"
					if !engine.ClipboardEnabled {
						status = "off"
					}
					fmt.Printf("clipboard: %s\n", status)
				} else {
					switch strings.ToLower(parts[2]) {
					case "on":
						engine.ClipboardEnabled = true
						cfg := engine.ReadAppConfig()
						t := true
						cfg.Clipboard = &t
						engine.SaveAppConfig(cfg)
						fmt.Println("clipboard: on")
					case "off":
						engine.ClipboardEnabled = false
						cfg := engine.ReadAppConfig()
						f := false
						cfg.Clipboard = &f
						engine.SaveAppConfig(cfg)
						fmt.Println("clipboard: off")
					default:
						fmt.Println(styleError("usage: setting clipboard on|off"))
					}
				}
			case "grouping":
				applyGrouping := func(display, clip *bool) {
					cfg := engine.ReadAppConfig()
					if display != nil {
						engine.GroupingDisplay = *display
						cfg.GroupingDisplay = display
					}
					if clip != nil {
						engine.GroupingClipboard = *clip
						cfg.GroupingClipboard = clip
					}
					engine.SaveAppConfig(cfg)
				}
				if len(parts) == 2 {
					fmt.Printf("grouping: display %s   clipboard %s\n",
						func() string {
							if engine.GroupingDisplay {
								return "on"
							}
							return "off"
						}(),
						func() string {
							if engine.GroupingClipboard {
								return "on"
							}
							return "off"
						}())
				} else {
					sub := strings.ToLower(parts[2])
					switch sub {
					case "on":
						t := true
						applyGrouping(&t, &t)
						fmt.Println("grouping: on")
					case "off":
						f := false
						applyGrouping(&f, &f)
						fmt.Println("grouping: off")
					case "display":
						if len(parts) < 4 {
							fmt.Println(styleError("usage: setting grouping display on|off"))
						} else {
							switch strings.ToLower(parts[3]) {
							case "on":
								t := true
								applyGrouping(&t, nil)
								fmt.Println("grouping display: on")
							case "off":
								f := false
								applyGrouping(&f, nil)
								fmt.Println("grouping display: off")
							default:
								fmt.Println(styleError("usage: setting grouping display on|off"))
							}
						}
					case "clipboard":
						if len(parts) < 4 {
							fmt.Println(styleError("usage: setting grouping clipboard on|off"))
						} else {
							switch strings.ToLower(parts[3]) {
							case "on":
								t := true
								applyGrouping(nil, &t)
								fmt.Println("grouping clipboard: on")
							case "off":
								f := false
								applyGrouping(nil, &f)
								fmt.Println("grouping clipboard: off")
							default:
								fmt.Println(styleError("usage: setting grouping clipboard on|off"))
							}
						}
					default:
						fmt.Println(styleError("usage: setting grouping [on|off|display on|off|clipboard on|off]"))
					}
				}
			case "prefix":
				applyPrefix := func(display, clip *bool) {
					cfg := engine.ReadAppConfig()
					if display != nil {
						engine.PrefixDisplay = *display
						cfg.PrefixDisplay = display
					}
					if clip != nil {
						engine.PrefixClipboard = *clip
						cfg.PrefixClipboard = clip
					}
					engine.SaveAppConfig(cfg)
				}
				if len(parts) == 2 {
					fmt.Printf("prefix: display %s   clipboard %s\n",
						func() string {
							if engine.PrefixDisplay {
								return "on"
							}
							return "off"
						}(),
						func() string {
							if engine.PrefixClipboard {
								return "on"
							}
							return "off"
						}())
				} else {
					sub := strings.ToLower(parts[2])
					switch sub {
					case "on":
						t := true
						applyPrefix(&t, &t)
						fmt.Println("prefix: on")
					case "off":
						f := false
						applyPrefix(&f, &f)
						fmt.Println("prefix: off")
					case "display":
						if len(parts) < 4 {
							fmt.Println(styleError("usage: setting prefix display on|off"))
						} else {
							switch strings.ToLower(parts[3]) {
							case "on":
								t := true
								applyPrefix(&t, nil)
								fmt.Println("prefix display: on")
							case "off":
								f := false
								applyPrefix(&f, nil)
								fmt.Println("prefix display: off")
							default:
								fmt.Println(styleError("usage: setting prefix display on|off"))
							}
						}
					case "clipboard":
						if len(parts) < 4 {
							fmt.Println(styleError("usage: setting prefix clipboard on|off"))
						} else {
							switch strings.ToLower(parts[3]) {
							case "on":
								t := true
								applyPrefix(nil, &t)
								fmt.Println("prefix clipboard: on")
							case "off":
								f := false
								applyPrefix(nil, &f)
								fmt.Println("prefix clipboard: off")
							default:
								fmt.Println(styleError("usage: setting prefix clipboard on|off"))
							}
						}
					default:
						fmt.Println(styleError("usage: setting prefix [on|off|display on|off|clipboard on|off]"))
					}
				}
			default:
				fmt.Println(styleError("unknown setting. try: setting clipboard / grouping / prefix"))
			}
			continue
		}

		// Drill mode.
		if lowerInput == "drill" {
			runDrill(line)
			continue
		}

		// Variable assignment: name = expression.

		if varName, exprStr, ok := engine.TryParseAssignment(rawInput); ok {
			if _, reserved := engine.ModeMap[strings.ToLower(varName)]; reserved {
				fmt.Println(styleError("error: '" + varName + "' is a reserved mode keyword"))
				continue
			}
			// Guard against overwriting builtin CalcEnv names (units, cast functions,
			// math functions, etc.). Allow re-assignment of names the user already set.
			if _, isBuiltin := engine.CalcEnv[varName]; isBuiltin {
				if _, isUserVar := engine.UserVars[varName]; !isUserVar {
					fmt.Println(styleError("error: '" + varName + "' is a reserved name"))
					continue
				}
			}

			cleaned := engine.StripNumericSeparators(exprStr)
			cleaned = engine.FixBaseTypos(cleaned)
			cleaned = engine.FixNakedBases(cleaned)
			ast := engine.BuildASTString(cleaned)
			env := engine.GetMergedEnv()

			prog, compErr := expr.Compile(ast, expr.Env(env))
			if compErr != nil {
				fmt.Println(styleError("error in assignment: " + compErr.Error()))
				fmt.Println(dimGray("  ast: " + ast))
				continue
			}
			res, runErr := expr.Run(prog, env)
			if runErr != nil {
				fmt.Println(styleError("error: " + runErr.Error()))
				continue
			}

			val := toFloat64(res)
			engine.StoreVar(varName, val)
			validTokens = engine.GetValidTokens()
			engine.PersistVars()
			fmt.Printf("%s  =  %s\n",
				styleVarName(varName),
				boldWhite(engine.FormatDecimal(val)),
			)
			continue
		}

		// Standard expression pipeline.

		cleanedInput := engine.StripNumericSeparators(rawInput)
		cleanedInput = engine.FixBaseTypos(cleanedInput)
		cleanedInput = engine.FixNakedBases(cleanedInput)
		cleanedInput = strings.ReplaceAll(cleanedInput, " into ", " to ")
		cleanedInput = strings.ReplaceAll(cleanedInput, " in to ", " to ")

		sizeCtx := engine.InputSizeUnitContext(cleanedInput)
		convTarget := engine.DetectConversionTarget(cleanedInput)

		// Autocorrect: suggest only if the fix compiles AND evaluates to a
		// non-function result (bare function names are not useful suggestions).
		sanitizedInput, changed := engine.SanitizeInput(cleanedInput, validTokens)
		if changed {
			testAST := engine.BuildASTString(sanitizedInput)
			testEnv := engine.GetMergedEnv()
			testProg, testCompErr := expr.Compile(testAST, expr.Env(testEnv))
			if testCompErr == nil {
				testRes, testRunErr := expr.Run(testProg, testEnv)
				isFn := false
				if testRunErr == nil {
					switch testRes.(type) {
					case func(float64) string, func(float64) float64, func(float64, float64) float64:
						isFn = true
					}
				}
				if testRunErr == nil && !isFn {
					fmt.Printf("%s %s? (y/n): ",
						styleAutocorrect("did you mean:"),
						styleAutocorrect(sanitizedInput),
					)
					confirmRaw, _ := line.Prompt("")
					confirm := strings.ToLower(strings.TrimSpace(confirmRaw))
					if confirm != "y" && confirm != "yes" {
						sanitizedInput = cleanedInput
					}
				} else {
					sanitizedInput = cleanedInput
				}
			} else {
				sanitizedInput = cleanedInput
			}
		}

		processedInput := engine.BuildASTString(sanitizedInput)
		env := engine.GetMergedEnv()

		program, compErr := expr.Compile(processedInput, expr.Env(env))
		if compErr != nil && engine.ContainsFormatFn(processedInput) {
			// Format functions return strings; strip wrappers and retry so that
			// "hex(a) + 1" → "a + 1" compiles and evaluates correctly.
			if stripped := engine.StripFormatWrappers(processedInput); stripped != processedInput {
				if p2, err2 := expr.Compile(stripped, expr.Env(env)); err2 == nil {
					program, compErr, processedInput = p2, nil, stripped
				}
			}
		}
		if compErr != nil {
			fmt.Println(styleError("error: could not parse expression"))
			fmt.Println(dimGray("  ast: " + processedInput))
			continue
		}
		result, runErr := expr.Run(program, env)
		if runErr != nil {
			fmt.Println(styleError("error: " + runErr.Error()))
			continue
		}

		// If the result is a string but the expression contains format functions
		// combined with arithmetic, it means we got string concatenation instead of
		// numeric addition (e.g. "bin(a) + bin(b)" -> "0b10100b101"). Retry with
		// format wrappers stripped.
		// BUT: only strip if the string is NOT a valid formatted value. A result like
		// "0b10000000" from bin(0x80) is correct and must be printed as-is; only
		// garbage like "0b10100b101" (concatenated strings) should trigger the retry.
		if sv, isStr := result.(string); isStr && engine.ContainsFormatFn(processedInput) {
			if _, validResult := engine.ParseResultString(sv); !validResult {
				if stripped := engine.StripFormatWrappers(processedInput); stripped != processedInput {
					if p2, err2 := expr.Compile(stripped, expr.Env(env)); err2 == nil {
						if res2, err2 := expr.Run(p2, env); err2 == nil {
							result = res2
							processedInput = stripped
						}
					}
				}
			}
		}

		switch v := result.(type) {
		case float64:
			outN(v, sizeCtx, convTarget)
		case float32:
			outN(float64(v), sizeCtx, convTarget)
		case int:
			outN(float64(v), sizeCtx, convTarget)
		case int64:
			outN(float64(v), sizeCtx, convTarget)
		case string:
			// Update _ so "bin32(-5)" then "_ + 1" works as expected.
			if f, ok := engine.ParseResultString(v); ok {
				engine.SetLastResult(f)
			}
			if engine.ClipboardEnabled {
				clipboard.WriteAll(engine.ApplyClipboardTransforms(v))
			}
			fmt.Println(colorizeResult(engine.ApplyDisplayTransforms(v)))
		case func(float64) string, func(float64) float64:
			fmt.Println(styleError("error: function needs arguments, e.g. bin(255)"))
		default:
			s := fmt.Sprintf("%v", v)
			if engine.ClipboardEnabled {
				clipboard.WriteAll(s)
			}
			fmt.Println(colorizeResult(s))
		}
	}
}

// outN applies type mode, updates _, formats, copies to clipboard, and prints.
func outN(val float64, sizeCtx engine.SizeUnitContext, convTarget string) {
	wrapped, overflowed := engine.ApplyTypeMode(val)
	engine.SetLastResult(wrapped)

	var typeHint string
	if engine.CurrentTypeMode != "auto" {
		if overflowed {
			typeHint = engine.CurrentTypeMode + " ovf"
		} else {
			typeHint = engine.CurrentTypeMode
		}
	}

	// mode all: render dec + hex + bin with per-base colors.
	if convTarget == "" && engine.CurrentMode == "all" {
		dec, hex, bin := engine.FormatAll(wrapped)
		line := boldWhite(dec) + "  " + styleHex(hex) + "  " + styleBin(bin)
		if typeHint != "" {
			line += dimGray("  [" + typeHint + "]")
		}
		if engine.ClipboardEnabled {
			clipboard.WriteAll(engine.FormatDecimal(wrapped))
		}
		fmt.Println(line)
		return
	}

	var terminal, clip string
	if convTarget != "" {
		terminal = engine.FormatWithTargetUnit(wrapped, convTarget)
		clip = engine.FormatDecimal(wrapped)
	} else {
		terminal = engine.FormatTerminal(wrapped, sizeCtx, typeHint)
		clip = engine.FormatClipboard(wrapped)
	}
	if engine.ClipboardEnabled {
		clipboard.WriteAll(clip)
	}
	fmt.Println(colorizeResult(terminal))
}

// toFloat64 coerces any numeric interface{} value to float64.
func toFloat64(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	}
	return 0
}
