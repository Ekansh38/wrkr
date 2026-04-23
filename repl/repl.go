package repl

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
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
		fmt.Println("  bitwise:  & | ^ ~ << >>  (standard C precedence)")
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
		fmt.Println("  cast functions return float64 — compose with arithmetic:")
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
		fmt.Println("  setting clipboard on|off    toggle clipboard copy (default: on)")
		fmt.Println("  setting clipboard           query current value")
		fmt.Println()
		fmt.Println("  settings are saved to ~/.wrkr_config.json")
	case "all":
		printHelp("math")
		printHelp("systems")
		printHelp("units")
		printHelp("modes")
		printHelp("types")
		printHelp("vars")
		printHelp("settings")
	default:
		fmt.Println("help math      trig, logs, pow")
		fmt.Println("help systems   base literals and conversion")
		fmt.Println("help units     unit conversion")
		fmt.Println("help modes     output modes")
		fmt.Println("help types     integer type modes (u8, s16, ...)")
		fmt.Println("help vars      variables")
		fmt.Println("help settings  clipboard and other settings")
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
					"exit", "quit", "clear", ":e", ":q",
				}, engine.GetValidTokens()...)
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
						candidates = []string{"clipboard"}
					case 2:
						if len(fields) >= 2 && fields[1] == "clipboard" {
							candidates = []string{"on", "off"}
						}
					}
				case "help":
					commandContext = true
					if numCompleted == 1 {
						candidates = []string{
							"math", "systems", "units", "modes", "types",
							"vars", "settings", "all",
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

	// Restore persisted settings (format mode, type mode, clipboard).
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

		// :e — open expression(s) in $EDITOR. Each non-empty line runs as a
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
			s1 := engine.FixBaseTypos(s0)
			s2 := engine.FixNakedBases(s1)
			s3 := strings.ReplaceAll(s2, " into ", " to ")
			s3 = strings.ReplaceAll(s3, " in to ", " to ")
			s4 := engine.ProcessConversions(s3)
			s5 := engine.ProcessFormatting(s4)
			s6 := engine.FixImplicitMultiplication(s5)
			s7 := engine.RewriteBitwiseOps(s6)
			s8 := engine.TranslateBases(s7)
			s9 := engine.ExpandConstants(s7) // unit names -> numbers (pre-translate view)

			steps := []struct{ label, val string }{
				{"input   ", s0},
				{"typos   ", s1},
				{"bases   ", s2},
				{"keywords", s3},
				{"convert ", s4},
				{"format  ", s5},
				{"multiply", s6},
				{"bitwise ", s7},
				{"ast     ", s8},
				{"expanded", s9},
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
			if prog, err := expr.Compile(s8, expr.Env(env)); err == nil {
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
				printHelp("settings")
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
			default:
				fmt.Println(styleError("unknown setting. try: setting clipboard on|off"))
			}
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

			cleaned := engine.FixBaseTypos(exprStr)
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

		cleanedInput := engine.FixBaseTypos(rawInput)
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
				clipboard.WriteAll(v)
			}
			fmt.Println(colorizeResult(v))
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
