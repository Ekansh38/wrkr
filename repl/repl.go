package repl

import (
	"fmt"
	"io"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/expr-lang/expr"
	"github.com/peterh/liner"

	"wrkr/engine"
)

func printHelp(topic string) {
	fmt.Println()
	switch strings.ToLower(topic) {
	case "math", "geometry", "gamedev":
		fmt.Println("--- Math & GameDev ---")
		fmt.Println("Trig:    sin, cos, tan, hypot, pi")
		fmt.Println("Tools:   sqrt, abs, round, floor, ceil, log2")
		fmt.Println("Example: hypot(3, 4)  →  5")
	case "systems", "computer", "hardware":
		fmt.Println("--- Systems & Hardware ---")
		fmt.Println("Sizes:   b (byte), bit, kb, mb, gb, tb")
		fmt.Println("Bases:   0x1F (hex)  0b1010 (bin)  0o17 (octal)")
		fmt.Println("Funcs:   hex(255)  bin(255)  octal(255)  dec(0xFF)")
	case "units", "conversions":
		fmt.Println("--- Units & Conversions ---")
		fmt.Println("Dist:    m, km, mi, ft, in")
		fmt.Println("Data:    b, bit, kb, mb, gb, tb")
		fmt.Println("Usage:   50 mi to km   |   2 gb to mb")
	case "modes", "state":
		fmt.Println("--- Output Modes ---")
		fmt.Println("Switch:  mode <type>   (or just type 'hex', 'bin', etc.)")
		fmt.Println()
		fmt.Println("  dec    terminal: 1048576  [1 MB]       clipboard: 1048576")
		fmt.Println("  size   terminal: 1 MB                  clipboard: 1")
		fmt.Println("  bytes  terminal: 1048576 B             clipboard: 1048576")
		fmt.Println("  bits   terminal: 8388608 bits          clipboard: 8388608")
		fmt.Println("  hex    terminal: 0x100000  [Hex]       clipboard: 0x100000")
		fmt.Println("  bin    terminal: 0b1010  [Bin]         clipboard: 0b1010")
		fmt.Println("  oct    terminal: 0o17  [Oct]           clipboard: 0o17")
		fmt.Println()
		fmt.Println("The Smart Hint (dec mode) adds a human-readable bracket when your")
		fmt.Println("expression contains a data-size unit but the result is ≥ 1 KB.")
		fmt.Println("bits mode multiplies the result by 8 (everything is stored as bytes internally).")
	case "vars", "variables", "memory":
		fmt.Println("--- User Variables ---")
		fmt.Println("Assign:  block = 4096")
		fmt.Println("         page  = 4 * kb")
		fmt.Println("Use:     (512 * mb) / block")
		fmt.Println("List:    vars")
		fmt.Println("Note:    variables persist for the life of the process.")
	case "all":
		printHelp("math")
		printHelp("systems")
		printHelp("units")
		printHelp("modes")
		printHelp("vars")
	default:
		fmt.Println("Help topics: math, systems, units, modes, vars, all")
	}
}

// Run starts the interactive REPL.
func Run() {
	validTokens := engine.GetValidTokens()

	line := liner.NewLiner()
	defer line.Close()
	line.SetCtrlCAborts(true)

	fmt.Println("wrkr — context-aware calculator")
	fmt.Println("type 'help all' for reference, 'exit' to quit")

	for {
		fmt.Println()
		rawInput, err := line.Prompt("> ")
		if err != nil {
			if err == liner.ErrPromptAborted || err == io.EOF {
				return
			}
			continue
		}

		rawInput = strings.TrimSpace(rawInput)
		if rawInput == "" {
			continue
		}
		line.AppendHistory(rawInput)
		lowerInput := strings.ToLower(rawInput)

		// ── Built-in commands ─────────────────────────────────────────────────

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

		// List all user-defined variables.
		if lowerInput == "vars" {
			if len(engine.UserVars) == 0 {
				fmt.Println("No variables defined.  Try: block = 4096")
			} else {
				fmt.Println("User-defined variables:")
				for k, v := range engine.UserVars {
					fmt.Printf("  %-12s = %v\n", k, v)
				}
			}
			continue
		}

		// ── Mode query / switch ───────────────────────────────────────────────

		if lowerInput == "mode" {
			fmt.Printf("Current output mode: %s\n", engine.CurrentMode)
			continue
		}
		modeCmd := lowerInput
		if strings.HasPrefix(lowerInput, "mode ") {
			modeCmd = strings.TrimSpace(strings.TrimPrefix(lowerInput, "mode "))
		}
		if newMode, ok := engine.ModeMap[modeCmd]; ok {
			engine.CurrentMode = newMode
			fmt.Printf("Output mode → %s\n", engine.CurrentMode)
			continue
		}

		// ── Variable assignment: name = expression ────────────────────────────

		if varName, exprStr, ok := engine.TryParseAssignment(rawInput); ok {
			// Guard reserved keywords.
			if _, reserved := engine.ModeMap[strings.ToLower(varName)]; reserved {
				fmt.Printf("Error: '%s' is a reserved mode keyword.\n", varName)
				continue
			}

			cleaned := engine.FixBaseTypos(exprStr)
			cleaned = engine.FixNakedBases(cleaned)
			ast := engine.BuildASTString(cleaned)
			env := engine.GetMergedEnv()

			prog, compErr := expr.Compile(ast, expr.Env(env))
			if compErr != nil {
				fmt.Printf("Error in assignment: %v\n", compErr)
				continue
			}
			res, runErr := expr.Run(prog, env)
			if runErr != nil {
				fmt.Printf("Error: %v\n", runErr)
				continue
			}

			val := toFloat64(res)
			engine.StoreVar(varName, val)
			validTokens = engine.GetValidTokens() // refresh so autocorrect knows the new var
			fmt.Printf("%s = %s\n", varName, engine.FormatDecimal(val))
			continue
		}

		// ── Standard expression pipeline ──────────────────────────────────────

		// 1. Early string cleanups.
		cleanedInput := engine.FixBaseTypos(rawInput)
		cleanedInput = engine.FixNakedBases(cleanedInput)
		cleanedInput = strings.ReplaceAll(cleanedInput, " into ", " to ")
		cleanedInput = strings.ReplaceAll(cleanedInput, " in to ", " to ")

		// 2. Remember whether a data-size unit is present (for Smart Hint).
		hasUnit := engine.InputHasSizeUnit(cleanedInput)

		// 2b. If this is a plain "X unit to targetUnit" conversion, record the
		// target so we can bypass the current output mode and label correctly.
		convTarget := engine.DetectConversionTarget(cleanedInput)

		// 3. Autocorrect: suggest the fix only if it actually compiles.
		sanitizedInput, changed := engine.SanitizeInput(cleanedInput, validTokens)
		if changed {
			testAST := engine.BuildASTString(sanitizedInput)
			testEnv := engine.GetMergedEnv()
			_, testErr := expr.Compile(testAST, expr.Env(testEnv))
			if testErr == nil {
				fmt.Printf("Did you mean: %s? (y/n): ", sanitizedInput)
				confirmRaw, _ := line.Prompt("")
				confirm := strings.ToLower(strings.TrimSpace(confirmRaw))
				if confirm != "y" && confirm != "yes" {
					sanitizedInput = cleanedInput
				}
			} else {
				// The suggested fix is mathematical garbage — silently discard it.
				sanitizedInput = cleanedInput
			}
		}

		// 4. Build AST string and evaluate.
		processedInput := engine.BuildASTString(sanitizedInput)
		env := engine.GetMergedEnv()

		program, compErr := expr.Compile(processedInput, expr.Env(env))
		if compErr != nil {
			fmt.Println("Error: Could not parse expression.")
			continue
		}
		result, runErr := expr.Run(program, env)
		if runErr != nil {
			fmt.Printf("Error: %v\n", runErr)
			continue
		}

		// 5. Format and output.
		switch v := result.(type) {
		case float64:
			outN(v, hasUnit, convTarget)
		case float32:
			outN(float64(v), hasUnit, convTarget)
		case int:
			outN(float64(v), hasUnit, convTarget)
		case int64:
			outN(float64(v), hasUnit, convTarget)
		case string:
			clipboard.WriteAll(v)
			fmt.Println(v)
		case func(float64) string, func(float64) float64:
			fmt.Println("[Error: function requires arguments — e.g., bin(255)]")
		default:
			s := fmt.Sprintf("%v", v)
			clipboard.WriteAll(s)
			fmt.Println(s)
		}
	}
}

// outN formats a numeric result and prints it.
// If convTarget is set (e.g. "bits", "km"), the result is labelled with that unit
// and the current output mode is bypassed — so "1 gb to mb" always shows "1024 MB"
// regardless of whether you're in size/hex/bin mode.
func outN(val float64, hasUnit bool, convTarget string) {
	var terminal, clip string
	if convTarget != "" {
		terminal = engine.FormatWithTargetUnit(val, convTarget)
		clip = engine.FormatDecimal(val)
	} else {
		terminal = engine.FormatTerminal(val, hasUnit)
		clip = engine.FormatClipboard(val)
	}
	clipboard.WriteAll(clip)
	fmt.Println(terminal)
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
