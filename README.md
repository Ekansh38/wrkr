# wrkr

> README and code are AI generated. Built for personal use. No guarantees. Use at your own risk.

A CLI calculator built for engineers who are tired of reaching for Python or a spreadsheet to do basic math while working. It understands units, remembers variables between expressions, and copies the right thing to your clipboard automatically.

---

## Why I built this

Standard calculators don't know what a megabyte is. They don't remember that your block size is 4096. They make you convert units manually. This one does not.

I work on filesystems and OS stuff. I constantly need to know how many blocks fit on a disk, how deep a B-tree will be, how much RAM a working set needs. I was tired of opening Python just to type one line. So I built a calculator that lives in the terminal and already knows the units I care about.

---

## Who it is for

- Systems and kernel engineers doing memory, storage, and bandwidth math
- Filesystem engineers working through block counts, inode ratios, RAID geometry
- Gamedevs calculating trajectories, distances, upgrade curves, production chains
- Any developer who does quick back-of-envelope math in the terminal

---

## How it works

Every input goes through a preprocessing pipeline before it hits the evaluator:

1. Typo fixes (`0b`, `0x`, `0o` prefix normalization)
2. Natural language base notation (`FF hex` becomes `0xFF`)
3. Base conversion detection (`0x123 hex to bin` becomes `bin(0x123)`)
4. Unit conversion expansion (`50 mi to km` becomes the actual math)
5. Implicit multiplication (`5 mb` becomes `(5 * 1048576)`)
6. Base literal translation (`0xFF` becomes `255.0` so the evaluator can handle it)
7. AST evaluation via expr-lang/expr (proper BODMAS, no regex math)

The pipeline runs on every input. The evaluator gets clean float64 expressions every time.

Units are stored as multipliers relative to a baseline (bytes for data, meters for distance). All math stays in that baseline. Output modes handle the presentation layer separately from the calculation layer.

---

## Features

### Units

Write a number next to a unit name and they multiply automatically.

Data: `b`, `bit`/`bits`, `kb`, `mb`, `gb`, `tb`
Distance: `m`, `km`, `cm`, `mm`, `mi`, `ft`, `in`

```
5 mb
(512 * kb) + (256 * kb)
2 * tb / (4 * kb)
hypot(3, 4)
```

### Unit conversions

```
50 mi to km
100 ft to m
1 gb to mb
1 mb to bits
30 cm to in
```

The result is always shown with the target unit label and bypasses your current output mode — so `1 gb to mb` shows `1024 MB` even if you're in hex mode.

### Base literals (input)

Three ways to write a number in a non-decimal base:

| Style | Example | Means |
|-------|---------|-------|
| Prefix | `0xFF`, `0b1010`, `0o17` | standard notation |
| Natural language | `FF hex`, `101 bin`, `17 octal` | suffix names the base the digits are in |
| Typo shorthand | `\xFF`, `\b1010`, `\o17` | backslash works as `0` |

### Base conversions (output)

Three ways to convert a number to a different base — pick whichever reads most naturally:

| Style | Example | Result |
|-------|---------|--------|
| Function call | `hex(255)` | `0xFF` |
| `to` keyword | `255 to bin` | `0b11111111` |
| Annotated source | `0x123 hex to bin` | `0b100100011` |

The annotated source form (`0x123 hex to bin`) is useful when you already have a prefixed literal and want to reformat it. The middle word tells the calculator what base you're coming from; `to X` says where to go. All three forms produce the same result.

```
hex(255)           →  0xFF
bin(255)           →  0b11111111
octal(255)         →  0o377
dec(0xFF)          →  255
255 to hex         →  0xFF
0xFF to bin        →  0b11111111
0x123 hex to bin   →  0b100100011
0b1010 bin to hex  →  0xA
```

### Output modes

Switch with `mode <name>`. Only the explicit `mode` prefix switches modes — bare words like `hex` or `bin` evaluate as expressions, not mode switches.

| mode | terminal | clipboard |
|------|----------|-----------|
| `dec` | `1048576  [1 MB]` | `1048576` |
| `size` | `1 MB` | `1` |
| `bytes` | `1048576 B` | `1048576` |
| `bits` | `8388608 bits` | `8388608` |
| `hex` | `0x100000  [Hex]` | `0x100000` |
| `bin` | `0b100000000000000000000  [Bin]` | `0b100000000000000000000` |
| `oct` | `0o4000000  [Oct]` | `0o4000000` |

The **Smart Hint** (dec mode) adds the `[1 MB]` bracket automatically when your expression involves a data size unit. If units cancel each other out (e.g. `(256 * mb) / (4 * gb) * 1000` = 62.5 — units cancelled, result is not bytes) the hint stays silent to avoid misleading you.

### User variables

Variables persist for the life of the process.

```
block = 4096
page  = 4 * kb
journal = 128 * mb

journal / block       use them in expressions
vars                  list all variables
del block             remove a variable
```

### Math functions

```
sin, cos, tan, hypot, sqrt, abs, log, log2, log10, pow, round, floor, ceil, pi
```

### Autocorrect

If you mistype a unit or function name, it finds the closest match via Levenshtein distance. Before asking "did you mean X?", it silently compiles the suggested fix. If the fix produces garbage math, it says nothing.

---

## Install

Requires Go 1.21+. Get it at https://go.dev/dl/

**Option 1: go install (recommended)**

```
go install github.com/Ekansh38/wrkr@latest
wrkr
```

If `wrkr` is not found, add Go's bin directory to your PATH:

```
export PATH="$HOME/go/bin:$PATH"
```

Then reload: `source ~/.zshrc`

**Updating**

`go install` does not auto-update. Re-run the same command to get the latest version:

```
go install github.com/Ekansh38/wrkr@latest
```

If the module proxy has a stale cached version, bypass it:

```
GOPROXY=direct go install github.com/Ekansh38/wrkr@latest
```

**Option 2: build from source**

```
git clone git@github.com:Ekansh38/wrkr.git
cd wrkr
go build -o wrkr .
sudo mv wrkr /usr/local/bin/wrkr
```

---

## Float precision

All values are IEEE 754 float64 (~15–16 significant digits). Output is capped at 12 decimal places to suppress representation noise. Without this, `1 mi to km` would show `1.60934400000000005` instead of `1.609344`. If you need more than 12 decimal places, this tool is the wrong choice.

---

## Quick reference

```
help math       trig and math functions
help systems    data sizes and base literals
help units      unit conversion syntax
help modes      output mode table
help vars       variable assignment
help all        everything
```
