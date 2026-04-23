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
		fmt.Println("math")
		fmt.Println()
		fmt.Println("  sin(x)  cos(x)  tan(x)  hypot(a,b)  pi")
		fmt.Println("  sqrt(x)  abs(x)  pow(base,exp)")
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
		fmt.Println()
		fmt.Println("  output:")
		fmt.Println("    hex(255)  bin(255)  octal(255)  dec(0xFF)    function")
		fmt.Println("    255 to hex  0xFF to bin                      to keyword")
		fmt.Println("    0x123 hex to bin  0b1010 bin to hex          annotated source")
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
	case "modes", "state":
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
		fmt.Println("  dec mode adds [1 MB] hint when expression uses a data unit.")
		fmt.Println("  suppressed when units cancel (e.g. mb/gb*1000 = dimensionless).")
		fmt.Println()
		fmt.Println("  bare 'hex'/'bin' evaluate as expressions. only 'mode hex' switches.")
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
		fmt.Println("  saved to ~/.wrkr_vars.json, reloaded on next launch")
		fmt.Println("  mode names (hex, bin, dec...) are reserved, cannot be used as var names")
	case "all":
		printHelp("math")
		printHelp("systems")
		printHelp("units")
		printHelp("modes")
		printHelp("vars")
	default:
		fmt.Println("help math      trig, logs, pow")
		fmt.Println("help systems   base literals and conversion")
		fmt.Println("help units     unit conversion")
		fmt.Println("help modes     output modes")
		fmt.Println("help vars      variables")
		fmt.Println("help all       everything")
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
		// Print saved vars in sorted order.
		keys := make([]string, 0, len(saved.Vars))
		for k := range saved.Vars {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		if engine.ReadAutoload() {
			// Autoload set — load silently, no prompt.
			engine.ApplySavedVars(saved.Vars)
			validTokens = engine.GetValidTokens()
			fmt.Println()
			fmt.Printf("  %s  %d variable(s) loaded  %s\n",
				boldWhite("↑"),
				len(saved.Vars),
				dimGray("(vars · del <name>)"),
			)
			fmt.Println()
		} else {
			fmt.Println()
			fmt.Printf("  %s  %d saved variable(s)\n", boldWhite("→"), len(saved.Vars))
			for _, k := range keys {
				fmt.Printf("    %s  =  %s\n",
					styleVarName(fmt.Sprintf("%-12s", k)),
					boldWhite(engine.FormatDecimal(saved.Vars[k])),
				)
			}
			fmt.Println()
			fmt.Println("  Enter  load & remember    s  this session only    d  delete")
			fmt.Println()

			choice, _ := line.Prompt("> ")
			switch strings.ToLower(strings.TrimSpace(choice)) {
			case "d", "delete":
				engine.DeletePersistedVars()
				fmt.Println("  variables deleted")
			case "s", "session":
				// Load for this session only — no autoload set.
				engine.ApplySavedVars(saved.Vars)
				validTokens = engine.GetValidTokens()
				fmt.Printf("  loaded for this session\n")
			default:
				// Empty input (Enter) or anything else = load + set autoload.
				engine.ApplySavedVars(saved.Vars)
				validTokens = engine.GetValidTokens()
				engine.SetAutoload(true)
				fmt.Printf("  loaded — will autoload from now on  %s\n",
					dimGray("(change: del all vars or edit ~/.wrkr_config.json)"))
			}
			fmt.Println()
		}
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
