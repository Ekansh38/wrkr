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
	line.SetCtrlCAborts(true) // returns ErrPromptAborted instead of sending SIGINT

	fmt.Println("wrkr — type 'help all' for reference, 'exit' to quit")

	for {
		// Mode tag on its own line so liner's \r redraws never overwrite it.
		fmt.Printf("\n%s\n", colorMode("["+engine.CurrentMode+"]"))
		rawInput, err := line.Prompt("> ")
		if err != nil {
			if err == liner.ErrPromptAborted {
				fmt.Println(dimGray("  type :q to quit"))
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
					fmt.Printf("  %s  =  %s\n",
						styleVarName(fmt.Sprintf("%-12s", k)),
						boldWhite(engine.FormatDecimal(v.(float64))),
					)
				}
			}
			continue
		}

		// ── Mode query / switch ───────────────────────────────────────────────

		if lowerInput == "mode" {
			fmt.Printf("Current output mode: %s\n", colorMode(engine.CurrentMode))
			continue
		}
		modeCmd := lowerInput
		if strings.HasPrefix(lowerInput, "mode ") {
			modeCmd = strings.TrimSpace(strings.TrimPrefix(lowerInput, "mode "))
		}
		if newMode, ok := engine.ModeMap[modeCmd]; ok {
			engine.CurrentMode = newMode
			fmt.Printf("Output mode → %s\n", colorMode(newMode))
			continue
		}

		// ── Variable assignment: name = expression ────────────────────────────

		if varName, exprStr, ok := engine.TryParseAssignment(rawInput); ok {
			// Guard reserved keywords.
			if _, reserved := engine.ModeMap[strings.ToLower(varName)]; reserved {
				fmt.Println(styleError("Error: '" + varName + "' is a reserved mode keyword."))
				continue
			}

			cleaned := engine.FixBaseTypos(exprStr)
			cleaned = engine.FixNakedBases(cleaned)
			ast := engine.BuildASTString(cleaned)
			env := engine.GetMergedEnv()

			prog, compErr := expr.Compile(ast, expr.Env(env))
			if compErr != nil {
				fmt.Println(styleError("Error in assignment: " + compErr.Error()))
				continue
			}
			res, runErr := expr.Run(prog, env)
			if runErr != nil {
				fmt.Println(styleError("Error: " + runErr.Error()))
				continue
			}

			val := toFloat64(res)
			engine.StoreVar(varName, val)
			validTokens = engine.GetValidTokens()
			fmt.Printf("%s  =  %s\n",
				styleVarName(varName),
				boldWhite(engine.FormatDecimal(val)),
			)
			continue
		}

		// ── Standard expression pipeline ──────────────────────────────────────

		// 1. Early string cleanups.
		cleanedInput := engine.FixBaseTypos(rawInput)
		cleanedInput = engine.FixNakedBases(cleanedInput)
		cleanedInput = strings.ReplaceAll(cleanedInput, " into ", " to ")
		cleanedInput = strings.ReplaceAll(cleanedInput, " in to ", " to ")

		// 2. Size unit context for Smart Hint.
		sizeCtx := engine.InputSizeUnitContext(cleanedInput)

		// 2b. Detect conversion target to bypass output mode.
		convTarget := engine.DetectConversionTarget(cleanedInput)

		// 3. Autocorrect: suggest only if the fix compiles.
		sanitizedInput, changed := engine.SanitizeInput(cleanedInput, validTokens)
		if changed {
			testAST := engine.BuildASTString(sanitizedInput)
			testEnv := engine.GetMergedEnv()
			_, testErr := expr.Compile(testAST, expr.Env(testEnv))
			if testErr == nil {
				fmt.Printf("%s %s? (y/n): ",
					styleAutocorrect("Did you mean:"),
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
		}

		// 4. Build AST and evaluate.
		processedInput := engine.BuildASTString(sanitizedInput)
		env := engine.GetMergedEnv()

		program, compErr := expr.Compile(processedInput, expr.Env(env))
		if compErr != nil {
			fmt.Println(styleError("Error: Could not parse expression."))
			continue
		}
		result, runErr := expr.Run(program, env)
		if runErr != nil {
			fmt.Println(styleError("Error: " + runErr.Error()))
			continue
		}

		// 5. Format and output.
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
			clipboard.WriteAll(v)
			fmt.Println(colorizeResult(v))
		case func(float64) string, func(float64) float64:
			fmt.Println(styleError("[Error: function requires arguments — e.g., bin(255)]"))
		default:
			s := fmt.Sprintf("%v", v)
			clipboard.WriteAll(s)
			fmt.Println(colorizeResult(s))
		}
	}
}

// outN formats a numeric result, writes clipboard, and prints with colors.
func outN(val float64, sizeCtx engine.SizeUnitContext, convTarget string) {
	var terminal, clip string
	if convTarget != "" {
		terminal = engine.FormatWithTargetUnit(val, convTarget)
		clip = engine.FormatDecimal(val)
	} else {
		terminal = engine.FormatTerminal(val, sizeCtx)
		clip = engine.FormatClipboard(val)
	}
	clipboard.WriteAll(clip)
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
