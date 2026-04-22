package engine_test

import (
	"math"
	"strings"
	"testing"

	"github.com/expr-lang/expr"

	"wrkr/engine"
)

// eval runs the full preprocessing pipeline on input and returns the float64 result.
// It mirrors exactly what the REPL does: FixBaseTypos → FixNakedBases → BuildASTString → Compile → Run.
func eval(t *testing.T, input string) float64 {
	t.Helper()
	s := engine.FixBaseTypos(input)
	s = engine.FixNakedBases(s)
	s = strings.ReplaceAll(s, " into ", " to ")
	s = engine.BuildASTString(s)
	env := engine.GetMergedEnv()
	prog, err := expr.Compile(s, expr.Env(env))
	if err != nil {
		t.Fatalf("eval compile(%q): %v", input, err)
	}
	res, err := expr.Run(prog, env)
	if err != nil {
		t.Fatalf("eval run(%q): %v", input, err)
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
