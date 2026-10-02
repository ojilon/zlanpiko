// Package config loads and saves JSON configuration and resolves the
// academic data root. It owns the install pointer file and the data-local
// user preferences; academic records live in the database (see docs/04).
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// EnvDataRoot overrides the data root when set (see docs/12).
const EnvDataRoot = "ZLANPIKO_DATA"

// appDirName is the application directory under the OS user-config dir:
// %AppData%\Zlanpiko\app-config.json on Windows.
const appDirName = "Zlanpiko"

// Pointer is the install pointer file: it records where the academic data
// lives so the app never has to guess (see docs/12).
type Pointer struct {
	DataRoot   string `json:"data_root"`
	InstallDir string `json:"install_dir,omitempty"`
}

// Dir returns the directory holding the pointer file.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config: locate user config dir: %w", err)
	}
	return filepath.Join(base, appDirName), nil
}

// PointerPath returns the pointer file path.
func PointerPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "app-config.json"), nil
}

// LoadPointer reads and parses the pointer file.
func LoadPointer() (*Pointer, error) {
	path, err := PointerPath()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read pointer %s: %w", path, err)
	}
	var p Pointer
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("config: parse pointer %s: %w", path, err)
	}
	return &p, nil
}

// SavePointer writes the pointer file, creating its directory.
func SavePointer(p *Pointer) error {
	path, err := PointerPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("config: create config dir: %w", err)
	}
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("config: encode pointer: %w", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		return fmt.Errorf("config: write pointer %s: %w", path, err)
	}
	return nil
}

// ResolveDataRoot returns the data root without ever inventing one:
// explicit flag first, then the environment, then the pointer file.
// A missing configuration is an error with setup guidance, never a silent
// second database (see docs/08 FR-C3).
func ResolveDataRoot(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if env := os.Getenv(EnvDataRoot); env != "" {
		return env, nil
	}
	p, err := LoadPointer()
	if err != nil {
		return "", fmt.Errorf("config: no data root configured (pass an explicit path, set %s, or run setup): %w", EnvDataRoot, err)
	}
	if p.DataRoot == "" {
		return "", fmt.Errorf("config: pointer file has empty data_root")
	}
	return p.DataRoot, nil
}

// User holds data-local display preferences (<root>/config/user.json).
// It is separate from the install pointer so reinstalls keep preferences.
type User struct {
	Theme      string `json:"theme"`       // auto | light-16 | none
	LastScreen string `json:"last_screen"` // screen id, empty = dashboard
}

// DefaultUser returns the default preferences.
func DefaultUser() User {
	return User{Theme: "auto", LastScreen: ""}
}

// UserPath returns the user preferences path for a data root.
func UserPath(dataRoot string) string {
	return filepath.Join(dataRoot, "config", "user.json")
}

// LoadUser reads preferences, returning defaults when the file is absent.
// A malformed file is an error (never silently replaced).
func LoadUser(dataRoot string) (User, error) {
	raw, err := os.ReadFile(UserPath(dataRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultUser(), nil
		}
		return User{}, fmt.Errorf("config: read user prefs: %w", err)
	}
	var u User
	if err := json.Unmarshal(raw, &u); err != nil {
		return User{}, fmt.Errorf("config: parse user prefs: %w", err)
	}
	if u.Theme == "" {
		u.Theme = "auto"
	}
	return u, nil
}

// SaveUser writes preferences, creating the config directory.
func SaveUser(dataRoot string, u User) error {
	path := UserPath(dataRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("config: create data config dir: %w", err)
	}
	raw, err := json.MarshalIndent(u, "", "  ")
	if err != nil {
		return fmt.Errorf("config: encode user prefs: %w", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		return fmt.Errorf("config: write user prefs: %w", err)
	}
	return nil
}
