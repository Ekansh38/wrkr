# wrkr

A context-aware CLI calculator built for engineers.  
It understands units, remembers your variables, auto-scales output, and
copies the right thing to your clipboard without asking.

---

## Install

```sh
go build -o wrkr .
./wrkr
```

Or put the binary on your `$PATH` for global access.

---

## Architecture overview

```
wrkr/
├── main.go            entry point
├── engine/
│   ├── units.go       unit definitions + CalcEnv initialisation
│   ├── format.go      result formatting and output modes
│   ├── vars.go        user variable store + GetMergedEnv
│   ├── pipeline.go    input transformation pipeline (AST builder)
│   └── correct.go     Levenshtein autocorrect + sanitisation
└── repl/
    └── repl.go        interactive REPL loop + help system
```

---

## Feature guide

### 1. Dimensional Data Engine

Every unit resolves to a **baseline** value before any math runs:

| Unit | Baseline (bytes) |
|------|-----------------|
| `bit` / `bits` | 0.125 |
| `b` / `byte` | 1 |
| `kb` | 1 024 |
| `mb` | 1 048 576 |
| `gb` | 1 073 741 824 |
| `tb` | 1 099 511 627 776 |

Numbers written immediately before a unit are automatically wrapped in a
multiplication — the **implicit multiplication** transform:

```
> 5 mb
5242880  [5 MB]

> 1.5 gb
1610612736  [1.5 GB]

> 512 kb + 256 kb
786432  [768 KB]
```

---

### 2. Output modes

Switch with `mode <name>` (or just type the mode name directly):

| Mode | Terminal output | Clipboard |
|------|----------------|-----------|
| `dec` | `1048576  [1 MB]` (Smart Hint when a size unit is present) | `1048576` |
| `size` | `1 MB` | `1` |
| `hex` | `0x100000  [Hex]` | `0x100000` |
| `bin` | `0b100000000000000000000  [Bin]` | `0b100000000000000000000` |
| `oct` | `0o4000000  [Oct]` | `0o4000000` |

The **Smart Hint** in `dec` mode activates automatically when your expression
contains a data-size unit and the result is ≥ 1 KB.  The terminal shows the
human-readable form in brackets; the clipboard always gets the raw number.

```
> 1 mb              # terminal: 1048576  [1 MB]   clipboard: 1048576
> mode size
> 1 mb              # terminal: 1 MB              clipboard: 1
> mode hex
> 255               # terminal: 0xFF  [Hex]        clipboard: 0xFF
> mode dec
```

---

### 3. User variables (persistent memory)

Variables live for the lifetime of the process.

```
> block = 4096
block = 4096

> page = 4 * kb
page = 4096

> (512 * mb) / block
131072

> cache = 256 * mb
cache = 268435456

> cache / block
65536

> vars
  block        = 4096
  page         = 4096
  cache        = 268435456
```

---

### 4. Dry-run autocorrect

Before offering a typo suggestion, wrkr silently compiles the proposed fix.
If the result is not a valid expression, the suggestion is discarded.

```
> 5 mbb           # "mbb" is close to "mb" — dry-run confirms it compiles
Did you mean: 5 mb? (y/n): y
5242880  [5 MB]

> 5 xyzmb         # too far from anything — no prompt, fall through to parse error
Error: Could not parse expression.
```

---

### 5. BODMAS / AST evaluation

Expressions are parsed into an AST by [`expr-lang/expr`](https://github.com/expr-lang/expr),
so operator precedence is always correct:

```
> 10 + 2 * 5 mb    →  10 + (2 * 5242880)  =  10485770
> (10 + 2) * 5 mb  →  (12 * 5242880)      =  62914560
```

---

### 6. Base-N arithmetic + Base-N Protection

```
> 0xFF + 1
256

> 0b1010 * 2
20

> 0xFF to bin
0b11111111

> 255 to hex
0xFF

> 17 octal       # natural-language base notation
0o17 → already stored as octal
```

**Protection:** the implicit-multiplication step recognises `0b`, `0x`, and `0o`
as protected prefixes and never splits `0b101` into `(0 * b)101`.

---

### 7. Unit conversions

```
> 50 mi to km
80.4672

> 100 ft to m
30.48

> 2 gb to mb
2048
```

---

### 8. Math functions

```
> sqrt(144)
12

> hypot(3, 4)
5

> sin(pi / 2)
1

> log2(1024)
10

> ceil(3.2)
4
```

---

## Things to try

```
# Systems programming
block = 4096
stripe = 64 * kb
(1 * gb) / block          # how many 4 KB blocks in 1 GB?
(1 * gb) / stripe         # how many 64 KB stripes?

# Mode switching
mode hex
255 + 1                   # 0x100  [Hex]
mode bin
0xFF                      # 0b11111111  [Bin]
mode size
4 * gb                    # 4 GB  (clipboard: 4)
mode dec

# Trig / geometry
hypot(1920, 1080)         # diagonal of a 1080p screen in pixels
sin(pi / 6)               # 0.5

# Unit conversions
100 ft to m
26.2 mi to km             # marathon distance

# Base conversions
0b11001100 to hex
0xDEAD to bin
255 to hex

# Autocorrect
5 mbs                     # → "Did you mean: 5 mb?"
```

---

## Reference

```
> help math       math functions and trig
> help systems    data sizes and base literals
> help units      unit conversion syntax
> help modes      output mode table
> help vars       variable assignment and listing
> help all        everything
```
