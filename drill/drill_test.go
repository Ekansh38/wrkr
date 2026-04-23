package drill

import (
	"math/rand"
	"testing"
)

// ── parseAnswer ───────────────────────────────────────────────────────────────

func TestParseAnswer(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{
		// decimal
		{"0", 0, true},
		{"15", 15, true},
		{"255", 255, true},
		{"1024", 1024, true},
		// hex with prefix
		{"0xF", 15, true},
		{"0XF", 15, true},
		{"0xff", 255, true},
		{"0xB4", 180, true},
		// bare hex (has a-f char)
		{"F", 15, true},
		{"f", 15, true},
		{"b4", 180, true},
		{"FF", 255, true},
		// binary with prefix
		{"0b1111", 15, true},
		{"0B10110100", 180, true},
		// ambiguous bare decimal digits — treated as decimal, not hex
		{"10", 10, true},
		{"9", 9, true},
		// empty / junk
		{"", 0, false},
		{"xyz", 0, false},
	}
	for _, tc := range cases {
		got, ok := parseAnswer(tc.in)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("parseAnswer(%q) = (%d, %v), want (%d, %v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

// ── Question.Check ────────────────────────────────────────────────────────────

func TestQuestionCheck(t *testing.T) {
	// hex target: value 15
	q := Question{Value: 15, From: "0b1111", ToBase: "hex"}
	for _, ans := range []string{"0xF", "0xf", "F", "f", "0XF"} {
		if !q.Check(ans) {
			t.Errorf("Check(%q) should be correct for value 15 → hex", ans)
		}
	}
	for _, ans := range []string{"0xe", "14", "0b1111"} {
		if q.Check(ans) {
			t.Errorf("Check(%q) should be wrong for value 15 → hex", ans)
		}
	}

	// bin target: value 10
	q2 := Question{Value: 10, From: "0xA", ToBase: "bin"}
	for _, ans := range []string{"0b1010", "0B1010", "1010"} {
		if !q2.Check(ans) {
			t.Errorf("Check(%q) should be correct for value 10 → bin", ans)
		}
	}
	// bare bin with wrong value
	if q2.Check("1111") {
		t.Error("Check(1111) should be wrong for value 10 → bin")
	}

	// dec target: value 255
	q3 := Question{Value: 255, From: "0xFF", ToBase: "dec"}
	if !q3.Check("255") {
		t.Error("Check(255) should be correct for value 255 → dec")
	}
	if q3.Check("254") {
		t.Error("Check(254) should be wrong for value 255 → dec")
	}
}

// ── CorrectAnswer ─────────────────────────────────────────────────────────────

func TestCorrectAnswer(t *testing.T) {
	cases := []struct {
		q    Question
		want string
	}{
		{Question{Value: 15, ToBase: "hex", Mode: ModeNibble}, "0xF"},
		{Question{Value: 255, ToBase: "hex", Mode: ModeByte}, "0xFF"},
		{Question{Value: 5, ToBase: "bin", Mode: ModeNibble}, "0b0101"},     // padded to 4
		{Question{Value: 10, ToBase: "bin", Mode: ModeNibble}, "0b1010"},    // already 4 bits
		{Question{Value: 180, ToBase: "bin", Mode: ModeByte}, "0b10110100"}, // already 8 bits
		{Question{Value: 3, ToBase: "bin", Mode: ModeByte}, "0b00000011"},   // padded to 8
		{Question{Value: 1024, ToBase: "bin", Mode: ModePowers}, "0b10000000000"}, // no padding
		{Question{Value: 255, ToBase: "dec", Mode: ModeByte}, "255"},
		{Question{Value: 1024, ToBase: "dec", Mode: ModePowers}, "1024"},
	}
	for _, tc := range cases {
		got := tc.q.CorrectAnswer()
		if got != tc.want {
			t.Errorf("CorrectAnswer(%v) = %q, want %q", tc.q, got, tc.want)
		}
	}
}

// ── Generate ──────────────────────────────────────────────────────────────────

func TestGenerateNibbleRange(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 200; i++ {
		q := Generate(ModeNibble, ConvToHex, rng)
		if q.Value < 0 || q.Value > 15 {
			t.Fatalf("nibble value out of range: %d", q.Value)
		}
	}
}

func TestGeneratePowersOfTwo(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 200; i++ {
		q := Generate(ModePowers, ConvToBin, rng)
		v := q.Value
		if v <= 0 || (v&(v-1)) != 0 {
			t.Fatalf("powers value not a power of 2: %d", v)
		}
		if v > (1 << 15) {
			t.Fatalf("powers value too large: %d (max 2^15=%d)", v, 1<<15)
		}
	}
}

func TestGenerateByteRange(t *testing.T) {
	rng := rand.New(rand.NewSource(13))
	for i := 0; i < 200; i++ {
		q := Generate(ModeByte, ConvToDec, rng)
		if q.Value < 0 || q.Value > 255 {
			t.Fatalf("byte value out of range: %d", q.Value)
		}
	}
}

func TestGenerateRandomMix(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	sawNibble, sawPow, sawByte := false, false, false
	for i := 0; i < 1000; i++ {
		q := Generate(ModeRandom, ConvToHex, rng)
		if q.Value < 0 || q.Value > 65535 {
			t.Fatalf("random value out of range: %d", q.Value)
		}
		v := q.Value
		if v <= 15 {
			sawNibble = true
		}
		if v > 15 && v <= 255 && (v&(v-1)) != 0 {
			sawByte = true
		}
		if v > 255 {
			sawPow = true
		}
	}
	if !sawNibble || !sawByte || !sawPow {
		t.Errorf("random mode not covering all sub-modes: nibble=%v byte=%v pow=%v",
			sawNibble, sawByte, sawPow)
	}
}

// ── round-trip: generate → correct answer → check ────────────────────────────

func TestRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(55))
	modes := []Mode{ModeNibble, ModePowers, ModeByte, ModeRandom}
	convs := []Conv{ConvToHex, ConvToBin, ConvToDec}
	for _, m := range modes {
		for _, c := range convs {
			for i := 0; i < 50; i++ {
				q := Generate(m, c, rng)
				ans := q.CorrectAnswer()
				if !q.Check(ans) {
					t.Errorf("round-trip failed: mode=%d conv=%d value=%d from=%q answer=%q",
						m, c, q.Value, q.From, ans)
				}
			}
		}
	}
}
