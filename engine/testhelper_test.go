package engine_test

import (
	"math"
	"strings"
	"testing"

	"github.com/expr-lang/expr"

	"github.com/Ekansh38/wrkr/engine"
)

// pipeline runs the full preprocessing pipeline and returns the AST string.
// Order matches the REPL's real evaluation path.
func pipeline(input string) string {
	s := engine.StripNumericSeparators(input)
	s = engine.FixBaseTypos(s)
	s = engine.FixNakedBases(s)
	s = strings.ReplaceAll(s, " into ", " to ")
	return engine.BuildASTString(s)
}

// eval runs the full preprocessing pipeline on input and returns the float64 result.
// tryCompileRun compiles and runs s, returning (result, ok).
// If the compile fails or the result is a string (format function in arithmetic
// context produces string concat), and s contains format functions, strips them
// and retries once.
func tryCompileRun(s string, env map[string]interface{}) (interface{}, error) {
	attempt := func(ast string) (interface{}, error) {
		prog, err := expr.Compile(ast, expr.Env(env))
		if err != nil {
			return nil, err
		}
		return expr.Run(prog, env)
	}

	res, err := attempt(s)
	// Retry with stripped format wrappers when:
	//   (a) compile/run failed, OR
	//   (b) result is a string but s contains format functions (string + string concat)
	if engine.ContainsFormatFn(s) {
		needRetry := err != nil
		if !needRetry {
			if _, isStr := res.(string); isStr {
				needRetry = true
			}
		}
		if needRetry {
			stripped := engine.StripFormatWrappers(s)
			if stripped != s {
				if res2, err2 := attempt(stripped); err2 == nil {
					return res2, nil
				}
			}
		}
	}
	return res, err
}

func eval(t *testing.T, input string) float64 {
	t.Helper()
	s := pipeline(input)
	env := engine.GetMergedEnv()
	res, err := tryCompileRun(s, env)
	if err != nil {
		t.Fatalf("eval(%q -> %q): %v", input, s, err)
	}
	switch v := res.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	t.Fatalf("eval(%q): unexpected result type %T = %v", input, res, res)
	return 0
}

// evalStr runs the full pipeline and returns the string result (for base conversion functions).
func evalStr(t *testing.T, input string) string {
	t.Helper()
	s := pipeline(input)
	env := engine.GetMergedEnv()
	prog, err := expr.Compile(s, expr.Env(env))
	if err != nil {
		t.Fatalf("evalStr compile(%q -> %q): %v", input, s, err)
	}
	res, err := expr.Run(prog, env)
	if err != nil {
		t.Fatalf("evalStr run(%q): %v", input, err)
	}
	if v, ok := res.(string); ok {
		return v
	}
	// numeric — format as decimal string so callers can strcmp
	switch v := res.(type) {
	case float64:
		return engine.FormatDecimal(v)
	case float32:
		return engine.FormatDecimal(float64(v))
	case int:
		return engine.FormatDecimal(float64(v))
	case int64:
		return engine.FormatDecimal(float64(v))
	}
	t.Fatalf("evalStr(%q): unexpected result type %T = %v", input, res, res)
	return ""
}

// near asserts that got ≈ want within a relative tolerance of 1e-9
// (falling back to absolute 1e-9 for values near zero).
// This avoids false failures from float64 representation noise.
func near(t *testing.T, got, want float64, label string) {
	t.Helper()
	// Relative epsilon scales with the magnitude of the expected value.
	eps := 1e-9 * math.Max(1.0, math.Abs(want))
	if math.Abs(got-want) > eps {
		t.Errorf("%s:\n  got  %.15g\n  want %.15g\n  Δ    %g", label, got, want, math.Abs(got-want))
	}
}
