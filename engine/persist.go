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
	Autoload   bool   `json:"autoload"`
	FormatMode string `json:"format_mode,omitempty"`
	TypeMode   string `json:"type_mode,omitempty"`
	Clipboard  *bool  `json:"clipboard,omitempty"` // nil = unset → default on
}

// ReadAppConfig reads ~/.wrkr_config.json and returns the parsed config.
// Returns a zero-value config on any error (safe to use as defaults).
func ReadAppConfig() wrkrConfig {
	path, err := configFilePath()
	if err != nil {
		return wrkrConfig{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return wrkrConfig{}
	}
	var cfg wrkrConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return wrkrConfig{}
	}
	return cfg
}

// SaveAppConfig writes cfg to ~/.wrkr_config.json.
func SaveAppConfig(cfg wrkrConfig) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ReadAutoload returns true if the user has chosen "always load" previously.
func ReadAutoload() bool {
	return ReadAppConfig().Autoload
}

// SetAutoload writes the autoload preference while preserving other config fields.
func SetAutoload(enabled bool) error {
	cfg := ReadAppConfig()
	cfg.Autoload = enabled
	return SaveAppConfig(cfg)
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
