# wrkr

> AI generated. Personal use. No guarantees.

Terminal calculator that knows units, remembers variables, and copies results to clipboard. Built because `python3 -c "print(128*1024*1024/4096)"` is too many keystrokes.

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
6. base translation       0xFF -> 255.000000
7. AST eval               expr-lang/expr, proper operator precedence
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

```
prefix:    0xFF    0b1010    0o17
natural:   FF hex  101 bin   17 octal   (suffix = base the digits are in)
typo:      \xFF    \b1010    \o17       (backslash = 0)
```

## Base conversion

Three equivalent ways to reformat a number:

```
hex(255)           -> 0xFF
255 to hex         -> 0xFF
0xFF to bin        -> 0b11111111
0x123 hex to bin   -> 0b100100011    (middle word = source base, to X = target)
0b1010 bin to hex  -> 0xA
dec(0xFF)          -> 255
```

## Output modes

`mode <name>` to switch. Bare `hex`/`bin` evaluate as expressions, not mode switches.

```
mode     terminal                          clipboard
dec      1048576  [1 MB]                   1048576
size     1 MB                              1
bytes    1048576 B                         1048576
bits     8388608 bits                      8388608
hex      0x100000  [Hex]                   0x100000
bin      0b100000000000000000000  [Bin]    0b100000000000000000000
oct      0o4000000  [Oct]                  0o4000000
```

dec mode adds a size hint `[1 MB]` when the expression involves a data unit. Suppressed when units cancel out (e.g. `(256 * mb) / (4 * gb) * 1000` = 62.5, units cancelled, result is dimensionless).

## Variables

Saved to `~/.wrkr_vars.json`. Offered for reload on next launch.

```
block = 4096
page  = 4 * kb
journal = 128 * mb

journal / block
vars              list
del block         remove
```

## Math

```
sin  cos  tan  hypot  sqrt  abs  log  log2  log10  pow  round  floor  ceil  pi
```

## Autocorrect

Levenshtein match on unknown tokens. Suggestion only shown if the corrected expression compiles cleanly. Silent otherwise.

## Debug

```
debug <expr>    show every pipeline stage and final result
```

Useful when a conversion doesn't produce what you expect.

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
help vars
help all
```
