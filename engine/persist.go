package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func varsFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".wrkr_vars.json"), nil
}

func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".wrkr_config.json"), nil
}

type wrkrConfig struct {
	Autoload bool `json:"autoload"`
}

// ReadAutoload returns true if the user has chosen "always load" previously.
func ReadAutoload() bool {
	path, err := configFilePath()
	if err != nil {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var cfg wrkrConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return false
	}
	return cfg.Autoload
}

// SetAutoload writes the autoload preference to ~/.wrkr_config.json.
func SetAutoload(enabled bool) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(wrkrConfig{Autoload: enabled}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// SavedVarsFile holds the contents of the on-disk vars file plus its path.
type SavedVarsFile struct {
	Path string
	Vars map[string]float64
}

// ReadSavedVars reads ~/.wrkr_vars.json from disk.
// Returns nil, nil if no file exists or the file is empty.
func ReadSavedVars() (*SavedVarsFile, error) {
	path, err := varsFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	vars := make(map[string]float64, len(raw))
	for k, v := range raw {
		if f, ok := v.(float64); ok {
			vars[k] = f
		}
	}
	if len(vars) == 0 {
		return nil, nil
	}
	return &SavedVarsFile{Path: path, Vars: vars}, nil
}

// ApplySavedVars injects a set of vars from disk into the running engine.
func ApplySavedVars(vars map[string]float64) {
	for k, v := range vars {
		StoreVar(k, v)
	}
}

// PersistVars writes the current UserVars to ~/.wrkr_vars.json.
// If UserVars is empty, the file is removed.
func PersistVars() error {
	path, err := varsFilePath()
	if err != nil {
		return err
	}
	if len(UserVars) == 0 {
		err = os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	m := make(map[string]float64, len(UserVars))
	for k, v := range UserVars {
		m[k] = v.(float64)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// DeletePersistedVars removes the vars file from disk.
func DeletePersistedVars() error {
	path, err := varsFilePath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
