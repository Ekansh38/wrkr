# wrkr

> AI generated. Personal use. No guarantees.

Terminal calculator that knows units, remembers variables, and copies results to clipboard. Built because `python3 -c "print(128*1024*1024/4096)"` is too confusing and weird.

## Why

`bc` doesn't know what a megabyte is. Spotlight doesn't remember your block size. This does.

Filesystem/OS work means constant block count math, B-tree depth estimates, working set sizing. One-liner Python works but the context switch is annoying. This lives in the terminal and already has the units loaded.

## Pipeline

Raw input goes through these stages before hitting the evaluator:

```
1. prefix normalization   \xFF -> 0xFF, ob101 -> 0b101
2. naked base notation    FF hex -> 0xFF, 101 bin -> 0b101
3. base conversion        0x123 hex to bin -> bin(0x123)
4. unit conversion        50 mi to km -> (50 * (1609.344 / 1000))
5. implicit multiply      5 mb -> (5 * 1048576)
6. bitwise rewrite        a & b -> band(a, b), ~x -> bnot(x)
7. base translation       0xFF -> 255.000000
8. AST eval               expr-lang/expr, proper operator precedence
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

```
setting clipboard on     copy results to clipboard (default)
setting clipboard off    disable clipboard copy
setting clipboard        query current value
```

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
sin  cos  tan  hypot  sqrt  abs  log  log2  log10  pow  round  floor  ceil  pi
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
