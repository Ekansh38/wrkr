package repl

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/expr-lang/expr"
	"github.com/peterh/liner"

	"github.com/Ekansh38/wrkr/engine"
)

func printHelp(topic string) {
	fmt.Println()
	switch strings.ToLower(topic) {
	case "math", "geometry", "gamedev":
		fmt.Println("--- Math & Functions ---")
		fmt.Println()
		fmt.Println("  Trig       sin(x)  cos(x)  tan(x)  hypot(a, b)  pi")
		fmt.Println("  Roots      sqrt(x)  abs(x)")
		fmt.Println("  Rounding   round(x)  floor(x)  ceil(x)")
		fmt.Println("  Logs       log(x)  log2(x)  log10(x)")
		fmt.Println("  Power      pow(base, exp)")
		fmt.Println()
		fmt.Println("  Examples:")
		fmt.Println("    hypot(3, 4)          →  5")
		fmt.Println("    pow(2, 10)           →  1024")
		fmt.Println("    floor(log2(1000))    →  9")
		fmt.Println("    sin(pi / 2)          →  1")
	case "systems", "computer", "hardware":
		fmt.Println("--- Base Literals & Conversion ---")
		fmt.Println()
		fmt.Println("  Input — three ways to write a non-decimal number:")
		fmt.Println("    Prefix          0xFF   0b1010   0o17")
		fmt.Println("    Natural lang    FF hex   101 bin   17 octal")
		fmt.Println("    Typo shorthand  \\xFF   \\b1010   \\o17")
		fmt.Println()
		fmt.Println("  Output — three ways to convert to a base:")
		fmt.Println("    Function        hex(255)   bin(255)   octal(255)   dec(0xFF)")
		fmt.Println("    to keyword      255 to hex   0xFF to bin   255 to octal")
		fmt.Println("    Annotated       0x123 hex to bin   0b1010 bin to hex")
		fmt.Println()
		fmt.Println("  The annotated form is handy when you already have a prefixed literal")
		fmt.Println("  and just want it in a different base. All three forms produce the same result.")
		fmt.Println()
		fmt.Println("  Data sizes:  b  bit  kb  mb  gb  tb")
		fmt.Println("    5 mb               →  5242880")
		fmt.Println("    2 * tb / (4 * kb)  →  536870912")
	case "units", "conversions":
		fmt.Println("--- Units & Conversions ---")
		fmt.Println()
		fmt.Println("  Distance:  m  km  cm  mm  mi  ft  in")
		fmt.Println("  Data:      b  bit  kb  mb  gb  tb")
		fmt.Println()
		fmt.Println("  Syntax:  <number> <unit> to <unit>")
		fmt.Println()
		fmt.Println("  Examples:")
		fmt.Println("    50 mi to km     →  80.4672 km")
		fmt.Println("    100 ft to m     →  30.48 m")
		fmt.Println("    1 gb to mb      →  1024 MB")
		fmt.Println("    1 mb to bits    →  8388608 bits")
		fmt.Println("    30 cm to in     →  11.811... in")
		fmt.Println()
		fmt.Println("  The result always shows the target unit label and ignores the current")
		fmt.Println("  output mode — so '1 gb to mb' shows '1024 MB' even in hex mode.")
	case "modes", "state":
		fmt.Println("--- Output Modes ---")
		fmt.Println()
		fmt.Println("  Switch:  mode <name>     Query:  mode")
		fmt.Println()
		fmt.Println("  Mode   Terminal                        Clipboard")
		fmt.Println("  -----  ------------------------------  ----------")
		fmt.Println("  dec    1048576  [1 MB]                 1048576")
		fmt.Println("  size   1 MB                            1")
		fmt.Println("  bytes  1048576 B                       1048576")
		fmt.Println("  bits   8388608 bits                    8388608")
		fmt.Println("  hex    0x100000  [Hex]                 0x100000")
		fmt.Println("  bin    0b100000000000000000000  [Bin]  0b100000000000000000000")
		fmt.Println("  oct    0o4000000  [Oct]                0o4000000")
		fmt.Println()
		fmt.Println("  Terminal and clipboard outputs differ by design — you paste what you need.")
		fmt.Println()
		fmt.Println("  Smart Hint (dec mode): when your expression uses a data-size unit, a")
		fmt.Println("  human-readable bracket like [1 MB] is added automatically. If units")
		fmt.Println("  cancel out (e.g. mb / gb), the hint stays silent to avoid misleading you.")
		fmt.Println()
		fmt.Println("  Unit conversions ('1 gb to mb') always show the target unit label and")
		fmt.Println("  bypass the current mode entirely.")
		fmt.Println()
		fmt.Println("  Note: bare words like 'hex' or 'bin' are evaluated as expressions.")
		fmt.Println("  Only 'mode hex' or 'mode bin' switches the mode.")
	case "vars", "variables", "memory":
		fmt.Println("--- User Variables ---")
		fmt.Println()
		fmt.Println("  Assign:   block = 4096")
		fmt.Println("            page  = 4 * kb")
		fmt.Println("            journal = 128 * mb")
		fmt.Println()
		fmt.Println("  Use:      journal / block")
		fmt.Println("            (512 * mb) / block")
		fmt.Println()
		fmt.Println("  List:     vars")
		fmt.Println("  Delete:   del block")
		fmt.Println()
		fmt.Println("  Variables are saved to ~/.wrkr_vars.json and offered for reload")
		fmt.Println("  the next time you start wrkr.")
		fmt.Println("  You cannot use a mode name (hex, bin, dec, ...) as a variable name.")
	case "all":
		printHelp("math")
		printHelp("systems")
		printHelp("units")
		printHelp("modes")
		printHelp("vars")
	default:
		fmt.Println("Help topics:")
		fmt.Println("  help math      trig, logs, pow, rounding")
		fmt.Println("  help systems   base literals, base conversions, data sizes")
		fmt.Println("  help units     unit conversion syntax and examples")
		fmt.Println("  help modes     output mode table with terminal vs clipboard")
		fmt.Println("  help vars      variable assignment, listing, deletion")
		fmt.Println("  help all       everything")
	}
}

// Run starts the interactive REPL.
func Run() {
	validTokens := engine.GetValidTokens()

	line := liner.NewLiner()
	defer line.Close()
	line.SetCtrlCAborts(true)

	// Tab completion: match the last partial token against all known names.
	line.SetCompleter(func(input string) []string {
		lastBoundary := strings.LastIndexAny(input, " \t(,+-*/^%")
		prefix := input
		before := ""
		if lastBoundary >= 0 {
			before = input[:lastBoundary+1]
			prefix = input[lastBoundary+1:]
		}
		if prefix == "" {
			return nil
		}
		lp := strings.ToLower(prefix)
		tokens := engine.GetValidTokens()
		var out []string
		for _, tok := range tokens {
			if strings.HasPrefix(strings.ToLower(tok), lp) && strings.ToLower(tok) != lp {
				out = append(out, before+tok)
			}
		}
		sort.Strings(out)
		return out
	})

	fmt.Println("wrkr — type 'help all' for reference, 'exit' to quit")

	// ── Saved variable prompt ─────────────────────────────────────────────────
	if saved, _ := engine.ReadSavedVars(); saved != nil {
		fmt.Println()
		fmt.Printf("  %d saved variable(s):\n", len(saved.Vars))

		// Print in sorted order for readability.
		keys := make([]string, 0, len(saved.Vars))
		for k := range saved.Vars {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("    %s  =  %s\n",
				styleVarName(fmt.Sprintf("%-12s", k)),
				boldWhite(engine.FormatDecimal(saved.Vars[k])),
			)
		}
		fmt.Println()
		fmt.Println("  [L] Load   [S] Skip (keep saved)   [D] Delete and start fresh")
		fmt.Println()

		choice, _ := line.Prompt("> ")
		switch strings.ToLower(strings.TrimSpace(choice)) {
		case "l", "load":
			engine.ApplySavedVars(saved.Vars)
			validTokens = engine.GetValidTokens()
			fmt.Printf("  loaded %d variable(s)\n", len(saved.Vars))
		case "d", "delete":
			engine.DeletePersistedVars()
			fmt.Println("  saved variables deleted")
		default:
			fmt.Println("  skipped (file kept)")
		}
		fmt.Println()
	}

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

		// Debug: show every pipeline transformation step for an expression.
		if strings.HasPrefix(lowerInput, "debug ") {
			debugExpr := strings.TrimSpace(rawInput[6:])
			s0 := debugExpr
			s1 := engine.FixBaseTypos(s0)
			s2 := engine.FixNakedBases(s1)
			s3 := strings.ReplaceAll(s2, " into ", " to ")
			s3 = strings.ReplaceAll(s3, " in to ", " to ")
			s4 := engine.ProcessConversions(s3)
			s5 := engine.ProcessFormatting(s4)
			s6 := engine.FixImplicitMultiplication(s5)
			s7 := engine.TranslateBases(s6)

			steps := []struct{ label, val string }{
				{"input   ", s0},
				{"typos   ", s1},
				{"bases   ", s2},
				{"keywords", s3},
				{"convert ", s4},
				{"format  ", s5},
				{"multiply", s6},
				{"ast     ", s7},
			}

			fmt.Println()
			prev := ""
			for _, step := range steps {
				changed := step.val != prev && prev != ""
				arrow := "  "
				if changed {
					arrow = dimGray("→ ")
				}
				fmt.Printf("  %s  %s%s\n", dimGray(step.label), arrow, boldWhite(step.val))
				prev = step.val
			}

			// Also evaluate and show the result.
			env := engine.GetMergedEnv()
			if prog, err := expr.Compile(s7, expr.Env(env)); err == nil {
				if res, err := expr.Run(prog, env); err == nil {
					fmt.Printf("\n  %s  %s\n", dimGray("result  "), colorizeResult(fmt.Sprintf("%v", res)))
				}
			}
			fmt.Println()
			continue
		}

		// List all user-defined variables.
		if lowerInput == "vars" {
			if len(engine.UserVars) == 0 {
				fmt.Println("No variables defined.  Try: block = 4096")
			} else {
				fmt.Println("User-defined variables:")
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
			if engine.DeleteVar(varName) {
				validTokens = engine.GetValidTokens()
				engine.PersistVars()
				fmt.Printf("deleted %s\n", styleVarName(varName))
			} else {
				fmt.Printf("%s  (not a user variable — use 'vars' to list)\n",
					styleError("unknown: "+varName))
			}
			continue
		}

		// ── Mode query / switch ───────────────────────────────────────────────

		if lowerInput == "mode" {
			fmt.Printf("Current output mode: %s\n", colorMode(engine.CurrentMode))
			continue
		}
		if strings.HasPrefix(lowerInput, "mode ") {
			modeCmd := strings.TrimSpace(strings.TrimPrefix(lowerInput, "mode "))
			if newMode, ok := engine.ModeMap[modeCmd]; ok {
				engine.CurrentMode = newMode
				fmt.Printf("Output mode → %s\n", colorMode(newMode))
				continue
			}
		}

		// ── Variable assignment: name = expression ────────────────────────────

		if varName, exprStr, ok := engine.TryParseAssignment(rawInput); ok {
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
				fmt.Println(dimGray("  ast: " + ast))
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
			engine.PersistVars()
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
			fmt.Println(dimGray("  ast: " + processedInput))
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
