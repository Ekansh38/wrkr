package engine

import (
	"regexp"
	"strings"
)

// UserVars stores user-defined variables for the lifetime of the process.
var UserVars = map[string]interface{}{}

var reAssign = regexp.MustCompile(`^\s*([a-zA-Z][a-zA-Z0-9_]*)\s*=\s*(.+)\s*$`)

// TryParseAssignment checks whether input is a variable assignment (e.g. "block = 4096").
// Returns (name, expression, true) if matched; ("", "", false) otherwise.
func TryParseAssignment(input string) (name, exprStr string, ok bool) {
	m := reAssign.FindStringSubmatch(input)
	if m == nil {
		return "", "", false
	}
	return strings.TrimSpace(m[1]), strings.TrimSpace(m[2]), true
}

// StoreVar saves a value into UserVars and also injects it into CalcEnv so
// subsequent expressions can reference it by name.
func StoreVar(name string, val float64) {
	UserVars[name] = val
	CalcEnv[name] = val
}

// DeleteVar removes a user-defined variable from both UserVars and CalcEnv.
// Returns false if the name was not a user variable.
func DeleteVar(name string) bool {
	if _, exists := UserVars[name]; !exists {
		return false
	}
	delete(UserVars, name)
	delete(CalcEnv, name)
	return true
}

// GetMergedEnv returns a fresh map combining CalcEnv and UserVars.
// Always call this immediately before Compile/Run so new variables are visible.
func GetMergedEnv() map[string]interface{} {
	env := make(map[string]interface{}, len(CalcEnv)+len(UserVars))
	for k, v := range CalcEnv {
		env[k] = v
	}
	for k, v := range UserVars {
		env[k] = v
	}
	return env
}
