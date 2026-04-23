package engine_test

// Gamedev / idle-game perspective.
//
// Covers projectile physics, distance calculations, splash damage falloff,
// and idle-game production chains with exponential upgrade curves.

import (
	"math"
	"testing"

	"github.com/Ekansh38/wrkr/engine"
)

func TestGame_Projectile_Range(t *testing.T) {
	// Projectile range formula: (v0² × sin(2θ)) / g
	// v0 = 100 m/s, θ = 45°, g = 9.81 m/s²
	// python: (100^2 * sin(pi/2)) / 9.81 = 1019.367991845056
	near(t,
		eval(t, "(pow(100, 2) * sin(2 * pi / 4)) / 9.81"),
		1019.367991845056,
		"projectile range at 45°, v=100")
}

func TestGame_Pythagorean_Distance(t *testing.T) {
	// Entity distance: hypot(dx=300, dy=400) = 500 (classic 3-4-5 scaled ×100)
	near(t, eval(t, "hypot(300, 400)"), 500, "2D entity distance")
}

func TestGame_SplashDamage_Falloff(t *testing.T) {
	// Inverse-square falloff: 1 / dist² where dist = hypot(300, 400) = 500
	// python: 1 / 500^2 = 4e-6
	near(t, eval(t, "1 / pow(hypot(300, 400), 2)"), 4e-6, "splash damage falloff")
}

func TestGame_IdleGame_OreToIngots(t *testing.T) {
	// 1 mine produces 10 ore/s; each ingot costs 5 ore → 2 ingots/s
	// python: 10 / 5 = 2.0
	near(t, eval(t, "10 / 5"), 2, "ore-to-ingot throughput")
}

func TestGame_IdleGame_InventoryCapacity(t *testing.T) {
	// 36 inventory slots × 64 stack size = 2304 max items
	near(t, eval(t, "36 * 64"), 2304, "inventory capacity")
}

func TestGame_IdleGame_SolarUpgrade(t *testing.T) {
	// Solar farm: 250 kW base, each of 4 upgrades multiplies by 1.5×
	// python: 250 * 1.5^4 = 1265.625
	near(t,
		eval(t, "250 * pow(1.5, 4)"),
		1265.625,
		"solar output after 4 upgrades")
}

func TestGame_IdleGame_PopulationGrowth(t *testing.T) {
	// Exponential growth: 100 settlers × 1.05^20 turns
	// python: 100 * 1.05^20 = 265.3297705144422
	near(t,
		eval(t, "100 * pow(1.05, 20)"),
		265.3297705144422,
		"population after 20 turns at 5% growth")
}

func TestGame_Trig_SinCos_Identity(t *testing.T) {
	// sin²(x) + cos²(x) = 1 for all x — verify floating point stays clean
	got := eval(t, "sin(pi / 4)")
	want := math.Sin(math.Pi / 4) // 0.7071067811865475
	near(t, got, want, "sin(pi/4)")

	gotC := eval(t, "cos(pi / 4)")
	wantC := math.Cos(math.Pi / 4)
	near(t, gotC, wantC, "cos(pi/4)")

	// sin² + cos² = 1 — use direct math, not eval, as the identity check
	near(t, got*got+gotC*gotC, 1.0, "sin²+cos² Pythagorean identity")
}

func TestGame_Trig_HypotAngle(t *testing.T) {
	// Screen-space diagonal for a 1920×1080 viewport.
	// python: hypot(1920, 1080) ≈ 2202.9071700822983
	near(t, eval(t, "hypot(1920, 1080)"), math.Hypot(1920, 1080), "1080p screen diagonal")
}

func TestGame_ResourceStorage_MaxTick(t *testing.T) {
	// Max gold that can accumulate: storage = 1 million, overflow resets.
	// Production: 125 gold/tick. Ticks until full: ceil(1e6 / 125) = 8000
	near(t, eval(t, "ceil(1000000 / 125)"), 8000, "ticks to fill gold storage")
}

func TestGame_EconomyScaling_Log2(t *testing.T) {
	// Upgrade cost doubles each level: level at which cost first exceeds 1 MB gold.
	// cost(n) = 100 * 2^n  →  n = log2(1e6 / 100) = log2(10000) ≈ 13.2877
	near(t, eval(t, "log2(1000000 / 100)"), math.Log2(10000), "upgrade level cap (log2)")
}

func TestGame_UserVar_ProductionChain(t *testing.T) {
	// Simulate storing intermediate results as user variables, as a player would.
	// mine_rate = 10, smelt_rate = 10/5 = 2, forge_rate = 2/3 ≈ 0.6667
	engine.StoreVar("mine_rate", 10)
	engine.StoreVar("smelt_cost", 5)
	engine.StoreVar("forge_cost", 3)

	near(t, eval(t, "mine_rate / smelt_cost"), 2, "ore → ingot rate")
	near(t, eval(t, "(mine_rate / smelt_cost) / forge_cost"), 2.0/3.0, "ingot → part rate")
}
