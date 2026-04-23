# wrkr

> README and code are vibe-coded. Built for personal use. No guarantees. Use at your own risk.

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

1. Typo fixes (0b, 0x, 0o prefix normalization)
2. Natural language base notation ("101 bin" becomes 0b101)
3. Unit conversion expansion ("50 mi to km" becomes the actual math)
4. Implicit multiplication ("5 mb" becomes "(5 * 1048576)")
5. Base literal translation (0xFF becomes 255.0 so the evaluator can handle it)
6. AST evaluation via expr-lang/expr (proper BODMAS, no regex math)

The pipeline runs on every input. The evaluator gets clean float64 expressions every time.

Units are stored as multipliers relative to a baseline (bytes for data, meters for distance). All math stays in that baseline. Output modes handle the presentation layer separately from the calculation layer.

---

## Features

**Units**

Data: b, bit, kb, mb, gb, tb
Distance: m, km, cm, mm, mi, ft, in

Write numbers next to units and they multiply automatically.

```
5 mb
(512 * kb) + (256 * kb)
2 * tb / (4 * kb)
```

**Unit conversions**

```
50 mi to km
100 ft to m
1 gb to mb
1 mb to bits
30 cm to in
```

**Output modes**

Switch with `mode <name>`. Only the explicit `mode` prefix switches modes — bare words like `hex` or `bits` evaluate as expressions.

- dec: raw number with a Smart Hint bracket when a size unit is involved
- size: human-readable auto-scaling (1 GB, 512 MB)
- bytes: raw number with B label
- bits: converts result to bits automatically
- hex: 0xFF format
- bin: 0b11111111 format
- oct: 0o377 format

The terminal output and clipboard output are different by design. Size mode shows "1 GB" on screen but copies "1" to clipboard. Dec mode shows "1048576 [1 MB]" but copies "1048576". You paste what you need.

**User variables**

Variables persist for the life of the process.

```
block = 4096
journal = 128 * mb
journal / block
vars            list all variables
del block       remove a variable
```

**Math functions**

sin, cos, tan, hypot, sqrt, abs, log, log2, log10, pow, round, floor, ceil, pi

**Base arithmetic**

```
0xFF + 1
0b1010 * 2
0xDEAD
1010 bin
```

**Autocorrect**

If you mistype a unit or function name, it finds the closest match via Levenshtein distance. Before asking "did you mean X?", it silently compiles the suggested fix. If the fix produces garbage math, it says nothing.

---

## Install

Requires Go 1.21+. Get it at https://go.dev/dl/

**Option 1: go install (recommended for Go users)**

This is the idiomatic Go way. It compiles the binary and drops it in `~/go/bin`, which Go adds to your PATH automatically.

```
go install github.com/Ekansh38/wrkr@latest
wrkr
```

If `wrkr` is not found after that, add this to your `~/.zshrc` or `~/.bashrc`:

```
export PATH="$HOME/go/bin:$PATH"
```

Then reload: `source ~/.zshrc`

**Option 2: build from source and install to /usr/local/bin**

This puts the binary somewhere already on your PATH system-wide. Any user on the machine can run it.

```
git clone git@github.com:Ekansh38/wrkr.git
cd wrkr
go build -o wrkr .
sudo mv wrkr /usr/local/bin/wrkr
wrkr
```

**Option 3: build from source and install to ~/bin**

No sudo required. Good if you do not have admin access or prefer keeping things in your home directory.

```
git clone git@github.com:Ekansh38/wrkr.git
cd wrkr
go build -o wrkr .
mkdir -p ~/bin
mv wrkr ~/bin/wrkr
```

Add this to your `~/.zshrc` or `~/.bashrc` if `~/bin` is not already on your PATH:

```
export PATH="$HOME/bin:$PATH"
```

Then reload: `source ~/.zshrc`

**Do not use an alias for this.** Aliases work but break in scripts and non-interactive shells. Putting the binary on your PATH is the right approach.

---

## Releases and distribution

Go compiles to a single static binary with no runtime dependencies. That binary runs on any machine with the same OS and architecture. You do not need Go installed on the target machine to run the binary.

To build for a different platform:

```
GOOS=linux GOARCH=amd64 go build -o wrkr-linux-amd64 .
GOOS=darwin GOARCH=arm64 go build -o wrkr-macos-arm64 .
GOOS=windows GOARCH=amd64 go build -o wrkr-windows-amd64.exe .
```

To create a GitHub release with pre-built binaries for all platforms, the standard Go tool is GoReleaser (goreleaser.com). You give it a config file and it cross-compiles, archives, checksums, and uploads everything to a GitHub release in one command. Most Go CLI projects use it. That is not set up here yet.

---

## Float precision

All values are IEEE 754 float64, which gives ~15-16 significant digits. Output is displayed at 12 decimal places to suppress representation noise. Without this cap, `1 mi to km` would show `1.60934400000000005` instead of `1.609344`. If you need more than 12 decimal places of precision, this tool is the wrong choice.

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
