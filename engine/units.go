package engine

import (
	"fmt"
	"math"
)

// Unit pairs a set of alias strings with a base-unit multiplier.
// Distance baseline: 1 meter. Data baseline: 1 byte.
type Unit struct {
	Aliases []string
	Rate    float64
}

// UnitRates maps every alias to its base-unit multiplier.
var UnitRates = make(map[string]float64)

// SizeUnitAliases is the set of data-size aliases; used for Smart Hint detection.
var SizeUnitAliases = map[string]bool{}

// CalcEnv is the expression-evaluation environment: math functions, constants, units.
var CalcEnv = map[string]interface{}{
	"_": float64(0), // last result; seeded so expr.Compile always finds it
	// Trig / geometry
	"sin":   math.Sin,
	"cos":   math.Cos,
	"tan":   math.Tan,
	"hypot": math.Hypot,
	"pi":    math.Pi,
	// Roots / rounding / logs
	"sqrt":  math.Sqrt,
	"abs":   math.Abs,
	"log2":  math.Log2,
	"log":   math.Log,
	"log10": math.Log10,
	"pow":   math.Pow,
	"round": math.Round,
	"floor": math.Floor,
	"ceil":  math.Ceil,
	// Base-conversion helpers (return formatted strings)
	"hex":     func(f float64) string { return FormatHex(f) },
	"bin":     func(f float64) string { return FormatBin(f) },
	"octal":   func(f float64) string { return FormatOct(f) },
	"oct":     func(f float64) string { return FormatOct(f) },
	"dec":     func(f float64) string { return FormatDecimal(f) },
	"decimal": func(f float64) string { return FormatDecimal(f) },
}

func init() {
	distUnits := []Unit{
		{[]string{"m", "meter", "meters"}, 1},
		{[]string{"km", "kilometer", "kilometers"}, 1000},
		{[]string{"cm", "centimeter", "centimeters"}, 0.01},
		{[]string{"mm", "millimeter", "millimeters"}, 0.001},
		{[]string{"mi", "mile", "miles"}, 1609.344},
		{[]string{"ft", "foot", "feet"}, 0.3048},
		{[]string{"in", "inch", "inches"}, 0.0254},
	}
	dataUnits := []Unit{
		{[]string{"b", "byte", "bytes"}, 1},
		{[]string{"bit", "bits"}, 0.125},
		{[]string{"kb", "kilobyte", "kilobytes"}, 1024},
		{[]string{"mb", "megabyte", "megabytes"}, math.Pow(1024, 2)},
		{[]string{"gb", "gigabyte", "gigabytes"}, math.Pow(1024, 3)},
		{[]string{"tb", "terabyte", "terabytes"}, math.Pow(1024, 4)},
	}

	for _, def := range distUnits {
		for _, alias := range def.Aliases {
			UnitRates[alias] = def.Rate
			CalcEnv[alias] = def.Rate
		}
	}
	for _, def := range dataUnits {
		for _, alias := range def.Aliases {
			UnitRates[alias] = def.Rate
			CalcEnv[alias] = def.Rate
			SizeUnitAliases[alias] = true
		}
	}

	// Two's complement width-specific functions.
	// bin8…bin512, hex8…hex128, oct8…oct64.
	for _, bits := range []int{8, 16, 32, 64, 128, 256, 512} {
		b := bits
		CalcEnv[fmt.Sprintf("bin%d", b)] = func(f float64) string { return FormatBinN(f, b) }
	}
	for _, bits := range []int{8, 16, 32, 64, 128} {
		b := bits
		CalcEnv[fmt.Sprintf("hex%d", b)] = func(f float64) string { return FormatHexN(f, b) }
	}
	for _, bits := range []int{8, 16, 32, 64} {
		b := bits
		CalcEnv[fmt.Sprintf("oct%d", b)] = func(f float64) string { return FormatOctN(f, b) }
	}
}
