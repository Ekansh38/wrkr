package engine

import (
	"regexp"
	"strings"

	"github.com/agnivade/levenshtein"
)

// GetValidTokens returns the names of all tokens currently known to CalcEnv.
// Called once at startup and again after new user variables are defined.
func GetValidTokens() []string {
	keys := make([]string, 0, len(CalcEnv))
	for k := range CalcEnv {
		keys = append(keys, k)
	}
	return keys
}

// FindClosestMatch uses Levenshtein distance to find the best-matching known token.
// Returns the input unchanged if no match is close enough.
func FindClosestMatch(input string, validOptions []string) string {
	bestMatch, minDist := "", -1
	for _, option := range validOptions {
		dist := levenshtein.ComputeDistance(strings.ToLower(input), strings.ToLower(option))
		if minDist == -1 || dist < minDist {
			minDist, bestMatch = dist, option
		}
	}
	if minDist > (len(input)/2)+1 {
		return input
	}
	return bestMatch
}

// SanitizeInput scans all word tokens in the expression, replacing unknown ones
// with their closest known match.  Returns the corrected string and whether
// any token was actually changed.
func SanitizeInput(raw string, validTokens []string) (string, bool) {
	re := regexp.MustCompile(`(?i)0x[0-9a-f]+|0b[01]+|0o[0-7]+|\b0[0-7]+\b|[0-9]*\.?[0-9]+(?:e[-+]?[0-9]+)?|[a-zA-Z][a-zA-Z0-9]*`)
	changed := false

	result := re.ReplaceAllStringFunc(raw, func(match string) string {
		lower := strings.ToLower(match)

		// Numeric literals: pass through, fixing legacy octal if needed.
		if (match[0] >= '0' && match[0] <= '9') || match[0] == '.' {
			if len(match) > 1 && match[0] == '0' && match[1] >= '0' && match[1] <= '7' &&
				!strings.ContainsAny(lower, "xbo.") {
				return "0o" + match[1:]
			}
			return match
		}

		// Structural keywords: never autocorrect.
		if lower == "to" || lower == "x" {
			return match
		}

		closest := FindClosestMatch(match, validTokens)
		if strings.ToLower(closest) != lower {
			changed = true
		}
		return closest
	})
	return result, changed
}
