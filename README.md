# wrkr

> AI generated. Personal use. No guarantees.

Terminal calculator that knows units, remembers variables, and copies results to clipboard. Built because `python3 -c "print(128*1024*1024/4096)"` is too confusing and weird.

## Why

`bc` doesn't know what a megabyte is. Spotlight doesn't remember your block size. This does.

Filesystem/OS work means constant block count math, B-tree depth estimates, working set sizing. One-liner Python works but the context switch is annoying. This lives in the terminal and already has the units loaded.

## Pipeline

Raw input goes through these stages before hitting the evaluator:

```
1. separator strip        1_000_000 -> 1000000, 0b1011_1011 -> 0b10111011
2. prefix normalization   \xFF -> 0xFF, ob101 -> 0b101
3. naked base notation    FF hex -> 0xFF, 101 bin -> 0b101
4. base conversion        0x123 hex to bin -> bin(0x123)
5. unit conversion        50 mi to km -> (50 * (1609.344 / 1000))
6. implicit multiply      5 mb -> (5 * 1048576)
7. bitwise rewrite        a & b -> band(a, b), ~x -> bnot(x)
8. base translation       0xFF -> 255.000000
9. AST eval               expr-lang/expr, proper operator precedence
```

All values are float64 internally. Units are multipliers against a baseline (bytes for data, meters for distance).

## Units

Number adjacent to a unit name = implicit multiply.

```
data:      b  bit  kb  mb  gb  tb
distance:  m  km  cm  mm  mi  ft  in
```

```
5 mb
(512 * kb) + (256 * kb)
2 * tb / (4 * kb)
```

## Unit conversions

```
50 mi to km
1 gb to mb
1 mb to bits
30 cm to in
```

Result shows the target unit label and ignores the current output mode.

## Base input

Three equivalent ways to write a non-decimal literal:

| style | example | note |
|-------|---------|------|
| prefix | `0xFF` `0b1010` `0o17` | standard |
| natural | `FF hex` `101 bin` `17 octal` | suffix = base the digits are in |
| typo | `\xFF` `\b1010` `\o17` | backslash = 0 |

## Numeric separators

Use `_` inside any numeric literal for readability. Stripped before evaluation.

```
1_000_000          -> 1000000
0b1011_1011        -> 0b10111011  (187)
0xDEAD_BEEF        -> 3735928559
1_048_576 / 4_096  -> 256
```

Underscores in variable names (`dead_beef`, `block_size`) are never touched.

## Base conversion

Three equivalent ways to reformat a number:

| style | example | result |
|-------|---------|--------|
| function | `hex(255)` `bin(255)` `octal(255)` `dec(0xFF)` | `0xFF` `0b11111111` `0o377` `255` |
| to keyword | `255 to hex` `0xFF to bin` | `0xFF` `0b11111111` |
| annotated source | `0x123 hex to bin` `0b1010 bin to hex` | `0b100100011` `0xA` |

Annotated source: middle word = source base, `to X` = target. Useful when you already have a prefixed literal and just want it in a different base.

## Output modes

`mode <name>` to switch. Bare `hex`/`bin` evaluate as expressions, not mode switches.

| mode | terminal | clipboard |
|------|----------|-----------|
| `dec` | `1048576  [1 MB]` | `1048576` |
| `size` | `1 MB` | `1` |
| `bytes` | `1048576 B` | `1048576` |
| `bits` | `8388608 bits` | `8388608` |
| `hex` | `0x100000  [Hex]` | `0x100000` |
| `bin` | `0b100000000000000000000  [Bin]` | `0b100000000000000000000` |
| `oct` | `0o4000000  [Oct]` | `0o4000000` |

dec mode adds a size hint `[1 MB]` when the expression involves a data unit. Suppressed when units cancel out (e.g. `(256 * mb) / (4 * gb) * 1000` = 62.5, units cancelled, result is dimensionless).

## Two's complement

`hex`/`bin`/`oct` show signed output: `-5` -> `-0x5`. For the actual bit pattern use the width-specific forms.

| width | bin | hex | oct |
|-------|-----|-----|-----|
| 8 | `bin8` | `hex8` | `oct8` |
| 16 | `bin16` | `hex16` | `oct16` |
| 32 | `bin32` | `hex32` | `oct32` |
| 64 | `bin64` | `hex64` | `oct64` |
| 128 | `bin128` | `hex128` | — |
| 256 | `bin256` | — | — |
| 512 | `bin512` | — | — |

Available as both modes and functions:

```
bin32(-5)           -> 0b11111111111111111111111111111011
hex32(-5)           -> 0xFFFFFFFB
hex64(-1)           -> 0xFFFFFFFFFFFFFFFF
hex128(-1)          -> 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF
bin8(300)           -> 0b00101100    (truncated, low 8 bits, matches hardware)

mode bin32
4 - 8               -> 0b11111111111111111111111111111100  [Bin32]
```

Positive values are zero-padded to the full width. Values outside the range truncate to the low N bits (same as a C cast). Composes with `_`:

```
4 - 8
hex32(_)            -> 0xFFFFFFFC
```

## Bitwise operators

Standard C bitwise operators, with standard C precedence.

| operator | name | example | result |
|----------|------|---------|--------|
| `a & b` | AND | `0b1100 & 0b1010` | `8` |
| `a \| b` | OR | `0b1100 \| 0b1010` | `14` |
| `a ^ b` | XOR | `0b1100 ^ 0b1010` | `6` |
| `~a` | NOT | `~0` | `-1` |
| `a << n` | left shift | `1 << 8` | `256` |
| `a >> n` | right shift | `256 >> 4` | `16` |

Precedence (high -> low): `~`, `* / + -`, `<< >>`, `&`, `^`, `|` — matching C.  
`>>` is arithmetic (sign-preserving). `&&` and `||` (logical) are passed through unchanged.

```
0xFF & 0x0F                  -> 15       low nibble
0xFF00 | 0x00FF              -> 65535    combine fields
0xAB ^ 0xCD                  -> 102      XOR checksum
~0b00001111                  -> -16      flip bits (int64 two's complement)
1 << 5                       -> 32       set bit 5
(0xAB >> 4) & 0xF            -> 10       extract high nibble of 0xAB
0x12345 & ~(4096-1)          -> 73728    page-align to 4 KB boundary
(0b10110100 >> 3) & 7        -> 6        extract bits [5:3]
7 & (2 | 4)                  -> 6        check collidable|active flags set
```

Combines with type mode and format mode:

```
mode hex
~0 & 0xFF                    -> 0xFF  [Hex]
(0xDEAD ^ key) to u16        -> wraps XOR result to u16 range
```

## Two modes: format and type

The calculator has two orthogonal mode settings:

| setting | what it does | examples |
|---------|-------------|---------|
| **format mode** (`mode`) | how results are displayed | `dec` `hex` `bin` `oct` `size` `bytes` `bits` `bin32` … |
| **type mode** (`type`) | integer semantics applied to results | `auto` `u8` `s8` `u16` `s16` `u32` `s32` `u64` `s64` `u128` `s128` |

They are independent: `mode hex` + `type u32` shows 32-bit unsigned results in hex.

**Format mode** was described above. **Type mode** is covered next.

## Type mode

`type auto` is the default. Pure float64 math — existing users are completely unaffected.

Setting a type mode wraps every numeric result to that integer range:

```
type u8

200 + 50    -> 250  [u8]
200 + 100   -> 44  [u8 ovf]     overflow detected, result wrapped
```

Signed types reinterpret the bit pattern:

```
type s8

-5 + 10     -> 5  [s8]
127 + 1     -> -128  [s8 ovf]   wraps from max to min
```

Switch back to pure math: `type auto` (or `type off`).

### Cast functions

`u8(x)` through `u128(x)` (unsigned), `s8(x)` through `s128(x)` (signed). Return float64 so they compose with arithmetic.

```
u8(246)              -> 246        stays in range
u8(256)              -> 0          wraps
u8(300)              -> 44         300 mod 256

s8(246)              -> -10        bit pattern reinterpret (246 - 256)
s8(255)              -> -1
s8(128)              -> -128       one past max wraps to min

s8(-5)               -> -5         in range, unchanged
s8(-129)             -> 127        underflow wraps to max

u8(200) + u8(100)    -> 300        functions return float64, no implicit wrap
u8(u8(200) + u8(100)) -> 44        explicit second cast wraps the sum
```

### `to` keyword with type names

```
246 to u8            -> 246
246 to s8            -> -10
0b11110110 to s8     -> -10       binary literal reinterpreted as signed
0xFFFFFFFB to s32    -> -5        32-bit hex to signed
-1 to u16            -> 65535     -1 as unsigned 16-bit
_ to s8              applies type cast to last result
```

The prompt shows `[mode/type]` when a type is active:

```
[dec/u8]
> 200 + 100
44  [u8 ovf]
```

### Type mode is persisted

Both `mode` and `type` are saved to `~/.wrkr_config.json` and restored on next launch.

## Settings

`setting` alone shows a table of all current settings.

```
setting clipboard on|off          toggle clipboard copy (default: on)

setting grouping on|off           _ separators in output + clipboard
setting grouping display on|off   _ separators in terminal only
setting grouping clipboard on|off _ separators in clipboard only

setting prefix on|off             0x/0b/0o prefix in output + clipboard
setting prefix display on|off     prefix in terminal only
setting prefix clipboard on|off   prefix in clipboard only
```

Defaults: grouping display **on**, grouping clipboard **off**, prefix display **on**, prefix clipboard **off** (raw hex/bin values without `0x`/`0b` on clipboard).

With defaults, `0xFF` in hex mode shows `0xFF  [Hex]` on screen and copies `FF` to clipboard.

## Variables

Saved to `~/.wrkr_vars.json`. Autoload preference in `~/.wrkr_config.json`.
On next launch: prompted to load, skip, or delete.
Choosing load sets autoload; subsequent launches restore silently.

```
block = 4096
page  = 4 * kb
journal = 128 * mb

journal / block
vars              list
del block         remove
```

`_` holds the last numeric result.

```
100 tb / (2 mb / 5)
_ / 60 / 60 / 24
```

## Math

```
sin  cos  tan  hypot  sqrt  abs  log  log2  log10  pow  round  floor  ceil  min  max  pi
```

## Autocorrect

Levenshtein match on unknown tokens. Suggestion only shown if the corrected expression compiles and produces a non-function result. Silent otherwise.

## Editor mode

`:e` opens `$EDITOR` (falls back to `vi`) with a temp file. Each non-empty line
runs as a separate command in sequence. Variables set on one line are available
to the next. Lines starting with `#` are ignored.

```
# in editor:
block = 4096
journal = 128 * mb
journal / block
```

Reopening `:e` pre-populates the file with your last session's content.
All lines are added to history individually.

## Debug

```
debug <expr>
```

Shows only the pipeline stages that changed, plus an `expanded` step with all
unit names and variables substituted for their numeric values (what the
evaluator actually computes).

```
debug (3 * tb) / 3 mb * 1000

  input        (3 * tb) / 3 mb * 1000
  multiply  -> (3 * tb) / (3 * mb) * 1000
  expanded  -> (3 * 1099511627776) / (3 * 1048576) * 1000

  result       1048576000
```

## Install

Requires Go 1.21+: https://go.dev/dl/

```
go install github.com/Ekansh38/wrkr@latest
```

If not found after install:

```
export PATH="$HOME/go/bin:$PATH"
source ~/.zshrc
```

To update, re-run the same command. If the proxy has a stale cache:

```
GOPROXY=direct go install github.com/Ekansh38/wrkr@latest
```

Build from source:

```
git clone git@github.com:Ekansh38/wrkr.git
cd wrkr
go build -o wrkr .
sudo mv wrkr /usr/local/bin/wrkr
```

## Precision

IEEE 754 float64. Output capped at 12 decimal places to suppress noise (`1.60934400000000005` becomes `1.609344`). Not suitable for >12 significant decimal places.

## Help

```
help math
help systems
help units
help modes
help types
help vars
help settings
help all
```

---

## Drill mode

Practice binary/hex/decimal conversions interactively. Stats persist to `~/.wrkr_drill.json` across sessions.

```
drill
```

**Games:**

| # | Game | Description |
|---|------|-------------|
| 1 | convert | Standard Q&A — type the conversion |
| 2 | flashcard | Answer flashes for 1.5s, then you recall from memory |
| 3 | vibes | Multiple choice — pick the closest decimal value (a/b/c) |
| 4 | sprint | 60-second timed blitz |
| 5 | bit scan | Given a hex value, which bit position is set? (0 = LSB) |

**Modes** (for games 1–4):

| # | Mode | Range | Purpose |
|---|------|-------|---------|
| 1 | nibble | 0–15 | Master the 16 core hex facts first |
| 2 | powers | 2^0–2^15 | Essential for fast decomposition |
| 3 | byte | 0–255 | Full 8-bit range |
| 4 | random | mix | All three combined |

**Convert to** (for games 1, 2, 4): `h` hex · `b` bin · `d` dec

**Accepted answer formats:**

- hex: `0xF` or bare with a-f letter (`F`, `b4`)
- bin: `0b1010` or bare 0s/1s (`1010`)
- dec: plain digits (`15`, `255`)
- bit: plain number, 0 = LSB (`7`)

Typing the wrong base is marked wrong — the point is to actually do the conversion.

**Recommended progression:** nibble → hex until instant, then powers → bin, then byte → hex.

---

## Use cases

A sample of real workflows to try. Each block is independent.

### Filesystem math

```
# How many 4 KB blocks fit on a 2 TB drive?
2 tb / (4 * kb)                 -> 536870912

# Journal size in blocks (128 MB journal, 4 KB blocks)
block = 4096
journal = 128 * mb
journal / block                 -> 32768

# B-tree depth for 1 billion records, 4 KB node, 8-byte keys
floor(log(1e9) / log(4096 / 8))  -> 3

# Inode table: 1 inode per 16 KB on a 1 TB filesystem
1 tb / (16 * kb)                -> 67108864
```

### Bitwise and registers

```
# Extract the high nibble of 0xAB
(0xAB >> 4) & 0xF               -> 10

# Pack two bytes into a 16-bit value
(0xAB << 8) | 0xCD              -> 0xABCD (in hex mode)

# Page-align an address to a 4 KB boundary
addr = 0x1234_5678
addr & ~(4096 - 1)              -> 0x12345000

# Clear bit 3, set bit 7
reg = 0b0001_1010
(reg & ~(1 << 3)) | (1 << 7)   -> 0b10010010

# Extract bits [5:3] from a value
val = 0b10110100
(val >> 3) & 7                  -> 6

# Check if a number is a power of 2 (result 0 = yes)
x = 256
x & (x - 1)                    -> 0
```

### Signed/unsigned interpretation

```
# What does 0xFFFB mean as a signed 16-bit integer?
0xFFFB to s16                   -> -5

# ADC returns 0xF6. Interpret as signed temperature.
0xF6 to s8                      -> -10

# -1 as an unsigned 16-bit port number
-1 to u16                       -> 65535

# Full 32-bit two's complement view of -129
type s32
mode bin32
-129                            -> 0b11111111111111111111111101111111  [Bin32]  [s32]
```

### Base conversion workflows

```
# Convert a hex colour to decimal channels
r = 0xDE
g = 0xAD
b = 0xBE
r                               -> 222
g                               -> 173

# Decimal to binary with grouping
255 to bin                      -> 0b1111_1111

# See IPv4 subnet mask in binary
0xFFFFFF00 to bin32             -> 0b11111111_11111111_11111111_00000000

# How many hosts in a /26?
pow(2, 32 - 26) - 2             -> 62
```

### Chained calculations with `_`

```
# Transfer time for 1 GB at 100 Mbps — in hours
1 gb * 8                        # bits
_ / (100 * 1e6)                 # seconds at 100 Mbps
_ / 3600                        -> ~0.02388 hours (~86 seconds)

# Cost of a 512 MB allocation at $0.023/GB-month
512 * mb / gb * 0.023           -> 0.01175 ($/month)
```

### Bitwise with format functions

```
# bin64 and hex64 compose directly with bitwise operators
bin64(-1) & bin64(0xFF)         -> 255
hex64(0xDEAD_BEEF) | hex64(0x0000_1111)  -> 0xDEADBFFF (in hex mode)

# Strip/extract using width-specific output and arithmetic
mode hex32
~0 & ~0xFF                      -> 0xFFFFFF00   (all bits except low byte)
(0xABCD_1234 >> 16) & 0xFFFF    -> 0xABCD       (high word)
```

### Type mode: simulating C integer overflow

```
type u8
200 + 100                       -> 44  [u8 ovf]    (wraps at 256)

type s8
127 + 1                         -> -128  [s8 ovf]  (wraps to min)

# Safe: cast only the value you want to clamp
u8(200 + 100)                   -> 44   (explicit, no global type mode)
```

---

## Limitations

### Float64 precision
All values are IEEE 754 float64 (~15–16 significant decimal digits). Integers above ~2^53 (9 quadrillion) lose precision.

**Workaround:** stay within int64 range for bit manipulation; use the `to s64`/`to u64` casts to clamp.

---

### Format functions in arithmetic lose their display hint
`hex(a) + hex(b)` evaluates correctly (values are extracted, arithmetic runs) but the result displays in the current output mode — the hex wrapping is stripped.

**Workaround:** wrap the whole expression: `hex(a + b)`. Or switch mode: `mode hex`, then `a + b`.

---

### `to format` only works on simple left-hand values
`(a + b) to hex` does not work. `to` matches a single number, identifier, or function call on the left.

**Workaround:** `hex(a + b)` or `_ to hex` after storing the result.

---

### Bitwise operates on int64
`~`, `&`, `|`, `^`, `<<`, `>>` truncate values to 64-bit signed integers. Numbers outside [-2^63, 2^63−1] clamp to the boundary.

**Workaround:** `u128`/`s128` cast functions for extra-wide arithmetic — though bitwise on those isn't supported, just value range.

---

### Width-specific format functions in bitwise are 64-bit-context values
`hex16(-1)` = `"0xFFFF"` = 65535 (positive) in a 64-bit expression. Only `hex64`/`bin64` produce strings that round-trip to -1 via two's complement.

**Workaround:** use `hex64`/`bin64` when you need signed 64-bit two's complement in bitwise expressions. For narrower math, use the cast functions: `s16(-1) & x`.

---

### Fractional parts are truncated in base output
`hex(1.5)` = `0x1`. Base format functions (`hex`, `bin`, `oct` and width variants) truncate to integer.

**Workaround:** `round(x)` or `floor(x)` before formatting if the fractional part matters.

---

### No multi-line expressions in the prompt
The REPL evaluates one expression per line.

**Workaround:** `:e` opens `$EDITOR` with a temp file — write multiple lines, each runs in sequence, variables persist across lines.
