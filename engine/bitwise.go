package engine

import "strings"

// RewriteBitwiseOps translates bitwise operator syntax into function calls that
// expr-lang/expr can evaluate, since that evaluator does not natively support
// &, |, ^, ~, <<, >>.
//
// Operator precedence (low → high), matching standard C precedence:
//
//	|    bor     lowest
//	^    bxor
//	&    band
//	<< >> blshift / brshift
//	~    bnot    highest (unary prefix)
//
// Arithmetic (+, -, *, /) is handled by expr-lang/expr and is treated as
// opaque by this stage — it has higher precedence than all binary bitwise ops.
//
// Examples:
//
//	0xFF & 0x0F              → band(0xFF, 0x0F)
//	a | b ^ c                → bor(a, bxor(b, c))
//	a + b & c                → band(a + b, c)   (&  is lower than +)
//	1 << 3                   → blshift(1, 3)
//	~a & 0xFF                → band(bnot(a), 0xFF)
//	(a | b) & c              → band((bor(a, b)), c)
//	a && b   (logical AND)   → unchanged  (tokenised as "&&", not "&")
//
// The fast-path returns the input unchanged when no bitwise operators are found.
func RewriteBitwiseOps(input string) string {
	toks := bwTokenize(input)
	if !bwHasOp(toks) {
		return input
	}
	return bwExpr(toks)
}

// ── token types ──────────────────────────────────────────────────────────────

type bwKind int

const (
	bwNum    bwKind = iota // numeric literal: 123, 0xFF, 0b101, 0o17, 1.5, 1e3
	bwIdent                // identifier: sin, x, mb, _
	bwOp                   // operator: + - * / & | ^ ~ << >> && || ** == != >= <= %
	bwLParen               // (
	bwRParen               // )
	bwComma                // ,
	bwSpace                // whitespace run
	bwOther                // anything else (pass through)
)

type bwTok struct {
	kind bwKind
	val  string
}

// ── tokenizer ────────────────────────────────────────────────────────────────

func bwTokenize(s string) []bwTok {
	var out []bwTok
	i := 0
	for i < len(s) {
		c := s[i]

		// Whitespace
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			j := i
			for j < len(s) && (s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r') {
				j++
			}
			out = append(out, bwTok{bwSpace, s[i:j]})
			i = j
			continue
		}

		// Numeric literal: starts with digit, or '.' followed by digit
		if bwIsDigit(c) || (c == '.' && i+1 < len(s) && bwIsDigit(s[i+1])) {
			j := i
			lo := func(k int) byte {
				if k < len(s) {
					b := s[k]
					if b >= 'A' && b <= 'Z' {
						return b + 32
					}
					return b
				}
				return 0
			}
			switch {
			case c == '0' && lo(i+1) == 'x':
				j += 2
				for j < len(s) && bwIsHex(s[j]) {
					j++
				}
			case c == '0' && lo(i+1) == 'b':
				j += 2
				for j < len(s) && (s[j] == '0' || s[j] == '1') {
					j++
				}
			case c == '0' && lo(i+1) == 'o':
				j += 2
				for j < len(s) && s[j] >= '0' && s[j] <= '7' {
					j++
				}
			default:
				for j < len(s) && (bwIsDigit(s[j]) || s[j] == '.') {
					j++
				}
				if j < len(s) && (s[j] == 'e' || s[j] == 'E') {
					j++
					if j < len(s) && (s[j] == '+' || s[j] == '-') {
						j++
					}
					for j < len(s) && bwIsDigit(s[j]) {
						j++
					}
				}
			}
			out = append(out, bwTok{bwNum, s[i:j]})
			i = j
			continue
		}

		// Identifier
		if bwIsLetter(c) || c == '_' {
			j := i
			for j < len(s) && (bwIsLetter(s[j]) || bwIsDigit(s[j]) || s[j] == '_') {
				j++
			}
			out = append(out, bwTok{bwIdent, s[i:j]})
			i = j
			continue
		}

		// Two-character operators (must be checked before single-char)
		if i+1 < len(s) {
			two := s[i : i+2]
			switch two {
			case ">>", "<<", "&&", "||", "**", "==", "!=", ">=", "<=":
				out = append(out, bwTok{bwOp, two})
				i += 2
				continue
			}
		}

		// Single-character
		switch c {
		case '(':
			out = append(out, bwTok{bwLParen, "("})
		case ')':
			out = append(out, bwTok{bwRParen, ")"})
		case ',':
			out = append(out, bwTok{bwComma, ","})
		case '&', '|', '^', '~', '+', '-', '*', '/', '%', '!', '<', '>', '=', '?', ':':
			out = append(out, bwTok{bwOp, string(c)})
		default:
			out = append(out, bwTok{bwOther, string(c)})
		}
		i++
	}
	return out
}

func bwIsDigit(c byte) bool { return c >= '0' && c <= '9' }
func bwIsHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
func bwIsLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// bwHasOp returns true if tokens contain any bitwise operator that needs rewriting.
func bwHasOp(toks []bwTok) bool {
	for _, t := range toks {
		if t.kind == bwOp {
			switch t.val {
			case "&", "|", "^", "~", ">>", "<<":
				return true
			}
		}
	}
	return false
}

// bwStr reconstructs the string value of a token slice.
func bwStr(toks []bwTok) string {
	var b strings.Builder
	for _, t := range toks {
		b.WriteString(t.val)
	}
	return b.String()
}

// bwTrimSpaceToks strips leading and trailing whitespace tokens.
func bwTrimSpaceToks(toks []bwTok) []bwTok {
	i := 0
	for i < len(toks) && toks[i].kind == bwSpace {
		i++
	}
	j := len(toks)
	for j > i && toks[j-1].kind == bwSpace {
		j--
	}
	return toks[i:j]
}

// ── recursive descent rewriter ────────────────────────────────────────────────
//
// Each level splits the token slice at depth-0 occurrences of its operator,
// then recurses into each sub-slice for higher-precedence operators.
// The bottom level (rewriteBWNotAndParens) handles the unary ~ and
// recursively rewrites the contents of every parenthesised group.

// bwExpr is the top-level entry point (lowest precedence = |).
func bwExpr(toks []bwTok) string { return bwRewriteOr(toks) }

func bwRewriteOr(toks []bwTok) string {
	parts := bwSplitAt(toks, "|")
	if len(parts) == 1 {
		return bwRewriteXor(toks)
	}
	res := bwRewriteXor(bwTrimSpaceToks(parts[0]))
	for _, p := range parts[1:] {
		res = "bor(" + res + ", " + bwRewriteXor(bwTrimSpaceToks(p)) + ")"
	}
	return res
}

func bwRewriteXor(toks []bwTok) string {
	parts := bwSplitAt(toks, "^")
	if len(parts) == 1 {
		return bwRewriteAnd(toks)
	}
	res := bwRewriteAnd(bwTrimSpaceToks(parts[0]))
	for _, p := range parts[1:] {
		res = "bxor(" + res + ", " + bwRewriteAnd(bwTrimSpaceToks(p)) + ")"
	}
	return res
}

func bwRewriteAnd(toks []bwTok) string {
	parts := bwSplitAt(toks, "&")
	if len(parts) == 1 {
		return bwRewriteShift(toks)
	}
	res := bwRewriteShift(bwTrimSpaceToks(parts[0]))
	for _, p := range parts[1:] {
		res = "band(" + res + ", " + bwRewriteShift(bwTrimSpaceToks(p)) + ")"
	}
	return res
}

func bwRewriteShift(toks []bwTok) string {
	// Collect depth-0 << and >> operators, preserving order (left-associative).
	var ops []string
	var parts [][]bwTok
	depth, start := 0, 0
	for i, t := range toks {
		if t.kind == bwLParen {
			depth++
		}
		if t.kind == bwRParen {
			depth--
		}
		if depth == 0 && t.kind == bwOp && (t.val == ">>" || t.val == "<<") {
			ops = append(ops, t.val)
			parts = append(parts, toks[start:i])
			start = i + 1
		}
	}
	parts = append(parts, toks[start:])

	if len(ops) == 0 {
		return bwRewriteNotAndParens(toks)
	}
	res := bwRewriteNotAndParens(bwTrimSpaceToks(parts[0]))
	for i, op := range ops {
		fn := "brshift"
		if op == "<<" {
			fn = "blshift"
		}
		res = fn + "(" + res + ", " + bwRewriteNotAndParens(bwTrimSpaceToks(parts[i+1])) + ")"
	}
	return res
}

// bwRewriteNotAndParens handles two jobs at the leaf level:
//  1. Unary ~ operator: wrap the immediately following atom with bnot().
//  2. Parenthesised groups: recursively rewrite their contents so that
//     bitwise operators inside grouping parens are processed correctly.
//     Function-call parens are handled per-argument to avoid misreading
//     commas as expression boundaries.
func bwRewriteNotAndParens(toks []bwTok) string {
	var b strings.Builder
	i := 0
	for i < len(toks) {
		t := toks[i]
		switch {
		case t.kind == bwOp && t.val == "~":
			// Unary NOT: consume the next atom and wrap with bnot().
			i++
			atom, n := bwConsumeAtom(toks, i)
			b.WriteString("bnot(")
			b.WriteString(atom)
			b.WriteString(")")
			i += n

		case t.kind == bwLParen:
			// Determine whether this ( is a function-call paren or a grouping paren.
			// A function-call paren is one whose immediately preceding non-space token
			// is an identifier.
			prevIsIdent := false
			for k := i - 1; k >= 0; k-- {
				if toks[k].kind == bwSpace {
					continue
				}
				prevIsIdent = toks[k].kind == bwIdent
				break
			}

			end := bwMatchParen(toks, i)
			inner := toks[i+1 : end]

			if prevIsIdent {
				// Function call: rewrite each comma-separated argument independently.
				args := bwSplitCommas(inner)
				b.WriteString("(")
				for j, arg := range args {
					if j > 0 {
						b.WriteString(", ")
					}
					b.WriteString(bwExpr(arg))
				}
				b.WriteString(")")
			} else {
				// Grouping parens: rewrite the whole inner expression.
				b.WriteString("(")
				b.WriteString(bwExpr(inner))
				b.WriteString(")")
			}
			i = end + 1

		default:
			b.WriteString(t.val)
			i++
		}
	}
	return b.String()
}

// bwConsumeAtom consumes the single "atom" that a ~ operator applies to,
// starting at toks[start].  Returns the rewritten atom string and the number
// of tokens consumed.
//
// An atom is one of:
//   - Another ~: recurse (e.g. ~~x → bnot(bnot(x)))
//   - A parenthesised group (recursively rewritten)
//   - An identifier optionally followed by (...) — function call
//   - A numeric literal
func bwConsumeAtom(toks []bwTok, start int) (string, int) {
	origStart := start // remember position before whitespace skip
	// Skip leading whitespace.
	for start < len(toks) && toks[start].kind == bwSpace {
		start++
	}
	if start >= len(toks) {
		return "0", start - origStart
	}
	skip := start - origStart // whitespace tokens to add to every returned count
	t := toks[start]

	// Recursive ~
	if t.kind == bwOp && t.val == "~" {
		inner, n := bwConsumeAtom(toks, start+1)
		return "bnot(" + inner + ")", skip + n + 1
	}

	// Parenthesised group
	if t.kind == bwLParen {
		end := bwMatchParen(toks, start)
		inner := toks[start+1 : end]
		return "(" + bwExpr(inner) + ")", skip + end - start + 1
	}

	// Identifier — check if it's a function call (ident immediately followed by '(')
	if t.kind == bwIdent {
		j := start + 1
		for j < len(toks) && toks[j].kind == bwSpace {
			j++
		}
		if j < len(toks) && toks[j].kind == bwLParen {
			end := bwMatchParen(toks, j)
			inner := toks[j+1 : end]
			args := bwSplitCommas(inner)
			var argStrs []string
			for _, arg := range args {
				argStrs = append(argStrs, bwExpr(arg))
			}
			fnCall := t.val + "(" + strings.Join(argStrs, ", ") + ")"
			return fnCall, skip + end - start + 1
		}
		return t.val, skip + 1
	}

	// Numeric literal or anything else
	return t.val, skip + 1
}

// ── utilities ─────────────────────────────────────────────────────────────────

// bwSplitAt splits toks at depth-0 occurrences of a single-character operator op.
// Does not split at doubled versions (e.g. "&" will not match "&&").
func bwSplitAt(toks []bwTok, op string) [][]bwTok {
	var parts [][]bwTok
	depth, start := 0, 0
	for i, t := range toks {
		if t.kind == bwLParen {
			depth++
		}
		if t.kind == bwRParen {
			depth--
		}
		if depth == 0 && t.kind == bwOp && t.val == op {
			parts = append(parts, toks[start:i])
			start = i + 1
		}
	}
	return append(parts, toks[start:])
}

// bwSplitCommas splits a token slice at depth-0 commas (for function argument lists).
func bwSplitCommas(toks []bwTok) [][]bwTok {
	var parts [][]bwTok
	depth, start := 0, 0
	for i, t := range toks {
		if t.kind == bwLParen {
			depth++
		}
		if t.kind == bwRParen {
			depth--
		}
		if depth == 0 && t.kind == bwComma {
			parts = append(parts, toks[start:i])
			start = i + 1
		}
	}
	return append(parts, toks[start:])
}

// bwMatchParen finds the index of the ')' that closes the '(' at position start.
// Returns len(toks)-1 on mismatched input (should not occur on valid expressions).
func bwMatchParen(toks []bwTok, start int) int {
	depth := 1
	for i := start + 1; i < len(toks); i++ {
		if toks[i].kind == bwLParen {
			depth++
		}
		if toks[i].kind == bwRParen {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return len(toks) - 1
}
